////////////////////////////////////////////////////////////////////////////////
//                                                                            //
//  Copyright 2026 SONiC contributors.                                       //
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

package translib

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/Azure/sonic-mgmt-common/translib/db"
	"github.com/Azure/sonic-mgmt-common/translib/internal/apis"
	"github.com/Azure/sonic-mgmt-common/translib/ocbinds"
	"github.com/Azure/sonic-mgmt-common/translib/path"
	"github.com/Azure/sonic-mgmt-common/translib/tlerr"
	log "github.com/golang/glog"
	"github.com/openconfig/gnmi/proto/gnmi"
	"github.com/openconfig/ygot/ygot"
)

const platformHealthzStatePath = "/openconfig-platform:components/component/healthz/state"

var platformHealthzPathNormalizer = strings.NewReplacer(
	"/openconfig-platform-healthz:healthz", "/healthz",
	"/openconfig-platform-healthz:state", "/state",
)

var platformHealthzTableSpec = db.TableSpec{Name: "COMPONENT_HEALTH_INFO", CompCt: 1}

type platformHealthzState struct {
	component      string
	status         ocbinds.E_OpenconfigPlatform_Components_Component_Healthz_State_Status
	lastUnhealthy  *uint64
	unhealthyCount uint64
}

type platformHealthzRow struct {
	key   db.Key
	entry db.Value
}

func platformPathNeedsHealthzState(targetPath string) bool {
	targetPath = platformHealthzPathNormalizer.Replace(targetPath)
	return targetPath == platformHealthzStatePath ||
		strings.HasPrefix(targetPath, platformHealthzStatePath+"/") ||
		strings.HasPrefix(platformHealthzStatePath, targetPath+"/")
}

func (app *PlatformApp) doGetHealthzState(stateDb *db.DB) (int, error) {
	table, err := stateDb.GetTable(&platformHealthzTableSpec)
	if err != nil {
		return 0, fmt.Errorf("COMPONENT_HEALTH_INFO table get failed: %w", err)
	}
	keys, err := table.GetKeys()
	if err != nil {
		return 0, fmt.Errorf("COMPONENT_HEALTH_INFO keys get failed: %w", err)
	}

	rows := make([]platformHealthzRow, 0, len(keys))
	for _, key := range keys {
		entry, err := table.GetEntry(key)
		if err != nil {
			log.Warningf("Skipping COMPONENT_HEALTH_INFO key %v: %v", key.Comp, err)
			continue
		}
		rows = append(rows, platformHealthzRow{key: key, entry: entry})
	}
	return addPlatformHealthzRows(app.getAppRootObject(), rows, app.path.Var("name")), nil
}

func addPlatformHealthzRows(components *ocbinds.OpenconfigPlatform_Components, rows []platformHealthzRow, componentFilter string) int {
	added := 0
	for _, row := range rows {
		state, err := parsePlatformHealthzState(row.key, row.entry)
		if err != nil {
			log.Warningf("Skipping malformed COMPONENT_HEALTH_INFO key %v: %v", row.key.Comp, err)
			continue
		}
		if componentFilter != "" && componentFilter != "*" && componentFilter != state.component {
			continue
		}
		if err := addPlatformHealthzState(components, state); err != nil {
			log.Warningf("Skipping COMPONENT_HEALTH_INFO key %v: %v", row.key.Comp, err)
			continue
		}
		added++
	}
	return added
}

func parsePlatformHealthzState(key db.Key, entry db.Value) (*platformHealthzState, error) {
	if key.Len() != 1 {
		return nil, fmt.Errorf("expected one key component, got %d", key.Len())
	}
	component, err := decodeFaultKeyComponent(key.Get(0))
	if err != nil || strings.TrimSpace(component) == "" {
		return nil, fmt.Errorf("invalid component key %q: %v", key.Get(0), err)
	}

	var status ocbinds.E_OpenconfigPlatform_Components_Component_Healthz_State_Status
	switch strings.TrimSpace(entry.Get("status")) {
	case "HEALTHY":
		status = ocbinds.OpenconfigPlatform_Components_Component_Healthz_State_Status_HEALTHY
	case "UNHEALTHY":
		status = ocbinds.OpenconfigPlatform_Components_Component_Healthz_State_Status_UNHEALTHY
	default:
		return nil, fmt.Errorf("unsupported component health status %q", entry.Get("status"))
	}

	unhealthyCount, err := strconv.ParseUint(strings.TrimSpace(entry.Get("unhealthy_count")), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid unhealthy_count: %w", err)
	}
	var lastUnhealthy *uint64
	if entry.Has("last_unhealthy") {
		value, err := strconv.ParseUint(strings.TrimSpace(entry.Get("last_unhealthy")), 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid last_unhealthy: %w", err)
		}
		lastUnhealthy = &value
	}
	return &platformHealthzState{
		component:      component,
		status:         status,
		lastUnhealthy:  lastUnhealthy,
		unhealthyCount: unhealthyCount,
	}, nil
}

