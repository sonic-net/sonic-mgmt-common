////////////////////////////////////////////////////////////////////////////////
//                                                                            //
//  Copyright 2022 Broadcom. The term Broadcom refers to Broadcom Inc. and/or //
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

package ocbinds

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/openconfig/gnmi/proto/gnmi"
	"github.com/openconfig/ygot/ygot"
	"github.com/openconfig/ygot/ytypes"
)

var (
	allTests []testCase
	tempDir  string
)

func TestMain(m *testing.M) {
	testdata := flag.String("testdata", "", "Testdata file patterns to load")
	flag.Parse()
	initTestData(*testdata)
	os.Exit(m.Run())
}

func TestNewEmitJSON(t *testing.T) {
	for _, test := range allTests {
		t.Run(test.name, test.verifyEmitJSON)
	}
}

func TestNewEmitJSON_noPrefix(t *testing.T) {
	opts := EmitJSONOptions{NoPrefix: true}
	for _, test := range allTests {
		test.opts = opts
		t.Run(test.name, test.verifyEmitJSON)
	}
}

func TestNewEmitJSON_sorted(t *testing.T) {
	opts := EmitJSONOptions{SortList: true}
	for _, test := range allTests {
		test.opts = opts
		t.Run(test.name, test.verifyEmitJSON)
	}
}

func BenchmarkNewEmitJSON(b *testing.B) {
	for _, test := range allTests {
		b.Run(test.name, test.benchmarkEmitJSON)
	}
}

func BenchmarkNewEmitJSON_natsort(b *testing.B) {
	for _, test := range allTests {
		test.opts.SortList = true
		b.Run(test.name, test.benchmarkEmitJSON)
	}
}

func BenchmarkYgotEmitJSON(b *testing.B) {
	for _, test := range allTests {
		b.Run(test.name, test.benchmarkYgotEmitJSON)
	}
}

type testCase struct {
	name string
	yObj ygot.ValidatedGoStruct
	jStr string // expected json string
	opts EmitJSONOptions
}

func (tc *testCase) verifyEmitJSON(t *testing.T) {
	data, err := EmitJSON(tc.yObj, &tc.opts)
	if err != nil {
		t.Fatalf("EmitJSON failed: %v", err)
	}

	jData := make(map[string]interface{})
	err = json.Unmarshal(data, &jData)
	if err != nil {
		t.Fatalf("EmitJSON returned invalid json: %v", err)
	}

	yStr := tc.jStr
	if yStr == "" {
		yStr, err = ygot.EmitJSON(tc.yObj, &ygot.EmitJSONConfig{
			Format:         ygot.RFC7951,
			RFC7951Config:  &ygot.RFC7951JSONConfig{AppendModuleName: !tc.opts.NoPrefix},
			SkipValidation: true,
		})
		if err != nil {
			t.Fatalf("ygot.EmitJSON failed: %v", err)
		}
	}

	yData := make(map[string]interface{})
	json.Unmarshal([]byte(yStr), &yData)
	if !jcompare(jData, yData) {
		t.Errorf("EmitJSON deviates from ygot.EmitJSON!")
		t.Log("Received ", dump(tc.name+".out.json", data))
		t.Log("Expected ", dump(tc.name+".exp.json", []byte(yStr)))
	}
}

func (tc *testCase) benchmarkEmitJSON(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := EmitJSON(tc.yObj, &tc.opts)
		if err != nil {
			b.Fatal(err.Error())
		}
	}
}

func (tc *testCase) benchmarkYgotEmitJSON(b *testing.B) {
	config := &ygot.EmitJSONConfig{
		Format:         ygot.RFC7951,
		RFC7951Config:  &ygot.RFC7951JSONConfig{AppendModuleName: true},
		SkipValidation: true,
	}
	for i := 0; i < b.N; i++ {
		_, err := ygot.EmitJSON(tc.yObj, config)
		if err != nil {
			b.Fatal(err.Error())
		}
	}
}

