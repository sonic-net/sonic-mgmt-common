//////////////////////////////////////////////////////////////////////////
//
// Copyright 2019 Dell, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
//////////////////////////////////////////////////////////////////////////

package transformer

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"inet.af/netaddr"

	"github.com/Azure/sonic-mgmt-common/translib/db"
	"github.com/Azure/sonic-mgmt-common/translib/ocbinds"
	"github.com/Azure/sonic-mgmt-common/translib/tlerr"
	"github.com/Azure/sonic-mgmt-common/translib/utils"
	log "github.com/golang/glog"
	"github.com/openconfig/ygot/ygot"
)

func init() {
	XlateFuncBind("intf_table_xfmr", intf_table_xfmr)
	XlateFuncBind("YangToDb_intf_tbl_key_xfmr", YangToDb_intf_tbl_key_xfmr)
	XlateFuncBind("DbToYang_intf_tbl_key_xfmr", DbToYang_intf_tbl_key_xfmr)
	XlateFuncBind("YangToDb_intf_mtu_xfmr", YangToDb_intf_mtu_xfmr)
	XlateFuncBind("DbToYang_intf_mtu_xfmr", DbToYang_intf_mtu_xfmr)
	XlateFuncBind("DbToYang_intf_admin_status_xfmr", DbToYang_intf_admin_status_xfmr)
	XlateFuncBind("YangToDb_intf_enabled_xfmr", YangToDb_intf_enabled_xfmr)
	XlateFuncBind("DbToYang_intf_enabled_xfmr", DbToYang_intf_enabled_xfmr)
	XlateFuncBind("YangToDb_intf_eth_port_config_xfmr", YangToDb_intf_eth_port_config_xfmr)
	XlateFuncBind("DbToYang_intf_eth_port_config_xfmr", DbToYang_intf_eth_port_config_xfmr)
	XlateFuncBind("DbToYangPath_intf_eth_port_config_path_xfmr", DbToYangPath_intf_eth_port_config_path_xfmr)
	XlateFuncBind("DbToYang_intf_eth_auto_neg_xfmr", DbToYang_intf_eth_auto_neg_xfmr)
	XlateFuncBind("DbToYang_intf_eth_port_speed_xfmr", DbToYang_intf_eth_port_speed_xfmr)

	XlateFuncBind("YangToDb_intf_subintfs_xfmr", YangToDb_intf_subintfs_xfmr)
	XlateFuncBind("DbToYang_intf_subintfs_xfmr", DbToYang_intf_subintfs_xfmr)

	XlateFuncBind("YangToDb_subintf_ip_addr_key_xfmr", YangToDb_subintf_ip_addr_key_xfmr)
	XlateFuncBind("DbToYang_subintf_ip_addr_key_xfmr", DbToYang_subintf_ip_addr_key_xfmr)
	XlateFuncBind("YangToDb_intf_ip_addr_xfmr", YangToDb_intf_ip_addr_xfmr)
	XlateFuncBind("DbToYang_intf_ip_addr_xfmr", DbToYang_intf_ip_addr_xfmr)

	XlateFuncBind("intf_subintfs_table_xfmr", intf_subintfs_table_xfmr)
	XlateFuncBind("YangToDb_subif_index_xfmr", YangToDb_subif_index_xfmr)
	XlateFuncBind("DbToYang_subif_index_xfmr", DbToYang_subif_index_xfmr)
	XlateFuncBind("DbToYangPath_intf_ip_path_xfmr", DbToYangPath_intf_ip_path_xfmr)
	XlateFuncBind("Subscribe_intf_ip_addr_xfmr", Subscribe_intf_ip_addr_xfmr)

	XlateFuncBind("YangToDb_subintf_ipv6_tbl_key_xfmr", YangToDb_subintf_ipv6_tbl_key_xfmr)
	XlateFuncBind("DbToYang_subintf_ipv6_tbl_key_xfmr", DbToYang_subintf_ipv6_tbl_key_xfmr)
	XlateFuncBind("YangToDb_ipv6_enabled_xfmr", YangToDb_ipv6_enabled_xfmr)
	XlateFuncBind("DbToYang_ipv6_enabled_xfmr", DbToYang_ipv6_enabled_xfmr)

	XlateFuncBind("intf_post_xfmr", intf_post_xfmr)
	XlateFuncBind("intf_pre_xfmr", intf_pre_xfmr)

}

const (
	PORT_ADMIN_STATUS = "admin_status"
	PORTCHANNEL_TN    = "PORTCHANNEL"
	PORT_SPEED        = "speed"
	PORT_AUTONEG      = "autoneg"
	DEFAULT_MTU       = "9100"
)

const (
	PIPE  = "|"
	COLON = ":"

	ETHERNET    = "Eth"
	MGMT        = "eth"
	VLAN        = "Vlan"
	PORTCHANNEL = "PortChannel"
	LOOPBACK    = "Loopback"
	VXLAN       = "vtep"
	MANAGEMENT  = "Management"
)

type TblData struct {
	portTN   string
	memberTN string
	intfTN   string
	keySep   string
}

type IntfTblData struct {
	cfgDb   TblData
	appDb   TblData
	stateDb TblData
}

var IntfTypeTblMap = map[E_InterfaceType]IntfTblData{
	IntfTypeEthernet: IntfTblData{
		cfgDb:   TblData{portTN: "PORT", intfTN: "INTERFACE", keySep: PIPE},
		appDb:   TblData{portTN: "PORT_TABLE", intfTN: "INTF_TABLE", keySep: COLON},
		stateDb: TblData{portTN: "PORT_TABLE", intfTN: "INTERFACE_TABLE", keySep: PIPE},
	},
	IntfTypeMgmt: IntfTblData{
		cfgDb:   TblData{portTN: "MGMT_PORT", intfTN: "MGMT_INTERFACE", keySep: PIPE},
		appDb:   TblData{portTN: "MGMT_PORT_TABLE", intfTN: "MGMT_INTF_TABLE", keySep: COLON},
		stateDb: TblData{portTN: "MGMT_PORT_TABLE", intfTN: "MGMT_INTERFACE_TABLE", keySep: PIPE},
	},
	IntfTypePortChannel: IntfTblData{
		cfgDb:   TblData{portTN: "PORTCHANNEL", intfTN: "PORTCHANNEL_INTERFACE", memberTN: "PORTCHANNEL_MEMBER", keySep: PIPE},
		appDb:   TblData{portTN: "LAG_TABLE", intfTN: "INTF_TABLE", keySep: COLON, memberTN: "LAG_MEMBER_TABLE"},
		stateDb: TblData{portTN: "LAG_TABLE", intfTN: "INTERFACE_TABLE", keySep: PIPE},
	},
	IntfTypeVlan: IntfTblData{
		cfgDb: TblData{portTN: "VLAN", memberTN: "VLAN_MEMBER", intfTN: "VLAN_INTERFACE", keySep: PIPE},
		appDb: TblData{portTN: "VLAN_TABLE", memberTN: "VLAN_MEMBER_TABLE", intfTN: "INTF_TABLE", keySep: COLON},
	},
	IntfTypeLoopback: IntfTblData{
		cfgDb: TblData{portTN: "LOOPBACK", intfTN: "LOOPBACK_INTERFACE", keySep: PIPE},
		appDb: TblData{portTN: "LOOPBACK_TABLE", intfTN: "INTF_TABLE", keySep: COLON},
	},
	IntfTypeSubIntf: IntfTblData{
		cfgDb:   TblData{portTN: "VLAN_SUB_INTERFACE", intfTN: "VLAN_SUB_INTERFACE", keySep: PIPE},
		appDb:   TblData{portTN: "PORT_TABLE", intfTN: "INTF_TABLE", keySep: COLON},
		stateDb: TblData{portTN: "PORT_TABLE", intfTN: "INTERFACE_TABLE", keySep: PIPE},
	},
}

var dbIdToTblMap = map[db.DBNum][]string{
	db.ConfigDB: {"PORT", "MGMT_PORT", "VLAN", "PORTCHANNEL", "LOOPBACK", "VXLAN_TUNNEL", "VLAN_SUB_INTERFACE"},
	db.ApplDB:   {"PORT_TABLE", "MGMT_PORT_TABLE", "VLAN_TABLE", "LAG_TABLE"},
	db.StateDB:  {"PORT_TABLE", "MGMT_PORT_TABLE", "LAG_TABLE"},
}

var intfOCToSpeedMap = map[ocbinds.E_OpenconfigIfEthernet_ETHERNET_SPEED]string{
	ocbinds.OpenconfigIfEthernet_ETHERNET_SPEED_SPEED_10MB:   "10",
	ocbinds.OpenconfigIfEthernet_ETHERNET_SPEED_SPEED_100MB:  "100",
	ocbinds.OpenconfigIfEthernet_ETHERNET_SPEED_SPEED_1GB:    "1000",
	ocbinds.OpenconfigIfEthernet_ETHERNET_SPEED_SPEED_2500MB: "2500",
	ocbinds.OpenconfigIfEthernet_ETHERNET_SPEED_SPEED_5GB:    "5000",
	ocbinds.OpenconfigIfEthernet_ETHERNET_SPEED_SPEED_10GB:   "10000",
	ocbinds.OpenconfigIfEthernet_ETHERNET_SPEED_SPEED_25GB:   "25000",
	ocbinds.OpenconfigIfEthernet_ETHERNET_SPEED_SPEED_40GB:   "40000",
	ocbinds.OpenconfigIfEthernet_ETHERNET_SPEED_SPEED_50GB:   "50000",
	ocbinds.OpenconfigIfEthernet_ETHERNET_SPEED_SPEED_100GB:  "100000",
	ocbinds.OpenconfigIfEthernet_ETHERNET_SPEED_SPEED_200GB:  "200000",
	ocbinds.OpenconfigIfEthernet_ETHERNET_SPEED_SPEED_400GB:  "400000",
}

type E_InterfaceType int64

const (
	IntfTypeUnset       E_InterfaceType = 0
	IntfTypeEthernet    E_InterfaceType = 1
	IntfTypeMgmt        E_InterfaceType = 2
	IntfTypeVlan        E_InterfaceType = 3
	IntfTypePortChannel E_InterfaceType = 4
	IntfTypeLoopback    E_InterfaceType = 5
	IntfTypeVxlan       E_InterfaceType = 6
	IntfTypeSubIntf     E_InterfaceType = 7
)

type E_InterfaceSubType int64

const (
	IntfSubTypeUnset       E_InterfaceSubType = 0
	IntfSubTypeVlanL2      E_InterfaceSubType = 1
	InterfaceSubTypeVlanL3 E_InterfaceSubType = 2
)

func getIntfTypeByName(name string) (E_InterfaceType, E_InterfaceSubType, error) {

	var err error
	if strings.Contains(name, ".") {
		if strings.HasPrefix(name, ETHERNET) || strings.HasPrefix(name, "Po") {
			return IntfTypeSubIntf, IntfSubTypeUnset, err
		}
	}
	if strings.HasPrefix(name, ETHERNET) {
		return IntfTypeEthernet, IntfSubTypeUnset, err
	} else {
		err = errors.New("Interface name prefix not matched with supported types")
		return IntfTypeUnset, IntfSubTypeUnset, err
	}
}

func getIntfsRoot(s *ygot.GoStruct) *ocbinds.OpenconfigInterfaces_Interfaces {
	deviceObj := (*s).(*ocbinds.Device)
	return deviceObj.Interfaces
}

func getPortTableNameByDBId(intftbl IntfTblData, curDb db.DBNum) (string, error) {

	var tblName string

	switch curDb {
	case db.ConfigDB:
		tblName = intftbl.cfgDb.portTN
	case db.ApplDB:
		tblName = intftbl.appDb.portTN
	case db.StateDB:
		tblName = intftbl.stateDb.portTN
	default:
		tblName = intftbl.cfgDb.portTN
	}

	return tblName, nil
}

/* Perform action based on the operation and Interface type wrt Interface name key */
/* It should handle only Interface name key xfmr operations */
func performIfNameKeyXfmrOp(inParams *XfmrParams, requestUriPath *string, ifName *string, ifType E_InterfaceType, subintfid uint32) error {
	var err error
	switch inParams.oper {
	case GET:
		if ifType == IntfTypeSubIntf && subintfid == 0 {
			errStr := "Invalid interface name: " + *ifName
			log.Infof("Invalid interface name: %s for GET path: %v", *ifName, *requestUriPath)
			err = tlerr.InvalidArgsError{Format: errStr}
			return err
		}
	case DELETE:
		if *requestUriPath == "/openconfig-interfaces:interfaces/interface/subinterfaces/subinterface" && subintfid != 0 {
			return nil
		}

		if *requestUriPath == "/openconfig-interfaces:interfaces/interface" {
			switch ifType {
			case IntfTypeEthernet:
				err = validateIntfExists(inParams.d, IntfTypeTblMap[IntfTypeEthernet].cfgDb.portTN, *ifName)
				if err != nil {
					// Not returning error from here since mgmt infra will return "Resource not found" error in case of non existence entries
					return nil
				}
				errStr := "Physical Interface: " + *ifName + " cannot be deleted"
				err = tlerr.InvalidArgsError{Format: errStr}
				return err
			default:
				errStr := "Invalid interface for delete:" + *ifName
				log.Error(errStr)
				return tlerr.InvalidArgsError{Format: errStr}
			}

		}
	case CREATE:
		fallthrough
	case UPDATE, REPLACE:
		if ifType == IntfTypeEthernet {
			err = validateIntfExists(inParams.d, IntfTypeTblMap[IntfTypeEthernet].cfgDb.portTN, *ifName)
			if err != nil { // Invalid Physical interface
				errStr := "Interface " + *ifName + " cannot be configured."
				return tlerr.InvalidArgsError{Format: errStr}
			}
			if inParams.oper == REPLACE {
				if strings.Contains(*requestUriPath, "/openconfig-interfaces:interfaces/interface") {
					if strings.Contains(*requestUriPath, "openconfig-if-ethernet:ethernet/openconfig-vlan:switched-vlan") ||
						strings.Contains(*requestUriPath, "mapped-vlans") {
						log.Infof("allow replace operation for switched-vlan")
					} else {
						// OC interfaces yang does not have attributes to set Physical interface critical attributes like speed, alias, lanes, index.
						// Replace/PUT request without the critical attributes would end up in deletion of the same in PORT table, which cannot be allowed.
						// Hence block the Replace/PUT request for Physical interfaces alone.
						err_str := "Replace/PUT request not allowed for Physical interfaces"
						return tlerr.NotSupported(err_str)
					}
				}
			}
		}
	}
	return err
}

