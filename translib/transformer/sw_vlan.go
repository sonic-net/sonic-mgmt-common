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
	"errors"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/Azure/sonic-mgmt-common/translib/db"
	"github.com/Azure/sonic-mgmt-common/translib/ocbinds"
	"github.com/Azure/sonic-mgmt-common/translib/tlerr"
	"github.com/Azure/sonic-mgmt-common/translib/utils"
	log "github.com/golang/glog"
	"github.com/openconfig/ygot/ygot"
)

type intfModeType int

const (
	MODE_UNSET intfModeType = iota
	ACCESS
	TRUNK
	ALL
)

type intfModeReq struct {
	ifName string
	mode   intfModeType
}

type ifVlan struct {
	ifName *string
	mode   intfModeType
	//accessVlan *string
	trunkVlans []string
}

type swVlanMemberPort_t struct {
	swEthMember         *ocbinds.OpenconfigInterfaces_Interfaces_Interface_Ethernet_SwitchedVlan
	swPortChannelMember *ocbinds.OpenconfigInterfaces_Interfaces_Interface_Aggregation_SwitchedVlan
}

func init() {
	XlateFuncBind("YangToDb_sw_vlans_xfmr", YangToDb_sw_vlans_xfmr)
	XlateFuncBind("DbToYang_sw_vlans_xfmr", DbToYang_sw_vlans_xfmr)
	XlateFuncBind("DbToYangPath_sw_vlans_path_xfmr", DbToYangPath_sw_vlans_path_xfmr)
}

/*
Param: port/portchannel name

	Return: tagged & untagged vlan list config for given port/portchannel
*/
func getIntfVlanConfig(d *db.DB, tblName string, ifName string) ([]string, string, error) {
	var taggedVlanList []string
	var untaggedVlan string
	vlanMemberKeys, err := d.GetKeysByPattern(&db.TableSpec{Name: tblName}, "*"+ifName)
	if err != nil {
		return nil, "", err
	}
	for _, vlanMember := range vlanMemberKeys {
		vlanId := vlanMember.Get(0)
		entry, err := d.GetEntry(&db.TableSpec{Name: tblName}, db.Key{Comp: []string{vlanId, ifName}})
		if err != nil {
			return nil, "", err
		}
		tagMode := entry.Field["tagging_mode"]
		if tagMode == "tagged" {
			taggedVlanList = append(taggedVlanList, vlanId)
		} else {
			untaggedVlan = vlanId
		}
	}
	return taggedVlanList, untaggedVlan, nil
}

/* Validate whether VLAN exists in DB */
func validateVlanExists(d *db.DB, vlanName *string) error {
	if len(*vlanName) == 0 {
		return errors.New("Length of VLAN name is zero")
	}
	entry, err := d.GetEntry(&db.TableSpec{Name: VLAN_TN}, db.Key{Comp: []string{*vlanName}})
	if err != nil || !entry.IsPopulated() {
		errStr := "Vlan:" + *vlanName + " does not exist!"
		log.V(3).Info(errStr)
		return errors.New(errStr)
	}
	return nil
}

/* Validates whether physical interface or port-channel interface configured as member of any existing VLAN */
func validateIntfAssociatedWithExistingVlan(d *db.DB, ifName *string) error {
	var err error

	if len(*ifName) == 0 {
		return errors.New("Interface name is empty!")
	}
	var vlanKeys []db.Key
	vlanKeys, err = d.GetKeysByPattern(&db.TableSpec{Name: VLAN_MEMBER_TN}, "*"+*ifName)

	if err != nil {
		return errors.New("Failed to get keys from table: " + VLAN_MEMBER_TN)
	}
	log.Infof("Interface member of %d Vlan(s)", len(vlanKeys))
	if len(vlanKeys) > 0 {
		errStr := "Vlan configuration exists on interface: " + *ifName
		log.Error(errStr)
		return tlerr.InvalidArgsError{Format: errStr}
	}
	return err
}

/* Check member port exists in the list and get Interface mode */
func checkMemberPortExistsInListAndGetMode(d *db.DB, memberPortsList []string, memberPort *string, vlanName *string, ifMode *intfModeType) bool {
	for _, port := range memberPortsList {
		if *memberPort == port {
			tagModeEntry, err := d.GetEntry(&db.TableSpec{Name: VLAN_MEMBER_TN}, db.Key{Comp: []string{*vlanName, *memberPort}})
			if err != nil {
				return false
			}
			tagMode := tagModeEntry.Field["tagging_mode"]
			convertTaggingModeToInterfaceModeType(&tagMode, ifMode)
			return true
		}
	}
	return false
}

/* Convert tagging mode to Interface Mode type */
func convertTaggingModeToInterfaceModeType(tagMode *string, ifMode *intfModeType) {
	switch *tagMode {
	case "untagged":
		*ifMode = ACCESS
	case "tagged":
		*ifMode = TRUNK
	}
}

/* Validate whether Port has any Untagged VLAN Config existing */
func validateUntaggedVlanCfgredForIf(d *db.DB, vlanMemberTs *string, ifName *string, accessVlan *string) (bool, error) {
	var err error

	var vlanMemberKeys []db.Key

	vlanMemberKeys, err = d.GetKeysPattern(&db.TableSpec{Name: *vlanMemberTs}, db.Key{Comp: []string{"*", *ifName}})
	if err != nil {
		return false, err
	}

	log.Infof("Found %d Vlan Member table keys", len(vlanMemberKeys))

	for _, vlanMember := range vlanMemberKeys {
		if len(vlanMember.Comp) < 2 {
			continue
		}
		memberPortEntry, err := d.GetEntry(&db.TableSpec{Name: *vlanMemberTs}, vlanMember)
		if err != nil || !memberPortEntry.IsPopulated() {
			errStr := "Get from VLAN_MEMBER table for Vlan: + " + vlanMember.Get(0) + " Interface:" + *ifName + " failed!"
			log.Error(errStr)
			return false, errors.New(errStr)
		}
		tagMode, ok := memberPortEntry.Field["tagging_mode"]
		if !ok {
			errStr := "tagging_mode entry is not present for VLAN: " + vlanMember.Get(0) + " Interface: " + *ifName
			log.Error(errStr)
			return false, errors.New(errStr)
		}
		if tagMode == "untagged" {
			*accessVlan = vlanMember.Get(0)
			return true, nil
		}
	}
	return false, nil
}

/* Fills all the trunk-vlans part of physical or port-channel interface */
func fillTrunkVlansForInterface(d *db.DB, ifName *string, ifVlanInfo *ifVlan) error {
	var err error
	var vlanKeys []db.Key

	vlanKeys, err = d.GetKeysByPattern(&db.TableSpec{Name: VLAN_MEMBER_TN}, "*"+*ifName)
	if err != nil {
		return err
	}

	for _, vlanKey := range vlanKeys {
		if len(vlanKey.Comp) < 2 {
			continue
		}
		if vlanKey.Get(1) == *ifName {
			memberPortEntry, err := d.GetEntry(&db.TableSpec{Name: VLAN_MEMBER_TN}, vlanKey)
			if err != nil {
				log.Errorf("Error found on fetching Vlan member info from App DB for Interface Name : %s", *ifName)
				return err
			}
			tagInfo, ok := memberPortEntry.Field["tagging_mode"]
			if ok {
				if tagInfo == "tagged" {
					ifVlanInfo.trunkVlans = append(ifVlanInfo.trunkVlans, vlanKey.Get(0))
				}
			}
		}
	}
	return err
}

/* Remove tagged port associated with VLAN and update VLAN_MEMBER table */
func removeTaggedVlanAndUpdateVlanMembTbl(d *db.DB, trunkVlan *string, ifName *string,
	vlanMemberMap map[string]db.Value) error {
	var err error

	memberPortEntry, err := d.GetEntry(&db.TableSpec{Name: VLAN_MEMBER_TN}, db.Key{Comp: []string{*trunkVlan, *ifName}})
	if err != nil || !memberPortEntry.IsPopulated() {
		errStr := "Tagged Vlan configuration: " + *trunkVlan + " doesn't exist for Interface: " + *ifName
		log.V(3).Info(errStr)
		return tlerr.InvalidArgsError{Format: errStr}
	}
	tagMode, ok := memberPortEntry.Field["tagging_mode"]
	if !ok {
		errStr := "tagging_mode entry is not present for VLAN: " + *trunkVlan + " Interface: " + *ifName
		log.V(3).Info(errStr)
		return errors.New(errStr)
	}
	vlanName := *trunkVlan
	if tagMode == "tagged" {
		vlanMemberKey := *trunkVlan + "|" + *ifName
		vlanMemberMap[vlanMemberKey] = db.Value{Field: map[string]string{}}
	} else {
		vlanId := vlanName[len("Vlan"):]
		errStr := "Tagged VLAN: " + vlanId + " configuration doesn't exist for Interface: " + *ifName
		log.V(3).Info(errStr)
		return tlerr.InvalidArgsError{Format: errStr}
	}
	return err
}

/* Remove untagged port associated with VLAN and update VLAN_MEMBER table */
func removeUntaggedVlanAndUpdateVlanMembTbl(d *db.DB, ifName *string,
	vlanMemberMap map[string]db.Value) (*string, error) {
	if len(*ifName) == 0 {
		return nil, errors.New("Interface name is empty for fetching list of VLANs!")
	}

	var vlanMemberKeys []db.Key
	var err error

	vlanMemberKeys, err = d.GetKeysByPattern(&db.TableSpec{Name: VLAN_MEMBER_TN}, "*"+*ifName)
	if err != nil {
		return nil, err
	}

	log.Infof("Found %d Vlan Member table keys", len(vlanMemberKeys))

	for _, vlanMember := range vlanMemberKeys {
		if len(vlanMember.Comp) < 2 {
			continue
		}
		if vlanMember.Get(1) != *ifName {
			continue
		}
		memberPortEntry, err := d.GetEntry(&db.TableSpec{Name: VLAN_MEMBER_TN}, vlanMember)
		if err != nil || !memberPortEntry.IsPopulated() {
			errStr := "Get from VLAN_MEMBER table for Vlan: + " + vlanMember.Get(0) + " Interface:" + *ifName + " failed!"
			return nil, errors.New(errStr)
		}
		tagMode, ok := memberPortEntry.Field["tagging_mode"]
		if !ok {
			errStr := "tagging_mode entry is not present for VLAN: " + vlanMember.Get(0) + " Interface: " + *ifName
			return nil, errors.New(errStr)
		}
		vlanName := vlanMember.Get(0)
		vlanMemberKey := vlanName + "|" + *ifName
		if tagMode == "untagged" {
			vlanMemberMap[vlanMemberKey] = db.Value{Field: map[string]string{}}
			return &vlanName, nil
		}
	}
	errStr := "Untagged VLAN configuration doesn't exist for Interface: " + *ifName
	log.Info(errStr)
	return nil, tlerr.InvalidArgsError{Format: errStr}
}

