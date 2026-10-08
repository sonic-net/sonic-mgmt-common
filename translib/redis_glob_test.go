package translib

import (
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/Azure/sonic-mgmt-common/translib/db"
	"github.com/Azure/sonic-mgmt-common/translib/tlerr"
)

func TestEscapeRedisGlob(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"plain", "plain"},
		{`slash\star*question?brackets[]`, `slash\\star\*question\?brackets\[\]`},
	}

	for _, tc := range tests {
		if got := escapeRedisGlob(tc.input); got != tc.want {
			t.Errorf("escapeRedisGlob(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestEscapedRedisGlobCascadeDeletesOnlyLiteralParent(t *testing.T) {
	d, err := db.NewDB(db.Options{
		DBNo:                    db.ConfigDB,
		TableNameSeparator:      "|",
		KeySeparator:            "|",
		DisableCVLCheck:         true,
		ForceNewRedisConnection: true,
	})
	if err != nil {
		t.Fatalf("NewDB() failed: %v", err)
	}
	defer d.DeleteDB()

	tests := []struct {
		name    string
		parent  string
		sibling string
	}{
		{"star", `literal*`, `literalX`},
		{"question", `literal?`, `literalY`},
		{"brackets", `literal[ab]`, `literala`},
		{"backslash", `literal\name`, `literalname`},
	}
	value := db.Value{Field: map[string]string{"field": "value"}}

	for i, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts := &db.TableSpec{Name: fmt.Sprintf("TEST_GLOB_CASCADE_%d_%d", os.Getpid(), i)}
			intended := db.NewKey(tc.parent, "RULE_1")
			sibling := db.NewKey(tc.sibling, "RULE_1")
			_ = d.DeleteTable(ts)
			t.Cleanup(func() { _ = d.DeleteTable(ts) })

			if err := d.SetEntry(ts, *intended, value); err != nil {
				t.Fatalf("SetEntry(%q) failed: %v", tc.parent, err)
			}
			if err := d.SetEntry(ts, *sibling, value); err != nil {
				t.Fatalf("SetEntry(%q) failed: %v", tc.sibling, err)
			}

			pattern := db.NewKey(escapeRedisGlob(tc.parent) + "|*")
			if err := d.DeleteKeys(ts, *pattern); err != nil {
				t.Fatalf("DeleteKeys(%q) failed: %v", pattern, err)
			}

			_, err := d.GetEntry(ts, *intended)
			var notFound tlerr.TranslibRedisClientEntryNotExist
			if !errors.As(err, &notFound) {
				t.Fatalf("literal parent %q still exists; err=%v", tc.parent, err)
			}
			if _, err := d.GetEntry(ts, *sibling); err != nil {
				t.Fatalf("sibling %q was removed with literal parent %q: %v", tc.sibling, tc.parent, err)
			}
		})
	}
}
