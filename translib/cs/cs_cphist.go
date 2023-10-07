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

// Config Session
package cs

import (
	"encoding/json"
	"io/ioutil"
	"os"

	"github.com/Azure/sonic-mgmt-common/translib/transformer"
	"github.com/golang/glog"
)

var checkPointPath = "/etc/sonic/checkpoints/"
var checkPointHistPath = checkPointPath + "cp_hist.json"

//type CpHistEntry struct {
//	CpName string `json:"name"`
//	CpHist CpHistData `json:"histentry"`
//}

type CpHistEntry struct {
	Label  string `json:"label"`
	Id     string `json:"id"`
	User   string `json:"user"`
	Origin string `json:"origin"`
	Time   int64  `json:"time"`
}

type CpHistEntries struct {
	CpHistEntries []CpHistEntry `json:"cphistentries"`
}

const MAX_HIST = 10

var cpHistEnts CpHistEntries
var is_hist_loaded bool

func init() {
	LoadCpHistEntries(checkPointHistPath)
	if len(cpHistEnts.CpHistEntries) > 0 {
		is_hist_loaded = true
	}
}

func isCpLabelExist(label string) bool {
	if len(label) == 0 {
		return false
	}
	for i := range cpHistEnts.CpHistEntries {
		if label == cpHistEnts.CpHistEntries[i].Label || label == cpHistEnts.CpHistEntries[i].Id {
			return true
		}
	}
	return false
}

func (c *CpHistEntries) AddCpHistEntry(entry CpHistEntry) {
	c.CpHistEntries = append(c.CpHistEntries, entry)
}

func SaveCpConfig(cpName string, user string) error {
	q_result := transformer.HostQuery("cphist_mgmt.cp_cfg_save", cpName, user)
	if q_result.Err != nil {
		glog.Errorf("check point config save Query failed: err=%+v", q_result.Err)
		return q_result.Err
	}
	return nil
}

func DeleteCpConfig(cpName string, user string) error {
	q_result := transformer.HostQuery("cphist_mgmt.cp_cfg_remove", cpName, user)
	if q_result.Err != nil {
		glog.Errorf("check point config delete Query failed: err=%+v", q_result.Err)
		return q_result.Err
	}
	return nil
}

func CreateCpHistEntry(cpName string, id string, user string, origin string, time int64) error {

	var fileName string
	if len(cpName) > 0 {
		fileName = cpName

	} else {
		fileName = id
	}

	if !is_hist_loaded {
		LoadCpHistEntries(checkPointHistPath)
		is_hist_loaded = true
	}

	if len(cpHistEnts.CpHistEntries) >= MAX_HIST {
		var delFileName string
		if len(cpHistEnts.CpHistEntries[0].Label) > 0 {
			delFileName = cpHistEnts.CpHistEntries[0].Label
		} else {
			delFileName = cpHistEnts.CpHistEntries[0].Id
		}
		DeleteCpConfig(delFileName, cpHistEnts.CpHistEntries[0].User)
		cpHistEnts.DeleteFirstCpHistEntry()
	}

	err := SaveCpConfig(fileName, user)
	if err != nil {
		return err
	}

	var entry CpHistEntry
	entry.Label = cpName
	entry.Id = id
	entry.User = user
	entry.Origin = origin
	entry.Time = time
	cpHistEnts.AddCpHistEntry(entry)
	err = UpdateCpHistFile(user)
	if err != nil {
		return err
	}
	return nil

}

func (c *CpHistEntries) DeleteFirstCpHistEntry() {
	c.CpHistEntries = c.CpHistEntries[1:]
}

func (c *CpHistEntries) DeleteCpHistEntry(entry CpHistEntry) {
	for idx, v := range c.CpHistEntries {
		if v == entry {
			//c.CpHistEntries = append(c.CpHistEntries[0:idx], c.CpHistEntries[idx+1:]...)
			c.CpHistEntries[idx] = c.CpHistEntries[len(c.CpHistEntries)-1]
			c.CpHistEntries[len(c.CpHistEntries)-1] = CpHistEntry{"", "", "", "", 0}
			c.CpHistEntries = c.CpHistEntries[:len(c.CpHistEntries)-1]
		}

	}
}

func LoadCpHistEntries(fileName string) {
	file, err := os.Open(fileName)

	if err != nil {
		if !os.IsNotExist(err) {
			glog.Errorf("failed to open checkpoint history file err=%+v", err)
		}
		return
	}

	defer file.Close()

	bData, _ := ioutil.ReadAll(file)

	err = json.Unmarshal(bData, &cpHistEnts)
	if err != nil {
		glog.Errorf("failed to unmarshal checkpoint history file err=%+v", err)
		return
	}
}

func UpdateCpHistFile(user string) error {
	jData, err := json.MarshalIndent(cpHistEnts, "", "  ")
	if err != nil {
		glog.Errorf("converting to json data failed: err=%+v", err)
		return err
	}
	js := string(jData)
	q_result := transformer.HostQuery("cphist_mgmt.cp_hist_save", user, js)
	if q_result.Err != nil {
		glog.Errorf("check point config history save Query failed: err=%+v", q_result.Err)
		return q_result.Err
	}
	return nil
}