func removeAllVlanMembrsForIfAndGetVlans(d *db.DB, ifName *string, ifMode intfModeType, vlanMemberMap map[string]db.Value) error {
	var err error
	var vlanKeys []db.Key

	vlanKeys, err = d.GetKeysByPattern(&db.TableSpec{Name: VLAN_MEMBER_TN}, "*"+*ifName)
	if err != nil {
		return err
	}

	for _, vlanKey := range vlanKeys {
		if len(vlanKeys) < 2 {
			continue
		}
		if vlanKey.Get(1) == *ifName {
			memberPortEntry, err := d.GetEntry(&db.TableSpec{Name: VLAN_MEMBER_TN}, vlanKey)
			if err != nil {
				log.Errorf("Error found on fetching Vlan member info from App DB for Interface Name : %s", *ifName)
				return err
			}
			tagInfo, ok := memberPortEntry.Field["tagging_mode"]
			if ok {
				switch ifMode {
				case ACCESS:
					if tagInfo != "tagged" {
						continue
					}
				case TRUNK:
					if tagInfo != "untagged" {
						continue
					}
				}
				vlanMemberKey := vlanKey.Get(0) + "|" + *ifName
				vlanMemberMap[vlanMemberKey] = db.Value{Field: make(map[string]string)}
				vlanMemberMap[vlanMemberKey] = memberPortEntry
			}
		}
	}
	return err
}

/*
	Function to compress vlan list

Param: string list of Vlan ids, e.g: ["Vlan1","Vlan2","Vlan30"] or ["1","2","30"]
Return: string list of Vlan range/ids, e.g: ["1-2","30"]
*/
func vlanIdstoRng(vlanIdsLst []string) ([]string, error) {
	var err error
	var idsLst []int
	var vlanRngLst []string
	for _, v := range vlanIdsLst {
		id, _ := strconv.Atoi(strings.TrimPrefix(v, "Vlan"))
		idsLst = append(idsLst, id)
	}
	sort.Ints(idsLst)
	for i, j := 0, 0; j < len(idsLst); j = j + 1 {
		if (j+1 < len(idsLst) && idsLst[j+1] == idsLst[j]+1) || (j+1 < len(idsLst) && idsLst[j] == idsLst[j+1]) {
			continue
		}
		if i == j {
			vlanid := strconv.Itoa(idsLst[i])
			vlanRngLst = append(vlanRngLst, (vlanid))
		} else {
			vlanidLow := strconv.Itoa(idsLst[i])
			vlanidHigh := strconv.Itoa(idsLst[j])
			vlanRngLst = append(vlanRngLst, (vlanidLow + "-" + vlanidHigh))
		}
		i = j + 1
	}
	return vlanRngLst, err
}

func intfAccessModeReqConfig(d *db.DB, ifName *string,
	vlanMap map[string]db.Value,
	vlanMemberMap map[string]db.Value) error {
	var err error
	if len(*ifName) == 0 {
		return errors.New("Empty Interface name received!")
	}

	err = removeAllVlanMembrsForIfAndGetVlans(d, ifName, ACCESS, vlanMemberMap)
	if err != nil {
		return err
	}

	return err
}

func intfModeReqConfig(d *db.DB, mode intfModeReq,
	vlanMap map[string]db.Value,
	vlanMemberMap map[string]db.Value) error {
	var err error
	switch mode.mode {
	case ACCESS:
		err := intfAccessModeReqConfig(d, &mode.ifName, vlanMap, vlanMemberMap)
		if err != nil {
			return err
		}
	case TRUNK:
	case MODE_UNSET:
		break
	}
	return err
}

/* Adding member to VLAN requires updation of VLAN Table and VLAN Member Table */
func processIntfVlanMemberAdd(d *db.DB, vlanMembersMap map[string]map[string]db.Value, vlanMap map[string]db.Value,
	vlanMemberMap map[string]db.Value) error {
	var err error

	/* Updating the VLAN member table */
	for vlanName, ifEntries := range vlanMembersMap {
		log.V(3).Info("Processing VLAN: ", vlanName)

		vlanEntry, _ := d.GetEntry(&db.TableSpec{Name: VLAN_TN}, db.Key{Comp: []string{vlanName}})
		if !vlanEntry.IsPopulated() {
			errStr := "Failed to retrieve memberPorts info of VLAN : " + vlanName
			log.Error(errStr)
			return errors.New(errStr)
		}

		for ifName, ifEntry := range ifEntries {
			log.V(3).Infof("Processing Interface: %s for VLAN: %s", ifName, vlanName)

			vlanMemberKey := vlanName + "|" + ifName
			vlanMemberMap[vlanMemberKey] = db.Value{Field: make(map[string]string)}
			vlanMemberMap[vlanMemberKey].Field["tagging_mode"] = ifEntry.Field["tagging_mode"]
			log.V(3).Infof("Updated Vlan Member Map with vlan member key: %s and tagging-mode: %s", vlanMemberKey, ifEntry.Field["tagging_mode"])
		}
		vlanMap[vlanName] = db.Value{Field: make(map[string]string)}
	}

	return err
}

func processIntfVlanMemberRemoval(inParams *XfmrParams, ifVlanInfoList []*ifVlan, vlanMap map[string]db.Value,
	vlanMemberMap map[string]db.Value) error {
	var err error

	d := inParams.d

	if len(ifVlanInfoList) == 0 {
		log.Info("No VLAN Info present for membership removal!")
		return nil
	}

	for _, ifVlanInfo := range ifVlanInfoList {
		if ifVlanInfo.ifName == nil {
			return errors.New("No Interface name present for membership removal from VLAN!")
		}

		ifName := ifVlanInfo.ifName
		ifMode := ifVlanInfo.mode
		trunkVlans := ifVlanInfo.trunkVlans

		switch ifMode {
		case ACCESS:
			/* Handling Access Vlan delete */
			log.Info("Access VLAN Delete!")
			_, err = removeUntaggedVlanAndUpdateVlanMembTbl(d, ifName, vlanMemberMap)
			if err != nil {
				return err
			}
		case TRUNK:
			/* Handling trunk-vlans delete */
			log.Info("Trunk VLAN Delete!")
			cfgredTrunkVlanList, _, _ := getIntfVlanConfig(inParams.d, VLAN_MEMBER_TN, *ifName)
			sort.Strings(cfgredTrunkVlanList)
			sort.Strings(trunkVlans)
			for _, trunkVlan := range trunkVlans {
				idx := sort.SearchStrings(cfgredTrunkVlanList, trunkVlan)
				//If Vlan Not exists in the Configured Tagged Vlan List then ignore
				if idx >= len(cfgredTrunkVlanList) || cfgredTrunkVlanList[idx] != trunkVlan {
					errStr := "Tagged Vlan : " + trunkVlan + " doesn't exist for Interface: " + *ifName
					log.V(3).Info(errStr)
					continue
				}
				if log.V(3) {
					log.Info("Tagged Vlan :", trunkVlan, " exists for the Interface", *ifName)
				}
				rerr := removeTaggedVlanAndUpdateVlanMembTbl(d, &trunkVlan, ifName, vlanMemberMap)
				if rerr != nil {
					//If trunkVlan config not present for ifname continue to next trunkVlan in list
					continue
				}
			}
		// Mode set to ALL, if you want to delete both access and trunk
		case ALL:
			log.Info("Handling All Access and Trunk VLAN delete!")
			//Access Vlan Delete
			_, _ = removeUntaggedVlanAndUpdateVlanMembTbl(d, ifName, vlanMemberMap)
			//Trunk Vlan Delete
			cfgredTrunkVlanList, _, _ := getIntfVlanConfig(inParams.d, VLAN_MEMBER_TN, *ifName)
			if len(cfgredTrunkVlanList) > 0 {
				if log.V(3) {
					log.Info("Configured Tagged Vlan list , cfgredTaggedVlan: ", cfgredTrunkVlanList)
				}
				for _, trunkVlan := range cfgredTrunkVlanList {
					rerr := removeTaggedVlanAndUpdateVlanMembTbl(d, &trunkVlan, ifName, vlanMemberMap)
					if rerr != nil {
						//If trunkVlan config not present for ifname continue to next trunkVlan in list
						continue
					}
				}
			} else {
				log.Info("Tagged Vlan doesn't exist for the interface :", *ifName)
			}
		}
	}
	return nil
}

/* Function performs VLAN Member removal from Interface */
/* Handles 4 cases
   case 1: Deletion of top-level container / list
   case 2: Deletion of entire leaf-list trunk-vlans
   case 3: Deletion of access-vlan leaf
   case 4: Deletion of trunk-vlan (leaf-list with instance)  */