/* Validate whether intf exists in DB */
func validateIntfExists(d *db.DB, intfTs string, ifName string) error {
	if len(ifName) == 0 {
		return errors.New("Length of Interface name is zero")
	}

	entry, err := d.GetEntry(&db.TableSpec{Name: intfTs}, db.Key{Comp: []string{ifName}})
	if err != nil || !entry.IsPopulated() {
		errStr := "Invalid Interface:" + ifName
		if log.V(3) {
			log.Error(errStr)
		}
		return tlerr.InvalidArgsError{Format: errStr}
	}
	return nil
}

func getMemTableNameByDBId(intftbl IntfTblData, curDb db.DBNum) (string, error) {

	var tblName string

	switch curDb {
	case db.ConfigDB:
		tblName = intftbl.cfgDb.memberTN
	case db.ApplDB:
		tblName = intftbl.appDb.memberTN
	case db.StateDB:
		tblName = intftbl.stateDb.memberTN
	default:
		tblName = intftbl.cfgDb.memberTN
	}

	return tblName, nil
}

func retrievePortChannelAssociatedWithIntf(inParams *XfmrParams, ifName *string) (*string, error) {
	var err error

	if strings.HasPrefix(*ifName, ETHERNET) {
		intTbl := IntfTypeTblMap[IntfTypePortChannel]
		tblName, _ := getMemTableNameByDBId(intTbl, inParams.curDb)
		var lagStr string

		lagKeys, err := inParams.d.GetKeysByPattern(&db.TableSpec{Name: tblName}, "*"+*ifName)
		/* Find the port-channel the given ifname is part of */
		if err != nil {
			return nil, err
		}
		var flag bool = false
		for i := range lagKeys {
			if *ifName == lagKeys[i].Get(1) {
				flag = true
				lagStr = lagKeys[i].Get(0)
				log.Info("Given interface part of PortChannel: ", lagStr)
				break
			}
		}
		if !flag {
			log.Info("Given Interface not part of any PortChannel")
			return nil, err
		}
		return &lagStr, err
	}
	return nil, err
}

func updateDefaultMtu(inParams *XfmrParams, ifName *string, ifType E_InterfaceType, resMap map[string]string) error {
	var err error
	subOpMap := make(map[db.DBNum]map[string]map[string]db.Value)
	intfMap := make(map[string]map[string]db.Value)

	intTbl := IntfTypeTblMap[ifType]
	resMap["mtu"] = DEFAULT_MTU

	intfMap[intTbl.cfgDb.portTN] = make(map[string]db.Value)
	intfMap[intTbl.cfgDb.portTN][*ifName] = db.Value{Field: resMap}

	subOpMap[db.ConfigDB] = intfMap
	inParams.subOpDataMap[UPDATE] = &subOpMap
	return err
}

func getDbToYangSpeed(speed string) (ocbinds.E_OpenconfigIfEthernet_ETHERNET_SPEED, error) {
	portSpeed := ocbinds.OpenconfigIfEthernet_ETHERNET_SPEED_SPEED_UNKNOWN
	var err error = errors.New("Not found in port speed map")
	for k, v := range intfOCToSpeedMap {
		if speed == v {
			portSpeed = k
			err = nil
		}
	}
	return portSpeed, err
}

var intf_table_xfmr TableXfmrFunc = func(inParams XfmrParams) ([]string, error) {
	var tblList []string
	var err error

	pathInfo := NewPathInfo(inParams.uri)
	targetUriPath := pathInfo.YangPath
	targetUriXpath, _, _ := XfmrRemoveXPATHPredicates(targetUriPath)

	ifName := pathInfo.Var("name")
	if ifName == "" {
		log.Info("TableXfmrFunc - intf_table_xfmr Intf key is not present")

		if _, ok := dbIdToTblMap[inParams.curDb]; !ok {
			if log.V(3) {
				log.Info("TableXfmrFunc - intf_table_xfmr db id entry not present")
			}
			return tblList, errors.New("Key not present")
		} else {
			return dbIdToTblMap[inParams.curDb], nil
		}
	}
	idx := pathInfo.Var("index")
	var i32 uint32
	i32 = 0
	if idx != "" {
		i64, _ := strconv.ParseUint(idx, 10, 32)
		i32 = uint32(i64)
	}

	intfType, _, ierr := getIntfTypeByName(ifName)
	if intfType == IntfTypeUnset || ierr != nil {
		return tblList, errors.New("Invalid interface type IntfTypeUnset")
	}
	intTbl := IntfTypeTblMap[intfType]
	if log.V(3) {
		log.Info("TableXfmrFunc - targetUriPath : ", targetUriPath)
		log.Info("TableXfmrFunc - targetUriXpath : ", targetUriXpath)
	}

	if inParams.oper == DELETE && (targetUriXpath == "/openconfig-interfaces:interfaces/interface/subinterfaces/subinterface/ipv4" ||
		targetUriXpath == "/openconfig-interfaces:interfaces/interface/subinterfaces/subinterface/ipv6") {
		errStr := "DELETE operation not allowed on this container"
		return tblList, tlerr.NotSupportedError{AppTag: "invalid-value", Path: "", Format: errStr}

	} else if strings.HasPrefix(targetUriPath, "/openconfig-interfaces:interfaces/interface/config") {
		if IntfTypeVxlan != intfType {
			tblList = append(tblList, intTbl.cfgDb.portTN)
		}

	} else if intfType != IntfTypeEthernet && intfType != IntfTypeMgmt &&
		strings.HasPrefix(targetUriPath, "/openconfig-interfaces:interfaces/interface/openconfig-if-ethernet:ethernet") {
		//Checking interface type at container level, if not Ethernet type return nil
		return nil, nil

	} else if intfType != IntfTypePortChannel &&
		strings.HasPrefix(targetUriPath, "/openconfig-interfaces:interfaces/interface/openconfig-if-aggregate:aggregation") {
		//Checking interface type at container level, if not PortChannel type return nil
		return nil, nil

	} else if intfType != IntfTypeVlan &&
		strings.HasPrefix(targetUriPath, "openconfig-interfaces:interfaces/interface/openconfig-vlan:routed-vlan") {
		//Checking interface type at container level, if not Vlan type return nil
		return nil, nil

	} else if intfType != IntfTypeVxlan &&
		strings.HasPrefix(targetUriPath, "/openconfig-interfaces:interfaces/interface/openconfig-vxlan:vxlan-if") {
		//Checking interface type at container level, if not Vxlan type return nil
		return nil, nil

	} else if strings.HasPrefix(targetUriPath, "/openconfig-interfaces:interfaces/interface/state/counters") {
		tblList = append(tblList, "NONE")

	} else if strings.HasPrefix(targetUriPath, "/openconfig-interfaces:interfaces/interface/state") ||
		strings.HasPrefix(targetUriPath, "/openconfig-interfaces:interfaces/interface/ethernet/state") ||
		strings.HasPrefix(targetUriPath, "/openconfig-interfaces:interfaces/interface/openconfig-if-ethernet:ethernet/state") {
		tblList = append(tblList, intTbl.appDb.portTN)
	} else if strings.HasPrefix(targetUriPath, "/openconfig-interfaces:interfaces/interface/subinterfaces/subinterface/ipv4/addresses/address/config") ||
		strings.HasPrefix(targetUriPath, "/openconfig-interfaces:interfaces/interface/subinterfaces/subinterface/openconfig-if-ip:ipv4/addresses/address/config") ||
		strings.HasPrefix(targetUriPath, "/openconfig-interfaces:interfaces/interface/subinterfaces/subinterface/openconfig-if-ip:ipv6/addresses/address/config") ||
		strings.HasPrefix(targetUriPath, "/openconfig-interfaces:interfaces/interface/subinterfaces/subinterface/ipv6/addresses/address/config") ||
		strings.HasPrefix(targetUriPath, "/openconfig-interfaces:interfaces/interface/subinterfaces/subinterface/ipv6/config") ||
		strings.HasPrefix(targetUriPath, "/openconfig-interfaces:interfaces/interface/subinterfaces/subinterface/openconfig-if-ip:ipv6/config") {
		if i32 > 0 {
			tblList = append(tblList, "VLAN_SUB_INTERFACE")
		} else {
			tblList = append(tblList, intTbl.cfgDb.intfTN)
		}
	} else if strings.HasPrefix(targetUriPath, "/openconfig-interfaces:interfaces/interface/subinterfaces/subinterface/ipv4/addresses/address/state") ||
		strings.HasPrefix(targetUriPath, "/openconfig-interfaces:interfaces/interface/subinterfaces/subinterface/openconfig-if-ip:ipv4/addresses/address/state") ||
		strings.HasPrefix(targetUriPath, "/openconfig-interfaces:interfaces/interface/subinterfaces/subinterface/openconfig-if-ip:ipv6/addresses/address/state") ||
		strings.HasPrefix(targetUriPath, "/openconfig-interfaces:interfaces/interface/subinterfaces/subinterface/ipv6/addresses/address/state") ||
		strings.HasPrefix(targetUriPath, "/openconfig-interfaces:interfaces/interface/subinterfaces/subinterface/ipv6/state") ||
		strings.HasPrefix(targetUriPath, "/openconfig-interfaces:interfaces/interface/subinterfaces/subinterface/openconfig-if-ip:ipv6/state") {
		tblList = append(tblList, intTbl.appDb.intfTN)
	} else if strings.HasPrefix(targetUriPath, "/openconfig-interfaces:interfaces/interface/subinterfaces/subinterface/ipv4/addresses") ||
		strings.HasPrefix(targetUriPath, "/openconfig-interfaces:interfaces/interface/subinterfaces/subinterface/openconfig-if-ip:ipv4/addresses") ||
		strings.HasPrefix(targetUriPath, "/openconfig-interfaces:interfaces/interface/subinterfaces/subinterface/openconfig-if-ip:ipv6/addresses") ||
		strings.HasPrefix(targetUriPath, "/openconfig-interfaces:interfaces/interface/subinterfaces/subinterface/ipv6/addresses") {
		tblList = append(tblList, intTbl.cfgDb.intfTN)
	} else if inParams.oper == GET && strings.HasPrefix(targetUriXpath, "/openconfig-interfaces:interfaces/interface/subinterfaces/subinterface/ipv4/neighbors") ||
		strings.HasPrefix(targetUriXpath, "/openconfig-interfaces:interfaces/interface/subinterfaces/subinterface/ipv6/neighbors") {
		tblList = append(tblList, "NONE")
	} else if strings.HasPrefix(targetUriPath, "/openconfig-interfaces:interfaces/interface/ethernet") ||
		strings.HasPrefix(targetUriPath, "/openconfig-interfaces:interfaces/interface/openconfig-if-ethernet:ethernet") {
		if inParams.oper != DELETE {
			tblList = append(tblList, intTbl.cfgDb.portTN)
		}
	} else if targetUriPath == "/openconfig-interfaces:interfaces/interface" {
		tblList = append(tblList, intTbl.cfgDb.portTN)
	} else if strings.HasPrefix(targetUriPath, "/openconfig-interfaces:interfaces/interface") {
		if inParams.oper != DELETE {
			tblList = append(tblList, intTbl.cfgDb.portTN)
		}
	} else {
		err = errors.New("Invalid URI")
	}

	if log.V(3) {
		log.Infof("TableXfmrFunc - Uri: (%v), targetUriPath: %s, tblList: (%v)", inParams.uri, targetUriPath, tblList)
	}

	return tblList, err
}

var YangToDb_intf_tbl_key_xfmr KeyXfmrYangToDb = func(inParams XfmrParams) (string, error) {
	var err error

	pathInfo := NewPathInfo(inParams.uri)
	reqpathInfo := NewPathInfo(inParams.requestUri)
	requestUriPath := reqpathInfo.YangPath

	log.Infof("YangToDb_intf_tbl_key_xfmr: inParams.uri: %s, pathInfo: %s, inParams.requestUri: %s", inParams.uri, pathInfo, requestUriPath)

	ifName := pathInfo.Var("name")
	idx := reqpathInfo.Var("index")
	var i32 uint32
	i32 = 0

	if idx != "" {
		i64, _ := strconv.ParseUint(idx, 10, 32)
		i32 = uint32(i64)
	}

	if ifName == "*" {
		return ifName, nil
	}

	if ifName != "" {
		log.Info("YangToDb_intf_tbl_key_xfmr: ifName: ", ifName)
		intfType, _, ierr := getIntfTypeByName(ifName)
		if ierr != nil {
			log.Errorf("Extracting Interface type for Interface: %s failed!", ifName)
			return "", tlerr.New(ierr.Error())
		}
		err = performIfNameKeyXfmrOp(&inParams, &requestUriPath, &ifName, intfType, i32)
		if err != nil {
			return "", tlerr.InvalidArgsError{Format: err.Error()}
		}
	}
	return ifName, err
}

var DbToYang_intf_tbl_key_xfmr KeyXfmrDbToYang = func(inParams XfmrParams) (map[string]interface{}, error) {
	/* Code for DBToYang - Key xfmr. */
	if log.V(3) {
		log.Info("Entering DbToYang_intf_tbl_key_xfmr")
	}
	res_map := make(map[string]interface{})
	log.Info("DbToYang_intf_tbl_key_xfmr: Interface Name = ", inParams.key)
	res_map["name"] = inParams.key
	return res_map, nil
}

var DbToYang_intf_admin_status_xfmr FieldXfmrDbtoYang = func(inParams XfmrParams) (map[string]interface{}, error) {
	var err error
	result := make(map[string]interface{})

	data := (*inParams.dbDataMap)[inParams.curDb]

	intfType, _, ierr := getIntfTypeByName(inParams.key)
	if intfType == IntfTypeUnset || ierr != nil {
		log.Info("DbToYang_intf_admin_status_xfmr - Invalid interface type IntfTypeUnset")
		return result, errors.New("Invalid interface type IntfTypeUnset")
	}

	intTbl := IntfTypeTblMap[intfType]

	tblName, _ := getPortTableNameByDBId(intTbl, inParams.curDb)
	if _, ok := data[tblName]; !ok {
		log.Info("DbToYang_intf_admin_status_xfmr table not found : ", tblName)
		return result, errors.New("table not found : " + tblName)
	}
	pTbl := data[tblName]
	if _, ok := pTbl[inParams.key]; !ok {
		log.Info("DbToYang_intf_admin_status_xfmr Interface not found : ", inParams.key)
		return result, errors.New("Interface not found : " + inParams.key)
	}
	prtInst := pTbl[inParams.key]
	adminStatus, ok := prtInst.Field[PORT_ADMIN_STATUS]
	var status ocbinds.E_OpenconfigInterfaces_Interfaces_Interface_State_AdminStatus
	if ok {
		if adminStatus == "up" {
			status = ocbinds.OpenconfigInterfaces_Interfaces_Interface_State_AdminStatus_UP
		} else {
			status = ocbinds.OpenconfigInterfaces_Interfaces_Interface_State_AdminStatus_DOWN
		}
		result["admin-status"] = ocbinds.E_OpenconfigInterfaces_Interfaces_Interface_State_AdminStatus.ΛMap(status)["E_OpenconfigInterfaces_Interfaces_Interface_State_AdminStatus"][int64(status)].Name
	} else {
		log.Info("Admin status field not found in DB")
	}

	return result, err
}