func addPlatformHealthzState(components *ocbinds.OpenconfigPlatform_Components, state *platformHealthzState) error {
	component := components.Component[state.component]
	if component == nil {
		var err error
		component, err = components.NewComponent(state.component)
		if err != nil {
			return err
		}
	}
	if component.Healthz == nil {
		component.Healthz = &ocbinds.OpenconfigPlatform_Components_Component_Healthz{}
	}
	component.Healthz.State = &ocbinds.OpenconfigPlatform_Components_Component_Healthz_State{
		Status:         state.status,
		LastUnhealthy:  state.lastUnhealthy,
		UnhealthyCount: &state.unhealthyCount,
	}
	return nil
}

func (app *PlatformApp) translateHealthzSubscribe(req translateSubRequest) (translateSubResponse, error) {
	targetPath, err := getYangPathFromUri(req.path)
	if err != nil || !platformPathNeedsHealthzState(targetPath) {
		return emptySubscribeResponse(req.path)
	}

	component := NewPathInfo(req.path).StringVar("name", "*")
	subscribePath, err := ygot.StringToStructuredPath(req.path)
	if err != nil {
		return translateSubResponse{}, err
	}
	info := &notificationAppInfo{
		dbno:                db.StateDB,
		table:               &platformHealthzTableSpec,
		key:                 &db.Key{Comp: []string{faultKeyPattern(component)}},
		path:                subscribePath,
		handlerFunc:         processPlatformHealthzOnChange,
		isOnChangeSupported: true,
		pType:               OnChange,
	}
	normalizedPath := platformHealthzPathNormalizer.Replace(targetPath)
	if normalizedPath == platformHealthzStatePath || strings.HasPrefix(normalizedPath, platformHealthzStatePath+"/") {
		return translateSubResponse{ntfAppInfoTrgt: []*notificationAppInfo{info}}, nil
	}
	response, err := emptySubscribeResponse(req.path)
	if err != nil {
		return translateSubResponse{}, err
	}
	response.ntfAppInfoTrgtChlds = []*notificationAppInfo{info}
	return response, nil
}

func (app *PlatformApp) processHealthzSubscribe(req processSubRequest) (processSubResponse, error) {
	if req.table == nil || req.table.Name != platformHealthzTableSpec.Name ||
		req.key == nil || req.key.Len() != platformHealthzTableSpec.CompCt {
		return processSubResponse{}, tlerr.New("unsupported component health subscription")
	}
	component, err := decodeFaultKeyComponent(req.key.Get(0))
	if err != nil {
		return processSubResponse{}, err
	}
	resolved := path.Clone(req.path)
	resolvePlatformHealthzPath(resolved, component)
	return processSubResponse{path: resolved}, nil
}

func processPlatformHealthzOnChange(ctx *apis.NotificationContext, sender apis.NotificationSender) {
	if ctx == nil || ctx.Key == nil || ctx.Key.Len() != 1 {
		return
	}
	oldState, oldErr := parsePlatformHealthzState(*ctx.Key, ctx.OldValue)
	newState, newErr := parsePlatformHealthzState(*ctx.Key, ctx.NewValue)
	if oldErr != nil && newErr != nil {
		return
	}
	if oldErr == nil && newErr == nil && !platformHealthzSubscribedValueChanged(ctx.Path, oldState, newState) {
		return
	}
	component := ""
	if newErr == nil {
		component = newState.component
	} else {
		component = oldState.component
	}
	target := platformHealthzNotificationPath(ctx.Path, component)
	if target == "" {
		return
	}
	parent, relative := path.SplitLastElem(target)
	if newErr == nil {
		sender.Send(&apis.Notification{Path: parent, UpdatePaths: []string{relative}})
	} else {
		sender.Send(&apis.Notification{Path: parent, Delete: []string{relative}})
	}
}

func platformHealthzSubscribedValueChanged(subscribePath *gnmi.Path, oldState, newState *platformHealthzState) bool {
	statusChanged := oldState.status != newState.status
	countChanged := oldState.unhealthyCount != newState.unhealthyCount
	lastChanged := !sameOptionalHealthzTime(oldState.lastUnhealthy, newState.lastUnhealthy)
	if subscribePath != nil && len(subscribePath.Elem) != 0 {
		leaf := strings.TrimPrefix(subscribePath.Elem[len(subscribePath.Elem)-1].Name, "openconfig-platform-healthz:")
		switch leaf {
		case "status":
			return statusChanged
		case "unhealthy-count":
			return countChanged
		case "last-unhealthy":
			return lastChanged
		}
	}
	return statusChanged || countChanged || lastChanged
}

func sameOptionalHealthzTime(a, b *uint64) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

func platformHealthzNotificationPath(subscribePath *gnmi.Path, component string) string {
	resolved := path.Clone(subscribePath)
	resolvePlatformHealthzPath(resolved, component)
	if path.Len(resolved) < 4 {
		resolved = &gnmi.Path{Elem: []*gnmi.PathElem{
			{Name: "openconfig-platform:components"},
			{Name: "component", Key: map[string]string{"name": component}},
			{Name: "healthz"},
			{Name: "state"},
		}}
	}
	result, err := ygot.PathToString(resolved)
	if err != nil {
		return ""
	}
	return result
}

func resolvePlatformHealthzPath(p *gnmi.Path, component string) {
	for i, elem := range p.Elem {
		if elem.Name == "component" {
			path.SetKeyAt(p, i, "name", component)
		}
	}
}