func intfVlanMemberRemoval(swVlanConfig *swVlanMemberPort_t,
	inParams *XfmrParams, ifName *string,
	vlanMap map[string]db.Value,
	vlanMemberMap map[string]db.Value,
	intfType E_InterfaceType) error {
	var err error
	var ifVlanInfo ifVlan
	var ifVlanInfoList []*ifVlan

	targetUriPath := (NewPathInfo(inParams.uri)).YangPath
	log.Info("Target URI Path = ", targetUriPath)
	switch intfType {
	case IntfTypeEthernet:
		if swVlanConfig.swPortChannelMember != nil {
			errStr := "Wrong yang path is used for member " + *ifName + " disassociation from vlan"
			log.Errorf(errStr)
			return errors.New(errStr)
		}
		//case 1
		if swVlanConfig.swEthMember == nil || swVlanConfig.swEthMember.Config == nil ||
			(swVlanConfig.swEthMember.Config.AccessVlan == nil && swVlanConfig.swEthMember.Config.TrunkVlans == nil) {

			log.Info("Container/list level delete for Interface: ", *ifName)
			ifVlanInfo.mode = ALL
			//case 2
			if targetUriPath == "/openconfig-interfaces:interfaces/interface/openconfig-if-ethernet:ethernet/openconfig-vlan:switched-vlan/config/trunk-vlans" {
				ifVlanInfo.mode = TRUNK
			}
			//Fill Trunk Vlans for interface
			err = fillTrunkVlansForInterface(inParams.d, ifName, &ifVlanInfo)
			if err != nil {
				return err
			}

			ifVlanInfo.ifName = ifName
			ifVlanInfoList = append(ifVlanInfoList, &ifVlanInfo)

			err = processIntfVlanMemberRemoval(inParams, ifVlanInfoList, vlanMap, vlanMemberMap)
			if err != nil {
				log.Errorf("Interface VLAN member removal for Interface: %s failed!", *ifName)
				return err
			}
			return err
		}
		//case 3
		if swVlanConfig.swEthMember.Config.AccessVlan != nil {
			ifVlanInfo.mode = ACCESS
		}
		//case 4
		if swVlanConfig.swEthMember.Config.TrunkVlans != nil {
			trunkVlansUnionList := swVlanConfig.swEthMember.Config.TrunkVlans
			ifVlanInfo.mode = TRUNK

			for _, trunkVlanUnion := range trunkVlansUnionList {
				trunkVlanUnionType := reflect.TypeOf(trunkVlanUnion).Elem()

				switch trunkVlanUnionType {

				case reflect.TypeOf(ocbinds.OpenconfigInterfaces_Interfaces_Interface_Ethernet_SwitchedVlan_Config_TrunkVlans_Union_String{}):
					val := (trunkVlanUnion).(*ocbinds.OpenconfigInterfaces_Interfaces_Interface_Ethernet_SwitchedVlan_Config_TrunkVlans_Union_String)
					vlansList := strings.Split(val.String, ",")
					for _, vlan := range vlansList {
						/* Handle case if multiple/range of VLANs given */
						if strings.Contains(vlan, "..") { //e.g vlan - 1..100
							err = utils.ExtractVlanIdsFromRange(vlan, &ifVlanInfo.trunkVlans)
							if err != nil {
								return err
							}
						} else {
							vlanName := "Vlan" + vlan
							err = validateVlanExists(inParams.d, &vlanName)
							if err != nil {
								return err
							}
							ifVlanInfo.trunkVlans = append(ifVlanInfo.trunkVlans, vlanName)
						}
					}
				case reflect.TypeOf(ocbinds.OpenconfigInterfaces_Interfaces_Interface_Ethernet_SwitchedVlan_Config_TrunkVlans_Union_Uint16{}):
					val := (trunkVlanUnion).(*ocbinds.OpenconfigInterfaces_Interfaces_Interface_Ethernet_SwitchedVlan_Config_TrunkVlans_Union_Uint16)
					ifVlanInfo.trunkVlans = append(ifVlanInfo.trunkVlans, "Vlan"+strconv.Itoa(int(val.Uint16)))
				}
			}
		}
	case IntfTypePortChannel:
		if swVlanConfig.swEthMember != nil {
			errStr := "Wrong yang path is used for Interface " + *ifName + " disassociation from Port-Channel Interface"
			log.Error(errStr)
			return errors.New(errStr)
		}
		//case 1
		if swVlanConfig.swPortChannelMember == nil || swVlanConfig.swPortChannelMember.Config == nil ||
			(swVlanConfig.swPortChannelMember.Config.AccessVlan == nil && swVlanConfig.swPortChannelMember.Config.TrunkVlans == nil) {

			log.Info("Container/list level delete for Interface: ", *ifName)
			ifVlanInfo.mode = ALL
			//case 2
			if targetUriPath == "/openconfig-interfaces:interfaces/interface/openconfig-if-aggregate:aggregation/openconfig-vlan:switched-vlan/config/trunk-vlans" {
				ifVlanInfo.mode = TRUNK
			}

			//Fill Trunk Vlans for interface
			err = fillTrunkVlansForInterface(inParams.d, ifName, &ifVlanInfo)
			if err != nil {
				return err
			}

			ifVlanInfo.ifName = ifName
			ifVlanInfoList = append(ifVlanInfoList, &ifVlanInfo)

			err = processIntfVlanMemberRemoval(inParams, ifVlanInfoList, vlanMap, vlanMemberMap)
			if err != nil {
				log.Errorf("Interface VLAN member removal for Interface: %s failed!", *ifName)
				return err
			}
			return err
		}
		//case 3
		if swVlanConfig.swPortChannelMember.Config.AccessVlan != nil {
			ifVlanInfo.mode = ACCESS
		}
		// case 4: Note:- Deletion request is for trunk-vlans with an instance
		if swVlanConfig.swPortChannelMember.Config.TrunkVlans != nil {
			trunkVlansUnionList := swVlanConfig.swPortChannelMember.Config.TrunkVlans
			ifVlanInfo.mode = TRUNK

			for _, trunkVlanUnion := range trunkVlansUnionList {
				trunkVlanUnionType := reflect.TypeOf(trunkVlanUnion).Elem()

				switch trunkVlanUnionType {

				case reflect.TypeOf(ocbinds.OpenconfigInterfaces_Interfaces_Interface_Aggregation_SwitchedVlan_Config_TrunkVlans_Union_String{}):
					val := (trunkVlanUnion).(*ocbinds.OpenconfigInterfaces_Interfaces_Interface_Aggregation_SwitchedVlan_Config_TrunkVlans_Union_String)
					vlansList := strings.Split(val.String, ",")
					for _, vlan := range vlansList {
						/* Handle case if multiple/range of VLANs given */
						if strings.Contains(vlan, "..") {
							err = utils.ExtractVlanIdsFromRange(vlan, &ifVlanInfo.trunkVlans)
							if err != nil {
								return err
							}
						} else {
							vlanName := "Vlan" + vlan
							err = validateVlanExists(inParams.d, &vlanName)
							if err != nil {
								return err
							}
							ifVlanInfo.trunkVlans = append(ifVlanInfo.trunkVlans, vlanName)
						}
					}
				case reflect.TypeOf(ocbinds.OpenconfigInterfaces_Interfaces_Interface_Aggregation_SwitchedVlan_Config_TrunkVlans_Union_Uint16{}):
					val := (trunkVlanUnion).(*ocbinds.OpenconfigInterfaces_Interfaces_Interface_Aggregation_SwitchedVlan_Config_TrunkVlans_Union_Uint16)
					ifVlanInfo.trunkVlans = append(ifVlanInfo.trunkVlans, "Vlan"+strconv.Itoa(int(val.Uint16)))
				}
			}
		}
	}
	if ifVlanInfo.mode != MODE_UNSET {
		ifVlanInfo.ifName = ifName
		ifVlanInfoList = append(ifVlanInfoList, &ifVlanInfo)
	}
	err = processIntfVlanMemberRemoval(inParams, ifVlanInfoList, vlanMap, vlanMemberMap)
	if err != nil {
		log.Errorf("Interface VLAN member removal for Interface: %s failed!", *ifName)
		return err
	}
	return err
}

