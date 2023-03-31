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

func TestDiff_empty(t *testing.T) {
	old := map[string]string{}
	new := map[string]string{}
	entryCompare(t, old, new, EntryDiff{})
}

func TestDiff_equal(t *testing.T) {
	old := map[string]string{"one": "1111", "two": "2222"}
	new := map[string]string{"one": "1111", "two": "2222"}
	entryCompare(t, old, new, EntryDiff{})
}

func TestDiff_equal_arr(t *testing.T) {
	old := map[string]string{"one": "1111", "two": "2222", "arr@": "1,2,3"}
	new := map[string]string{"one": "1111", "two": "2222", "arr@": "1,2,3"}
	entryCompare(t, old, new, EntryDiff{})
}

func TestDiff_equal_null(t *testing.T) {
	old := map[string]string{"NULL": "NULL"}
	new := map[string]string{"NULL": "NULL"}
	entryCompare(t, old, new, EntryDiff{})
}

func TestDiff_create_entry(t *testing.T) {
	old := map[string]string{}
	new := map[string]string{"one": "1111", "two": "2222", "arr@": "1,2,3"}
	entryCompare(t, old, new, EntryDiff{EntryCreated: true})
}

func TestDiff_create_null_entry(t *testing.T) {
	old := map[string]string{}
	new := map[string]string{"NULL": "NULL"}
	entryCompare(t, old, new, EntryDiff{EntryCreated: true})
}

func TestDiff_delete_entry(t *testing.T) {
	old := map[string]string{"one": "1111", "two": "2222", "arr@": "1,2,3"}
	new := map[string]string{}
	entryCompare(t, old, new, EntryDiff{EntryDeleted: true})
}

func TestDiff_add_field(t *testing.T) {
	old := map[string]string{"one": "1111"}
	new := map[string]string{"one": "1111", "two": "2222"}
	entryCompare(t, old, new, EntryDiff{CreatedFields: []string{"two"}})
}

func TestDiff_add_fields(t *testing.T) {
	old := map[string]string{"one": "1111"}
	new := map[string]string{"one": "1111", "two": "2222", "arr@": "1,2,3", "foo": "bar"}
	entryCompare(t, old, new, EntryDiff{CreatedFields: []string{"two", "arr", "foo"}})
}

func TestDiff_add_null(t *testing.T) {
	old := map[string]string{"one": "1111"}
	new := map[string]string{"one": "1111", "NULL": "NULL"}
	entryCompare(t, old, new, EntryDiff{})
}

func TestDiff_del_fields(t *testing.T) {
	old := map[string]string{"one": "1111", "two": "2222", "foo": "bar"}
	new := map[string]string{"one": "1111"}
	entryCompare(t, old, new, EntryDiff{DeletedFields: []string{"two", "foo"}})
}

func TestDiff_del_arr_fields(t *testing.T) {
	old := map[string]string{"one": "1111", "arr@": "1,2", "foo@": "bar"}
	new := map[string]string{"foo@": "bar"}
	entryCompare(t, old, new, EntryDiff{DeletedFields: []string{"one", "arr"}})
}

func TestDiff_del_null(t *testing.T) {
	old := map[string]string{"one": "1111", "NULL": "NULL"}
	new := map[string]string{"one": "1111"}
	entryCompare(t, old, new, EntryDiff{})
}

func TestDiff_mod_fields(t *testing.T) {
	old := map[string]string{"one": "1111", "two": "2222"}
	new := map[string]string{"one": "0001", "two": "2222"}
	entryCompare(t, old, new, EntryDiff{UpdatedFields: []string{"one"}})
}

func TestDiff_mod_arr_fields(t *testing.T) {
	old := map[string]string{"one": "1111", "foo@": "2222"}
	new := map[string]string{"one": "0001", "foo@": "1,2,3"}
	entryCompare(t, old, new, EntryDiff{UpdatedFields: []string{"one", "foo"}})
}

func TestDiff_cru_fields(t *testing.T) {
	old := map[string]string{"one": "1111", "foo@": "2222", "NULL": "NULL"}
	new := map[string]string{"one": "0001", "two": "2222"}
	entryCompare(t, old, new, EntryDiff{
		CreatedFields: []string{"two"},
		UpdatedFields: []string{"one"},
		DeletedFields: []string{"foo"},
	})
}

func entryCompare(t *testing.T, old, new map[string]string, exp EntryDiff) {
	t.Logf("EntryCompare(old=%v, new=%v)", old, new)
	d := EntryCompare(db.Value{Field: old}, db.Value{Field: new})
	ok := reflect.DeepEqual(d.OldValue.Field, old) &&
		reflect.DeepEqual(d.NewValue.Field, new) &&
		d.EntryCreated == exp.EntryCreated &&
		d.EntryDeleted == exp.EntryDeleted &&
		arrayEquals(d.CreatedFields, exp.CreatedFields) &&
		arrayEquals(d.UpdatedFields, exp.UpdatedFields) &&
		arrayEquals(d.DeletedFields, exp.DeletedFields)
	if !ok {
		t.Errorf("Expected=%v", exp.String())
		t.Errorf("Found=%v", d.String())
	}
}

func arrayEquals(a1, a2 []string) bool {
	sort.Strings(a1)
	sort.Strings(a2)
	return reflect.DeepEqual(a1, a2)
}
