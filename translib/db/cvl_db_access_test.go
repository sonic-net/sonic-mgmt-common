package db

import (
	"os"
	"strconv"
	"testing"
)

func TestCvlDBAccessExistsUsesLiteralKey(t *testing.T) {
	d := newTestDB(t, Options{
		DBNo:                    ConfigDB,
		DisableCVLCheck:         true,
		ForceNewRedisConnection: true,
		TableNameSeparator:      "|",
		KeySeparator:            "|",
	})
	table := "CVL_EXISTS_LITERAL_" + strconv.Itoa(os.Getpid())
	value := map[string]interface{}{"field": "value"}
	data := make(map[string]map[string]interface{})

	existing := []string{`literal*`, `literal?`, `literal[ab]`, `literal\name`}
	missing := []struct {
		key     string
		sibling string
	}{
		{`absent*`, `absentX`},
		{`absent?`, `absentY`},
		{`absent[ab]`, `absenta`},
		{`absent\name`, `absentname`},
	}

	for _, key := range existing {
		data[table+"|"+key] = value
	}
	for _, tc := range missing {
		data[table+"|"+tc.sibling] = value
	}
	setupTestData(t, d.client, data)

	access := &cvlDBAccess{Db: d}
	for _, key := range existing {
		t.Run("existing/"+key, func(t *testing.T) {
			got, err := access.Exists(table + "|" + key).Result()
			if err != nil || got != 1 {
				t.Fatalf("Exists(%q) = (%d, %v), want (1, nil)", key, got, err)
			}
		})
	}
	for _, tc := range missing {
		t.Run("missing/"+tc.key, func(t *testing.T) {
			got, err := access.Exists(table + "|" + tc.key).Result()
			if err != nil || got != 0 {
				t.Fatalf("Exists(%q) = (%d, %v), want (0, nil)", tc.key, got, err)
			}
		})
	}
}