/* Function performs VLAN Member addition to Interface */
func intfVlanMemberAdd(swVlanConfig *swVlanMemberPort_t,
	inParams *XfmrParams, ifName *string,
	uriIfName *string,
	vlanMap map[string]db.Value,
	vlanMemberMap map[string]db.Value, intfType E_InterfaceType) error {

	var err error
	var accessVlanId uint16 = 0
	var trunkVlanSlice []string
	var accessVlan string
	var ifMode ocbinds.E_OpenconfigVlan_VlanModeType

	accessVlanFound := false
	trunkVlanFound := false

	intTbl := IntfTypeTblMap[IntfTypeVlan]

	vlanMembersListMap := make(map[string]map[string]db.Value)

	switch intfType {
	case IntfTypeEthernet:
		/* Retrieve the Access VLAN Id */
		if swVlanConfig.swEthMember == nil || swVlanConfig.swEthMember.Config == nil {
			errStr := "Not supported switched-vlan request for Interface: " + *ifName
			log.Error(errStr)
			return errors.New(errStr)
		}
		if swVlanConfig.swEthMember.Config.AccessVlan != nil {
			accessVlanId = *swVlanConfig.swEthMember.Config.AccessVlan
			log.Infof("Vlan id : %d observed for Untagged Member port addition configuration!", accessVlanId)
			accessVlanFound = true
		}

		/* Retrieve the list of trunk-vlans */
		if swVlanConfig.swEthMember.Config.TrunkVlans != nil {
			vlanUnionList := swVlanConfig.swEthMember.Config.TrunkVlans
			if len(vlanUnionList) != 0 {
				trunkVlanFound = true
			}
			for _, vlanUnion := range vlanUnionList {
				vlanUnionType := reflect.TypeOf(vlanUnion).Elem()

				switch vlanUnionType {

				case reflect.TypeOf(ocbinds.OpenconfigInterfaces_Interfaces_Interface_Ethernet_SwitchedVlan_Config_TrunkVlans_Union_String{}):
					val := (vlanUnion).(*ocbinds.OpenconfigInterfaces_Interfaces_Interface_Ethernet_SwitchedVlan_Config_TrunkVlans_Union_String)
					err = utils.ExtractVlanIdsFromRange(val.String, &trunkVlanSlice)
					if err != nil {
						return err
					}
				case reflect.TypeOf(ocbinds.OpenconfigInterfaces_Interfaces_Interface_Ethernet_SwitchedVlan_Config_TrunkVlans_Union_Uint16{}):
					val := (vlanUnion).(*ocbinds.OpenconfigInterfaces_Interfaces_Interface_Ethernet_SwitchedVlan_Config_TrunkVlans_Union_Uint16)
					trunkVlanSlice = append(trunkVlanSlice, "Vlan"+strconv.Itoa(int(val.Uint16)))
				}
			}
		}
		if swVlanConfig.swEthMember.Config.InterfaceMode != ocbinds.OpenconfigVlan_VlanModeType_UNSET {
			ifMode = swVlanConfig.swEthMember.Config.InterfaceMode
		}
	case IntfTypePortChannel:
		/* Retrieve the Access VLAN Id */
		if swVlanConfig.swPortChannelMember == nil || swVlanConfig.swPortChannelMember.Config == nil {
			errStr := "Not supported switched-vlan request for Interface: " + *ifName
			log.Error(errStr)
			return errors.New(errStr)
		}
		if swVlanConfig.swPortChannelMember.Config.AccessVlan != nil {
			accessVlanId = *swVlanConfig.swPortChannelMember.Config.AccessVlan
			log.Infof("Vlan id : %d observed for Untagged Member port addition configuration!", accessVlanId)
			accessVlanFound = true
		}

		/* Retrieve the list of trunk-vlans */
		if swVlanConfig.swPortChannelMember.Config.TrunkVlans != nil {
			vlanUnionList := swVlanConfig.swPortChannelMember.Config.TrunkVlans
			if len(vlanUnionList) != 0 {
				trunkVlanFound = true
			}
			for _, vlanUnion := range vlanUnionList {
				vlanUnionType := reflect.TypeOf(vlanUnion).Elem()

				switch vlanUnionType {

				case reflect.TypeOf(ocbinds.OpenconfigInterfaces_Interfaces_Interface_Aggregation_SwitchedVlan_Config_TrunkVlans_Union_String{}):
					val := (vlanUnion).(*ocbinds.OpenconfigInterfaces_Interfaces_Interface_Aggregation_SwitchedVlan_Config_TrunkVlans_Union_String)
					err = utils.ExtractVlanIdsFromRange(val.String, &trunkVlanSlice)
					if err != nil {
						return err
					}
				case reflect.TypeOf(ocbinds.OpenconfigInterfaces_Interfaces_Interface_Aggregation_SwitchedVlan_Config_TrunkVlans_Union_Uint16{}):
					val := (vlanUnion).(*ocbinds.OpenconfigInterfaces_Interfaces_Interface_Aggregation_SwitchedVlan_Config_TrunkVlans_Union_Uint16)
					trunkVlanSlice = append(trunkVlanSlice, "Vlan"+strconv.Itoa(int(val.Uint16)))
				}
			}
		}
		if swVlanConfig.swPortChannelMember.Config.InterfaceMode != ocbinds.OpenconfigVlan_VlanModeType_UNSET {
			ifMode = swVlanConfig.swPortChannelMember.Config.InterfaceMode
		}
	}
	/* Update the DS based on access-vlan/trunk-vlans config */
	if accessVlanFound {
		accessVlan = "Vlan" + strconv.Itoa(int(accessVlanId))
		var cfgredAccessVlan string
		log.Info(accessVlan)

		exists, err := validateUntaggedVlanCfgredForIf(inParams.d, &intTbl.cfgDb.memberTN, ifName, &cfgredAccessVlan)
		if err != nil {
			return err
		}
		if exists {
			if cfgredAccessVlan == accessVlan {
				log.Infof("Untagged VLAN: %s already configured, not updating the cache!", accessVlan)
				goto TRUNKCONFIG
			}
			//Replace existing untagged vlan config(cfgredAccessVlan) with new config
			del_res_map := make(map[string]map[string]db.Value)
			vlanMapDel := make(map[string]db.Value)
			vlanMemberMapDel := make(map[string]db.Value)
			untagdVlan, err := removeUntaggedVlanAndUpdateVlanMembTbl(inParams.d, ifName, vlanMemberMapDel)
			if err != nil {
				return err
			}

			// Update vlanMapDel with access VLAN
			if untagdVlan != nil {
				ts := db.TableSpec{Name: intTbl.cfgDb.memberTN + inParams.d.Opts.KeySeparator + *untagdVlan}
				memberKeys, err := inParams.d.GetKeys(&ts)

				memberFound := false
				if err == nil {
					for key := range memberKeys {
						if memberKeys[key].Get(1) == *ifName {
							memberFound = true
							break
						}
					}
					if memberFound {
						vlanMapDel[*untagdVlan] = db.Value{Field: make(map[string]string)}
					}
				}

				// // TODO: Or only this is necessary
				// vlanMapDel[*untagdVlan] = db.Value{Field: make(map[string]string)}
			}

			if len(vlanMemberMapDel) != 0 {
				del_res_map[VLAN_MEMBER_TN] = vlanMemberMapDel
			}
			if len(vlanMapDel) != 0 {
				del_res_map[VLAN_TN] = vlanMapDel
			}
			vlanId := cfgredAccessVlan[len("Vlan"):]

			if inParams.subOpDataMap[DELETE] != nil && (*inParams.subOpDataMap[DELETE])[db.ConfigDB] != nil {
				if map_val, exists := (*inParams.subOpDataMap[DELETE])[db.ConfigDB][VLAN_TN]; exists {
					for vlanName := range vlanMapDel {
						if _, ok := map_val[vlanName]; !ok {
							map_val[vlanName] = db.Value{Field: make(map[string]string)}
						}
					}
					del_res_map[VLAN_TN] = map_val
				}
				mapCopy((*inParams.subOpDataMap[DELETE])[db.ConfigDB], del_res_map)
			} else {
				del_subOpMap := make(map[db.DBNum]map[string]map[string]db.Value)
				del_subOpMap[db.ConfigDB] = del_res_map
				inParams.subOpDataMap[DELETE] = &del_subOpMap
			}
			log.Info("Removing existing untagged VLAN: "+vlanId+" configuration, vlan delete subopmap:", (*inParams.subOpDataMap[DELETE])[db.ConfigDB])

		}
		err = validateVlanExists(inParams.d, &accessVlan)
		if err == nil {
			//If VLAN exists add to vlanMembersListMap
			if vlanMembersListMap[accessVlan] == nil {
				vlanMembersListMap[accessVlan] = make(map[string]db.Value)
			}
			vlanMembersListMap[accessVlan][*ifName] = db.Value{Field: make(map[string]string)}
			vlanMembersListMap[accessVlan][*ifName].Field["tagging_mode"] = "untagged"
		}
	}

TRUNKCONFIG:
	if trunkVlanFound {
		memberPortEntryMap := make(map[string]string)
		memberPortEntry := db.Value{Field: memberPortEntryMap}
		memberPortEntry.Field["tagging_mode"] = "tagged"
		for _, vlanId := range trunkVlanSlice {
			vlanName := vlanId
			log.Infof("%s", vlanName)

			err = validateVlanExists(inParams.d, &vlanId)
			if err == nil {
				if vlanMembersListMap[vlanId] == nil {
					vlanMembersListMap[vlanId] = make(map[string]db.Value)
				}
				vlanMembersListMap[vlanId][*ifName] = db.Value{Field: make(map[string]string)}
				vlanMembersListMap[vlanId][*ifName].Field["tagging_mode"] = "tagged"
			}
		}
	}

	if accessVlanFound || trunkVlanFound {
		err = processIntfVlanMemberAdd(inParams.d, vlanMembersListMap, vlanMap, vlanMemberMap)
		if err != nil {
			log.Info("Processing Interface VLAN addition failed!")
			return err
		}
		return err
	}

	if ifMode == ocbinds.OpenconfigVlan_VlanModeType_UNSET {
		return nil
	}
	/* Handling the request just for setting Interface Mode */
	log.Info("Request is for Configuring just the Mode for Interface: ", *ifName)
	var mode intfModeReq

	switch ifMode {
	case ocbinds.OpenconfigVlan_VlanModeType_ACCESS:
		/* Configuring Interface Mode as ACCESS only without VLAN info*/
		mode = intfModeReq{ifName: *ifName, mode: ACCESS}
		log.Info("Access Mode Config for Interface: ", *ifName)
	case ocbinds.OpenconfigVlan_VlanModeType_TRUNK:
	}
	/* Switchport access/trunk mode config without VLAN */
	/* This mode will be set in the translate fn, when request is just for mode without VLAN info. */
	if mode.mode != MODE_UNSET {
		err = intfModeReqConfig(inParams.d, mode, vlanMap, vlanMemberMap)
		if err != nil {
			return err
		}
	}
	return nil
}

