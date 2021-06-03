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
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"strings"

	"github.com/Azure/sonic-mgmt-common/translib/db"
	log "github.com/golang/glog"
)

const DEFAULT_EVPROFILE_DIR = "/etc/evprofile/"
const DEFAULT_EVPROFILE_NAME = "default.json"
const DEFAULT_EVPROFILE_PATH = DEFAULT_EVPROFILE_DIR + DEFAULT_EVPROFILE_NAME
const ACTIVE_EVPROFILE_SYMLINK_NAME = ".current"
const ACTIVE_EVPROFILE_SYMLINK_PATH = DEFAULT_EVPROFILE_DIR + ACTIVE_EVPROFILE_SYMLINK_NAME

// Event struct which contains all the fields
type Event struct {
	Name     string `json:"name"`
	Severity string `json:"severity"`
	Enable   string `json:"enable"`
	Message  string `json:"message"`
}

// Events struct which contains array of Event
type Events struct {
	Events []Event `json:"events"`
}

var ev_severity = []string{"informational", "warning", "minor", "major", "critical"}
var ev_enable = []string{"true", "false"}

func init() {
	XlateFuncBind("rpc_getevprofile_cb", rpc_getevprofile_cb)
	XlateFuncBind("rpc_setevprofile_cb", rpc_setevprofile_cb)
}

func validate_evprofile(ev_sev string, ev_en string) bool {
	for _, en_item := range ev_enable {
		if strings.EqualFold(en_item, ev_en) {
			for _, sev_item := range ev_severity {
				if strings.EqualFold(sev_item, ev_sev) {
					return true
				}
			}
			log.Errorf("Invalid value for severity field %s", ev_sev)
			return false
		}
	}
	log.Errorf("Invalid value for enable field %s", ev_en)
	return false
}

var rpc_setevprofile_cb RpcCallpoint = func(body []byte, dbs [db.MaxDB]*db.DB) ([]byte, error) {
	var setevprofile struct {
		Output struct {
			Result string `json:"status"`
		} `json:"sonic-set-evprofile:output"`
	}

	s := string(body)
	log.Info("received body", s)

	var inputData map[string]interface{}
	err := json.Unmarshal(body, &inputData)
	if err != nil {
		return nil, err
	}

	var filePath string
	var db_field [2]string

	db_field[0] = "name"

	input := inputData["sonic-evprofile:input"]
	if input != nil {
		inputData = input.(map[string]interface{})
		if value, ok := inputData["file-name"].(string); ok {
			if value != "" {
				filePath = DEFAULT_EVPROFILE_DIR + value
				db_field[1] = value
			} else {
				filePath = DEFAULT_EVPROFILE_PATH
				db_field[1] = DEFAULT_EVPROFILE_NAME
			}
		}
	}

	log.Info("input file path ", filePath)

	if _, err := os.Lstat(filePath); err != nil {
		log.Error("Event Profile doesnt exist")
		setevprofile.Output.Result = "Event Profile doesnt exist"
		result, _ := json.Marshal(&setevprofile)
		return result, err
	}

	// Open jsonFile
	jsonFile, _ := os.Open(filePath)

	// read opened jsonFile as a byte array.
	byteValue, _ := ioutil.ReadAll(jsonFile)

	// initialize Events array
	var events Events

	// unmarshal byteArray which contains jsonFile's content into 'events'
	json.Unmarshal(byteValue, &events)

	// iterate through every event within events array
	for i := 0; i < len(events.Events); i++ {
		log.Info("Event Name: " + events.Events[i].Name)
		log.Info("Event Severity: " + events.Events[i].Severity)
		log.Info("Event Enable: " + events.Events[i].Enable)
		log.Info("Event message: " + events.Events[i].Message)
		if !validate_evprofile(events.Events[i].Severity, events.Events[i].Enable) {
			setevprofile.Output.Result = "Event Profile has invalid values for parameters"
			result, _ := json.Marshal(&setevprofile)
			defer jsonFile.Close()
			return result, errors.New("Event Profile has invalid values for parameters")
		}
	}

	d := dbs[db.EventDB]
	key := "evprofile"
	filename_field := db.Value{Field: make(map[string]string)}
	filename_field.Set(db_field[0], db_field[1])

	existingEntry, _ := d.GetEntry(&db.TableSpec{Name: "EVPROFILE_TABLE"}, db.Key{Comp: []string{key}})
	if existingEntry.IsPopulated() {
		log.Info("EVPROFILE table is populdated. Modifying the filename")
		err = d.ModEntry(&db.TableSpec{Name: "EVPROFILE_TABLE"}, db.Key{Comp: []string{key}}, filename_field)
	} else {
		log.Info("EVPROFILE table is empty. Creating the filename entry..")
		err = d.CreateEntry(&db.TableSpec{Name: "EVPROFILE_TABLE"}, db.Key{Comp: []string{key}}, filename_field)
	}

	if err != nil {
		log.Error("Unable to announce validated event profile name to backend")
		setevprofile.Output.Result = "Event Profile can not be processed."
		result, _ := json.Marshal(&setevprofile)
		defer jsonFile.Close()
		return result, err
	}

	setevprofile.Output.Result = "Event Profile is applied."
	result, _ := json.Marshal(&setevprofile)
	defer jsonFile.Close()
	return result, nil
}

var rpc_getevprofile_cb RpcCallpoint = func(body []byte, dbs [db.MaxDB]*db.DB) ([]byte, error) {

	var getevprofile struct {
		Output struct {
			Result   string   `json:"file-name"`
			Filelist []string `json:"file-list"`
		} `json:"sonic-get-evprofile:output"`
	}

	// read symlink
	n, e := os.Readlink(ACTIVE_EVPROFILE_SYMLINK_PATH)
	if e != nil {
		getevprofile.Output.Result = "No event profile is active."
	} else {
		getevprofile.Output.Result = string(n[15:])
	}

	// read all the files under evprofile directory
	files, err := ioutil.ReadDir(DEFAULT_EVPROFILE_DIR)
	if err == nil {
		for _, file := range files {
			if file.Name() != ACTIVE_EVPROFILE_SYMLINK_NAME {
				getevprofile.Output.Filelist = append(getevprofile.Output.Filelist, file.Name())
			}
		}
	} else {
		getevprofile.Output.Filelist = append(getevprofile.Output.Filelist, "Can not fetch Event Profile list.")
		log.Error("Can not fetch Event Profile list.")
	}

	result, _ := json.Marshal(&getevprofile)

	return result, nil
}