var YangToDb_intf_enabled_xfmr FieldXfmrYangToDb = func(inParams XfmrParams) (map[string]string, error) {
	res_map := make(map[string]string)

	intfsObj := getIntfsRoot(inParams.ygRoot)
	if intfsObj == nil || len(intfsObj.Interface) < 1 {
		return res_map, nil
	}

	enabled, _ := inParams.param.(*bool)
	var enStr string
	if *enabled {
		enStr = "up"
	} else {
		enStr = "down"
	}
	res_map[PORT_ADMIN_STATUS] = enStr

	return res_map, nil
}

var DbToYang_intf_enabled_xfmr FieldXfmrDbtoYang = func(inParams XfmrParams) (map[string]interface{}, error) {
	var err error
	result := make(map[string]interface{})

	data := (*inParams.dbDataMap)[inParams.curDb]

	intfType, _, ierr := getIntfTypeByName(inParams.key)
	if intfType == IntfTypeUnset || ierr != nil {
		log.Info("DbToYang_intf_enabled_xfmr - Invalid interface type IntfTypeUnset")
		return result, errors.New("Invalid interface type IntfTypeUnset")
	}

	intTbl := IntfTypeTblMap[intfType]

	tblName, _ := getPortTableNameByDBId(intTbl, inParams.curDb)
	if _, ok := data[tblName]; !ok {
		log.Info("DbToYang_intf_enabled_xfmr table not found : ", tblName)
		return result, errors.New("table not found : " + tblName)
	}

	pTbl := data[tblName]
	if _, ok := pTbl[inParams.key]; !ok {
		log.Info("DbToYang_intf_enabled_xfmr Interface not found : ", inParams.key)
		return result, errors.New("Interface not found : " + inParams.key)
	}
	prtInst := pTbl[inParams.key]
	adminStatus, ok := prtInst.Field[PORT_ADMIN_STATUS]
	if ok {
		if adminStatus == "up" {
			result["enabled"] = true
		} else {
			result["enabled"] = false
		}
	} else {
		log.Info("Admin status field not found in DB")
	}
	return result, err
}

var YangToDb_intf_mtu_xfmr FieldXfmrYangToDb = func(inParams XfmrParams) (map[string]string, error) {
	res_map := make(map[string]string)
	var ifName string
	intfsObj := getIntfsRoot(inParams.ygRoot)
	if intfsObj == nil || len(intfsObj.Interface) < 1 {
		return res_map, nil
	} else {
		for infK := range intfsObj.Interface {
			ifName = infK
		}
	}
	intfType, _, _ := getIntfTypeByName(ifName)

	if inParams.oper == DELETE {
		log.Infof("Updating the Interface: %s with default MTU", ifName)
		/* Note: For the mtu delete request, res_map with delete operation and
		   subOp map with update operation (default MTU value) is filled. This is because, transformer default
		   updates the result DS for delete oper with table and key. This needs to be fixed by transformer
		   for deletion of an attribute */
		err := updateDefaultMtu(&inParams, &ifName, intfType, res_map)
		if err != nil {
			log.Errorf("Updating Default MTU for Interface: %s failed", ifName)
			return res_map, err
		}
		return res_map, nil
	}
	// Handles all the operations other than Delete
	intfTypeVal, _ := inParams.param.(*uint16)
	intTypeValStr := strconv.FormatUint(uint64(*intfTypeVal), 10)

	res_map["mtu"] = intTypeValStr
	return res_map, nil
}

var DbToYang_intf_mtu_xfmr FieldXfmrDbtoYang = func(inParams XfmrParams) (map[string]interface{}, error) {
	var err error
	result := make(map[string]interface{})

	data := (*inParams.dbDataMap)[inParams.curDb]

	intfType, _, ierr := getIntfTypeByName(inParams.key)
	if intfType == IntfTypeUnset || ierr != nil {
		log.Info("DbToYang_intf_mtu_xfmr - Invalid interface type IntfTypeUnset")
		return result, errors.New("Invalid interface type IntfTypeUnset")
	}

	intTbl := IntfTypeTblMap[intfType]

	tblName, _ := getPortTableNameByDBId(intTbl, inParams.curDb)
	if _, ok := data[tblName]; !ok {
		log.Info("DbToYang_intf_mtu_xfmr table not found : ", tblName)
		return result, errors.New("table not found : " + tblName)
	}

	pTbl := data[tblName]
	if _, ok := pTbl[inParams.key]; !ok {
		log.Info("DbToYang_intf_mtu_xfmr Interface not found : ", inParams.key)
		return result, errors.New("Interface not found : " + inParams.key)
	}
	prtInst := pTbl[inParams.key]
	mtuStr, ok := prtInst.Field["mtu"]
	if ok {
		mtuVal, err := strconv.ParseUint(mtuStr, 10, 16)
		if err != nil {
			return result, err
		}
		result["mtu"] = mtuVal
	}
	return result, err
}

// YangToDb_intf_eth_port_config_xfmr handles port-speed, and auto-neg config.
var YangToDb_intf_eth_port_config_xfmr SubTreeXfmrYangToDb = func(inParams XfmrParams) (map[string]map[string]db.Value, error) {
	var err error
	memMap := make(map[string]map[string]db.Value)

	pathInfo := NewPathInfo(inParams.uri)
	requestUriPath := (NewPathInfo(inParams.requestUri)).YangPath
	uriIfName := pathInfo.Var("name")
	ifName := uriIfName

	intfType, _, err := getIntfTypeByName(ifName)
	if err != nil {
		errStr := "Invalid Interface"
		err = tlerr.InvalidArgsError{Format: errStr}
		return nil, err
	}

	intfsObj := getIntfsRoot(inParams.ygRoot)
	intfObj := intfsObj.Interface[uriIfName]

	// Need to differentiate between config container delete and any other attribute delete
	if inParams.oper == DELETE {
		/* Handles 3 cases
		   case 1: Deletion request at top-level container / list
		   case 2: Deletion request at ethernet container level
		   case 3: Deletion request at ethernet/config container level */

		//case 1
		if intfObj.Ethernet == nil ||
			//case 2
			intfObj.Ethernet.Config == nil ||
			//case 3
			(intfObj.Ethernet.Config != nil && requestUriPath == "/openconfig-interfaces:interfaces/interface/openconfig-if-ethernet:ethernet/config") {
			return memMap, err
		}
	}

	/* Handle PortSpeed config */
	if intfObj.Ethernet.Config.PortSpeed != 0 {
		res_map := make(map[string]string)
		value := db.Value{Field: res_map}
		intTbl := IntfTypeTblMap[intfType]

		portSpeed := intfObj.Ethernet.Config.PortSpeed
		val, ok := intfOCToSpeedMap[portSpeed]
		if ok {
			err = nil
			res_map[PORT_SPEED] = val
			res_map[PORT_AUTONEG] = "off"
		} else {
			err = tlerr.InvalidArgs("Invalid speed %s", val)
		}

		if err == nil {
			if _, ok := memMap[intTbl.cfgDb.portTN]; !ok {
				memMap[intTbl.cfgDb.portTN] = make(map[string]db.Value)
			}
			memMap[intTbl.cfgDb.portTN][ifName] = value
		}
	} else if requestUriPath == "/openconfig-interfaces:interfaces/interface/openconfig-if-ethernet:ethernet/config/port-speed" {
		if inParams.oper == DELETE {
			err_str := "DELETE request not allowed for port-speed"
			return nil, tlerr.NotSupported(err_str)
		} else {
			log.Error("Unexpected oper ", inParams.oper)
		}
	}

	/* Prepare the map to handle multiple entries */
	res_map := make(map[string]string)
	value := db.Value{Field: res_map}

	/* Handle AutoNegotiate config */
	if intfObj.Ethernet.Config.AutoNegotiate != nil {
		if intfType == IntfTypeEthernet {
			intTbl := IntfTypeTblMap[intfType]
			autoNeg := intfObj.Ethernet.Config.AutoNegotiate
			var enStr string
			if *autoNeg {
				enStr = "on"
			} else {
				enStr = "off"
			}
			res_map[PORT_AUTONEG] = enStr

			if _, ok := memMap[intTbl.cfgDb.portTN]; !ok {
				memMap[intTbl.cfgDb.portTN] = make(map[string]db.Value)
			}
			memMap[intTbl.cfgDb.portTN][ifName] = value
		} else {
			return nil, errors.New("AutoNegotiate config not supported for given Interface type")
		}
	}

	return memMap, err
}

// DbToYang_intf_eth_port_config_xfmr is to handle DB to yang translation of port-speed, auto-neg and aggregate-id config.
var DbToYang_intf_eth_port_config_xfmr SubTreeXfmrDbToYang = func(inParams XfmrParams) error {
	var err error
	intfsObj := getIntfsRoot(inParams.ygRoot)
	pathInfo := NewPathInfo(inParams.uri)
	uriIfName := pathInfo.Var("name")
	ifName := uriIfName

	intfType, _, err := getIntfTypeByName(ifName)
	if err != nil {
		errStr := "Invalid Interface"
		err = tlerr.InvalidArgsError{Format: errStr}
		return err
	}
	intTbl := IntfTypeTblMap[intfType]
	tblName := intTbl.cfgDb.portTN
	entry, dbErr := inParams.dbs[db.ConfigDB].GetEntry(&db.TableSpec{Name: tblName}, db.Key{Comp: []string{ifName}})
	if dbErr != nil {
		errStr := "Invalid Interface"
		err = tlerr.InvalidArgsError{Format: errStr}
		return err
	}
	targetUriPath := pathInfo.YangPath
	if strings.HasPrefix(targetUriPath, "/openconfig-interfaces:interfaces/interface/openconfig-if-ethernet:ethernet/config") {
		get_cfg_obj := false
		var intfObj *ocbinds.OpenconfigInterfaces_Interfaces_Interface
		if intfsObj != nil && intfsObj.Interface != nil && len(intfsObj.Interface) > 0 {
			var ok bool = false
			if intfObj, ok = intfsObj.Interface[uriIfName]; !ok {
				intfObj, _ = intfsObj.NewInterface(uriIfName)
			}
			ygot.BuildEmptyTree(intfObj)
		} else {
			ygot.BuildEmptyTree(intfsObj)
			intfObj, _ = intfsObj.NewInterface(uriIfName)
			ygot.BuildEmptyTree(intfObj)
		}
		ygot.BuildEmptyTree(intfObj.Ethernet)
		ygot.BuildEmptyTree(intfObj.Ethernet.Config)

		if targetUriPath == "/openconfig-interfaces:interfaces/interface/openconfig-if-ethernet:ethernet/config" {
			get_cfg_obj = true
		}
		var errStr string

		if entry.IsPopulated() {
			if get_cfg_obj || targetUriPath == "/openconfig-interfaces:interfaces/interface/openconfig-if-ethernet:ethernet/config/auto-negotiate" {
				autoNeg, ok := entry.Field[PORT_AUTONEG]
				if ok {
					var oc_auto_neg bool
					if autoNeg == "on" || autoNeg == "true" {
						oc_auto_neg = true
					} else {
						oc_auto_neg = false
					}
					intfObj.Ethernet.Config.AutoNegotiate = &oc_auto_neg
				} else {
					errStr = "auto-negotiate not set"
				}
			}
			if get_cfg_obj || targetUriPath == "/openconfig-interfaces:interfaces/interface/openconfig-if-ethernet:ethernet/config/port-speed" {
				speed, ok := entry.Field[PORT_SPEED]
				portSpeed := ocbinds.OpenconfigIfEthernet_ETHERNET_SPEED_UNSET
				if ok {
					portSpeed, err = getDbToYangSpeed(speed)
					intfObj.Ethernet.Config.PortSpeed = portSpeed
				} else {
					errStr = "port-speed not set"
				}
			}

		} else {
			errStr = "Attribute not set"
		}
		if !get_cfg_obj && errStr != "" {
			err = tlerr.InvalidArgsError{Format: errStr}
		}
	}

	return err
}

var DbToYangPath_intf_eth_port_config_path_xfmr PathXfmrDbToYangFunc = func(params XfmrDbToYgPathParams) error {
	log.Info("DbToYangPath_intf_eth_port_config_path_xfmr: params: ", params)

	intfRoot := "/openconfig-interfaces:interfaces/interface"
	if params.tblName != "PORT" {
		log.Info("DbToYangPath_intf_eth_port_config_path_xfmr: from wrong table: ", params.tblName)
		return nil
	}

	if (params.tblName == "PORT") && (len(params.tblKeyComp) > 0) {
		params.ygPathKeys[intfRoot+"/name"] = params.tblKeyComp[0]
	} else {
		log.Info("DbToYangPath_intf_eth_port_config_path_xfmr, wrong param: tbl ", params.tblName, " key ", params.tblKeyComp)
		return nil
	}

	log.Info("DbToYangPath_intf_eth_port_config_path_xfmr: params.ygPathkeys: ", params.ygPathKeys)

	return nil
}

var DbToYang_intf_eth_auto_neg_xfmr FieldXfmrDbtoYang = func(inParams XfmrParams) (map[string]interface{}, error) {
	var err error
	result := make(map[string]interface{})

	data := (*inParams.dbDataMap)[inParams.curDb]
	intfType, _, ierr := getIntfTypeByName(inParams.key)
	if intfType == IntfTypeUnset || ierr != nil {
		log.Info("DbToYang_intf_eth_auto_neg_xfmr - Invalid interface type IntfTypeUnset")
		return result, errors.New("Invalid interface type IntfTypeUnset")
	}
	if IntfTypeEthernet != intfType {
		return result, nil
	}
	intTbl := IntfTypeTblMap[intfType]

	tblName, _ := getPortTableNameByDBId(intTbl, inParams.curDb)
	pTbl := data[tblName]
	prtInst := pTbl[inParams.key]
	autoNeg, ok := prtInst.Field[PORT_AUTONEG]
	if ok {
		if autoNeg == "on" || autoNeg == "true" {
			result["auto-negotiate"] = true
		} else {
			result["auto-negotiate"] = false
		}
	} else {
		log.Info("auto-negotiate field not found in DB")
	}
	return result, err
}