/* Function performs VLAN Member replace to Interface */
func intfVlanMemberReplace(swVlanConfig *swVlanMemberPort_t,
	inParams *XfmrParams, ifName *string,
	vlanMap map[string]db.Value,
	vlanMemberMap map[string]db.Value,
	intfType E_InterfaceType) error {

	var err error
	var accessVlanId uint16 = 0
	var trunkVlanSlice []string
	var accessVlan string

	accessVlanFound := false
	trunkVlanFound := false

	vlanMembersListMap := make(map[string]map[string]db.Value)

	var cfgredTaggedVlan []string
	var cfgredAccessVlan string

	accessVlanInPath := true
	trunkVlanInPath := true

	// check the request uri path to see if need to handle accessVlan or trunkVlan under switched-vlan
	xpath, _, _ := XfmrRemoveXPATHPredicates(inParams.requestUri)
	log.V(3).Info("intfVlanMemberReplace, xpath: ", xpath)

	if (xpath == "/openconfig-interfaces:interfaces/interface/ethernet/switched-vlan/config/access-vlan") ||
		(xpath == "/openconfig-interfaces:interfaces/interface/aggregation/switched-vlan/config/access-vlan") {
		trunkVlanInPath = false
	}
	if (xpath == "/openconfig-interfaces:interfaces/interface/ethernet/switched-vlan/config/trunk-vlans") ||
		(xpath == "/openconfig-interfaces:interfaces/interface/aggregation/switched-vlan/config/trunk-vlans") {
		accessVlanInPath = false
	}

	switch intfType {
	case IntfTypeEthernet:
		/* Retrieve the Access VLAN Id */
		if swVlanConfig.swEthMember == nil || swVlanConfig.swEthMember.Config == nil {
			errStr := "Not supported switched-vlan request for Interface: " + *ifName
			log.Error(errStr)
			return errors.New(errStr)
		}
		if swVlanConfig.swEthMember.Config.AccessVlan != nil {
			accessVlanId = *swVlanConfig.swEthMember.Config.AccessVlan
			log.Infof("Vlan id : %d observed for Untagged Member port addition configuration!", accessVlanId)
			accessVlanFound = true
		}

		/* Retrieve the list of trunk-vlans */
		if swVlanConfig.swEthMember.Config.TrunkVlans != nil {
			vlanUnionList := swVlanConfig.swEthMember.Config.TrunkVlans
			if len(vlanUnionList) != 0 {
				trunkVlanFound = true
			}
			for _, vlanUnion := range vlanUnionList {
				vlanUnionType := reflect.TypeOf(vlanUnion).Elem()

				switch vlanUnionType {

				case reflect.TypeOf(ocbinds.OpenconfigInterfaces_Interfaces_Interface_Ethernet_SwitchedVlan_Config_TrunkVlans_Union_String{}):
					val := (vlanUnion).(*ocbinds.OpenconfigInterfaces_Interfaces_Interface_Ethernet_SwitchedVlan_Config_TrunkVlans_Union_String)
					err = utils.ExtractVlanIdsFromRange(val.String, &trunkVlanSlice)
					if err != nil {
						return err
					}
				case reflect.TypeOf(ocbinds.OpenconfigInterfaces_Interfaces_Interface_Ethernet_SwitchedVlan_Config_TrunkVlans_Union_Uint16{}):
					val := (vlanUnion).(*ocbinds.OpenconfigInterfaces_Interfaces_Interface_Ethernet_SwitchedVlan_Config_TrunkVlans_Union_Uint16)
					trunkVlanSlice = append(trunkVlanSlice, "Vlan"+strconv.Itoa(int(val.Uint16)))
				}
			}
		}
	case IntfTypePortChannel:
		/* Retrieve the Access VLAN Id */
		if swVlanConfig.swPortChannelMember == nil || swVlanConfig.swPortChannelMember.Config == nil {
			errStr := "Not supported switched-vlan request for Interface: " + *ifName
			log.Error(errStr)
			return errors.New(errStr)
		}
		if swVlanConfig.swPortChannelMember.Config.AccessVlan != nil {
			accessVlanId = *swVlanConfig.swPortChannelMember.Config.AccessVlan
			log.Infof("Vlan id : %d observed for Untagged Member port addition configuration!", accessVlanId)
			accessVlanFound = true
		}

		/* Retrieve the list of trunk-vlans */
		if swVlanConfig.swPortChannelMember.Config.TrunkVlans != nil {
			vlanUnionList := swVlanConfig.swPortChannelMember.Config.TrunkVlans
			if len(vlanUnionList) != 0 {
				trunkVlanFound = true
			}
			for _, vlanUnion := range vlanUnionList {
				vlanUnionType := reflect.TypeOf(vlanUnion).Elem()

				switch vlanUnionType {

				case reflect.TypeOf(ocbinds.OpenconfigInterfaces_Interfaces_Interface_Aggregation_SwitchedVlan_Config_TrunkVlans_Union_String{}):
					val := (vlanUnion).(*ocbinds.OpenconfigInterfaces_Interfaces_Interface_Aggregation_SwitchedVlan_Config_TrunkVlans_Union_String)
					err = utils.ExtractVlanIdsFromRange(val.String, &trunkVlanSlice)
					if err != nil {
						return err
					}
				case reflect.TypeOf(ocbinds.OpenconfigInterfaces_Interfaces_Interface_Aggregation_SwitchedVlan_Config_TrunkVlans_Union_Uint16{}):
					val := (vlanUnion).(*ocbinds.OpenconfigInterfaces_Interfaces_Interface_Aggregation_SwitchedVlan_Config_TrunkVlans_Union_Uint16)
					trunkVlanSlice = append(trunkVlanSlice, "Vlan"+strconv.Itoa(int(val.Uint16)))
				}
			}
		}
	}

	log.V(3).Infof("intfVlanMemberReplace, accessVlanId: %v, trunkVlanSlice:%v", accessVlanId, trunkVlanSlice)

	//Get existing tagged and untagged vlan config on interface
	if !accessVlanInPath {
		cfgredTaggedVlan, _, _ = getIntfVlanConfig(inParams.d, VLAN_MEMBER_TN, *ifName)
	} else if !trunkVlanInPath {
		_, cfgredAccessVlan, _ = getIntfVlanConfig(inParams.d, VLAN_MEMBER_TN, *ifName)
	} else {
		cfgredTaggedVlan, cfgredAccessVlan, _ = getIntfVlanConfig(inParams.d, VLAN_MEMBER_TN, *ifName)
	}

	log.V(3).Infof("intfVlanMemberReplace, cfgredAccessVlan: %v, cfgredTaggedVlan: %v", cfgredAccessVlan, cfgredTaggedVlan)

	delTrunkVlansList := utils.VlanDifference(cfgredTaggedVlan, trunkVlanSlice)
	log.V(3).Info("REPLACE oper - delTrunkVlansList: ", delTrunkVlansList)
	addTrunkVlansList := utils.VlanDifference(trunkVlanSlice, cfgredTaggedVlan)
	log.V(3).Info("REPLACE oper - addTrunkVlansList: ", addTrunkVlansList)

	vlanMapDel := make(map[string]db.Value)
	vlanMemberMapDel := make(map[string]db.Value)

	del_res_map := make(map[string]map[string]db.Value)
	add_res_map := make(map[string]map[string]db.Value)

	if accessVlanInPath {
		newAccessVlanFound := cfgredAccessVlan == ""
		accessVlan = ""

		if accessVlanFound {
			accessVlan = "Vlan" + strconv.Itoa(int(accessVlanId))

			err = validateVlanExists(inParams.d, &accessVlan)
			if err == nil {
				if cfgredAccessVlan != accessVlan {
					newAccessVlanFound = true
				}
				if newAccessVlanFound {
					// If new accessVlanExist, add it
					//Adding VLAN to be configured(accessVlan) to the vlanMembersListMap
					if vlanMembersListMap[accessVlan] == nil {
						vlanMembersListMap[accessVlan] = make(map[string]db.Value)
					}
					vlanMembersListMap[accessVlan][*ifName] = db.Value{Field: make(map[string]string)}
					vlanMembersListMap[accessVlan][*ifName].Field["tagging_mode"] = "untagged"
				}
			}
		}
		if cfgredAccessVlan != "" {
			if cfgredAccessVlan == accessVlan {
				log.Infof("Untagged VLAN: %s already configured, not updating the cache!", accessVlan)
				goto TRUNKCONFIG
			}

			//Delete existing untagged vlan config(cfgredAccessVlan)
			_, err := removeUntaggedVlanAndUpdateVlanMembTbl(inParams.d, ifName, vlanMemberMapDel)
			if err != nil {
				return err
			}
		}
	}

TRUNKCONFIG:

	if trunkVlanInPath {
		memberPortEntryMap := make(map[string]string)
		memberPortEntry := db.Value{Field: memberPortEntryMap}
		memberPortEntry.Field["tagging_mode"] = "tagged"
		//Update vlanMembersListMap with trunk vlans to be configured
		for _, vlanName := range addTrunkVlansList {
			err = validateVlanExists(inParams.d, &vlanName)
			if err == nil && accessVlan != vlanName {
				//Update vlanMembersListMap if the VLAN exists and there is no conflicting untagged configuration.
				if vlanMembersListMap[vlanName] == nil {
					vlanMembersListMap[vlanName] = make(map[string]db.Value)
				}
				vlanMembersListMap[vlanName][*ifName] = db.Value{Field: make(map[string]string)}
				vlanMembersListMap[vlanName][*ifName].Field["tagging_mode"] = "tagged"
			}
		}

		//Delete existing Vlans already configured and are not in VLANs to be configured list
		if len(cfgredTaggedVlan) != 0 {
			//Not including the vlans to be configured in the delete map
			for _, vlan := range delTrunkVlansList {
				err = removeTaggedVlanAndUpdateVlanMembTbl(inParams.d, &vlan, ifName, vlanMemberMapDel)
				if err != nil {
					return err
				}
			}
		}
	}

	if len(vlanMemberMapDel) != 0 {
		del_res_map[VLAN_MEMBER_TN] = vlanMemberMapDel
	}
	if len(vlanMapDel) != 0 {
		del_res_map[VLAN_TN] = vlanMapDel
	}

	del_subOpMap := make(map[db.DBNum]map[string]map[string]db.Value)
	del_subOpMap[db.ConfigDB] = del_res_map
	inParams.subOpDataMap[DELETE] = &del_subOpMap
	log.V(3).Info("REPLACE oper - vlan delete subopmap:", del_subOpMap)

	if accessVlanFound || trunkVlanFound {
		err = processIntfVlanMemberAdd(inParams.d, vlanMembersListMap, vlanMap, vlanMemberMap)
		if err != nil {
			log.Error("Processing Interface VLAN addition failed!")
			return err
		}
		if len(vlanMemberMap) != 0 {
			add_res_map[VLAN_MEMBER_TN] = vlanMemberMap
		}
		if len(vlanMap) != 0 {
			add_res_map[VLAN_TN] = vlanMap
		}

		add_subOpMap := make(map[db.DBNum]map[string]map[string]db.Value)
		add_subOpMap[db.ConfigDB] = add_res_map
		inParams.subOpDataMap[UPDATE] = &add_subOpMap
		return err
	}

	return nil
}

/* Function to delete VLAN and all its member ports */
func deleteVlanIntfAndMembers(inParams *XfmrParams, vlanName *string) error {
	var err error
	subOpMap := make(map[db.DBNum]map[string]map[string]db.Value)
	resMap := make(map[string]map[string]db.Value)
	vlanMap := make(map[string]db.Value)
	vlanIntfMap := make(map[string]db.Value)
	vlanMemberMap := make(map[string]db.Value)

	intTbl := IntfTypeTblMap[IntfTypeVlan]
	vlanMap[*vlanName] = db.Value{Field: map[string]string{}}
	subOpMap[db.ConfigDB] = resMap
	inParams.subOpDataMap[DELETE] = &subOpMap

	_, err = inParams.d.GetEntry(&db.TableSpec{Name: VLAN_TN}, db.Key{Comp: []string{*vlanName}})
	if err != nil {
		errStr := "Retrieving data from VLAN table for VLAN: " + *vlanName + " failed!"
		log.Error(errStr)
		// Not returning error from here since mgmt infra will return "Resource not found" error in case of non existence entries
		return nil
	}
	/* Validation is needed, if oper is not DELETE. Cleanup for sub-interfaces is done as part of Delete. */
	if inParams.oper != DELETE {
		err = validateL3ConfigExists(inParams.d, vlanName)
		if err != nil {
			return err
		}
	}

	/* Handle VLAN_MEMBER TABLE */
	var flag bool = false
	ts := db.TableSpec{Name: intTbl.cfgDb.memberTN + inParams.d.Opts.KeySeparator + *vlanName}
	memberKeys, err := inParams.d.GetKeys(&ts)
	if err == nil {
		for key := range memberKeys {
			flag = true
			log.Info("Member port", memberKeys[key].Get(1))
			memberKey := *vlanName + "|" + memberKeys[key].Get(1)
			vlanMemberMap[memberKey] = db.Value{Field: map[string]string{}}
		}
		if flag {
			resMap[VLAN_MEMBER_TN] = vlanMemberMap
		}
	}

	/* Handle VLAN_INTERFACE TABLE */
	processIntfTableRemoval(inParams.d, *vlanName, VLAN_INTERFACE_TN, vlanIntfMap)
	if len(vlanIntfMap) != 0 {
		resMap[VLAN_INTERFACE_TN] = vlanIntfMap
	}

	if len(vlanMap) != 0 {
		resMap[VLAN_TN] = vlanMap
	}
	subOpMap[db.ConfigDB] = resMap
	inParams.subOpDataMap[DELETE] = &subOpMap
	return err
}

