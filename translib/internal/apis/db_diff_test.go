////////////////////////////////////////////////////////////////////////////////
//                                                                            //
//  Copyright 2021 Broadcom. The term Broadcom refers to Broadcom Inc. and/or //
//  its subsidiaries.                                                         //
//                                                                            //
//  Licensed under the Apache License, Version 2.0 (the "License");           //
//  you may not use this file except in compliance with the License.          //
//  You may obtain a copy of the License at                                   //
//                                                                            //
//     http://www.apache.org/licenses/LICENSE-2.0                             //
//                                                                            //
//  Unless required by applicable law or agreed to in writing, software       //
//  distributed under the License is distributed on an "AS IS" BASIS,         //
//  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.  //
//  See the License for the specific language governing permissions and       //
//  limitations under the License.                                            //
//                                                                            //
////////////////////////////////////////////////////////////////////////////////

package apis

import (
	"reflect"
	"sort"
	"testing"

	"github.com/Azure/sonic-mgmt-common/translib/db"
)

func TestDiff(t *testing.T) {
	var testCases = []diffTestCase{
		{
			Name: "empty",
			Old:  map[string]string{},
			New:  map[string]string{},
			Diff: EntryDiff{},
		}, {
			Name: "equal",
			Old:  map[string]string{"one": "1111", "two": "2222"},
			New:  map[string]string{"one": "1111", "two": "2222"},
			Diff: EntryDiff{},
		}, {
			Name: "equal_arr",
			Old:  map[string]string{"one": "1111", "two": "2222", "arr@": "1,2,3"},
			New:  map[string]string{"one": "1111", "two": "2222", "arr@": "1,2,3"},
			Diff: EntryDiff{},
		}, {
			Name: "equal_null",
			Old:  map[string]string{"NULL": "NULL"},
			New:  map[string]string{"NULL": "NULL"},
			Diff: EntryDiff{},
		}, {
			Name: "create_entry",
			Old:  map[string]string{},
			New:  map[string]string{"one": "1111", "two": "2222", "arr@": "1,2,3"},
			Diff: EntryDiff{EntryCreated: true},
		}, {
			Name: "create_null_entry",
			Old:  map[string]string{},
			New:  map[string]string{"NULL": "NULL"},
			Diff: EntryDiff{EntryCreated: true},
		}, {
			Name: "delete_entry",
			Old:  map[string]string{"one": "1111", "two": "2222", "arr@": "1,2,3"},
			New:  map[string]string{},
			Diff: EntryDiff{EntryDeleted: true},
		}, {
			Name: "add_field",
			Old:  map[string]string{"one": "1111"},
			New:  map[string]string{"one": "1111", "two": "2222"},
			Diff: EntryDiff{CreatedFields: []string{"two"}},
		}, {
			Name: "add_fields",
			Old:  map[string]string{"one": "1111"},
			New:  map[string]string{"one": "1111", "two": "2222", "arr@": "1,2,3", "foo": "bar"},
			Diff: EntryDiff{CreatedFields: []string{"two", "arr", "foo"}},
		}, {
			Name: "add_null",
			Old:  map[string]string{"one": "1111"},
			New:  map[string]string{"one": "1111", "NULL": "NULL"},
			Diff: EntryDiff{},
		}, {
			Name: "del_fields",
			Old:  map[string]string{"one": "1111", "two": "2222", "foo": "bar"},
			New:  map[string]string{"one": "1111"},
			Diff: EntryDiff{DeletedFields: []string{"two", "foo"}},
		}, {
			Name: "del_arr_fields",
			Old:  map[string]string{"one": "1111", "arr@": "1,2", "foo@": "bar"},
			New:  map[string]string{"foo@": "bar"},
			Diff: EntryDiff{DeletedFields: []string{"one", "arr"}},
		}, {
			Name: "del_null",
			Old:  map[string]string{"one": "1111", "NULL": "NULL"},
			New:  map[string]string{"one": "1111"},
			Diff: EntryDiff{},
		}, {
			Name: "mod_fields",
			Old:  map[string]string{"one": "1111", "two": "2222"},
			New:  map[string]string{"one": "0001", "two": "2222"},
			Diff: EntryDiff{UpdatedFields: []string{"one"}},
		}, {
			Name: "mod_arr_fields",
			Old:  map[string]string{"one": "1111", "foo@": "2222"},
			New:  map[string]string{"one": "0001", "foo@": "1,2,3"},
			Diff: EntryDiff{UpdatedFields: []string{"one", "foo"}},
		}, {
			Name: "cru_fields",
			Old:  map[string]string{"one": "1111", "foo@": "2222", "NULL": "NULL"},
			New:  map[string]string{"one": "0001", "two": "2222"},
			Diff: EntryDiff{
				CreatedFields: []string{"two"},
				UpdatedFields: []string{"one"},
				DeletedFields: []string{"foo"},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(tt *testing.T) { tc.run(tt) })
	}
}

type diffTestCase struct {
	Name string
	Old  map[string]string
	New  map[string]string
	Diff EntryDiff
}

func (tc *diffTestCase) run(t *testing.T) {
	t.Helper()
	old, new, exp := tc.Old, tc.New, tc.Diff
	d := EntryCompare(db.Value{Field: old}, db.Value{Field: new})
	ok := reflect.DeepEqual(d.OldValue.Field, old) &&
		reflect.DeepEqual(d.NewValue.Field, new) &&
		d.EntryCreated == exp.EntryCreated &&
		d.EntryDeleted == exp.EntryDeleted &&
		arrayEquals(d.CreatedFields, exp.CreatedFields) &&
		arrayEquals(d.UpdatedFields, exp.UpdatedFields) &&
		arrayEquals(d.DeletedFields, exp.DeletedFields)
	if !ok {
		t.Errorf("Old values  = %v", old)
		t.Errorf("New values  = %v", new)
		t.Errorf("Expect diff = %v", exp.String())
		t.Errorf("Actual diff = %v", d.String())
	}
}

func arrayEquals(a1, a2 []string) bool {
	sort.Strings(a1)
	sort.Strings(a2)
	return reflect.DeepEqual(a1, a2)
}