var DbToYang_intf_eth_port_speed_xfmr FieldXfmrDbtoYang = func(inParams XfmrParams) (map[string]interface{}, error) {
	var err error
	result := make(map[string]interface{})

	data := (*inParams.dbDataMap)[inParams.curDb]
	intfType, _, ierr := getIntfTypeByName(inParams.key)
	if intfType == IntfTypeUnset || ierr != nil {
		log.Info("DbToYang_intf_eth_port_speed_xfmr - Invalid interface type IntfTypeUnset")
		return result, errors.New("Invalid interface type IntfTypeUnset")
	}

	intTbl := IntfTypeTblMap[intfType]

	tblName, _ := getPortTableNameByDBId(intTbl, inParams.curDb)
	pTbl := data[tblName]
	prtInst := pTbl[inParams.key]
	speed, ok := prtInst.Field[PORT_SPEED]
	portSpeed := ocbinds.OpenconfigIfEthernet_ETHERNET_SPEED_UNSET
	if ok {
		portSpeed, err = getDbToYangSpeed(speed)
		result["port-speed"] = ocbinds.E_OpenconfigIfEthernet_ETHERNET_SPEED.ΛMap(portSpeed)["E_OpenconfigIfEthernet_ETHERNET_SPEED"][int64(portSpeed)].Name
	} else {
		log.Info("Speed field not found in DB")
	}

	return result, err
}

var intf_post_xfmr PostXfmrFunc = func(inParams XfmrParams) (map[string]map[string]db.Value, error) {

	requestUriPath := (NewPathInfo(inParams.requestUri)).YangPath
	retDbDataMap := (*inParams.dbDataMap)[inParams.curDb]
	log.Info("Entering intf_post_xfmr")
	log.Info(requestUriPath)
	xpath, _, _ := XfmrRemoveXPATHPredicates(inParams.requestUri)

	if inParams.oper == DELETE {

		err_str := "Delete not allowed at this container"
		/* Preventing delete at IPv6 config level*/
		if xpath == "/openconfig-interfaces:interfaces/interface/subinterfaces/subinterface/ipv6/config" {
			log.Info("In interface Post transformer for DELETE op ==> URI : ", inParams.requestUri)
			return retDbDataMap, tlerr.NotSupported(err_str)
		}

		/* For delete request and for fields with default value, transformer adds subOp map with update operation (to update with default value).
		   So, adding code to clear the update SubOp map for delete operation to go through for the following requestUriPath */
		if xpath == "/openconfig-interfaces:interfaces/interface/subinterfaces/subinterface/ipv6/config/enabled" {
			if len(inParams.subOpDataMap) > 0 {
				dbMap := make(map[string]map[string]db.Value)
				if inParams.subOpDataMap[4] != nil && (*inParams.subOpDataMap[4])[db.ConfigDB] != nil {
					(*inParams.subOpDataMap[4])[db.ConfigDB] = dbMap
				}
				log.Info("intf_post_xfmr inParams.subOpDataMap :", inParams.subOpDataMap)
			}
		}
	} else if inParams.oper == UPDATE {
		if replace, ok := inParams.subOpDataMap[REPLACE]; ok {
			if (*replace)[db.ConfigDB] != nil {
				if portTable, ok := (*replace)[db.ConfigDB]["PORT"]; ok {
					for key := range portTable {
						delete(inParams.yangDefValMap["PORT"], key)
					}
				}
			}
		}
	}
	return retDbDataMap, nil
}

var intf_pre_xfmr PreXfmrFunc = func(inParams XfmrParams) error {
	var err error
	if inParams.oper == DELETE || inParams.oper == REPLACE {
		requestUriPath := (NewPathInfo(inParams.requestUri)).YangPath
		if log.V(3) {
			log.Info("intf_pre_xfmr:- Request URI path = ", requestUriPath)
		}
		errStr := "Delete operation not supported for this path - "
		if inParams.oper == REPLACE {
			errStr = "Replace operation not supported for this path - "
		}
		switch requestUriPath {
		case "/openconfig-interfaces:interfaces":
			errStr += requestUriPath
			return tlerr.InvalidArgsError{Format: errStr}
		case "/openconfig-interfaces:interfaces/interface":
			pathInfo := NewPathInfo(inParams.uri)
			if len(pathInfo.Vars) == 0 {
				errStr += requestUriPath
				return tlerr.InvalidArgsError{Format: errStr}
			}
		}
	}
	return err
}

func populateVlanSubIntfTblKeys(inParams XfmrParams) error {
	var key string

	(*inParams.dbDataMap)[db.ConfigDB]["VLAN_SUB_INTERFACE"] = make(map[string]db.Value)
	mapIntfKeys, _ := inParams.d.GetKeys(&db.TableSpec{Name: "VLAN_SUB_INTERFACE"})
	if len(mapIntfKeys) > 0 {
		for _, intfKey := range mapIntfKeys {
			key = intfKey.Get(0)
			key = *(&key)
			if _, ok := (*inParams.dbDataMap)[db.ConfigDB]["VLAN_SUB_INTERFACE"][key]; !ok {
				(*inParams.dbDataMap)[db.ConfigDB]["VLAN_SUB_INTERFACE"][key] = db.Value{Field: make(map[string]string)}
				(*inParams.dbDataMap)[db.ConfigDB]["VLAN_SUB_INTERFACE"][key].Field["NULL"] = "NULL"
			}
		}
	}

	if log.V(3) {
		log.Infof("populateVlanSubIntfTblKeys, configDB dbdataMap[\"VLAN_SUB_INTERFACE\"]: %v ", (*inParams.dbDataMap)[db.ConfigDB]["VLAN_SUB_INTERFACE"])
	}

	return nil
}

var intf_subintfs_table_xfmr TableXfmrFunc = func(inParams XfmrParams) ([]string, error) {
	var tblList []string

	pathInfo := NewPathInfo(inParams.uri)
	ifName := pathInfo.Var("name")
	idx := pathInfo.Var("index")

	if inParams.oper == SUBSCRIBE {
		var _intfTypeList []E_InterfaceType

		_addSubIntfToList := func() {
			if idx == "*" || idx != "0" {
				_intfTypeList = append(_intfTypeList, IntfTypeSubIntf)
			}
		}

		if ifName == "*" {
			_intfTypeList = append(_intfTypeList, IntfTypeEthernet, IntfTypeMgmt, IntfTypePortChannel, IntfTypeLoopback)
			_addSubIntfToList()
		} else {
			_ifType, _, _err := getIntfTypeByName(ifName)
			if _ifType == IntfTypeUnset || _err != nil {
				return tblList, errors.New("Invalid interface type IntfTypeUnset")
			}
			if IntfTypeVlan == _ifType || IntfTypeVxlan == _ifType {
				return tblList, nil
			}
			_intfTypeList = append(_intfTypeList, _ifType)
			_addSubIntfToList()
		}

		for _, _ifType := range _intfTypeList {
			_intfTblName, _ := getIntfTableNameByDBId(IntfTypeTblMap[_ifType], inParams.curDb)
			tblList = append(tblList, _intfTblName)
		}

		log.V(3).Info("intf_subintfs_table_xfmr: URI: ", inParams.uri, " OP:", inParams.oper, " ifName:", ifName, " idx:", idx, " tblList:", tblList)
		return tblList, nil
	}

	//if GET at top level, populate the VLAN_SUB_INTERFACE table with keys and store the flag to txCache
	val, present := inParams.txCache.Load("vlan_sub_intf_tbl_keys_read")
	reqPathInfo := NewPathInfo(inParams.requestUri)
	requestUriPath := reqPathInfo.YangPath
	var reqUriIfName string = reqPathInfo.Var("name")
	if inParams.oper == GET && (requestUriPath == "/openconfig-interfaces:interfaces" ||
		requestUriPath == "/openconfig-interfaces:interfaces/interface") && reqUriIfName == "" {
		if !present || val != true {
			if inParams.dbDataMap != nil {
				populateVlanSubIntfTblKeys(inParams)
				inParams.txCache.Store("vlan_sub_intf_tbl_keys_read", true)
				val = true
				log.Info("intf_subintfs_table_xfmr, cached vlan_sub_intf_tbl_keys_read")
			}
		}
	}

	if idx == "" {
		if inParams.oper == GET || inParams.oper == DELETE {
			if inParams.dbDataMap != nil {
				(*inParams.dbDataMap)[db.ConfigDB]["SUBINTF_TBL"] = make(map[string]db.Value)
				(*inParams.dbDataMap)[db.ConfigDB]["SUBINTF_TBL"]["0"] = db.Value{Field: make(map[string]string)}
				tblList = append(tblList, "SUBINTF_TBL")
				tblList = append(tblList, "VLAN_SUB_INTERFACE")
			}
		}
		log.Info("intf_subintfs_table_xfmr - Subinterface get operation ")
	} else {
		if idx == "0" {
			if inParams.dbDataMap != nil {
				(*inParams.dbDataMap)[db.ConfigDB]["SUBINTF_TBL"] = make(map[string]db.Value)
				(*inParams.dbDataMap)[db.ConfigDB]["SUBINTF_TBL"]["0"] = db.Value{Field: make(map[string]string)}
				(*inParams.dbDataMap)[db.ConfigDB]["SUBINTF_TBL"]["0"].Field["NULL"] = "NULL"
			}
			tblList = append(tblList, "SUBINTF_TBL")
		} else {
			if inParams.dbDataMap != nil {
				(*inParams.dbDataMap)[db.ConfigDB]["VLAN_SUB_INTERFACE"] = make(map[string]db.Value)
				if val == true {
					//reset cached flag
					inParams.txCache.Store("vlan_sub_intf_tbl_keys_read", false)
					log.Info("intf_subintfs_table_xfmr, reset vlan_sub_intf_tbl_keys_read cache")
				}
			}
			tblList = append(tblList, "VLAN_SUB_INTERFACE")
			*inParams.isVirtualTbl = false
		}
		if log.V(3) {
			log.Info("intf_subintfs_table_xfmr - Subinterface get operation ")
		}
	}

	return tblList, nil
}

var YangToDb_intf_subintfs_xfmr KeyXfmrYangToDb = func(inParams XfmrParams) (string, error) {
	var subintf_key string
	var err error

	log.Info("YangToDb_intf_subintfs_xfmr - inParams.uri ", inParams.uri)

	pathInfo := NewPathInfo(inParams.uri)
	ifName := pathInfo.Var("name")
	if ifName == "*" {
		return ifName, nil
	}

	intfType, _, ierr := getIntfTypeByName(ifName)
	if intfType == IntfTypeUnset || ierr != nil {
		return ifName, errors.New("Invalid interface type IntfTypeUnset")
	}
	if IntfTypeVlan == intfType {
		log.Info("YangToDb_intf_subintfs_xfmr - IntfTypeVlan")
		return ifName, nil
	}

	idx := pathInfo.Var("index")

	if idx != "0" && idx != "*" && idx != "" {
		subintf_key = ifName + "." + idx
	} else { /* For get 0 index case & subscribe index * case */
		subintf_key = idx
	}

	log.Info("YangToDb_intf_subintfs_xfmr - return subintf_key ", subintf_key)
	return subintf_key, err
}

var DbToYang_intf_subintfs_xfmr KeyXfmrDbToYang = func(inParams XfmrParams) (map[string]interface{}, error) {

	if log.V(3) {
		log.Info("Entering DbToYang_intf_subintfs_xfmr")
	}
	var idx string

	if strings.Contains(inParams.key, ".") {
		key_split := strings.Split(inParams.key, ".")
		idx = key_split[1]
	} else {
		idx = inParams.key
	}

	rmap := make(map[string]interface{})
	var err error
	i64, _ := strconv.ParseUint(idx, 10, 32)
	rmap["index"] = i64

	log.Info("DbToYang_intf_subintfs_xfmr rmap ", rmap)
	return rmap, err
}

var YangToDb_subintf_ip_addr_key_xfmr KeyXfmrYangToDb = func(inParams XfmrParams) (string, error) {
	if log.V(3) {
		log.Info("Entering YangToDb_subintf_ip_addr_key_xfmr")
	}
	var err error
	var inst_key string
	pathInfo := NewPathInfo(inParams.uri)
	inst_key = pathInfo.Var("ip")
	log.Infof("URI:%v Interface IP:%v", inParams.uri, inst_key)
	return inst_key, err
}
var DbToYang_subintf_ip_addr_key_xfmr KeyXfmrDbToYang = func(inParams XfmrParams) (map[string]interface{}, error) {
	if log.V(3) {
		log.Info("Entering DbToYang_subintf_ip_addr_key_xfmr")
	}
	rmap := make(map[string]interface{})
	return rmap, nil
}

var YangToDb_subif_index_xfmr FieldXfmrYangToDb = func(inParams XfmrParams) (map[string]string, error) {
	res_map := make(map[string]string)
	var err error

	pathInfo := NewPathInfo(inParams.uri)
	uriIfName := pathInfo.Var("name")
	log.Info(uriIfName)
	ifName := uriIfName

	res_map["parent"] = ifName

	log.Info("YangToDb_subif_index_xfmr: res_map:", res_map)
	return res_map, err
}

var DbToYang_subif_index_xfmr FieldXfmrDbtoYang = func(inParams XfmrParams) (map[string]interface{}, error) {
	res_map := make(map[string]interface{})

	pathInfo := NewPathInfo(inParams.uri)
	id := pathInfo.Var("index")
	log.Info("DbToYang_subif_index_xfmr: Sub-interface Index = ", id)
	i64, _ := strconv.ParseUint(id, 10, 32)
	res_map["index"] = i64
	return res_map, nil
}

var DbToYangPath_intf_ip_path_xfmr PathXfmrDbToYangFunc = func(params XfmrDbToYgPathParams) error {
	ifRoot := "/openconfig-interfaces:interfaces/interface"
	subIf := ifRoot + "/subinterfaces/subinterface"
	dbKey := ""

	log.Info("DbToYangPath_intf_ip_path_xfmr: params: ", params)

	uiName := &params.tblKeyComp[0]
	ifParts := strings.Split(*uiName, ".")

	params.ygPathKeys[ifRoot+"/name"] = ifParts[0]

	if params.tblName == "INTERFACE" || params.tblName == "VLAN_INTERFACE" ||
		params.tblName == "INTF_TABLE" || params.tblName == "MGMT_INTERFACE" ||
		params.tblName == "VLAN_SUB_INTERFACE" || params.tblName == "MGMT_INTF_TABLE" ||
		params.tblName == "PORTCHANNEL_INTERFACE" {

		addrPath := "/openconfig-if-ip:ipv4/addresses/address/ip"

		/* For APPL_DB IPv6 case, addr is split [fe80  56bf 64ff feba 3bc0/64] instead of
		   [fe80::56bf:64ff:feba:3bc0/64]
		   Handle this case
		*/
		dbKey = strings.Join(params.tblKeyComp[1:], ":")

		if len(params.tblKeyComp) > 2 || strings.Contains(dbKey, ":") {
			addrPath = "/openconfig-if-ip:ipv6/addresses/address/ip"
		}

		ipKey := strings.Split(dbKey, "/")

		if strings.HasPrefix(params.tblKeyComp[0], "Vlan") {
			return nil
		} else {
			if len(ifParts) > 1 {
				params.ygPathKeys[subIf+"/index"] = ifParts[1]
			} else {
				params.ygPathKeys[subIf+"/index"] = "0"
			}
			params.ygPathKeys[subIf+addrPath] = ipKey[0]
		}
	}

	log.Infof("DbToYangPath_intf_ip_path_xfmr:  tblName:%v dbKey:[%v] params.ygPathKeys: %v", params.tblName, dbKey, params.ygPathKeys)
	return nil
}