// YangToDb_sw_vlans_xfmr is a Yang to DB Subtree transformer supports CREATE, UPDATE and DELETE operations
var YangToDb_sw_vlans_xfmr SubTreeXfmrYangToDb = func(inParams XfmrParams) (map[string]map[string]db.Value, error) {
	var err error
	res_map := make(map[string]map[string]db.Value)
	vlanMap := make(map[string]db.Value)
	vlanMemberMap := make(map[string]db.Value)
	log.Info("YangToDb_sw_vlans_xfmr: ", inParams.uri)

	var swVlanConfig swVlanMemberPort_t
	pathInfo := NewPathInfo(inParams.uri)
	uriIfName := pathInfo.Var("name")
	ifName := uriIfName

	deviceObj := (*inParams.ygRoot).(*ocbinds.Device)
	intfObj := deviceObj.Interfaces

	log.Info("Switched vlans request for ", ifName)
	intf := intfObj.Interface[uriIfName]

	intfType, _, err := getIntfTypeByName(ifName)
	if err != nil {
		errStr := "Extraction of Interface type from Interface: " + ifName + " failed!"
		return nil, errors.New(errStr)
	}
	if intfType != IntfTypeEthernet && intfType != IntfTypePortChannel {
		return nil, nil
	}

	/* Set invokeCRUSubtreeOnce flag to invoke subtree once */
	if inParams.invokeCRUSubtreeOnce != nil {
		*inParams.invokeCRUSubtreeOnce = true
	}

	if (inParams.oper == DELETE) && ((intf.Ethernet == nil || intf.Ethernet.SwitchedVlan == nil ||
		intf.Ethernet.SwitchedVlan.Config == nil) && (intf.Aggregation == nil || intf.Aggregation.SwitchedVlan == nil ||
		intf.Aggregation.SwitchedVlan.Config == nil)) {
		err = intfVlanMemberRemoval(&swVlanConfig, &inParams, &ifName, vlanMap, vlanMemberMap, intfType)
		if err != nil {
			log.Errorf("Interface VLAN member port removal failed for Interface: %s!", ifName)
			return nil, err
		}
		if len(vlanMemberMap) != 0 {
			res_map[VLAN_MEMBER_TN] = vlanMemberMap
		}
		if len(vlanMap) != 0 {
			res_map[VLAN_TN] = vlanMap
		}
		return res_map, err
	}

	if intf.Ethernet == nil && intf.Aggregation == nil {
		return nil, errors.New("Wrong Config Request")
	}
	if intf.Ethernet != nil {
		if intf.Ethernet.SwitchedVlan == nil || intf.Ethernet.SwitchedVlan.Config == nil {
			return nil, errors.New("Wrong config request for Ethernet!")
		}
		swVlanConfig.swEthMember = intf.Ethernet.SwitchedVlan
		if inParams.oper == REPLACE || inParams.oper == UPDATE {
			if swVlanConfig.swEthMember.Config.TrunkVlans != nil {
				vlanUnionList := swVlanConfig.swEthMember.Config.TrunkVlans
				if len(vlanUnionList) == 0 {
					log.Errorf("patch/replace operation not supported with empty trunk vlans ; ifname %s!", ifName)
					return nil, errors.New("patch/replace operation not supported with empty trunk vlans !")
				}
			}
		}
	}
	if intf.Aggregation != nil {
		if intf.Aggregation.SwitchedVlan == nil || intf.Aggregation.SwitchedVlan.Config == nil {
			return nil, errors.New("Wrong Config Request for Port Channel")
		}
		swVlanConfig.swPortChannelMember = intf.Aggregation.SwitchedVlan
		if inParams.oper == REPLACE || inParams.oper == UPDATE {
			if swVlanConfig.swPortChannelMember.Config.TrunkVlans != nil {
				vlanUnionList := swVlanConfig.swPortChannelMember.Config.TrunkVlans
				if len(vlanUnionList) == 0 {
					log.Errorf("patch/replace operation not supported with empty trunk vlans ; ifname %s!", ifName)
					return nil, errors.New("patch/replace operation not supported with empty trunk vlans !")
				}
			}
		}
	}

	switch inParams.oper {
	case REPLACE:
		err = intfVlanMemberReplace(&swVlanConfig, &inParams, &ifName, vlanMap, vlanMemberMap, intfType)
		if err != nil {
			log.Errorf("Interface VLAN member port replace failed for Interface: %s!", ifName)
			return nil, err
		}

	case CREATE:
		fallthrough
	case UPDATE:
		err = intfVlanMemberAdd(&swVlanConfig, &inParams, &ifName, &uriIfName, vlanMap, vlanMemberMap, intfType)
		if err != nil {
			log.Errorf("Interface VLAN member port addition failed for Interface: %s!", ifName)
			return nil, err
		}
		if len(vlanMap) != 0 {
			res_map[VLAN_TN] = vlanMap
			if inParams.subOpDataMap[inParams.oper] != nil && (*inParams.subOpDataMap[inParams.oper])[db.ConfigDB] != nil {
				map_val := (*inParams.subOpDataMap[inParams.oper])[db.ConfigDB][VLAN_TN]
				for vlanName := range vlanMap {
					if _, ok := map_val[vlanName]; !ok {
						map_val[vlanName] = db.Value{Field: make(map[string]string)}
					}
				}
			} else {
				subOpMap := make(map[db.DBNum]map[string]map[string]db.Value)
				subOpMap[db.ConfigDB] = res_map
				inParams.subOpDataMap[inParams.oper] = &subOpMap
			}
		}

		if len(vlanMemberMap) != 0 { //make sure this map filled only with vlans existing
			res_map[VLAN_MEMBER_TN] = vlanMemberMap
		}

	case DELETE:
		err = intfVlanMemberRemoval(&swVlanConfig, &inParams, &ifName, vlanMap, vlanMemberMap, intfType)
		if err != nil {
			log.Errorf("Interface VLAN member port removal failed for Interface: %s!", ifName)
			return nil, err
		}
		if len(vlanMemberMap) != 0 {
			res_map[VLAN_MEMBER_TN] = vlanMemberMap
		}
		if len(vlanMap) != 0 {
			res_map[VLAN_TN] = vlanMap
		}
	}
	log.Info("YangToDb_sw_vlans_xfmr: vlan res map:", res_map)
	log.Info("YangToDb_sw_vlans_xfmr: inParams.subOpDataMap UPDATE: ", inParams.subOpDataMap[UPDATE])
	log.Info("YangToDb_sw_vlans_xfmr: inParams.subOpDataMap DELETE: ", inParams.subOpDataMap[DELETE])
	log.Info("YangToDb_sw_vlans_xfmr: inParams.subOpDataMap REPLACE: ", inParams.subOpDataMap[REPLACE])
	return res_map, err
}

func fillDBSwitchedVlanInfoForIntf(d *db.DB, ifName *string, vlanMemberMap map[string]map[string]db.Value) error {
	if log.V(5) {
		log.Info("fillDBSwitchedVlanInfoForIntf() called!")
	}
	var err error

	vlanMemberKeys, err := d.GetKeysByPattern(&db.TableSpec{Name: VLAN_MEMBER_TN}, "*"+*ifName)
	if err != nil {
		return err
	}
	if log.V(5) {
		log.Infof("Found %d vlan-member-table keys", len(vlanMemberKeys))
	}

	for _, vlanMember := range vlanMemberKeys {
		if len(vlanMember.Comp) < 2 {
			continue
		}
		vlanId := vlanMember.Get(0)
		ifName := vlanMember.Get(1)
		if log.V(5) {
			log.Infof("Received Vlan: %s for Interface: %s", vlanId, ifName)
		}

		memberPortEntry, err := d.GetEntry(&db.TableSpec{Name: VLAN_MEMBER_TN}, vlanMember)
		if err != nil {
			return err
		}
		if !memberPortEntry.IsPopulated() {
			errStr := "Tagging Info not present for Vlan: " + vlanId + " Interface: " + ifName + " from VLAN_MEMBER_TABLE"
			return errors.New(errStr)
		}

		/* vlanMembersTableMap is used as DS for ifName to list of VLANs */
		if vlanMemberMap[ifName] == nil {
			vlanMemberMap[ifName] = make(map[string]db.Value)
			vlanMemberMap[ifName][vlanId] = memberPortEntry
		} else {
			vlanMemberMap[ifName][vlanId] = memberPortEntry
		}
	}
	if log.V(5) {
		log.Infof("Updated the vlan-member-table ds for Interface: %s", *ifName)
	}
	return err
}

func getIntfVlanAttr(ifName *string, ifMode intfModeType, vlanMemberMap map[string]map[string]db.Value) ([]string, *string, error) {

	if log.V(5) {
		log.Info("getIntfVlanAttr() called")
	}
	vlanEntries, ok := vlanMemberMap[*ifName]
	if !ok {
		errStr := "Cannot find info for Interface: " + *ifName + " from VLAN_MEMBERS_TABLE!"
		log.Info(errStr)
		return nil, nil, nil
	}
	switch ifMode {
	case ACCESS:
		for vlanKey, tagEntry := range vlanEntries {
			tagMode, ok := tagEntry.Field["tagging_mode"]
			if ok {
				if tagMode == "untagged" {
					log.Info("Untagged VLAN found!")
					return nil, &vlanKey, nil
				}
			}
		}
	case TRUNK:
		var trunkVlans []string
		for vlanKey, tagEntry := range vlanEntries {
			tagMode, ok := tagEntry.Field["tagging_mode"]
			if ok {
				if tagMode == "tagged" {
					trunkVlans = append(trunkVlans, vlanKey)
				}
			}
		}
		return trunkVlans, nil, nil
	}
	return nil, nil, nil
}