func initTestData(pattern string) {
	if len(pattern) == 0 {
		pattern = "*.yangjson"
		addBasicTests()
	}

	paths, _ := filepath.Glob(pattern)
	if len(paths) == 0 {
		paths, _ = filepath.Glob(filepath.Join("testdata", pattern))
	}

	loadFailed := false
	for _, fp := range paths {
		root, err := loadGoStruct(fp)
		if err != nil {
			fmt.Printf("WARNING: error loading %s: %v\n", fp, err)
			loadFailed = true
			continue
		}
		allTests = append(allTests, testCase{
			name: strings.Split(filepath.Base(fp), ".")[0],
			yObj: root,
		})
	}

	if loadFailed {
		os.Exit(1)
	}

	if len(allTests) == 0 {
		panic("No test data found")
	}
}

func loadGoStruct(fileName string) (ygot.ValidatedGoStruct, error) {
	data, err := ioutil.ReadFile(fileName)
	if err != nil {
		return nil, err
	}

	var root ygot.ValidatedGoStruct = new(Device)
	// first line is a path
	if data[0] == '/' {
		k := bytes.Index(data, []byte("\n"))
		if k == -1 {
			return nil, fmt.Errorf("bad path line")
		}

		p, err := ygot.StringToStructuredPath(string(data[:k]))
		if err != nil {
			return nil, err
		}

		removeModulePrefix(p)
		node, _, err := ytypes.GetOrCreateNode(SchemaTree["Device"], root, p)
		if err != nil {
			return nil, err
		}

		root = node.(ygot.ValidatedGoStruct)
		data = data[k+1:]
	}

	err = Unmarshal(data, root, new(ytypes.IgnoreExtraFields))
	return root, err
}

func addBasicTests() {
	// Empty root struct -- contains lots of empty child nodes
	root := new(Device)
	ygot.BuildEmptyTree(root)
	allTests = append(allTests, testCase{name: "empty_root", yObj: root})
	// Empty acl struct -- contains few empty child nodes
	allTests = append(allTests, testCase{name: "empty_acltop", yObj: root.Acl})
	// Empty acl struct -- contains 1 empty child node
	allTests = append(allTests, testCase{name: "empty_aclsets", yObj: root.Acl.AclSets})

	// Empty map for a list field
	ntpServers := new(OpenconfigSystem_System_Ntp_Servers)
	ntpServers.Server = make(map[string]*OpenconfigSystem_System_Ntp_Servers_Server)
	allTests = append(allTests, testCase{name: "empty_map", yObj: ntpServers, jStr: "{}"})
}

func jcompare(j1, j2 interface{}) bool {
	if reflect.TypeOf(j1) != reflect.TypeOf(j2) {
		return false
	}
	switch j1 := j1.(type) {
	case map[string]interface{}:
		j2 := j2.(map[string]interface{})
		if len(j1) != len(j2) {
			return false
		}
		for k1, v1 := range j1 {
			v2, ok := j2[k1]
			if !ok || !jcompare(v1, v2) {
				return false
			}
		}
	case []interface{}:
		j2 := j2.([]interface{})
		if len(j1) != len(j2) {
			return false
		}
		done := make([]bool, len(j2))
	outer:
		for _, v1 := range j1 {
			for i, v2 := range j2 {
				if !done[i] && jcompare(v1, v2) {
					done[i] = true
					continue outer
				}
			}
			return false
		}
	default:
		return j1 == j2
	}
	return true
}

func dump(fileName string, data []byte) string {
	var err error
	if len(tempDir) == 0 {
		if tempDir, err = ioutil.TempDir("", "emitjson."); err != nil {
			panic(err.Error())
		}
	}
	outFile := filepath.Join(tempDir, fileName)
	if err = ioutil.WriteFile(outFile, data, 0644); err != nil {
		panic(err.Error())
	}
	return outFile
}

func removeModulePrefix(p *gnmi.Path) {
	for _, ele := range p.Elem {
		if k := strings.IndexByte(ele.Name, ':'); k != -1 {
			ele.Name = ele.Name[k+1:]
		}
	}
}