/* Get interface to IP mapping for all interfaces in the given table */
func getCachedAllIntfIpMap(dbCl *db.DB, tblName string, ipv4 bool, ipv6 bool, ip string, tblPattern *db.Table) (map[string]map[string]db.Value, error) {
	var err error
	all := true
	intfIpMap := make(map[string]map[string]db.Value)
	if !ipv4 || !ipv6 {
		all = false
	}
	log.V(3).Info("Inside getCachedAllIntfIpMap: Get Interface IP Info from table cache to Internal DS")

	//Get keys from tblPattern
	keys, err := tblPattern.GetKeys()
	if err != nil {
		return intfIpMap, err
	}

	for _, key := range keys {
		ifName := key.Get(0)
		intfType, _, ierr := getIntfTypeByName(ifName)
		if intfType == IntfTypeUnset || ierr != nil {
			continue
		}

		if !all {
			ipB, _, _ := parseCIDR(key.Get(1))
			if (validIPv4(ipB.String()) && (!ipv4)) ||
				(validIPv6(ipB.String()) && (!ipv6)) {
				continue
			}
			if ip != "" {
				if ipB.String() != ip {
					continue
				}
			}
		}

		ipInfo, _ := tblPattern.GetEntry(db.Key{Comp: []string{ifName, key.Get(1)}})

		if _, ok := intfIpMap[ifName]; !ok {
			intfIpMap[ifName] = make(map[string]db.Value)
		}

		intfIpMap[ifName][key.Get(1)] = ipInfo
	}
	return intfIpMap, err
}

func handleAllIntfIPGetForTable(inParams XfmrParams, tblName string, isAppDb bool) error {
	var err error
	intfsObj := getIntfsRoot(inParams.ygRoot)
	var intfObj *ocbinds.OpenconfigInterfaces_Interfaces_Interface

	var tblPattern db.Table

	currDb := inParams.dbs[db.ConfigDB]
	if isAppDb {
		currDb = inParams.dbs[db.ApplDB]
	}

	dbTbl := db.TableSpec{Name: tblName, CompCt: 2}
	keyPattern := db.Key{Comp: []string{"*", "*"}}
	tblPattern, err = currDb.GetTablePattern(&dbTbl, keyPattern)

	if err != nil {
		log.Error("handleAllIntfIPGetForTable: GetTablePattern() returns err: %v", err)
		return nil
	}

	var intfIpMap map[string]map[string]db.Value
	if isAppDb {
		intfIpMap, err = getCachedAllIntfIpMap(inParams.dbs[db.ApplDB], tblName, true, true, "", &tblPattern)
	} else {
		intfIpMap, err = getCachedAllIntfIpMap(inParams.dbs[db.ConfigDB], tblName, true, true, "", &tblPattern)
	}

	if log.V(3) {
		log.Infof("handleAllIntfIPGetForTable, tbl: %v, intfIpMap: %v", tblName, intfIpMap)
	}

	if len(intfIpMap) == 0 {
		return nil
	}

	i32 := uint32(0)

	// YGOT filling
	for intfName, ipMapDB := range intfIpMap {
		if strings.HasPrefix(intfName, "Vlan") {
			continue
		}

		var subIdxStr string
		var name string
		if strings.Contains(intfName, ".") {
			intfLongName := *(&intfName)
			parts := strings.Split(intfLongName, ".")
			name = *(&parts[0])
			subIdxStr = parts[1]
			tmpIdx, _ := strconv.Atoi(subIdxStr)
			i32 = uint32(tmpIdx)
		} else {
			name = *(&intfName)
		}

		if intfsObj != nil && intfsObj.Interface != nil && len(intfsObj.Interface) > 0 {
			var ok bool = false
			if intfObj, ok = intfsObj.Interface[name]; !ok {
				intfObj, _ = intfsObj.NewInterface(name)
			}
			ygot.BuildEmptyTree(intfObj)
			if intfObj.Subinterfaces == nil {
				ygot.BuildEmptyTree(intfObj.Subinterfaces)
			}
		} else {
			ygot.BuildEmptyTree(intfsObj)
			intfObj, _ = intfsObj.NewInterface(name)
			ygot.BuildEmptyTree(intfObj)
		}

		if log.V(3) {
			log.Infof("handleAllIntfIPGetForTable, intfName: %v, name: %v, subidx: %v, ipmap: %v", intfName, name, i32, ipMapDB)
		}

		if isAppDb {
			convertIpMapToOC(ipMapDB, intfObj, true, i32)
		} else {
			convertIpMapToOC(ipMapDB, intfObj, false, i32)
		}
	}
	return nil
}

// ValidateIntfProvisionedForRelay helper function to validate IP address deletion if DHCP relay is provisioned
func ValidateIntfProvisionedForRelay(d *db.DB, ifName string, prefixIp string, entry *db.Value) (bool, error) {
	var tblList string

	intfType, _, ierr := getIntfTypeByName(ifName)
	if intfType == IntfTypeUnset || ierr != nil {
		log.Info("ValidateIntfProvisionedForRelay - Invalid interface type IntfTypeUnset")
		return false, errors.New("Invalid InterfaceType")
	}

	// get all the IP addresses on this interface, refer to the intf table name
	intTbl := IntfTypeTblMap[intfType]
	tblList = intTbl.cfgDb.intfTN

	// for VLAN - DHCP info is stored in the VLAN Table
	if intfType == IntfTypeVlan {
		tblList = intTbl.cfgDb.portTN
	}

	if entry == nil || intfType == IntfTypeVlan {
		ent, dbErr := d.GetEntry(&db.TableSpec{Name: tblList}, db.Key{Comp: []string{ifName}})
		entry = &ent
		if dbErr != nil {
			log.Warning("Failed to read entry from config DB, " + tblList + " " + ifName)
			return false, nil
		}
	}

	//check if dhcp_sever is provisioned for ipv4
	if strings.Contains(prefixIp, ".") || strings.Contains(prefixIp, "ipv4") {
		log.V(2).Info("ValidateIntfProvisionedForRelay  - IPv4Check")
		log.V(2).Info(entry)
		if len(entry.Field["dhcp_servers@"]) > 0 {
			return true, nil
		}
		//} else if (strings.Contains(prefixIp, ":") && numIpv6 < 2) || strings.Contains(prefixIp, "ipv6") {
	} else if (strings.Contains(prefixIp, ":")) || strings.Contains(prefixIp, "ipv6") {
		//check if dhcpv6_sever is provisioned for ipv6
		log.V(2).Info("ValidateIntfProvisionedForRelay  - IPv6Check")
		log.V(2).Info(entry)
		if len(entry.Field["dhcpv6_servers@"]) > 0 {
			return true, nil
		}
	}
	return false, nil
}

func handleIntfIPGetByTargetURI(inParams XfmrParams, targetUriPath string, ifName string, intfObj *ocbinds.OpenconfigInterfaces_Interfaces_Interface) error {
	var ipMap map[string]db.Value
	var err error

	pathInfo := NewPathInfo(inParams.uri)
	ipAddr := pathInfo.Var("ip")
	idx := pathInfo.Var("index")
	i32 := uint32(0)
	if idx != "0" {
		ifName = *utils.GetSubInterfaceDBKeyfromParentInterfaceAndSubInterfaceID(&ifName, &idx)
		i64, _ := strconv.ParseUint(idx, 10, 32)
		i32 = uint32(i64)
	}
	intfType, _, ierr := getIntfTypeByName(ifName)
	if intfType == IntfTypeUnset || ierr != nil {
		errStr := "Invalid interface type IntfTypeUnset"
		log.Info("YangToDb_intf_subintf_ip_xfmr : uri:" + inParams.uri + ": " + errStr)
		return errors.New(errStr)
	}
	intTbl := IntfTypeTblMap[intfType]

	if len(ipAddr) > 0 {
		// Check if the given IP is configured on interface
		keyPattern := ifName + ":" + ipAddr + "/*"
		ipKeys, err := inParams.dbs[db.ApplDB].GetKeysByPattern(&db.TableSpec{Name: intTbl.appDb.intfTN}, keyPattern)
		if err != nil || len(ipKeys) == 0 {
			return tlerr.NotFound("Resource not found")
		}
	}

	if strings.HasPrefix(targetUriPath, "/openconfig-interfaces:interfaces/interface/subinterfaces/subinterface/ipv4/addresses/address/config") ||
		strings.HasPrefix(targetUriPath, "/openconfig-interfaces:interfaces/interface/subinterfaces/subinterface/openconfig-if-ip:ipv4/addresses/address/config") {
		ipMap, err = getIntfIpByName(inParams.dbs[db.ConfigDB], intTbl.cfgDb.intfTN, ifName, true, false, ipAddr)
		log.Info("handleIntfIPGetByTargetURI : ipv4 config ipMap - : ", ipMap)
		convertIpMapToOC(ipMap, intfObj, false, i32)
	} else if strings.HasPrefix(targetUriPath, "/openconfig-interfaces:interfaces/interface/subinterfaces/subinterface/openconfig-if-ip:ipv6/addresses/address/config") ||
		strings.HasPrefix(targetUriPath, "/openconfig-interfaces:interfaces/interface/subinterfaces/subinterface/ipv6/addresses/address/config") {
		ipMap, err = getIntfIpByName(inParams.dbs[db.ConfigDB], intTbl.cfgDb.intfTN, ifName, false, true, ipAddr)
		log.Info("handleIntfIPGetByTargetURI : ipv6 config ipMap - : ", ipMap)
		convertIpMapToOC(ipMap, intfObj, false, i32)
	} else if strings.HasPrefix(targetUriPath, "/openconfig-interfaces:interfaces/interface/subinterfaces/subinterface/ipv4/addresses/address/state") ||
		strings.HasPrefix(targetUriPath, "/openconfig-interfaces:interfaces/interface/subinterfaces/subinterface/openconfig-if-ip:ipv4/addresses/address/state") {
		ipMap, err = getIntfIpByName(inParams.dbs[db.ApplDB], intTbl.appDb.intfTN, ifName, true, false, ipAddr)
		log.Info("handleIntfIPGetByTargetURI : ipv4 state ipMap - : ", ipMap)
		convertIpMapToOC(ipMap, intfObj, true, i32)
	} else if strings.HasPrefix(targetUriPath, "/openconfig-interfaces:interfaces/interface/subinterfaces/subinterface/openconfig-if-ip:ipv6/addresses/address/state") ||
		strings.HasPrefix(targetUriPath, "/openconfig-interfaces:interfaces/interface/subinterfaces/subinterface/ipv6/addresses/address/state") {
		ipMap, err = getIntfIpByName(inParams.dbs[db.ApplDB], intTbl.appDb.intfTN, ifName, false, true, ipAddr)
		log.Info("handleIntfIPGetByTargetURI : ipv6 state ipMap - : ", ipMap)
		convertIpMapToOC(ipMap, intfObj, true, i32)
	} else if strings.HasPrefix(targetUriPath, "/openconfig-interfaces:interfaces/interface/subinterfaces/subinterface/ipv4/addresses") ||
		strings.HasPrefix(targetUriPath, "/openconfig-interfaces:interfaces/interface/subinterfaces/subinterface/openconfig-if-ip:ipv4/addresses") {
		ipMap, err = getIntfIpByName(inParams.dbs[db.ConfigDB], intTbl.cfgDb.intfTN, ifName, true, false, ipAddr)
		if err == nil {
			log.Info("handleIntfIPGetByTargetURI : ipv4 config ipMap - : ", ipMap)
			convertIpMapToOC(ipMap, intfObj, false, i32)
		}
		ipMap, err = getIntfIpByName(inParams.dbs[db.ApplDB], intTbl.appDb.intfTN, ifName, true, false, ipAddr)
		if err == nil {
			log.Info("handleIntfIPGetByTargetURI : ipv4 state ipMap - : ", ipMap)
			convertIpMapToOC(ipMap, intfObj, true, i32)
		}
	} else if strings.HasPrefix(targetUriPath, "/openconfig-interfaces:interfaces/interface/subinterfaces/subinterface/ipv6/addresses") ||
		strings.HasPrefix(targetUriPath, "/openconfig-interfaces:interfaces/interface/subinterfaces/subinterface/openconfig-if-ip:ipv6/addresses") {
		ipMap, err = getIntfIpByName(inParams.dbs[db.ConfigDB], intTbl.cfgDb.intfTN, ifName, false, true, ipAddr)
		if err == nil {
			log.Info("handleIntfIPGetByTargetURI : ipv6 config ipMap - : ", ipMap)
			convertIpMapToOC(ipMap, intfObj, false, i32)
		}
		ipMap, err = getIntfIpByName(inParams.dbs[db.ApplDB], intTbl.appDb.intfTN, ifName, false, true, ipAddr)
		if err == nil {
			log.Info("handleIntfIPGetByTargetURI : ipv6 state ipMap - : ", ipMap)
			convertIpMapToOC(ipMap, intfObj, true, i32)
		}
	}
	return err
}
func convertIpMapToOC(intfIpMap map[string]db.Value, ifInfo *ocbinds.OpenconfigInterfaces_Interfaces_Interface, isState bool, subintfid uint32) error {
	var subIntf *ocbinds.OpenconfigInterfaces_Interfaces_Interface_Subinterfaces_Subinterface
	var err error

	if _, ok := ifInfo.Subinterfaces.Subinterface[subintfid]; !ok {
		_, err = ifInfo.Subinterfaces.NewSubinterface(subintfid)
		if err != nil {
			log.Error("Creation of subinterface subtree failed!")
			return err
		}
	}

	subIntf = ifInfo.Subinterfaces.Subinterface[subintfid]
	ygot.BuildEmptyTree(subIntf)
	ygot.BuildEmptyTree(subIntf.Ipv4)
	ygot.BuildEmptyTree(subIntf.Ipv6)

	for ipKey, _ := range intfIpMap {
		log.Info("IP address = ", ipKey)
		ipB, ipNetB, _ := parseCIDR(ipKey)
		v4Flag := false
		v6Flag := false

		var v4Address *ocbinds.OpenconfigInterfaces_Interfaces_Interface_Subinterfaces_Subinterface_Ipv4_Addresses_Address
		var v6Address *ocbinds.OpenconfigInterfaces_Interfaces_Interface_Subinterfaces_Subinterface_Ipv6_Addresses_Address
		if validIPv4(ipB.String()) {
			if _, ok := subIntf.Ipv4.Addresses.Address[ipB.String()]; !ok {
				_, err = subIntf.Ipv4.Addresses.NewAddress(ipB.String())
			}
			v4Address = subIntf.Ipv4.Addresses.Address[ipB.String()]
			v4Flag = true
		} else if validIPv6(ipB.String()) {
			if _, ok := subIntf.Ipv6.Addresses.Address[ipB.String()]; !ok {
				_, err = subIntf.Ipv6.Addresses.NewAddress(ipB.String())
			}
			v6Address = subIntf.Ipv6.Addresses.Address[ipB.String()]
			v6Flag = true
		} else {
			log.Error("Invalid IP address " + ipB.String())
			continue
		}
		if err != nil {
			log.Error("Creation of address subtree failed!")
			return err
		}
		if v4Flag {
			ygot.BuildEmptyTree(v4Address)
			ipStr := new(string)
			*ipStr = ipB.String()
			v4Address.Ip = ipStr
			prfxLen := new(uint8)
			*prfxLen = ipNetB.Bits()
			ipv4Str := new(string)
			*ipv4Str = "ipv4"
			if isState {
				v4Address.State.Ip = ipStr
				v4Address.State.PrefixLength = prfxLen
				v4Address.State.Family = ipv4Str
			} else {
				v4Address.Config.Ip = ipStr
				v4Address.Config.PrefixLength = prfxLen
			}
		}
		if v6Flag {
			ygot.BuildEmptyTree(v6Address)
			ipStr := new(string)
			*ipStr = ipB.String()
			v6Address.Ip = ipStr
			prfxLen := new(uint8)
			*prfxLen = ipNetB.Bits()
			ipv6Str := new(string)
			*ipv6Str = "ipv6"
			if isState {
				v6Address.State.Ip = ipStr
				v6Address.State.PrefixLength = prfxLen
				v6Address.State.Family = ipv6Str
			} else {
				v6Address.Config.Ip = ipStr
				v6Address.Config.PrefixLength = prfxLen
			}
		}
	}
	return err
}