func getSpecificSwitchedVlanStateAttr(targetUriPath *string, ifKey *string,
	vlanMemberMap map[string]map[string]db.Value,
	swVlan *swVlanMemberPort_t, intfType E_InterfaceType) (bool, error) {
	log.Info("Specific Switched-vlan attribute!")
	var config bool = true
	switch *targetUriPath {
	case "/openconfig-interfaces:interfaces/interface/openconfig-if-ethernet:ethernet/openconfig-vlan:switched-vlan/state/access-vlan":
		fallthrough
	case "/openconfig-interfaces:interfaces/interface/openconfig-if-aggregate:aggregation/openconfig-vlan:switched-vlan/state/access-vlan":
		config = false
		fallthrough
	case "/openconfig-interfaces:interfaces/interface/openconfig-if-ethernet:ethernet/openconfig-vlan:switched-vlan/config/access-vlan":
		fallthrough
	case "/openconfig-interfaces:interfaces/interface/openconfig-if-aggregate:aggregation/openconfig-vlan:switched-vlan/config/access-vlan":

		_, accessVlanName, e := getIntfVlanAttr(ifKey, ACCESS, vlanMemberMap)
		if e != nil {
			return true, e
		}
		if accessVlanName == nil {
			return true, nil
		}
		log.Info("Access VLAN - ", accessVlanName)
		vlanName := *accessVlanName
		vlanIdStr := vlanName[len("Vlan"):]
		vlanId, err := strconv.Atoi(vlanIdStr)
		if err != nil {
			errStr := "Conversion of string to int failed for " + vlanIdStr
			return true, errors.New(errStr)
		}
		vlanIdCast := uint16(vlanId)

		switch intfType {
		case IntfTypeEthernet:
			if config {
				swVlan.swEthMember.Config.AccessVlan = &vlanIdCast
			} else {
				if swVlan.swEthMember.State == nil {
					ygot.BuildEmptyTree(swVlan.swEthMember)
				}
				swVlan.swEthMember.State.AccessVlan = &vlanIdCast
			}
		case IntfTypePortChannel:
			if config {
				swVlan.swPortChannelMember.Config.AccessVlan = &vlanIdCast
			} else {
				if swVlan.swPortChannelMember.State == nil {
					ygot.BuildEmptyTree(swVlan.swPortChannelMember)
				}
				swVlan.swPortChannelMember.State.AccessVlan = &vlanIdCast
			}
		}
		return true, nil
	case "/openconfig-interfaces:interfaces/interface/openconfig-if-ethernet:ethernet/openconfig-vlan:switched-vlan/state/trunk-vlans":
		fallthrough
	case "/openconfig-interfaces:interfaces/interface/openconfig-if-aggregate:aggregation/openconfig-vlan:switched-vlan/state/trunk-vlans":
		config = false
		fallthrough
	case "/openconfig-interfaces:interfaces/interface/openconfig-if-ethernet:ethernet/openconfig-vlan:switched-vlan/config/trunk-vlans":
		fallthrough
	case "/openconfig-interfaces:interfaces/interface/openconfig-if-aggregate:aggregation/openconfig-vlan:switched-vlan/config/trunk-vlans":

		trunkVlans, _, e := getIntfVlanAttr(ifKey, TRUNK, vlanMemberMap)
		if e != nil {
			return true, e
		}

		switch intfType {
		case IntfTypeEthernet:

			for _, vlanName := range trunkVlans {
				log.Info("Trunk VLAN - ", vlanName)
				vlanIdStr := vlanName[len("Vlan"):]
				vlanId, err := strconv.Atoi(vlanIdStr)
				if err != nil {
					errStr := "Conversion of string to int failed for " + vlanIdStr
					return true, errors.New(errStr)
				}
				vlanIdCast := uint16(vlanId)
				if config {
					trunkVlan, _ := swVlan.swEthMember.Config.To_OpenconfigInterfaces_Interfaces_Interface_Ethernet_SwitchedVlan_Config_TrunkVlans_Union(vlanIdCast)
					swVlan.swEthMember.Config.TrunkVlans = append(swVlan.swEthMember.Config.TrunkVlans, trunkVlan)
				} else {
					trunkVlan, _ := swVlan.swEthMember.State.To_OpenconfigInterfaces_Interfaces_Interface_Ethernet_SwitchedVlan_State_TrunkVlans_Union(vlanIdCast)
					swVlan.swEthMember.State.TrunkVlans = append(swVlan.swEthMember.State.TrunkVlans, trunkVlan)
				}
			}
		case IntfTypePortChannel:
			for _, vlanName := range trunkVlans {
				log.Info("Trunk VLAN - ", vlanName)
				vlanIdStr := vlanName[len("Vlan"):]
				vlanId, err := strconv.Atoi(vlanIdStr)
				if err != nil {
					errStr := "Conversion of string to int failed for " + vlanIdStr
					return true, errors.New(errStr)
				}
				vlanIdCast := uint16(vlanId)
				if config {
					trunkVlan, _ := swVlan.swPortChannelMember.Config.To_OpenconfigInterfaces_Interfaces_Interface_Aggregation_SwitchedVlan_Config_TrunkVlans_Union(vlanIdCast)
					swVlan.swPortChannelMember.Config.TrunkVlans = append(swVlan.swPortChannelMember.Config.TrunkVlans, trunkVlan)
				} else {
					trunkVlan, _ := swVlan.swPortChannelMember.State.To_OpenconfigInterfaces_Interfaces_Interface_Aggregation_SwitchedVlan_State_TrunkVlans_Union(vlanIdCast)
					swVlan.swPortChannelMember.State.TrunkVlans = append(swVlan.swPortChannelMember.State.TrunkVlans, trunkVlan)
				}
			}
		}
		return true, nil
	case "/openconfig-interfaces:interfaces/interface/openconfig-if-ethernet:ethernet/openconfig-vlan:switched-vlan/state/interface-mode":
		fallthrough
	case "/openconfig-interfaces:interfaces/interface/openconfig-if-aggregate:aggregation/openconfig-vlan:switched-vlan/state/interface-mode":
		config = false
		fallthrough
	case "/openconfig-interfaces:interfaces/interface/openconfig-if-ethernet:ethernet/openconfig-vlan:switched-vlan/config/interface-mode":
		fallthrough
	case "/openconfig-interfaces:interfaces/interface/openconfig-if-aggregate:aggregation/openconfig-vlan:switched-vlan/config/interface-mode":

		_, accessVlanName, e := getIntfVlanAttr(ifKey, ACCESS, vlanMemberMap)
		if e != nil {
			return true, e
		}

		trunkVlans, _, e := getIntfVlanAttr(ifKey, TRUNK, vlanMemberMap)
		if e != nil {
			return true, e
		}

		switch intfType {
		case IntfTypeEthernet:
			if accessVlanName != nil {
				if config {
					swVlan.swEthMember.Config.InterfaceMode = ocbinds.OpenconfigVlan_VlanModeType_ACCESS
				} else {
					swVlan.swEthMember.State.InterfaceMode = ocbinds.OpenconfigVlan_VlanModeType_ACCESS
				}
			}
			if len(trunkVlans) > 0 {
				if config {
					swVlan.swEthMember.Config.InterfaceMode = ocbinds.OpenconfigVlan_VlanModeType_TRUNK
				} else {
					swVlan.swEthMember.State.InterfaceMode = ocbinds.OpenconfigVlan_VlanModeType_TRUNK
				}
			}
		case IntfTypePortChannel:
			if accessVlanName != nil {
				if config {
					swVlan.swPortChannelMember.Config.InterfaceMode = ocbinds.OpenconfigVlan_VlanModeType_ACCESS
				} else {
					swVlan.swPortChannelMember.State.InterfaceMode = ocbinds.OpenconfigVlan_VlanModeType_ACCESS
				}
			}
			if len(trunkVlans) > 0 {
				if config {
					swVlan.swPortChannelMember.Config.InterfaceMode = ocbinds.OpenconfigVlan_VlanModeType_TRUNK
				} else {
					swVlan.swPortChannelMember.State.InterfaceMode = ocbinds.OpenconfigVlan_VlanModeType_TRUNK
				}
			}
		}
		return true, nil
	}
	return false, nil
}

