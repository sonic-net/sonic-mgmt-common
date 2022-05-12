////////////////////////////////////////////////////////////////////////////////
//                                                                            //
//  Copyright 2019 Broadcom. The term Broadcom refers to Broadcom Inc. and/or //
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


package utils

import (
    "github.com/Azure/sonic-mgmt-common/translib/db"
    "github.com/Azure/sonic-mgmt-common/cvl"
    "fmt"
    "time"
    "strconv"
    log "github.com/golang/glog"
)

// SortAsPerTblDeps - sort transformer result table list based on dependencies (using CVL API) tables to be used for CRUD operations
func SortAsPerTblDeps(tblLst []string) ([]string, error) {
        var resultTblLst []string
        var err error
        logStr := "Failure in CVL API to sort table list as per dependencies."

        cvSess, cvlRetSess := cvl.ValidationSessOpen()
        if cvlRetSess != cvl.CVL_SUCCESS {

                log.Errorf("Failure in creating CVL validation session object required to use CVl API(sort table list as per dependencies) - %v", cvlRetSess)
                err = fmt.Errorf("%v", logStr)
                return resultTblLst, err
        }
        cvlSortDepTblList, cvlRetDepTbl := cvSess.SortDepTables(tblLst)
        if cvlRetDepTbl != cvl.CVL_SUCCESS {
                log.Warningf("Failure in cvlSess.SortDepTables: %v", cvlRetDepTbl)
                cvl.ValidationSessClose(cvSess)
                err = fmt.Errorf("%v", logStr)
                return resultTblLst, err
        }
        log.Info("cvlSortDepTblList = ", cvlSortDepTblList)
        resultTblLst = cvlSortDepTblList

        cvl.ValidationSessClose(cvSess)
        return resultTblLst, err

}

// RemoveElement - Remove a specific string from a list of strings
func RemoveElement(sl []string, str string) []string {
    for i := 0; i < len(sl); i++ {
        if sl[i] == str {
            sl = append(sl[:i], sl[i+1:]...)
            i--
            break
        }
    }
    return sl
}

func GetActionStr(action int) string {
	switch action {
	case CLEAR:
		return "CLEAR"
	case RAISE:
		return "RAISE"
	case ACKNOWLEDGE:
		return "ACKNOWLEDGE"
	case UNACKNOWLEDGE:
		return "UNACKNOWLEDGE"
	default:
		return ""
	}
}

var seq_id = 1

const (
	RAISE = iota
	CLEAR
	ACKNOWLEDGE
	UNACKNOWLEDGE
)

func EventNotify(dbs [db.MaxDB]*db.DB, name string, source string, action int, format string, a ...string) error {

	timeCreated := time.Now().UnixNano()
	seq_id++
	tblKey := name + strconv.Itoa(seq_id)

	s := make([]interface{}, len(a))
	for i, v := range a {
		s[i] = v
	}

	text := fmt.Sprintf(format, s...)
	evtFields := db.Value{Field: make(map[string]string)}
	evtFields.Set("type-id", name)
	evtFields.Set("text", text)
	evtFields.Set("action", GetActionStr(action))
	evtFields.Set("time-created", strconv.FormatInt(timeCreated, 10))
	evtFields.Set("reckey", tblKey)
	evtFields.Set("resource", source)

	d := dbs[db.EventDB]

	err := d.CreateEntry(&db.TableSpec{Name: "EVENTPUBSUB"}, db.Key{Comp: []string{tblKey}}, evtFields)
	if err != nil {
		log.Info("Unable to write event  ", tblKey)
	}
	return err
}