var DbToYang_intf_ip_addr_xfmr SubTreeXfmrDbToYang = func(inParams XfmrParams) error {
	var err error
	intfsObj := getIntfsRoot(inParams.ygRoot)
	pathInfo := NewPathInfo(inParams.uri)
	uriIfName := pathInfo.Var("name")
	ifName := uriIfName

	targetUriPath := pathInfo.YangPath
	log.Infof("DbToYang_intf_ip_addr_xfmr: uri:%v path:%v", inParams.uri, targetUriPath)

	reqPathInfo := NewPathInfo(inParams.requestUri)
	requestUriPath := reqPathInfo.YangPath
	var reqUriIfName string = reqPathInfo.Var("name")

	if (inParams.oper == GET) &&
		((requestUriPath == "/openconfig-interfaces:interfaces" || requestUriPath == "/openconfig-interfaces:interfaces/interface") && reqUriIfName == "") {
		_, present := inParams.txCache.Load("interface_subinterface_ip_read_once")
		if present {
			log.Info("DbToYang_intf_ip_addr_xfmr, top level GET, interface_subinterface_ip_read_once already cached")
			return nil
		}

		intfTypeList := [5]E_InterfaceType{IntfTypeEthernet, IntfTypeMgmt, IntfTypePortChannel, IntfTypeLoopback, IntfTypeSubIntf}

		// Get IP from all configDb table interfaces
		for i := 0; i < len(intfTypeList); i++ {
			intfTbl := IntfTypeTblMap[intfTypeList[i]]

			handleAllIntfIPGetForTable(inParams, intfTbl.cfgDb.intfTN, false)
		}

		// Get IP from applDb INTF_TABLE interfaces except vlan intf
		handleAllIntfIPGetForTable(inParams, "INTF_TABLE", true)

		inParams.txCache.Store("interface_subinterface_ip_read_once", true)
		return nil
	} else {
		// Handle GET requests for given interface
		var intfObj *ocbinds.OpenconfigInterfaces_Interfaces_Interface
		ifName = *(&uriIfName)

		intfType, _, _ := getIntfTypeByName(ifName)
		if IntfTypeVlan == intfType {
			return nil
		}

		if strings.HasPrefix(targetUriPath, "/openconfig-interfaces:interfaces/interface/subinterfaces") {
			if intfsObj != nil && intfsObj.Interface != nil && len(intfsObj.Interface) > 0 {
				var ok bool = false
				if intfObj, ok = intfsObj.Interface[uriIfName]; !ok {
					intfObj, _ = intfsObj.NewInterface(uriIfName)
				}
				ygot.BuildEmptyTree(intfObj)
				if intfObj.Subinterfaces == nil {
					ygot.BuildEmptyTree(intfObj.Subinterfaces)
				}
			} else {
				ygot.BuildEmptyTree(intfsObj)
				intfObj, _ = intfsObj.NewInterface(uriIfName)
				ygot.BuildEmptyTree(intfObj)
			}

			err = handleIntfIPGetByTargetURI(inParams, targetUriPath, ifName, intfObj)
			if err != nil {
				return err
			}

		} else {
			err = errors.New("Invalid URI : " + targetUriPath)
		}
	}

	return err
}

var YangToDb_intf_ip_addr_xfmr SubTreeXfmrYangToDb = func(inParams XfmrParams) (map[string]map[string]db.Value, error) {
	var err, oerr error
	subOpMap := make(map[db.DBNum]map[string]map[string]db.Value)
	subIntfmap := make(map[string]map[string]db.Value)
	subIntfmap_del := make(map[string]map[string]db.Value)
	var value db.Value
	var overlapIP string

	pathInfo := NewPathInfo(inParams.uri)
	uriIfName := pathInfo.Var("name")
	idx := pathInfo.Var("index")
	i64, err := strconv.ParseUint(idx, 10, 32)
	i32 := uint32(i64)
	ifName := uriIfName

	sonicIfName := &uriIfName

	log.Infof("YangToDb_intf_ip_addr_xfmr: Interface name retrieved from alias : %s is %s", ifName, *sonicIfName)
	ifName = *sonicIfName
	intfType, _, ierr := getIntfTypeByName(ifName)
	if i32 > 0 {
		intfType = IntfTypeSubIntf
		ifName = *utils.GetSubInterfaceDBKeyfromParentInterfaceAndSubInterfaceID(&ifName, &idx)
	}

	if IntfTypeVxlan == intfType || IntfTypeVlan == intfType {
		return subIntfmap, nil
	}

	intfsObj := getIntfsRoot(inParams.ygRoot)
	if intfsObj == nil || len(intfsObj.Interface) < 1 {
		log.Info("YangToDb_intf_subintf_ip_xfmr : IntfsObj/interface list is empty.")
		return subIntfmap, errors.New("IntfsObj/Interface is not specified")
	}

	if ifName == "" {
		errStr := "Interface KEY not present"
		log.Info("YangToDb_intf_subintf_ip_xfmr : " + errStr)
		return subIntfmap, errors.New(errStr)
	}

	if intfType == IntfTypeUnset || ierr != nil {
		errStr := "Invalid interface type IntfTypeUnset"
		log.Info("YangToDb_intf_subintf_ip_xfmr : " + errStr)
		return subIntfmap, errors.New(errStr)
	}
	/* Set invokeCRUSubtreeOnce flag to invoke subtree once */
	if inParams.invokeCRUSubtreeOnce != nil {
		*inParams.invokeCRUSubtreeOnce = true
	}

	/* Validate if DHCP_Relay is provisioned on the interface */
	prefixType := ""
	if strings.Contains(inParams.uri, "ipv4") {
		prefixType = "ipv4"
	} else if strings.Contains(inParams.uri, "ipv6") {
		prefixType = "ipv6"
	}

	if inParams.oper == DELETE {
		dhcpProv, _ := ValidateIntfProvisionedForRelay(inParams.d, ifName, prefixType, nil)
		if dhcpProv {
			errStr := "IP address cannot be deleted. DHCP Relay is configured on the interface."
			return subIntfmap, tlerr.InvalidArgsError{Format: errStr}
		}
	}

	if _, ok := intfsObj.Interface[uriIfName]; !ok {
		errStr := "Interface entry not found in Ygot tree, ifname: " + ifName
		log.Info("YangToDb_intf_subintf_ip_xfmr : " + errStr)
		return subIntfmap, errors.New(errStr)
	}

	intTbl := IntfTypeTblMap[intfType]
	tblName, _ := getIntfTableNameByDBId(intTbl, inParams.curDb)
	intfObj := intfsObj.Interface[uriIfName]

	if intfObj.Subinterfaces == nil || len(intfObj.Subinterfaces.Subinterface) < 1 {
		// Handling the scenario for Interface instance delete at interfaces/interface[name] level or subinterfaces container level
		if inParams.oper == DELETE {
			log.Info("Top level Interface instance delete or subinterfaces container delete for Interface: ", ifName)
			return intf_ip_addr_del(inParams.d, ifName, tblName, nil)
		}
		errStr := "SubInterface node doesn't exist"
		log.Info("YangToDb_intf_subintf_ip_xfmr : " + errStr)
		err = tlerr.InvalidArgsError{Format: errStr}
		return subIntfmap, err
	}
	if _, ok := intfObj.Subinterfaces.Subinterface[i32]; !ok {
		log.Info("YangToDb_intf_subintf_ip_xfmr : No IP address handling required")
		errStr := "SubInterface index 0 doesn't exist"
		err = tlerr.InvalidArgsError{Format: errStr}
		return subIntfmap, err
	}

	subIntfObj := intfObj.Subinterfaces.Subinterface[i32]
	if inParams.oper == DELETE {
		return intf_ip_addr_del(inParams.d, ifName, tblName, subIntfObj)
	}

	entry, dbErr := inParams.d.GetEntry(&db.TableSpec{Name: intTbl.cfgDb.intfTN}, db.Key{Comp: []string{ifName}})
	if dbErr != nil || !entry.IsPopulated() {
		ifdb := make(map[string]string)
		ifdb["NULL"] = "NULL"
		value := db.Value{Field: ifdb}
		if _, ok := subIntfmap[tblName]; !ok {
			subIntfmap[tblName] = make(map[string]db.Value)
		}
		subIntfmap[tblName][ifName] = value

	}
	if subIntfObj.Ipv4 != nil && subIntfObj.Ipv4.Addresses != nil {
		for ip := range subIntfObj.Ipv4.Addresses.Address {
			addr := subIntfObj.Ipv4.Addresses.Address[ip]
			if addr.Config != nil {
				if addr.Config.Ip == nil {
					addr.Config.Ip = new(string)
					*addr.Config.Ip = ip
				}
				log.Info("Ip:=", *addr.Config.Ip)
				if addr.Config.PrefixLength == nil {
					log.Error("Prefix Length empty!")
					errStr := "Prefix Length not present"
					err = tlerr.InvalidArgsError{Format: errStr}
					return subIntfmap, err
				}
				log.Info("prefix:=", *addr.Config.PrefixLength)
				if addr.Config.Family == nil {
					addr.Config.Family = new(string)
					*addr.Config.Family = "ipv4"

				} else if *addr.Config.Family == "ipv6" {
					log.Error("Incorrect family ipv6!")
					errStr := "IPv4 Family not present"
					err = tlerr.InvalidArgsError{Format: errStr}
					return subIntfmap, err
				}
				log.Info("family:=", *addr.Config.Family)

				ipPref := *addr.Config.Ip + "/" + strconv.Itoa(int(*addr.Config.PrefixLength))
				/* Check for IP overlap */
				overlapIP, oerr = validateIpOverlap(inParams.d, ifName, ipPref, tblName, true)

				ipEntry, _ := inParams.d.GetEntry(&db.TableSpec{Name: intTbl.cfgDb.intfTN}, db.Key{Comp: []string{ifName, ipPref}})
				ipMap, _ := getIntfIpByName(inParams.d, intTbl.cfgDb.intfTN, ifName, true, false, "")

				m := make(map[string]string)
				alrdyCfgredIP, primaryIpAlrdyCfgred, err := utlValidateIpTypeForCfgredDiffIp(m, ipMap, &ipEntry, &ipPref, &ifName)
				if err != nil {
					return nil, err
				}
				// Primary IP config already happened and replacing it with new one
				if primaryIpAlrdyCfgred && len(alrdyCfgredIP) != 0 && alrdyCfgredIP != ipPref {
					subIntfmap_del[tblName] = make(map[string]db.Value)
					key := ifName + "|" + alrdyCfgredIP
					subIntfmap_del[tblName][key] = value
					subOpMap[db.ConfigDB] = subIntfmap_del
					log.Info("subOpMap: ", subOpMap)
					inParams.subOpDataMap[DELETE] = &subOpMap
				}

				intf_key := intf_intf_tbl_key_gen(ifName, *addr.Config.Ip, int(*addr.Config.PrefixLength), "|")

				value := db.Value{Field: m}
				if _, ok := subIntfmap[tblName]; !ok {
					subIntfmap[tblName] = make(map[string]db.Value)
				}
				subIntfmap[tblName][intf_key] = value
				if log.V(3) {
					log.Info("tblName :", tblName, " intf_key: ", intf_key, " data : ", value)
				}
			}
		}
	}
	if subIntfObj.Ipv6 != nil && subIntfObj.Ipv6.Addresses != nil {
		for ip := range subIntfObj.Ipv6.Addresses.Address {
			addr := subIntfObj.Ipv6.Addresses.Address[ip]
			if addr.Config != nil {
				if addr.Config.Ip == nil {
					addr.Config.Ip = new(string)
					*addr.Config.Ip = ip
				}
				log.Info("Ipv6 IP:=", *addr.Config.Ip)
				if addr.Config.PrefixLength == nil {
					log.Error("Prefix Length empty!")
					errStr := "Prefix Length not present"
					err = tlerr.InvalidArgsError{Format: errStr}
					return subIntfmap, err
				}
				log.Info("Ipv6 prefix:=", *addr.Config.PrefixLength)
				if addr.Config.Family == nil {
					addr.Config.Family = new(string)
					*addr.Config.Family = "ipv6"

				} else if *addr.Config.Family == "ipv4" {
					log.Error("Incorrect family!")
					errStr := "IPv6 Family not present"
					err = tlerr.InvalidArgsError{Format: errStr}
					return subIntfmap, err
				}
				log.Info("family:=", *addr.Config.Family)

				/* Check for IPv6 overlap */
				ipPref := *addr.Config.Ip + "/" + strconv.Itoa(int(*addr.Config.PrefixLength))
				overlapIP, oerr = validateIpOverlap(inParams.d, ifName, ipPref, tblName, true)

				m := make(map[string]string)

				intf_key := intf_intf_tbl_key_gen(ifName, *addr.Config.Ip, int(*addr.Config.PrefixLength), "|")

				value := db.Value{Field: m}
				if _, ok := subIntfmap[tblName]; !ok {
					subIntfmap[tblName] = make(map[string]db.Value)
				}
				subIntfmap[tblName][intf_key] = value
				log.Info("tblName :", tblName, "intf_key: ", intf_key, "data : ", value)
			}
		}
	}

	if oerr != nil {
		if overlapIP == "" {
			log.Error(oerr)
			return nil, tlerr.InvalidArgsError{Format: oerr.Error()}
		} else {
			subIntfmap_del[tblName] = make(map[string]db.Value)
			key := ifName + "|" + overlapIP
			subIntfmap_del[tblName][key] = value
			subOpMap[db.ConfigDB] = subIntfmap_del
			log.Info("subOpMap: ", subOpMap)
			inParams.subOpDataMap[DELETE] = &subOpMap
		}
	}

	log.Info("YangToDb_intf_subintf_ip_xfmr : subIntfmap : ", subIntfmap)
	return subIntfmap, err
}