func getSwitchedVlanState(ifKey *string, vlanMemberMap map[string]map[string]db.Value,
	swVlan *swVlanMemberPort_t, intfType E_InterfaceType, config bool) error {
	/* Get Access VLAN info for Interface */
	_, accessVlanName, e := getIntfVlanAttr(ifKey, ACCESS, vlanMemberMap)
	if e != nil {
		return e
	}

	/* Get Trunk VLAN info for Interface */
	trunkVlans, _, e := getIntfVlanAttr(ifKey, TRUNK, vlanMemberMap)
	if e != nil {
		return e
	}

	switch intfType {
	case IntfTypeEthernet:

		if swVlan.swEthMember.State == nil {
			ygot.BuildEmptyTree(swVlan.swEthMember)
		}

		if accessVlanName != nil {
			vlanName := *accessVlanName
			vlanIdStr := vlanName[len("Vlan"):]
			vlanId, err := strconv.Atoi(vlanIdStr)
			if err != nil {
				errStr := "Conversion of string to int failed for " + vlanIdStr
				return errors.New(errStr)
			}
			vlanIdCast := uint16(vlanId)
			if config {
				swVlan.swEthMember.Config.AccessVlan = &vlanIdCast
				swVlan.swEthMember.Config.InterfaceMode = ocbinds.OpenconfigVlan_VlanModeType_ACCESS
			} else {
				swVlan.swEthMember.State.AccessVlan = &vlanIdCast
				swVlan.swEthMember.State.InterfaceMode = ocbinds.OpenconfigVlan_VlanModeType_ACCESS
			}
		}
		for _, vlanName := range trunkVlans {
			vlanIdStr := vlanName[len("Vlan"):]
			vlanId, err := strconv.Atoi(vlanIdStr)
			if err != nil {
				errStr := "Conversion of string to int failed for " + vlanIdStr
				return errors.New(errStr)
			}
			vlanIdCast := uint16(vlanId)

			if config {
				trunkVlan, _ := swVlan.swEthMember.Config.To_OpenconfigInterfaces_Interfaces_Interface_Ethernet_SwitchedVlan_Config_TrunkVlans_Union(vlanIdCast)
				swVlan.swEthMember.Config.TrunkVlans = append(swVlan.swEthMember.Config.TrunkVlans, trunkVlan)
				swVlan.swEthMember.Config.InterfaceMode = ocbinds.OpenconfigVlan_VlanModeType_TRUNK
			} else {
				trunkVlan, _ := swVlan.swEthMember.State.To_OpenconfigInterfaces_Interfaces_Interface_Ethernet_SwitchedVlan_State_TrunkVlans_Union(vlanIdCast)
				swVlan.swEthMember.State.TrunkVlans = append(swVlan.swEthMember.State.TrunkVlans, trunkVlan)
				swVlan.swEthMember.State.InterfaceMode = ocbinds.OpenconfigVlan_VlanModeType_TRUNK
			}
		}
	case IntfTypePortChannel:

		if swVlan.swPortChannelMember.State == nil {
			ygot.BuildEmptyTree(swVlan.swPortChannelMember)
		}

		if accessVlanName != nil {
			vlanName := *accessVlanName
			vlanIdStr := vlanName[len("Vlan"):]
			vlanId, err := strconv.Atoi(vlanIdStr)
			if err != nil {
				errStr := "Conversion of string to int failed for " + vlanIdStr
				return errors.New(errStr)
			}
			vlanIdCast := uint16(vlanId)
			if config {
				swVlan.swPortChannelMember.Config.AccessVlan = &vlanIdCast
				swVlan.swPortChannelMember.Config.InterfaceMode = ocbinds.OpenconfigVlan_VlanModeType_ACCESS
			} else {
				swVlan.swPortChannelMember.State.AccessVlan = &vlanIdCast
				swVlan.swPortChannelMember.State.InterfaceMode = ocbinds.OpenconfigVlan_VlanModeType_ACCESS
			}
		}
		for _, vlanName := range trunkVlans {
			vlanIdStr := vlanName[len("Vlan"):]
			vlanId, err := strconv.Atoi(vlanIdStr)
			if err != nil {
				errStr := "Conversion of string to int failed for " + vlanIdStr
				return errors.New(errStr)
			}

			vlanIdCast := uint16(vlanId)
			if config {
				trunkVlan, _ := swVlan.swPortChannelMember.Config.To_OpenconfigInterfaces_Interfaces_Interface_Aggregation_SwitchedVlan_Config_TrunkVlans_Union(vlanIdCast)
				swVlan.swPortChannelMember.Config.TrunkVlans = append(swVlan.swPortChannelMember.Config.TrunkVlans, trunkVlan)
				swVlan.swPortChannelMember.Config.InterfaceMode = ocbinds.OpenconfigVlan_VlanModeType_TRUNK
			} else {
				trunkVlan, _ := swVlan.swPortChannelMember.State.To_OpenconfigInterfaces_Interfaces_Interface_Aggregation_SwitchedVlan_State_TrunkVlans_Union(vlanIdCast)
				swVlan.swPortChannelMember.State.TrunkVlans = append(swVlan.swPortChannelMember.State.TrunkVlans, trunkVlan)
				swVlan.swPortChannelMember.State.InterfaceMode = ocbinds.OpenconfigVlan_VlanModeType_TRUNK
			}
		}
	}
	return nil
}

// DbToYang_sw_vlans_xfmr is a DB to Yang Subtree transformer method handles GET operation
var DbToYang_sw_vlans_xfmr SubTreeXfmrDbToYang = func(inParams XfmrParams) error {
	var err error
	var swVlan swVlanMemberPort_t
	intfsObj := getIntfsRoot(inParams.ygRoot)
	if intfsObj == nil {
		errStr := "Nil root object received for Ethernet-Switched VLAN Get!"
		log.Errorf(errStr)
		return errors.New(errStr)
	}
	pathInfo := NewPathInfo(inParams.uri)

	uriIfName := pathInfo.Var("name")
	ifName := uriIfName

	if log.V(5) {
		log.Infof("Ethernet-Switched Vlan Get observed for Interface: %s", ifName)
	}
	intfType, _, err := getIntfTypeByName(ifName)
	if intfType != IntfTypeEthernet && intfType != IntfTypePortChannel || err != nil {
		intfTypeStr := strconv.Itoa(int(intfType))
		errStr := "TableXfmrFunc - Invalid interface type" + intfTypeStr
		log.Warning(errStr)
		return errors.New(errStr)
	}

	if (strings.Contains(inParams.uri, "ethernet") && (intfType == IntfTypePortChannel)) ||
		(strings.Contains(inParams.uri, "aggregation") && (intfType == IntfTypeEthernet)) {
		return nil
	}
	targetUriPath := pathInfo.YangPath
	if log.V(5) {
		log.Info("targetUriPath is ", targetUriPath)
	}

	intfObj := intfsObj.Interface[uriIfName]
	if intfObj == nil {
		intfObj, _ = intfsObj.NewInterface(uriIfName)
		ygot.BuildEmptyTree(intfObj)
	}

	if intfObj.Ethernet == nil && intfObj.Aggregation == nil {
		return errors.New("Wrong GET request for switched-vlan!")
	}
	if intfObj.Ethernet != nil {
		if intfObj.Ethernet.SwitchedVlan == nil {
			ygot.BuildEmptyTree(intfObj.Ethernet)
		}
		swVlan.swEthMember = intfObj.Ethernet.SwitchedVlan
	}
	if intfObj.Aggregation != nil {
		if intfObj.Aggregation.SwitchedVlan == nil {
			ygot.BuildEmptyTree(intfObj.Aggregation)
		}
		swVlan.swPortChannelMember = intfObj.Aggregation.SwitchedVlan
	}
	switch intfType {
	case IntfTypeEthernet:
		if intfObj.Ethernet == nil {
			errStr := "Switched-vlan state tree not built correctly for Interface: " + ifName
			log.Error(errStr)
			return errors.New(errStr)
		}
		if intfObj.Ethernet.SwitchedVlan == nil {
			ygot.BuildEmptyTree(intfObj.Ethernet)
		}

		vlanMemberMap := make(map[string]map[string]db.Value)
		err = fillDBSwitchedVlanInfoForIntf(inParams.d, &ifName, vlanMemberMap)
		if err != nil {
			log.Errorf("Filiing Switched Vlan Info for Interface: %s failed!", ifName)
			return err
		}
		if log.V(5) {
			log.Info("Succesfully completed DB population for Ethernet!")
		}

		attrPresent, err := getSpecificSwitchedVlanStateAttr(&targetUriPath, &ifName, vlanMemberMap, &swVlan, intfType)
		if err != nil {
			return err
		}
		if !attrPresent {
			if log.V(5) {
				log.Infof("Get is for Switched Vlan State Container!")
			}
			switch targetUriPath {
			case "/openconfig-interfaces:interfaces/interface/openconfig-if-ethernet:ethernet/openconfig-vlan:switched-vlan/config":
				err = getSwitchedVlanState(&ifName, vlanMemberMap, &swVlan, intfType, true)
				if err != nil {
					return err
				}
			case "/openconfig-interfaces:interfaces/interface/openconfig-if-ethernet:ethernet/openconfig-vlan:switched-vlan/state":
				err = getSwitchedVlanState(&ifName, vlanMemberMap, &swVlan, intfType, false)
				if err != nil {
					return err
				}
			case "/openconfig-interfaces:interfaces/interface/openconfig-if-ethernet:ethernet/openconfig-vlan:switched-vlan":
				fallthrough
			case "/openconfig-interfaces:interfaces/interface/openconfig-if-ethernet:ethernet/switched-vlan":
				fallthrough
			case "/openconfig-interfaces:interfaces/interface/ethernet/switched-vlan":
				err = getSwitchedVlanState(&ifName, vlanMemberMap, &swVlan, intfType, true)
				if err != nil {
					return err
				}
				err = getSwitchedVlanState(&ifName, vlanMemberMap, &swVlan, intfType, false)
				if err != nil {
					return err
				}
			}
		}

	case IntfTypePortChannel:
		if intfObj.Aggregation == nil {
			errStr := "Switched-vlan state tree not built correctly for Interface: " + ifName
			log.Error(errStr)
			return errors.New(errStr)
		}

		if intfObj.Aggregation.SwitchedVlan == nil {
			ygot.BuildEmptyTree(intfObj.Aggregation)
		}

		vlanMemberMap := make(map[string]map[string]db.Value)
		err = fillDBSwitchedVlanInfoForIntf(inParams.d, &ifName, vlanMemberMap)
		if err != nil {
			log.Errorf("Filiing Switched Vlan Info for Interface: %s failed!", ifName)
			return err
		}
		log.Info("Succesfully completed DB population for Port-Channel!")
		attrPresent, err := getSpecificSwitchedVlanStateAttr(&targetUriPath, &ifName, vlanMemberMap, &swVlan, intfType)
		if err != nil {
			return err
		}
		if !attrPresent {
			log.Infof("Get is for Switched Vlan State Container!")
			switch targetUriPath {
			case "/openconfig-interfaces:interfaces/interface/openconfig-if-aggregate:aggregation/openconfig-vlan:switched-vlan/config":
				err = getSwitchedVlanState(&ifName, vlanMemberMap, &swVlan, intfType, true)
				if err != nil {
					return err
				}
			case "/openconfig-interfaces:interfaces/interface/openconfig-if-aggregate:aggregation/openconfig-vlan:switched-vlan/state":
				err = getSwitchedVlanState(&ifName, vlanMemberMap, &swVlan, intfType, false)
				if err != nil {
					return err
				}
			case "/openconfig-interfaces:interfaces/interface/openconfig-if-aggregate:aggregation/openconfig-vlan:switched-vlan":
				fallthrough
			case "/openconfig-interfaces:interfaces/interface/openconfig-if-aggregate:aggregation/switched-vlan":
				fallthrough
			case "/openconfig-interfaces:interfaces/interface/aggregation/switched-vlan":
				err = getSwitchedVlanState(&ifName, vlanMemberMap, &swVlan, intfType, true)
				if err != nil {
					return err
				}
				err = getSwitchedVlanState(&ifName, vlanMemberMap, &swVlan, intfType, false)
				if err != nil {
					return err
				}
			}
		}
	}
	return err
}

var DbToYangPath_sw_vlans_path_xfmr PathXfmrDbToYangFunc = func(params XfmrDbToYgPathParams) error {
	log.Info("DbToYangPath_sw_vlans_path_xfmr : params ", params)

	if (params.tblName != "PORT") &&
		(params.tblName != "PORTCHANNEL") {
		log.Info("DbToYangPath_sw_vlans_path_xfmr: unsupported table: ", params.tblName)
		return nil
	}

	log.Info("DbToYangPath_sw_vlans_path_xfmr : params.ygPathkeys: ", params.ygPathKeys)

	return nil
}
