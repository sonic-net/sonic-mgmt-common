////////////////////////////////////////////////////////////////////////////////
//                                                                            //
//  Copyright 2019 Dell, Inc.                                                 //
//                                                                            //
//  Licensed under the Apache License, Version 2.0 (the "License");           //
//  you may not use this file except in compliance with the License.          //
//  You may obtain a copy of the License at                                   //
//                                                                            //
//  http://www.apache.org/licenses/LICENSE-2.0                                //
//                                                                            //
//  Unless required by applicable law or agreed to in writing, software       //
//  distributed under the License is distributed on an "AS IS" BASIS,         //
//  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.  //
//  See the License for the specific language governing permissions and       //
//  limitations under the License.                                            //
//                                                                            //
////////////////////////////////////////////////////////////////////////////////

package transformer

import (
	"bufio"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"

	log "github.com/golang/glog"

	"github.com/Azure/sonic-mgmt-common/translib/ocbinds"
	"github.com/openconfig/goyang/pkg/yang"
)

var YangPath = "/usr/models/yang/" // OpenConfig-*.yang and sonic yang models path
var ModelsListFile = "models_list"
var TblInfoJsonFile = "sonic_table_info.json"

func getModelsList() ([]string, map[string]bool) {
	var fileList []string
	excludeSonicList := make(map[string]bool)
	file, err := os.Open(YangPath + ModelsListFile)
	if err != nil {
		return fileList, excludeSonicList
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fileEntry := scanner.Text()
		if strings.HasPrefix(fileEntry, "-sonic") {
			_, err := os.Stat(YangPath + fileEntry[1:])
			if err != nil {
				continue
			}
			excludeSonicList[fileEntry[1:]] = true
			continue
		}
		if !strings.HasPrefix(fileEntry, "#") {
			_, err := os.Stat(YangPath + fileEntry)
			if err != nil {
				continue
			}
			fileList = append(fileList, fileEntry)
		}
	}
	return fileList, excludeSonicList
}

func getDefaultModelsList(excludeList map[string]bool) []string {
	var files []string
	fileInfo, err := ioutil.ReadDir(YangPath)
	if err != nil {
		return files
	}

	for _, file := range fileInfo {
		if strings.HasPrefix(file.Name(), "sonic-") && !strings.HasSuffix(file.Name(), "-dev.yang") && filepath.Ext(file.Name()) == ".yang" {
			if _, ok := excludeList[file.Name()]; !ok {
				files = append(files, file.Name())
			}
		}
	}
	return files
}

func init() {
	initYangModelsPath()
	initRegex()
	modelsList, excludeSncList := getModelsList()
	yangFiles := getDefaultModelsList(excludeSncList)
	yangFiles = append(yangFiles, modelsList...)
	xfmrLogInfo("Yang model List: %v", yangFiles)
	err := loadYangModules(yangFiles...)
	if err != nil {
		log.Error(err)
	}
	debug.FreeOSMemory()
}

func initYangModelsPath() {
	if path, ok := os.LookupEnv("YANG_MODELS_PATH"); ok {
		if !strings.HasSuffix(path, "/") {
			path += "/"
		}
		YangPath = path
	}

	xfmrLogDebug("Yang modles path: %v", YangPath)
}

type ModProfile map[string]interface{}

func loadYangModules(files ...string) error {
	var err error

	paths := []string{YangPath}

	for _, path := range paths {
		expanded, err := yang.PathsWithModules(path)
		if err != nil {
			xfmrLogInfo("Couldn't load yang module %v due to %v ", path, err)
			continue
		}
		yang.AddPath(expanded...)
	}

	ygSchema, err := ocbinds.Schema()
	if err != nil {
		return err
	}

	modProfiles := make(map[string]ModProfile)
	rootSchema := ygSchema.RootSchema()
	for _, schema := range rootSchema.Dir {
		if _, found := schema.Annotation["modulename"]; found {
			m := schema.Annotation["modulename"].(string)
			for _, fn := range files {
				f := strings.Split(fn, ".yang")
				if m == f[0] {
					modProfiles[m] = schema.Annotation
					break
				}
			}
		}
	}

	annotMs := yang.NewModules()
	for _, name := range files {
		if strings.Contains(name, "-annot") {
			if err := annotMs.Read(name); err != nil {
				xfmrLogInfo("Couldn't read yang annotation %v due to %v", name, err)
				continue
			}
		}
	}

	oc_annot_entries := make([]*yang.Entry, 0)
	sonic_annot_entries := make([]*yang.Entry, 0)

	for _, m := range annotMs.Modules {
		if strings.Contains(m.Name, "sonic") {
			sonic_annot_entries = append(sonic_annot_entries, yang.ToEntry(m))
		} else {
			yangMdlNmDt := strings.Split(m.Name, "-annot")
			if len(yangMdlNmDt) > 0 {
				addMdlCpbltEntry(yangMdlNmDt[0])
			}
			oc_annot_entries = append(oc_annot_entries, yang.ToEntry(m))
		}
	}

	sonic_entries := make([]*yang.Entry, 0)
	oc_entries := make(map[string]*yang.Entry)

	// Iterate over SchemaTree
	for k, v := range ygSchema.SchemaTree["Device"].Dir {
		mod := strings.Split(v.Annotation["schemapath"].(string), "/")
		if _, found := modProfiles[mod[1]]; found {
			if strings.Contains(k, "sonic-") {
				sonic_entries = append(sonic_entries, v)
			} else if oc_entries[k] == nil {
				oc_entries[k] = v
			}
		}
	}

	// populate model capabilities data
	for yngMdlNm := range xMdlCpbltMap {
		org := ""
		ver := ""
		ocVerSet := false
		yngEntry := oc_entries[yngMdlNm]
		if yngEntry != nil {
			// OC yang has version in standard extension oc-ext:openconfig-version
			if strings.HasPrefix(yngMdlNm, "openconfig-") {
				for _, ext := range yngEntry.Exts {
					dataTagArr := strings.Split(ext.Keyword, ":")
					tagType := dataTagArr[len(dataTagArr)-1]
					if tagType == "openconfig-version" {
						ver = ext.NName()
						xfmrLogDebug("Found version %v for yang module %v", ver, yngMdlNm)
						if len(strings.TrimSpace(ver)) > 0 {
							ocVerSet = true
						}
						break
					}

				}
			}
		}
		if (strings.HasPrefix(yngMdlNm, "ietf-")) || (!ocVerSet) {
			// as per RFC7895 revision date to be used as version
			if value, found := modProfiles[yngMdlNm]["revison"]; found {
				ver = value.(string)
			}
		}
		if value, found := modProfiles[yngMdlNm]["organization"]; found {
			org = value.(string)
		}
		addMdlCpbltData(yngMdlNm, ver, org)
	}

	dbMapBuild(sonic_entries)
	annotDbSpecMap(sonic_annot_entries)
	annotToDbMapBuild(oc_annot_entries)
	yangToDbMapBuild(oc_entries)

	return err
}