/* Check for IP overlap */
func validateIpOverlap(d *db.DB, intf string, ipPref string, tblName string, isIntfIp bool) (string, error) {
	log.Info("Checking for IP overlap ....")

	ipA, ipNetA, err := parseCIDR(ipPref)
	if err != nil {
		log.Info("Failed to parse IP address: ", ipPref)
		return "", err
	}

	var allIntfKeys []db.Key

	for key := range IntfTypeTblMap {
		intTbl := IntfTypeTblMap[key]
		keys, err := d.GetKeys(&db.TableSpec{Name: intTbl.cfgDb.intfTN})
		if err != nil {
			log.Info("Failed to get keys; err=%v", err)
			return "", err
		}
		allIntfKeys = append(allIntfKeys, keys...)
	}

	if len(allIntfKeys) > 0 {
		for _, key := range allIntfKeys {
			if len(key.Comp) < 2 {
				continue
			}
			ipB, ipNetB, perr := parseCIDR(key.Get(1))
			//Check if key has IP, if not continue
			if perr != nil {
				continue
			}
			if ipNetA.Contains(ipB) || ipNetB.Contains(ipA) {
				if log.V(3) {
					log.Info("IP: ", ipPref, " overlaps with ", key.Get(1), " of ", key.Get(0))
				}

				errStr := "IP " + ipPref + " overlaps with IP or IP Anycast " + key.Get(1) + " of Interface " + key.Get(0)
				return "", errors.New(errStr)
			}
		}
	}
	return "", nil
}

func utlCheckAndRetrievePrimaryIPConfigured(ipMap map[string]db.Value) (bool, string) {
	for ipKey, _ := range ipMap {
		return true, ipKey
	}
	return false, ""
}

func utlValidateIpTypeForCfgredDiffIp(m map[string]string, ipMap map[string]db.Value, ipEntry *db.Value, ipPref *string, ifName *string) (string, bool, error) {

	dbgStr := "IPv4 address"
	checkPrimIPCfgred, cfgredPrimIP := utlCheckAndRetrievePrimaryIPConfigured(ipMap)
	if checkPrimIPCfgred && !ipEntry.IsPopulated() {
		infoStr := "Primary " + dbgStr + " is already configured for interface: " + *ifName
		log.Info(infoStr)
		return cfgredPrimIP, true, nil
	}

	return "", false, nil
}

func intf_intf_tbl_key_gen(intfName string, ip string, prefixLen int, keySep string) string {
	return intfName + keySep + ip + "/" + strconv.Itoa(prefixLen)
}
func parseCIDR(ipPref string) (netaddr.IP, netaddr.IPPrefix, error) {
	prefIdx := strings.LastIndexByte(ipPref, '/')
	if prefIdx <= 0 {
		return netaddr.IP{}, netaddr.IPPrefix{}, fmt.Errorf("Invalid Prefix(%q): no'/'", ipPref)
	}
	prefLen, _ := strconv.Atoi(ipPref[prefIdx+1:])
	ipA, err := netaddr.ParseIP(ipPref[:prefIdx])
	if err != nil {
		log.Infof("parseCIDR: Failed to parse IP address:%s : err : %s ", ipPref, err)
		return netaddr.IP{}, netaddr.IPPrefix{}, fmt.Errorf("Failed to parse IP address: %s", ipPref)
	}

	ipNetA, _ := ipA.Prefix(uint8(prefLen))
	return ipA, ipNetA, nil
}
func getIntfIpByName(dbCl *db.DB, tblName string, ifName string, ipv4 bool, ipv6 bool, ip string) (map[string]db.Value, error) {
	var err error
	intfIpMap := make(map[string]db.Value)
	all := true
	if !ipv4 || !ipv6 {
		all = false
	}
	log.V(3).Info("Updating Interface IP Info from DB to Internal DS for Interface Name : ", ifName)

	keys, err := doGetIntfIpKeys(dbCl, tblName, ifName)
	if log.V(3) {
		log.Infof("Found %d keys for (%v)(%v)", len(keys), tblName, ifName)
	}
	if err != nil {
		return intfIpMap, err
	}
	for _, key := range keys {
		if len(key.Comp) < 2 {
			continue
		}
		if key.Get(0) != ifName {
			continue
		}
		if len(key.Comp) > 2 {
			for i := range key.Comp {
				if i == 0 || i == 1 {
					continue
				}
				key.Comp[1] = key.Comp[1] + ":" + key.Comp[i]
			}
		}
		if !all {
			ipB, _, _ := parseCIDR(key.Get(1))
			if (validIPv4(ipB.String()) && (!ipv4)) ||
				(validIPv6(ipB.String()) && (!ipv6)) {
				continue
			}
			if ip != "" {
				if ipB.String() != ip {
					continue
				}
			}
		}

		ipInfo, _ := dbCl.GetEntry(&db.TableSpec{Name: tblName}, db.Key{Comp: []string{key.Get(0), key.Get(1)}})
		intfIpMap[key.Get(1)] = ipInfo
	}
	return intfIpMap, err
}

/* Get all IP keys for given interface */
func doGetIntfIpKeys(d *db.DB, tblName string, intfName string) ([]db.Key, error) {
	var ipKeys []db.Key
	var err error

	if intfName != "" {
		ts := db.TableSpec{Name: tblName + d.Opts.KeySeparator + intfName}
		ipKeys, err = d.GetKeys(&ts)
	} else {
		ipKeys, err = d.GetKeys(&db.TableSpec{Name: tblName})
	}
	if log.V(3) {
		log.Infof("doGetIntfIpKeys for intfName: %v  tblName:%v  ipKeys: %v", intfName, tblName, ipKeys)
	}
	return ipKeys, err
}
func validIPv4(ipAddress string) bool {
	/* Dont allow ip addresses that start with "0." or "255."*/
	if strings.HasPrefix(ipAddress, "0.") || strings.HasPrefix(ipAddress, "255.") {
		log.Info("validIP: IP is reserved ", ipAddress)
		return false
	}

	ip, err := netaddr.ParseIP(ipAddress)
	if err != nil {
		log.Infof("validIPv4: Failed to parse IP address %s : err : %s", ipAddress, err)
		return false
	}

	ipAddress = strings.Trim(ipAddress, " ")

	re, _ := regexp.Compile(`^(([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])\.){3}([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])$`)
	if re.MatchString(ipAddress) {
		return validIP(ip)
	}
	return false
}

func validIPv6(ipAddress string) bool {
	ip, err := netaddr.ParseIP(ipAddress)
	if err != nil {
		log.Infof("validIPv6: Failed to parse IP address %s : err : %s", ipAddress, err)
		return false
	}
	ipAddress = strings.Trim(ipAddress, " ")

	re, _ := regexp.Compile(`(([0-9a-fA-F]{1,4}:){7,7}[0-9a-fA-F]{1,4}|([0-9a-fA-F]{1,4}:){1,7}:|([0-9a-fA-F]{1,4}:){1,6}:[0-9a-fA-F]{1,4}|([0-9a-fA-F]{1,4}:){1,5}(:[0-9a-fA-F]{1,4}){1,2}|([0-9a-fA-F]{1,4}:){1,4}(:[0-9a-fA-F]{1,4}){1,3}|([0-9a-fA-F]{1,4}:){1,3}(:[0-9a-fA-F]{1,4}){1,4}|([0-9a-fA-F]{1,4}:){1,2}(:[0-9a-fA-F]{1,4}){1,5}|[0-9a-fA-F]{1,4}:((:[0-9a-fA-F]{1,4}){1,6})|:((:[0-9a-fA-F]{1,4}){1,7}|:)|fe80:(:[0-9a-fA-F]{0,4}){0,4}%[0-9a-zA-Z]{1,}|::(ffff(:0{1,4}){0,1}:){0,1}((25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])\.){3,3}(25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])|([0-9a-fA-F]{1,4}:){1,4}:((25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])\.){3,3}(25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9]))`)
	if re.MatchString(ipAddress) {
		return validIP(ip)
	}
	return false
}
func validIP(ip netaddr.IP) bool {
	if ip.IsUnspecified() || ip.IsLoopback() || ip.IsMulticast() {
		return false
	}
	return true
}

func getIntfTableNameByDBId(intftbl IntfTblData, curDb db.DBNum) (string, error) {

	var tblName string

	switch curDb {
	case db.ConfigDB:
		tblName = intftbl.cfgDb.intfTN
	case db.ApplDB:
		tblName = intftbl.appDb.intfTN
	case db.StateDB:
		tblName = intftbl.stateDb.intfTN
	default:
		tblName = intftbl.cfgDb.intfTN
	}

	return tblName, nil
}
func intf_ip_addr_del(d *db.DB, ifName string, tblName string, subIntf *ocbinds.OpenconfigInterfaces_Interfaces_Interface_Subinterfaces_Subinterface) (map[string]map[string]db.Value, error) {
	var err error
	subIntfmap := make(map[string]map[string]db.Value)
	intfIpMap := make(map[string]db.Value)

	// Handles the case when the delete request at subinterfaces/subinterface[index = 0]
	if subIntf == nil || (subIntf.Ipv4 == nil && subIntf.Ipv6 == nil) {
		ipMap, _ := getIntfIpByName(d, tblName, ifName, true, true, "")
		if len(ipMap) > 0 {
			for k, v := range ipMap {
				intfIpMap[k] = v
			}
		}
	}

	// This handles the delete for a specific IPv4 address or a group of IPv4 addresses
	if subIntf != nil && subIntf.Ipv4 != nil {
		if subIntf.Ipv4.Addresses != nil {
			if len(subIntf.Ipv4.Addresses.Address) < 1 {
				ipMap, _ := getIntfIpByName(d, tblName, ifName, true, false, "")
				if len(ipMap) > 0 {
					for k, v := range ipMap {
						intfIpMap[k] = v
					}
				}
			} else {
				for ip := range subIntf.Ipv4.Addresses.Address {
					ipMap, _ := getIntfIpByName(d, tblName, ifName, true, false, ip)

					if len(ipMap) > 0 {
						for k, v := range ipMap {
							// Primary IPv4 delete
							ifIpMap, _ := getIntfIpByName(d, tblName, ifName, true, false, "")

							checkIPCfgred, _ := utlCheckAndRetrievePrimaryIPConfigured(ifIpMap)

							if checkIPCfgred {
								intfIpMap[k] = v
							}
						}
					}
				}
			}
		} else {
			// Case when delete request is at IPv4 container level
			ipMap, _ := getIntfIpByName(d, tblName, ifName, true, false, "")
			if len(ipMap) > 0 {
				for k, v := range ipMap {
					intfIpMap[k] = v
				}
			}
		}
	}

	// This handles the delete for a specific IPv6 address or a group of IPv6 addresses
	if subIntf != nil && subIntf.Ipv6 != nil {
		if subIntf.Ipv6.Addresses != nil {
			if len(subIntf.Ipv6.Addresses.Address) < 1 {
				ipMap, _ := getIntfIpByName(d, tblName, ifName, false, true, "")
				if len(ipMap) > 0 {
					for k, v := range ipMap {
						intfIpMap[k] = v
					}
				}
			} else {
				for ip := range subIntf.Ipv6.Addresses.Address {
					ipMap, _ := getIntfIpByName(d, tblName, ifName, false, true, ip)

					if len(ipMap) > 0 {
						for k, v := range ipMap {
							intfIpMap[k] = v
						}
					}
				}
			}
		} else {
			// Case when the delete request is at IPv6 container level
			ipMap, _ := getIntfIpByName(d, tblName, ifName, false, true, "")
			if len(ipMap) > 0 {
				for k, v := range ipMap {
					intfIpMap[k] = v
				}
			}
		}
	}
	if len(intfIpMap) > 0 {
		if _, ok := subIntfmap[tblName]; !ok {
			subIntfmap[tblName] = make(map[string]db.Value)
		}
		var data db.Value
		for k := range intfIpMap {
			ifKey := ifName + "|" + k
			subIntfmap[tblName][ifKey] = data
		}
		intfIpCnt := 0
		_ = interfaceIPcount(tblName, d, &ifName, &intfIpCnt)
		/* Delete interface from interface table if no other interface attributes/ip */
		ipCntAfterDeletion := intfIpCnt - len(intfIpMap)
		if check_if_delete_l3_intf_entry(d, tblName, ifName, ipCntAfterDeletion, nil) {
			if _, ok := subIntfmap[tblName]; !ok {
				subIntfmap[tblName] = make(map[string]db.Value)
			}
			subIntfmap[tblName][ifName] = data
		}
	}
	log.Info("Delete IP address list ", subIntfmap, " ", err)
	return subIntfmap, err
}
func interfaceIPcount(tblName string, d *db.DB, intfName *string, ipCnt *int) error {
	ipKeys, _ := d.GetKeysPattern(&db.TableSpec{Name: tblName}, db.Key{Comp: []string{*intfName, "*"}})
	*ipCnt = len(ipKeys)
	return nil
}
func check_if_delete_l3_intf_entry(d *db.DB, tblName string, ifName string, ipCnt int, intfEntry *db.Value) bool {
	if strings.HasPrefix(ifName, VLAN) {
		sagIpKey, _ := d.GetKeysPattern(&db.TableSpec{Name: "SAG"}, db.Key{Comp: []string{ifName, "*"}})
		if len(sagIpKey) != 0 {
			return false
		}
	}
	if intfEntry == nil {
		entry, err := d.GetEntry(&db.TableSpec{Name: tblName}, db.Key{Comp: []string{ifName}})
		if err != nil {
			// Failed to read entry from config DB
			return false
		}
		intfEntry = &entry
	}
	if ipCnt == 0 && intfEntry.IsPopulated() {
		intfEntryMap := intfEntry.Field
		_, nullValPresent := intfEntryMap["NULL"]
		_, natZoneValPresent := intfEntryMap["nat_zone"]
		/* Note: Unbinding shouldn't happen if VRF config is associated with interface.
		   Hence, we check for map length and only if either NULL or NAT value is present */
		if (len(intfEntryMap) == 1 && nullValPresent) || (len(intfEntryMap) == 2 && nullValPresent && natZoneValPresent) {
			return true
		}
	}
	return false
}

