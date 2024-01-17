////////////////////////////////////////////////////////////////////////////////
//                                                                            //
//  Copyright 2023 Dell, Inc.                                                 //
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
	"fmt"
	"github.com/Azure/sonic-mgmt-common/translib/db"
	"io/ioutil"
	"os"

	"github.com/Azure/sonic-mgmt-common/cvl"
	log "github.com/golang/glog"
)

type xTablelist struct {
	TblInfo []xTblInfo `json:"tablelist"`
}

type xTblInfo struct {
	Name   string `json:"tablename"`
	Parent string `json:"parent"`
}

type gphNode struct {
	tableName string
	childTbl  []*gphNode
	visited   bool
}

func tblInfoAdd(tnMap map[string]*gphNode, tname, pname string) error {
	if _, ok := tnMap[tname]; !ok {
		node := new(gphNode)
		node.tableName = tname
		node.visited = false
		tnMap[tname] = node
	}
	node := tnMap[tname]
	if _, ok := tnMap[pname]; !ok {
		pnode := new(gphNode)
		pnode.tableName = pname
		pnode.visited = false
		tnMap[pname] = pnode
	}
	tnMap[pname].childTbl = append(tnMap[pname].childTbl, node)
	return nil
}

func childtblListGet(tnode *gphNode, ordTblList map[string][]string) ([]string, error) {
	var ctlist []string
	if len(tnode.childTbl) <= 0 {
		return ctlist, nil
	}

	if _, ok := ordTblList[tnode.tableName]; ok {
		return ordTblList[tnode.tableName], nil
	}

	for _, ctnode := range tnode.childTbl {
		if !ctnode.visited {
			ctnode.visited = true

			curTblList, err := childtblListGet(ctnode, ordTblList)
			if err != nil {
				ctlist = make([]string, 0)
				return ctlist, err
			}

			ordTblList[ctnode.tableName] = curTblList
			ctlist = append(ctlist, ctnode.tableName)
			ctlist = append(ctlist, curTblList...)
		} else {
			ctlist = append(ctlist, ctnode.tableName)
			ctlist = append(ctlist, ordTblList[ctnode.tableName]...)
		}
	}

	return ctlist, nil
}

func ordTblListCreate(ordTblList map[string][]string, tnMap map[string]*gphNode) {
	var tnodelist []*gphNode

	for _, tnode := range tnMap {
		tnodelist = append(tnodelist, tnode)
	}

	for _, tnode := range tnodelist {
		if (tnode != nil) && (!tnode.visited) {
			tnode.visited = true
			tlist, _ := childtblListGet(tnode, ordTblList)
			ordTblList[tnode.tableName] = tlist
		}
	}
}

// sort transformer result table list based on dependenciesi(using CVL API) tables to be used for CRUD operations
func sortPerTblDeps(ordTblListMap map[string][]string) error {
	var err error

	errStr := "Failed to create cvl session"
	cvSess, status := db.NewValidationSession()
	if status != nil {
		log.Warningf("CVL validation session creation failed(%v).", status)
		err = fmt.Errorf("%v", errStr)
		return err
	}

	for tname, tblList := range ordTblListMap {
		sortedTblList, status := cvSess.SortDepTables(tblList)
		if status != cvl.CVL_SUCCESS {
			log.Warningf("Failure in cvlSess.SortDepTables: %v", status)
			cvl.ValidationSessClose(cvSess)
			err = fmt.Errorf("%v", errStr)
			return err
		}
		ordTblListMap[tname] = sortedTblList
	}
	cvl.ValidationSessClose(cvSess)
	return err
}

func xlateJsonTblInfoLoad(ordTblListMap map[string][]string, jsonFileName string) error {
	var tlist xTablelist

	jsonFile, err := os.Open(jsonFileName)
	if err != nil {
		errStr := fmt.Sprintf("Error: Unable to open table list file(%v)", jsonFileName)
		return errors.New(errStr)
	}
	defer jsonFile.Close()

	xfmrLogDebug("Successfully Opened users.json\r\n")

	byteValue, _ := ioutil.ReadAll(jsonFile)

	json.Unmarshal(byteValue, &tlist)
	tnMap := make(map[string]*gphNode)

	for i := 0; i < len(tlist.TblInfo); i++ {
		err := tblInfoAdd(tnMap, tlist.TblInfo[i].Name, tlist.TblInfo[i].Parent)
		if err != nil {
			log.Warningf("Failed to add table dependency(tbl:%v, par:%v) into tablenode list.(%v)\r\n",
				tlist.TblInfo[i].Name, tlist.TblInfo[i].Parent, err)
			break
		}
	}

	if err == nil {
		ordTblListCreate(ordTblListMap, tnMap)
		for tname, tlist := range ordTblListMap {
			ordTblListMap[tname] = append([]string{tname}, tlist...)
		}
		if len(ordTblListMap) > 0 {
			sortPerTblDeps(ordTblListMap)
		}
	}
	return nil
}
