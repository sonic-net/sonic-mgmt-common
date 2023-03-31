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
	"fmt"
	"strings"

	"github.com/Azure/sonic-mgmt-common/translib/db"
)

// EntryDiff holds diff of two versions of a single db entry.
// It contains both old & new db.Value objects and list changed field names.
// Dummy "NULL" fields are ignored; array field names will have "@" suffix.
type EntryDiff struct {
	OldValue      db.Value // value before change; empty during entry create
	NewValue      db.Value // changed db value; empty during entry delete
	EntryCreated  bool     // true if entry being created
	EntryDeleted  bool     // true if entry being deleted
	CreatedFields []string // fields added during entry update
	UpdatedFields []string // fields modified during entry update
	DeletedFields []string // fields deleted during entry update
}

func (d *EntryDiff) String() string {
	return fmt.Sprintf(
		"{EntryCreated=%t, EntryDeleted=%t, CreatedFields=%v, UpdatedFields=%v, DeletedFields=%v}",
		d.EntryCreated, d.EntryDeleted, d.CreatedFields, d.UpdatedFields, d.DeletedFields)
}

// IsEmpty returns true if this EntryDiff has no diff data -- either not initialized
// or both old and new values are identical.
func (d *EntryDiff) IsEmpty() bool {
	return !d.EntryCreated && !d.EntryDeleted &&
		len(d.CreatedFields) == 0 && len(d.UpdatedFields) == 0 && len(d.DeletedFields) == 0
}

// EntryCompare function compares two db.Value objects representing two versions
// of a single db entry. Changes are returned as a DBEntryDiff pointer.
func EntryCompare(old, new db.Value) *EntryDiff {
	diff := &EntryDiff{
		OldValue: old,
		NewValue: new,
	}

	if old.IsPopulated() {
		if !new.IsPopulated() {
			diff.EntryDeleted = true
			return diff
		}
	} else {
		if new.IsPopulated() {
			diff.EntryCreated = true
		}
		return diff
	}

	for fldName := range old.Field {
		if fldName == "NULL" {
			continue
		}
		if _, fldOk := new.Field[fldName]; !fldOk {
			diff.DeletedFields = append(
				diff.DeletedFields, strings.TrimSuffix(fldName, "@"))
		}
	}

	for nf, nv := range new.Field {
		if nf == "NULL" {
			continue
		}
		if ov, exists := old.Field[nf]; !exists {
			diff.CreatedFields = append(
				diff.CreatedFields, strings.TrimSuffix(nf, "@"))
		} else if ov != nv {
			diff.UpdatedFields = append(
				diff.UpdatedFields, strings.TrimSuffix(nf, "@"))
		}
	}

	return diff
}

// EntryFields returns the list of field names in a DB entry.
// Ignores the dummy NULL field and also removes @ suffix of array fields.
func EntryFields(v db.Value) []string {
	var fields []string
	for f := range v.Field {
		if f != "NULL" {
			fields = append(fields, strings.TrimSuffix(f, "@"))
		}
	}
	return fields
}