var Subscribe_intf_ip_addr_xfmr = func(inParams XfmrSubscInParams) (XfmrSubscOutParams, error) {
	if log.V(3) {
		log.Info("Entering Subscribe_intf_ip_addr_xfmr")
	}
	var err error
	var result XfmrSubscOutParams

	pathInfo := NewPathInfo(inParams.uri)
	origTargetUriPath := pathInfo.YangPath

	log.Infof("Subscribe_intf_ip_addr_xfmr:- subscProc:%v URI: %s", inParams.subscProc, inParams.uri)
	log.Infof("Subscribe_intf_ip_addr_xfmr:- Target URI path: %s", origTargetUriPath)

	// When the subscribe subtree is invoked in the GET or CRUD context the inParams.subscProc is set to TRANSLATE_EXISTS
	if inParams.subscProc == TRANSLATE_EXISTS {
		// Defer the DB resource check done by infra by setting the virtual table to true.
		// Resource checks are now performed within the DbToYang or YangToDb subtree callback.
		result.isVirtualTbl = true
		return result, nil
	}
	if inParams.subscProc == TRANSLATE_SUBSCRIBE {
		isRoutedVlan := false

		ifBasePath := "/openconfig-interfaces:interfaces/interface"
		targetUriPath := origTargetUriPath[len(ifBasePath):]

		if strings.HasPrefix(targetUriPath, "/subinterfaces") {
			targetUriPath = targetUriPath[len("/subinterfaces/subinterface"):]
		} else {
			isRoutedVlan = true
			targetUriPath = targetUriPath[len("/openconfig-vlan:routed-vlan"):]
		}

		if strings.HasPrefix(targetUriPath, "/openconfig-if-ip:ipv4") {
			targetUriPath = targetUriPath[len("/openconfig-if-ip:ipv4/addresses"):]
		} else {
			targetUriPath = targetUriPath[len("/openconfig-if-ip:ipv6/addresses"):]
		}

		if targetUriPath == "" || targetUriPath == "/address" {
			result.isVirtualTbl = true
			log.Info("Subscribe_intf_ip_addr_xfmr:- result.isVirtualTbl: ", result.isVirtualTbl)
			return result, err
		}

		result.onChange = OnchangeEnable
		result.nOpts = &notificationOpts{}
		result.nOpts.pType = OnChange
		result.isVirtualTbl = false

		uriIfName := pathInfo.Var("name")
		tableName := ""
		ipKey := ""
		ifKey := ""
		subIfKey := "*"

		if uriIfName == "" || uriIfName == "*" {
			ifKey = "*"
		} else {
			sonicIfName := &uriIfName
			ifKey = *sonicIfName
		}

		addressConfigPath := "/address/config"
		addressStatePath := "/address/state"

		idx := pathInfo.Var("index")
		if ifKey != "" {
			if idx == "0" {
				intfType, _, _ := getIntfTypeByName(ifKey)
				intTbl := IntfTypeTblMap[intfType]
				if targetUriPath == addressStatePath {
					tableName = intTbl.appDb.intfTN
				} else {
					tableName = intTbl.cfgDb.intfTN
				}
			} else if idx == "*" || idx == "" {
				subIfKey = *utils.GetSubInterfaceDBKeyfromParentInterfaceAndSubInterfaceID(&ifKey, &idx)
			} else {
				tableName = "VLAN_SUB_INTERFACE"
				ifKey = *utils.GetSubInterfaceDBKeyfromParentInterfaceAndSubInterfaceID(&ifKey, &idx)
			}
		}

		ipKey = pathInfo.Var("ip")
		if ipKey == "" {
			ipKey = "*"
		}

		if ipKey != "*" {
			ipKey = ipKey + "/*"
		}

		log.Infof("path:%v ifKey:%v, ipKey:%v tbl:[%v]", origTargetUriPath, ifKey, ipKey, tableName)

		keyName := ""
		if targetUriPath == addressConfigPath {
			keyName = ifKey + "|" + ipKey
			if tableName != "" {
				result.dbDataMap = RedisDbSubscribeMap{db.ConfigDB: {tableName: {keyName: {}}}}
			} else {
				if isRoutedVlan {
					result.dbDataMap = RedisDbSubscribeMap{db.ConfigDB: {"VLAN_INTERFACE": {keyName: {}}}}
				} else {
					subIfKeyName := subIfKey + "|" + ipKey
					result.dbDataMap = RedisDbSubscribeMap{db.ConfigDB: {"INTERFACE": {keyName: {}},
						"MGMT_INTERFACE":        {keyName: {}},
						"LOOPBACK_INTERFACE":    {keyName: {}},
						"VLAN_SUB_INTERFACE":    {subIfKeyName: {}},
						"PORTCHANNEL_INTERFACE": {keyName: {}}}}
				}
			}
		} else if targetUriPath == addressStatePath {
			keyName = ifKey + ":" + ipKey
			if tableName != "" {
				result.dbDataMap = RedisDbSubscribeMap{db.ApplDB: {tableName: {keyName: {KEY_COMP_CNT: "2", DEL_AS_UPDATE: "true"}}}}
			} else {
				if isRoutedVlan {
					result.dbDataMap = RedisDbSubscribeMap{db.ApplDB: {"INTF_TABLE": {keyName: {KEY_COMP_CNT: "2", DEL_AS_UPDATE: "true"}}}}
				} else {
					result.dbDataMap = RedisDbSubscribeMap{db.ApplDB: {"INTF_TABLE": {keyName: {KEY_COMP_CNT: "2", DEL_AS_UPDATE: "true"}},
						"MGMT_INTF_TABLE": {keyName: {KEY_COMP_CNT: "2", DEL_AS_UPDATE: "true"}}}}
				}
			}
		}

		log.Info("Subscribe_intf_ip_addr_xfmr:- result dbDataMap: ", result.dbDataMap)
		log.Info("Subscribe_intf_ip_addr_xfmr:- result secDbDataMap: ", result.secDbDataMap)

		return result, err
	}

	result.isVirtualTbl = false

	result.dbDataMap = make(RedisDbSubscribeMap)
	uriIfName := pathInfo.Var("name")
	idx := pathInfo.Var("index")
	sonicIfName := &uriIfName
	keyName := *sonicIfName

	if keyName != "" {
		intfType, _, _ := getIntfTypeByName(keyName)
		intTbl := IntfTypeTblMap[intfType]
		tblName := intTbl.cfgDb.intfTN
		if idx != "" && idx != "0" {
			tblName = "VLAN_SUB_INTERFACE"
			keyName = *utils.GetSubInterfaceDBKeyfromParentInterfaceAndSubInterfaceID(&keyName, &idx)
		}
		result.dbDataMap = RedisDbSubscribeMap{db.ConfigDB: {tblName: {keyName: {}}}}
	}
	log.Info("Returning Subscribe_intf_ip_addr_xfmr, result:", result)
	result.needCache = true
	result.nOpts = new(notificationOpts)
	result.nOpts.mInterval = 15
	result.nOpts.pType = OnChange
	log.Info("Returning Subscribe_intf_ip_addr_xfmr, result:", result)
	return result, err

}

// YangToDb_subintf_ipv6_tbl_key_xfmr is a YangToDB Key transformer for IPv6 config.
var YangToDb_subintf_ipv6_tbl_key_xfmr KeyXfmrYangToDb = func(inParams XfmrParams) (string, error) {
	if log.V(3) {
		log.Info("Entering YangToDb_subintf_ipv6_tbl_key_xfmr")
	}

	var err error
	var inst_key string
	pathInfo := NewPathInfo(inParams.uri)
	ifName := pathInfo.Var("name")

	if log.V(3) {
		log.Info("inParams.requestUri: ", inParams.requestUri)
	}
	idx := pathInfo.Var("index")
	var i32 uint32
	i32 = 0
	if idx != "" {
		i64, _ := strconv.ParseUint(idx, 10, 32)
		i32 = uint32(i64)
	}
	inst_key = ifName
	if i32 > 0 {
		inst_key = ifName + "." + idx
	}
	log.Info("Exiting YangToDb_subintf_ipv6_tbl_key_xfmr, key ", inst_key)
	return inst_key, err
}

// DbToYang_subintf_ipv6_tbl_key_xfmr is a DbToYang key transformer for IPv6 config.
var DbToYang_subintf_ipv6_tbl_key_xfmr KeyXfmrDbToYang = func(inParams XfmrParams) (map[string]interface{}, error) {
	if log.V(3) {
		log.Info("Entering DbToYang_subintf_ipv6_tbl_key_xfmr")
	}

	rmap := make(map[string]interface{})
	return rmap, nil
}

// YangToDb_ipv6_enabled_xfmr is a YangToDB Field transformer for IPv6 config "enabled".
var YangToDb_ipv6_enabled_xfmr FieldXfmrYangToDb = func(inParams XfmrParams) (map[string]string, error) {
	if log.V(3) {
		log.Info("Entering YangToDb_ipv6_enabled_xfmr")
	}
	var err error
	res_map := make(map[string]string)
	pathInfo := NewPathInfo(inParams.uri)
	ifUIName := pathInfo.Var("name")
	idx := pathInfo.Var("index")
	var i32 uint32
	i32 = 0
	if idx != "" {
		i64, _ := strconv.ParseUint(idx, 10, 32)
		i32 = uint32(i64)
	}

	intfType, _, ierr := getIntfTypeByName(ifUIName)
	if ierr != nil || intfType == IntfTypeUnset || intfType == IntfTypeVxlan || intfType == IntfTypeMgmt {
		return res_map, errors.New("YangToDb_ipv6_enabled_xfmr, Error: Unsupported Interface: " + ifUIName)
	}

	if ifUIName == "" {
		errStr := "Interface KEY not present"
		log.Info("YangToDb_ipv6_enabled_xfmr: " + errStr)
		return res_map, errors.New(errStr)
	}

	if inParams.param == nil {
		return res_map, err
	}

	// Vlan Interface (routed-vlan) contains only one Key "ifname"
	// For all other interfaces (subinterfaces/subintfaces) will have 2 keys "ifname" & "subintf-index"
	if len(pathInfo.Vars) < 2 && intfType != IntfTypeVlan {
		return res_map, errors.New("YangToDb_ipv6_enabled_xfmr, Error: Invalid Key length")
	}

	if log.V(3) {
		log.Info("YangToDb_ipv6_enabled_xfmr, inParams.key: ", inParams.key)
	}

	ifName := &ifUIName

	intTbl := IntfTypeTblMap[intfType]
	tblName := intTbl.cfgDb.intfTN
	if i32 > 0 {
		tblName = "VLAN_SUB_INTERFACE"
		ifName = utils.GetSubInterfaceDBKeyfromParentInterfaceAndSubInterfaceID(ifName, &idx)
	}
	ipMap, _ := getIntfIpByName(inParams.d, tblName, *ifName, true, true, "")
	var enStr string
	subOpMap := make(map[db.DBNum]map[string]map[string]db.Value)
	subOpTblMap := make(map[string]map[string]db.Value)
	field_map := make(map[string]db.Value)
	res_values := db.Value{Field: map[string]string{}}
	IntfMap := make(map[string]string)

	enabled, _ := inParams.param.(*bool)
	if *enabled {
		enStr = "enable"
	} else {
		enStr = "disable"
	}

	IntfMapObj, err := inParams.d.GetEntry(&db.TableSpec{Name: tblName}, db.Key{Comp: []string{*ifName}})
	if err == nil || IntfMapObj.IsPopulated() {
		IntfMap = IntfMapObj.Field
	}
	val, fieldExists := IntfMap["ipv6_use_link_local_only"]
	if fieldExists && val == enStr {
		// Check if already set to required value
		log.Info("IPv6 is already %s.", enStr)
		return nil, nil
	}

	res_map["ipv6_use_link_local_only"] = enStr
	if log.V(3) {
		log.Info("YangToDb_ipv6_enabled_xfmr, res_map: ", res_map)
	}

	if enStr == "disable" {

		if len(IntfMap) == 0 {
			return nil, nil
		}

		keys := make([]string, 0, len(IntfMap))
		for k := range IntfMap {
			keys = append(keys, k)
		}
		check_keys := []string{"NULL", "ipv6_use_link_local_only"}
		sort.Strings(keys)
		/* Delete interface from interface table if disabling IPv6 and no other interface attributes/ip
		   else remove ipv6_use_link_local_only field */
		if !((reflect.DeepEqual(keys, check_keys) || reflect.DeepEqual(keys, check_keys[1:])) && len(ipMap) == 0) {
			//Checking if field entry exists
			if !fieldExists {
				//Nothing to delete
				return nil, nil
			}
			log.Info("YangToDb_ipv6_enabled_xfmr, deleting ipv6_use_link_local_only field")
			//Delete field entry
			(&res_values).Set("ipv6_use_link_local_only", enStr)
		}
		field_map[*ifName] = res_values
		subOpTblMap[tblName] = field_map
		subOpMap[db.ConfigDB] = subOpTblMap
		inParams.subOpDataMap[DELETE] = &subOpMap
		if log.V(3) {
			log.Info("YangToDb_ipv6_enabled_xfmr, subOpMap: ", subOpMap)
		}
		return nil, nil
	}
	return res_map, nil
}

// DbToYang_ipv6_enabled_xfmr is a DbToYang Field transformer for IPv6 config "enabled". */
var DbToYang_ipv6_enabled_xfmr FieldXfmrDbtoYang = func(inParams XfmrParams) (map[string]interface{}, error) {
	if log.V(3) {
		log.Info("Entering DbToYang_ipv6_enabled_xfmr")
	}
	res_map := make(map[string]interface{})

	if log.V(3) {
		log.Info("DbToYang_ipv6_enabled_xfmr, inParams.key ", inParams.key)
	}
	pathInfo := NewPathInfo(inParams.uri)
	ifName := pathInfo.Var("name")

	ifUIName := &ifName
	log.Info("Interface Name = ", *ifUIName)

	intfType, _, _ := getIntfTypeByName(inParams.key)
	if intfType == IntfTypeVxlan || intfType == IntfTypeMgmt {
		return res_map, nil
	}

	intTbl := IntfTypeTblMap[intfType]
	tblName, _ := getIntfTableNameByDBId(intTbl, inParams.curDb)

	data := (*inParams.dbDataMap)[inParams.curDb]

	res_map["enabled"] = false
	ipv6_status, ok := data[tblName][inParams.key].Field["ipv6_use_link_local_only"]

	if ok && ipv6_status == "enable" {
		res_map["enabled"] = true
	}
	return res_map, nil
}
