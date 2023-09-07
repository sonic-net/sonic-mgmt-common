////////////////////////////////////////////////////////////////////////////////
//                                                                            //
//  Copyright 2023 Broadcom. The term Broadcom refers to Broadcom Inc. and/or //
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

package db

import (
	"testing"
)

func verifyValueContainsAll(t *testing.T, v1, v2 Value, exp bool) {
	if v1.ContainsAll(&v2) != exp {
		t.Errorf("v1.ContainsAll(v2) != %v\nv1 = %v\nv2 = %v", exp, v1.Field, v2.Field)
	}
}

func verifyValueEquals(t *testing.T, v1, v2 Value, exp bool) {
	if v1.Equals(&v2) != exp {
		t.Errorf("v1.Equals(v2) != %v\nv1 = %v\nv2 = %v", exp, v1.Field, v2.Field)
	}
}

func TestValueContainsAll(t *testing.T) {
	testContainsAll := func(v1, v2 Value, exp bool) func(*testing.T) {
		return func(tt *testing.T) { verifyValueContainsAll(tt, v1, v2, exp) }
	}

	/* both equal cases */
	t.Run("nil-nil", testContainsAll(Value{}, Value{}, true))
	t.Run("empty-empty", testContainsAll(
		Value{Field: map[string]string{}},
		Value{Field: map[string]string{}},
		true,
	))
	t.Run("xyz-xyz", testContainsAll(
		Value{Field: map[string]string{"one": "1", "two": "2"}},
		Value{Field: map[string]string{"one": "1", "two": "2"}},
		true,
	))

	/* contains more */
	t.Run("xyz-empty", testContainsAll(
		Value{Field: map[string]string{"one": "1", "two": "2"}},
		Value{Field: map[string]string{}},
		true,
	))
	t.Run("xyz-xy", testContainsAll(
		Value{Field: map[string]string{"one": "1", "two": "2"}},
		Value{Field: map[string]string{"one": "1"}},
		true,
	))

	/* not contains cases */
	t.Run("empty-xyz", testContainsAll(
		Value{Field: map[string]string{}},
		Value{Field: map[string]string{"one": "1", "two": "2"}},
		false,
	))
	t.Run("xyz-XYZ", testContainsAll(
		Value{Field: map[string]string{"one": "1", "two": "2"}},
		Value{Field: map[string]string{"one": "1", "two": "002"}},
		false,
	))
	t.Run("xyz-abc", testContainsAll(
		Value{Field: map[string]string{"one": "1", "two": "2"}},
		Value{Field: map[string]string{"one": "1", "hello": "world"}},
		false,
	))
	t.Run("xyz-xyzL", testContainsAll(
		Value{Field: map[string]string{"one": "1", "two": "2"}},
		Value{Field: map[string]string{"one": "1", "two": "2", "L@": "foo"}},
		false,
	))

	/* leaf-list cases */
	t.Run("list,empty", testContainsAll(
		Value{Field: map[string]string{"L@": "", "one": "1", "two": "002"}},
		Value{Field: map[string]string{"L@": ""}},
		true,
	))
	t.Run("list,1inst", testContainsAll(
		Value{Field: map[string]string{"L@": "foo", "one": "1"}},
		Value{Field: map[string]string{"L@": "foo", "one": "1"}},
		true,
	))
	t.Run("list,equal", testContainsAll(
		Value{Field: map[string]string{"L@": "foo,bar", "one": "1"}},
		Value{Field: map[string]string{"L@": "foo,bar"}},
		true,
	))
	t.Run("list,out_of_order", testContainsAll(
		Value{Field: map[string]string{"L@": "foo,bar,01,002,0003,00004"}},
		Value{Field: map[string]string{"L@": "0003,bar,01,00004,foo,002"}},
		true,
	))

	/* leaf-list mismatch cases */
	t.Run("list,more_inst", testContainsAll(
		Value{Field: map[string]string{"L@": "foo,bar"}},
		Value{Field: map[string]string{"L@": "foo+bar"}},
		false,
	))
	t.Run("list,less_inst", testContainsAll(
		Value{Field: map[string]string{"L@": "foo+bar"}},
		Value{Field: map[string]string{"L@": "foo,bar"}},
		false,
	))
	t.Run("list,diff", testContainsAll(
		Value{Field: map[string]string{"L@": "foo,bar,001"}},
		Value{Field: map[string]string{"L@": "foo,bar,002"}},
		false,
	))
	t.Run("list,diff_len", testContainsAll(
		Value{Field: map[string]string{"L@": "foo,bar"}},
		Value{Field: map[string]string{"L@": "hello,world"}},
		false,
	))

}

func TestValueEquals(t *testing.T) {
	testEquals := func(v1, v2 Value, exp bool) func(*testing.T) {
		return func(tt *testing.T) { verifyValueEquals(tt, v1, v2, exp) }
	}

	t.Run("nil-nil", testEquals(Value{}, Value{}, true))
	t.Run("empty-empty", testEquals(
		Value{Field: map[string]string{}},
		Value{Field: map[string]string{}},
		true,
	))
	t.Run("xyz-xyz", testEquals(
		Value{Field: map[string]string{"one": "1", "two": "2"}},
		Value{Field: map[string]string{"one": "1", "two": "2"}},
		true,
	))

	t.Run("list,empty", testEquals(
		Value{Field: map[string]string{"L@": ""}},
		Value{Field: map[string]string{"L@": ""}},
		true,
	))
	t.Run("list,1inst", testEquals(
		Value{Field: map[string]string{"L@": "foo", "one": "1"}},
		Value{Field: map[string]string{"L@": "foo", "one": "1"}},
		true,
	))
	t.Run("list,equal", testEquals(
		Value{Field: map[string]string{"L@": "foo,bar"}},
		Value{Field: map[string]string{"L@": "foo,bar"}},
		true,
	))
	t.Run("list,out_of_order", testEquals(
		Value{Field: map[string]string{"L@": "foo,bar,01,002,0003,00004"}},
		Value{Field: map[string]string{"L@": "0003,bar,01,00004,foo,002"}},
		true,
	))

	t.Run("empty-xyz", testEquals(
		Value{Field: map[string]string{}},
		Value{Field: map[string]string{"one": "1", "two": "2"}},
		false,
	))
	t.Run("xyz-empty", testEquals(
		Value{Field: map[string]string{"one": "1", "two": "2"}},
		Value{Field: map[string]string{}},
		false,
	))
	t.Run("xyz-XYZ", testEquals(
		Value{Field: map[string]string{"one": "1", "two": "2"}},
		Value{Field: map[string]string{"one": "01", "two": "02"}},
		false,
	))
	t.Run("list,diff", testEquals(
		Value{Field: map[string]string{"L@": "foo,bar,001"}},
		Value{Field: map[string]string{"L@": "foo,bar,002"}},
		false,
	))

}
