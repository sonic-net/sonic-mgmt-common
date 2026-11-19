////////////////////////////////////////////////////////////////////////////////
//                                                                            //
//  Copyright 2026 Nokia.                                                     //
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

package transformer

import (
	"strconv"
	"strings"

	"github.com/Azure/sonic-mgmt-common/translib/db"
	"github.com/Azure/sonic-mgmt-common/translib/ocbinds"
	log "github.com/golang/glog"
)

// The openconfig-system-ext cleanup containers all map to fields of the
// CONFIG_DB AUTO_TECHSUPPORT|GLOBAL row.
const (
	AUTO_TECHSUPPORT_TABLE      = "AUTO_TECHSUPPORT"
	AUTO_TECHSUPPORT_GLOBAL_KEY = "GLOBAL"

	autoTsStateField        = "state"
	autoTsMaxCoreLimitField = "max_core_limit"

	// The cleanup scripts treat 0.0 as "no cleanup" but fail on a missing
	// field, so a delete writes 0.0 instead of removing the field.
	autoTsLimitDisabled = "0.0"
)

func init() {
	XlateFuncBind("YangToDb_core_file_cleanup_xfmr", YangToDb_core_file_cleanup_xfmr)
	XlateFuncBind("DbToYang_core_file_cleanup_xfmr", DbToYang_core_file_cleanup_xfmr)
	XlateFuncBind("Subscribe_core_file_cleanup_xfmr", Subscribe_auto_techsupport_global_xfmr)
}

// formatAutoTsLimit renders a percent the way the AUTO_TECHSUPPORT defaults
// store it, for example "5.0" or "8.25".
func formatAutoTsLimit(v float64) string {
	s := strconv.FormatFloat(v, 'f', -1, 64)
	if !strings.Contains(s, ".") {
		s += ".0"
	}
	return s
}

// autoTsLimitToDb maps the limit leaf of a cleanup config container to its
// AUTO_TECHSUPPORT|GLOBAL field. The rest of the row is left untouched.
func autoTsLimitToDb(inParams XfmrParams, field string, limit *float64) map[string]map[string]db.Value {
	res := make(map[string]map[string]db.Value)
	if inParams.oper == DELETE {
		autoTsGlobalSubOpUpdate(inParams, field, autoTsLimitDisabled)
		return res
	}
	if limit != nil {
		res[AUTO_TECHSUPPORT_TABLE] = map[string]db.Value{
			AUTO_TECHSUPPORT_GLOBAL_KEY: {Field: map[string]string{field: formatAutoTsLimit(*limit)}},
		}
	}
	return res
}

// autoTsGlobalSubOpUpdate adds field=value to the AUTO_TECHSUPPORT|GLOBAL
// update that runs alongside the request, keeping fields other cleanup
// containers have already added.
func autoTsGlobalSubOpUpdate(inParams XfmrParams, field, value string) {
	if inParams.subOpDataMap[UPDATE] == nil {
		subOpMap := make(RedisDbMap)
		inParams.subOpDataMap[UPDATE] = &subOpMap
	}
	subOpMap := *inParams.subOpDataMap[UPDATE]
	if subOpMap[db.ConfigDB] == nil {
		subOpMap[db.ConfigDB] = make(map[string]map[string]db.Value)
	}
	tbl := subOpMap[db.ConfigDB][AUTO_TECHSUPPORT_TABLE]
	if tbl == nil {
		tbl = make(map[string]db.Value)
		subOpMap[db.ConfigDB][AUTO_TECHSUPPORT_TABLE] = tbl
	}
	entry := tbl[AUTO_TECHSUPPORT_GLOBAL_KEY]
	if entry.Field == nil {
		entry.Field = make(map[string]string)
	}
	entry.Field[field] = value
	tbl[AUTO_TECHSUPPORT_GLOBAL_KEY] = entry
}

// autoTsGlobal reads AUTO_TECHSUPPORT|GLOBAL. ok is false if the row is not
// configured.
func autoTsGlobal(inParams XfmrParams) (entry db.Value, ok bool) {
	d := inParams.dbs[db.ConfigDB]
	if d == nil {
		return entry, false
	}
	entry, err := d.GetEntry(&db.TableSpec{Name: AUTO_TECHSUPPORT_TABLE}, db.Key{Comp: []string{AUTO_TECHSUPPORT_GLOBAL_KEY}})
	if err != nil {
		log.V(3).Infof("AUTO_TECHSUPPORT|GLOBAL not read: %v", err)
		return entry, false
	}
	return entry, true
}

// autoTsLimitFromDb returns the percent stored in field, or nil if it is
// unset or not a number.
func autoTsLimitFromDb(entry db.Value, field string) *float64 {
	v, err := strconv.ParseFloat(entry.Get(field), 64)
	if err != nil {
		return nil
	}
	return &v
}

// autoTsEnabledFromDb reports whether the AUTO_TECHSUPPORT state field is
// "enabled", or nil if it is unset. The same field turns event-driven
// tech-support collection on and off, so it is read-only in OpenConfig.
func autoTsEnabledFromDb(entry db.Value) *bool {
	v := entry.Get(autoTsStateField)
	if v == "" {
		return nil
	}
	enabled := v == "enabled"
	return &enabled
}

// YangToDb_core_file_cleanup_xfmr maps core-file-cleanup/config/max-core-limit
// to AUTO_TECHSUPPORT|GLOBAL:max_core_limit, which coredump_gen_handler.py
// reads each time a core file is written.
var YangToDb_core_file_cleanup_xfmr SubTreeXfmrYangToDb = func(inParams XfmrParams) (map[string]map[string]db.Value, error) {
	var limit *float64
	if sys := getAppRootObject(inParams); sys != nil && sys.CoreFileCleanup != nil && sys.CoreFileCleanup.Config != nil {
		limit = sys.CoreFileCleanup.Config.MaxCoreLimit
	}
	return autoTsLimitToDb(inParams, autoTsMaxCoreLimitField, limit), nil
}

var DbToYang_core_file_cleanup_xfmr SubTreeXfmrDbToYang = func(inParams XfmrParams) error {
	entry, ok := autoTsGlobal(inParams)
	if !ok {
		return nil
	}
	deviceObj := (*inParams.ygRoot).(*ocbinds.Device)
	if deviceObj.System == nil {
		deviceObj.System = &ocbinds.OpenconfigSystem_System{}
	}
	cleanup := &ocbinds.OpenconfigSystem_System_CoreFileCleanup{
		Config: &ocbinds.OpenconfigSystem_System_CoreFileCleanup_Config{},
		State:  &ocbinds.OpenconfigSystem_System_CoreFileCleanup_State{},
	}
	cleanup.Config.MaxCoreLimit = autoTsLimitFromDb(entry, autoTsMaxCoreLimitField)
	cleanup.State.MaxCoreLimit = autoTsLimitFromDb(entry, autoTsMaxCoreLimitField)
	cleanup.State.Enabled = autoTsEnabledFromDb(entry)
	deviceObj.System.CoreFileCleanup = cleanup
	return nil
}

var Subscribe_auto_techsupport_global_xfmr SubTreeXfmrSubscribe = func(inParams XfmrSubscInParams) (XfmrSubscOutParams, error) {
	return XfmrSubscOutParams{
		dbDataMap: RedisDbSubscribeMap{
			db.ConfigDB: {AUTO_TECHSUPPORT_TABLE: {AUTO_TECHSUPPORT_GLOBAL_KEY: {}}}},
		onChange: OnchangeEnable,
		nOpts:    &notificationOpts{mInterval: 0, pType: OnChange},
	}, nil
}
