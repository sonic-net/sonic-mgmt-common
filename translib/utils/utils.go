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
	"fmt"
	"strconv"
	"strings"
	"sync"
	//"github.com/Azure/sonic-mgmt-common/translib/db"
	"github.com/Azure/sonic-mgmt-common/cvl"
	"github.com/Azure/sonic-mgmt-common/translib/db"
	log "github.com/golang/glog"
)

/* VLAN to tagged & untagged member sets map */
var vlanMemberCache *sync.Map

type vlan_member_list struct {
	tagged   Set
	untagged Set //set of ethernet or portchannel interfaces
}

// Set a sets representation of ports list.
type Set struct {
	items *sync.Map
}

// SetAddItem adds port to the Set.
func (s *Set) SetAddItem(port string) error {
	if s.items == nil {
		s.items = new(sync.Map)
	}

	_, ok := s.items.Load(port)
	if !ok {
		s.items.Store(port, struct{}{})
	}

	return nil
}

// SetDelItem removes the item from the Set.
func (s *Set) SetDelItem(key string) bool {
	if s.items == nil {
		return false
	}

	_, ok := s.items.Load(key)
	if ok {
		s.items.Delete(key)
	}
	return ok
}

// SetContains return true if Set contains the item.
func (s *Set) SetContains(item string) bool {
	if s.items == nil {
		return false
	}

	_, ok := s.items.Load(item)
	return ok
}

// SetSize returns the size of the set
func (s *Set) SetSize() int {
	if s.items == nil {
		return 0
	}

	count := 0

	s.items.Range(func(key, value interface{}) bool {
		count++
		return true
	})

	return count
}

// SetItems returns the stored ports list
func (s *Set) SetItems() []string {
	PortsSlice := []string{}

	if s.items == nil {
		return PortsSlice
	}

	s.items.Range(func(key, value interface{}) bool {
		PortsSlice = append(PortsSlice, key.(string))
		return true
	})

	return PortsSlice
}

func NewSet(list []string) Set {
	var s Set
	for _, item := range list {
		s.SetAddItem(item)
	}
	return s
}

// SortAsPerTblDeps - sort transformer result table list based on dependencies (using CVL API) tables to be used for CRUD operations
func SortAsPerTblDeps(tblLst []string) ([]string, error) {
	var resultTblLst []string
	var err error
	logStr := "Failure in CVL API to sort table list as per dependencies."

	cvSess, cvlRetSess := db.NewValidationSession()
	if cvlRetSess != nil {
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

// VlanDifference returns difference between existing list of Vlans and new list of Vlans.
func VlanDifference(vlanList1, vlanList2 []string) []string {
	mb := make(map[string]struct{}, len(vlanList2))
	for _, ifName := range vlanList2 {
		mb[ifName] = struct{}{}
	}
	var diff []string
	for _, ifName := range vlanList1 {
		if _, found := mb[ifName]; !found {
			diff = append(diff, ifName)
		}
	}
	return diff
}

// GenerateMemberPortsSliceFromString Convert string to slice
func GenerateMemberPortsSliceFromString(memberPortsStr *string) []string {
	if len(*memberPortsStr) == 0 {
		return nil
	}
	memberPorts := strings.Split(*memberPortsStr, ",")
	return memberPorts
}

// GetFromCacheVlanMemberList Get tagged/untagged Set for given vlan
func GetFromCacheVlanMemberList(vlanName string) (Set, Set) {
	if memberlist, ok := vlanMemberCache.Load(vlanName); ok {
		return memberlist.(*vlan_member_list).tagged, memberlist.(*vlan_member_list).untagged
	}
	return Set{}, Set{}
}

// ExtractVlanIdsFromRange expands given range into list of individual VLANs
// Param: A Range e.g. 1-3 or 1..3
// Return: Expanded list e.g. [Vlan1, Vlan2, Vlan3] */
func ExtractVlanIdsFromRange(rngStr string, vlanLst *[]string) error {
	var err error
	var res []string
	if strings.Contains(rngStr, "..") {
		res = strings.Split(rngStr, "..")
	}
	if strings.Contains(rngStr, "-") {
		res = strings.Split(rngStr, "-")
	}
	if len(res) != 0 {
		low, _ := strconv.Atoi(res[0])
		high, _ := strconv.Atoi(res[1])
		for id := low; id <= high; id++ {
			*vlanLst = append(*vlanLst, "Vlan"+strconv.Itoa(id))
		}
	}
	return err
}
