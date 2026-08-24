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
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/url"
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

const (
	platformFaultPath         = "/openconfig-platform:components/component/healthz/faults"
	platformFaultModule       = "openconfig-platform-healthz-fault"
	platformFaultModulePrefix = "oc-platform-healthz-fault"
)

var platformFaultPathNormalizer = strings.NewReplacer(
	"/openconfig-platform-healthz:healthz", "/healthz",
	"/openconfig-platform-healthz-fault:faults", "/faults",
)

type platformFault struct {
	component         string
	symptomName       string
	symptom           ocbinds.E_OpenconfigPlatformHealthzFault_SYMPTOM_BASE
	status            ocbinds.E_OpenconfigPlatform_Components_Component_Healthz_Faults_Fault_State_Status
	originTime        uint64
	lastDetectionTime uint64
	occurrences       uint64
	description       string
	repairActions     []ocbinds.E_OpenconfigPlatformHealthzFault_ACTION_BASE
}

type platformFaultRow struct {
	key   db.Key
	entry db.Value
}

type faultRepairAction struct {
	Action string `json:"action"`
}

func platformPathNeedsFaults(targetPath string) bool {
	targetPath = platformFaultPathNormalizer.Replace(targetPath)
	return strings.HasPrefix(targetPath, platformFaultPath) ||
		strings.HasPrefix(platformFaultPath, targetPath)
}

func platformPathNeedsEeprom(targetPath, componentName string) bool {
	switch targetPath {
	case "/openconfig-platform:components":
		return true
	case "/openconfig-platform:components/component":
		return componentName == "" || componentName == "System Eeprom"
	default:
		return strings.HasPrefix(targetPath, "/openconfig-platform:components/component/state")
	}
}

func (app *PlatformApp) doGetFaults(stateDb *db.DB) error {
	table, err := stateDb.GetTable(app.faultInfoTs)
	if err != nil {
		return fmt.Errorf("FAULT_INFO table get failed: %w", err)
	}

	keys, err := table.GetKeys()
	if err != nil {
		return fmt.Errorf("FAULT_INFO keys get failed: %w", err)
	}

	rows := make([]platformFaultRow, 0, len(keys))
	for _, key := range keys {
		entry, err := table.GetEntry(key)
		if err != nil {
			log.Warningf("Skipping FAULT_INFO key %v: %v", key.Comp, err)
			continue
		}
		rows = append(rows, platformFaultRow{key: key, entry: entry})
	}

	componentFilter := app.path.Var("name")
	symptomFilter, err := platformFaultIdentityName(app.path.Var("symptom"))
	if err != nil {
		return fmt.Errorf("invalid symptom filter: %w", err)
	}
	addPlatformFaultRows(app.getAppRootObject(), rows, componentFilter, symptomFilter)
	return nil
}

func addPlatformFaultRows(components *ocbinds.OpenconfigPlatform_Components, rows []platformFaultRow, componentFilter, symptomFilter string) int {
	added := 0
	for _, row := range rows {
		fault, err := parsePlatformFault(row.key, row.entry)
		if err != nil {
			log.Warningf("Skipping malformed FAULT_INFO key %v: %v", row.key.Comp, err)
			continue
		}
		if componentFilter != "" && componentFilter != "*" && componentFilter != fault.component {
			continue
		}
		if symptomFilter != "" && symptomFilter != "*" && symptomFilter != fault.symptomName {
			continue
		}
		if err := addPlatformFault(components, fault); err != nil {
			log.Warningf("Skipping FAULT_INFO key %v: %v", row.key.Comp, err)
			continue
		}
		added++
	}
	return added
}

func parsePlatformFault(key db.Key, entry db.Value) (*platformFault, error) {
	if key.Len() != 2 {
		return nil, fmt.Errorf("expected two key components, got %d", key.Len())
	}

	if strings.TrimSpace(entry.Get("component_type")) == "" {
		return nil, errors.New("component_type is empty")
	}
	componentName := strings.TrimSpace(entry.Get("component_name"))
	if componentName == "" {
		return nil, errors.New("component_name is empty")
	}

	symptomName, err := platformFaultIdentityName(entry.Get("symptom"))
	if err != nil {
		return nil, err
	}
	keyComponent, err := decodeFaultKeyComponent(key.Get(0))
	if err != nil {
		return nil, fmt.Errorf("invalid component key: %w", err)
	}
	keySymptom, err := decodeFaultKeyComponent(key.Get(1))
	if err != nil {
		return nil, fmt.Errorf("invalid symptom key: %w", err)
	}
	keySymptomName, err := platformFaultIdentityName(keySymptom)
	if err != nil {
		return nil, fmt.Errorf("invalid symptom key: %w", err)
	}
	if keyComponent != componentName || keySymptomName != symptomName {
		return nil, errors.New("key does not match component_name and symptom")
	}

	symptom, err := platformFaultSymptom(symptomName)
	if err != nil {
		return nil, err
	}
	status, err := platformFaultStatus(strings.TrimSpace(entry.Get("status")))
	if err != nil {
		return nil, err
	}
	originTime, err := decimalEpochToNanoseconds(entry.Get("origin_time"))
	if err != nil {
		return nil, fmt.Errorf("invalid origin_time: %w", err)
	}
	lastDetectionTime, err := decimalEpochToNanoseconds(entry.Get("last_detection_time"))
	if err != nil {
		return nil, fmt.Errorf("invalid last_detection_time: %w", err)
	}
	occurrences, err := strconv.ParseUint(entry.Get("occurrences"), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid occurrences: %w", err)
	}
	if occurrences == 0 {
		return nil, errors.New("occurrences must be at least 1")
	}

	actions, err := parsePlatformRepairActions(entry.Get("repair_actions"))
	if err != nil {
		return nil, err
	}
	if status == ocbinds.OpenconfigPlatform_Components_Component_Healthz_Faults_Fault_State_Status_ACTIVE && len(actions) == 0 {
		return nil, errors.New("active fault has no repair_actions")
	}
	description := entry.Get("description")
	if strings.TrimSpace(description) == "" {
		return nil, errors.New("description is empty")
	}

	return &platformFault{
		component:         componentName,
		symptomName:       symptomName,
		symptom:           symptom,
		status:            status,
		originTime:        originTime,
		lastDetectionTime: lastDetectionTime,
		occurrences:       occurrences,
		description:       description,
		repairActions:     actions,
	}, nil
}

func encodeFaultKeyComponent(value string) string {
	// DLDD uses Python's quote(value, safe=""), which preserves only RFC 3986
	// unreserved bytes. net/url.PathEscape preserves additional path bytes and
	// therefore cannot be used to construct a matching Redis key.
	const hexadecimal = "0123456789ABCDEF"
	var encoded strings.Builder
	for i := 0; i < len(value); i++ {
		character := value[i]
		if (character >= 'a' && character <= 'z') ||
			(character >= 'A' && character <= 'Z') ||
			(character >= '0' && character <= '9') ||
			character == '-' || character == '.' || character == '_' || character == '~' {
			encoded.WriteByte(character)
			continue
		}
		encoded.WriteByte('%')
		encoded.WriteByte(hexadecimal[character>>4])
		encoded.WriteByte(hexadecimal[character&0x0f])
	}
	return encoded.String()
}

func faultKeyPattern(value string) string {
	if value == "*" {
		return value
	}
	return encodeFaultKeyComponent(value)
}

func decodeFaultKeyComponent(value string) (string, error) {
	return url.PathUnescape(value)
}

func parsePlatformRepairActions(value string) ([]ocbinds.E_OpenconfigPlatformHealthzFault_ACTION_BASE, error) {
	if strings.TrimSpace(value) == "" {
		return nil, errors.New("repair_actions is missing")
	}

	var raw []faultRepairAction
	if err := json.Unmarshal([]byte(value), &raw); err != nil {
		return nil, fmt.Errorf("invalid repair_actions: %w", err)
	}
	if raw == nil {
		return nil, errors.New("repair_actions must be an array")
	}

	actions := make([]ocbinds.E_OpenconfigPlatformHealthzFault_ACTION_BASE, 0, len(raw))
	for i, item := range raw {
		action, err := platformFaultAction(item.Action)
		if err != nil {
			return nil, fmt.Errorf("invalid repair_actions[%d]: %w", i, err)
		}
		actions = append(actions, action)
	}
	return actions, nil
}

func addPlatformFault(components *ocbinds.OpenconfigPlatform_Components, fault *platformFault) error {
	component := components.Component[fault.component]
	if component == nil {
		var err error
		component, err = components.NewComponent(fault.component)
		if err != nil {
			return err
		}
	}
	if component.Healthz == nil {
		component.Healthz = &ocbinds.OpenconfigPlatform_Components_Component_Healthz{}
	}
	if component.Healthz.Faults == nil {
		component.Healthz.Faults = &ocbinds.OpenconfigPlatform_Components_Component_Healthz_Faults{}
	}

	faultNode, err := component.Healthz.Faults.NewFault(fault.symptom)
	if err != nil {
		return err
	}
	faultNode.State = &ocbinds.OpenconfigPlatform_Components_Component_Healthz_Faults_Fault_State{
		Counters: &ocbinds.OpenconfigPlatform_Components_Component_Healthz_Faults_Fault_State_Counters{
			Occurrences: &fault.occurrences,
		},
		LastDetectionTime: &fault.lastDetectionTime,
		OriginTime:        &fault.originTime,
		Status:            fault.status,
		Symptom:           fault.symptom,
	}
	if fault.description != "" {
		faultNode.State.Description = &fault.description
	}

	if len(fault.repairActions) == 0 {
		return nil
	}
	faultNode.Remediations = &ocbinds.OpenconfigPlatform_Components_Component_Healthz_Faults_Fault_Remediations{}
	for i, action := range fault.repairActions {
		index := uint64(i)
		remediation, err := faultNode.Remediations.NewRemediation(index)
		if err != nil {
			return err
		}
		remediation.State = &ocbinds.OpenconfigPlatform_Components_Component_Healthz_Faults_Fault_Remediations_Remediation_State{
			Action: action,
			Index:  &index,
			Target: &fault.component,
		}
	}
	return nil
}

func decimalEpochToNanoseconds(value string) (uint64, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, errors.New("empty timestamp")
	}
	if strings.HasPrefix(value, "+") || strings.HasPrefix(value, "-") {
		return 0, errors.New("timestamp must be non-negative decimal seconds")
	}

	parts := strings.Split(value, ".")
	if len(parts) > 2 || parts[0] == "" {
		return 0, errors.New("timestamp must be decimal seconds")
	}
	seconds, err := strconv.ParseUint(parts[0], 10, 64)
	if err != nil {
		return 0, err
	}
	if seconds > math.MaxUint64/1_000_000_000 {
		return 0, errors.New("timestamp overflows nanoseconds")
	}

	var fraction uint64
	if len(parts) == 2 {
		if parts[1] == "" {
			return 0, errors.New("timestamp fraction is empty")
		}
		fractionText := strings.TrimRight(parts[1], "0")
		if fractionText == "" {
			fractionText = "0"
		}
		if len(fractionText) > 9 {
			return 0, errors.New("timestamp has sub-nanosecond precision")
		}
		for _, digit := range fractionText {
			if digit < '0' || digit > '9' {
				return 0, errors.New("timestamp must be decimal seconds")
			}
		}
		fraction, err = strconv.ParseUint(fractionText, 10, 64)
		if err != nil {
			return 0, err
		}
		for i := len(fractionText); i < 9; i++ {
			fraction *= 10
		}
	}

	nanoseconds := seconds * 1_000_000_000
	if nanoseconds > math.MaxUint64-fraction {
		return 0, errors.New("timestamp overflows nanoseconds")
	}
	return nanoseconds + fraction, nil
}

func platformFaultIdentityName(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" || value == "*" {
		return value, nil
	}

	parts := strings.Split(value, ":")
	if len(parts) == 1 {
		return value, nil
	}
	if len(parts) != 2 || parts[1] == "" {
		return "", fmt.Errorf("invalid identity %q", value)
	}
	if parts[0] != platformFaultModule && parts[0] != platformFaultModulePrefix {
		return "", fmt.Errorf("unsupported identity namespace %q", parts[0])
	}
	return parts[1], nil
}

func platformFaultSymptom(value string) (ocbinds.E_OpenconfigPlatformHealthzFault_SYMPTOM_BASE, error) {
	const enumName = "E_OpenconfigPlatformHealthzFault_SYMPTOM_BASE"
	identities, ok := ocbinds.OpenconfigPlatformHealthzFault_SYMPTOM_BASE_UNSET.ΛMap()[enumName]
	if !ok {
		return ocbinds.OpenconfigPlatformHealthzFault_SYMPTOM_BASE_UNSET,
			fmt.Errorf("generated identity map %q is unavailable", enumName)
	}
	identityValue, err := platformFaultIdentityValue(value, identities)
	if err != nil {
		return ocbinds.OpenconfigPlatformHealthzFault_SYMPTOM_BASE_UNSET, err
	}
	return ocbinds.E_OpenconfigPlatformHealthzFault_SYMPTOM_BASE(identityValue), nil
}

func platformFaultAction(value string) (ocbinds.E_OpenconfigPlatformHealthzFault_ACTION_BASE, error) {
	const enumName = "E_OpenconfigPlatformHealthzFault_ACTION_BASE"
	identities, ok := ocbinds.OpenconfigPlatformHealthzFault_ACTION_BASE_UNSET.ΛMap()[enumName]
	if !ok {
		return ocbinds.OpenconfigPlatformHealthzFault_ACTION_BASE_UNSET,
			fmt.Errorf("generated identity map %q is unavailable", enumName)
	}
	identityValue, err := platformFaultIdentityValue(value, identities)
	if err != nil {
		return ocbinds.OpenconfigPlatformHealthzFault_ACTION_BASE_UNSET, err
	}
	return ocbinds.E_OpenconfigPlatformHealthzFault_ACTION_BASE(identityValue), nil
}

func platformFaultIdentityValue(value string, identities map[int64]ygot.EnumDefinition) (int64, error) {
	value = strings.TrimSpace(value)
	parts := strings.Split(value, ":")
	module := platformFaultModule
	identity := ""
	switch len(parts) {
	case 1:
		identity = parts[0]
	case 2:
		module = parts[0]
		identity = parts[1]
		if module == platformFaultModulePrefix {
			module = platformFaultModule
		}
	default:
		return 0, fmt.Errorf("invalid identity %q", value)
	}
	if module == "" || identity == "" {
		return 0, fmt.Errorf("invalid identity %q", value)
	}
	for enumValue, definition := range identities {
		if definition.Name == identity && definition.DefiningModule == module {
			return enumValue, nil
		}
	}
	return 0, fmt.Errorf("unsupported identity %q", value)
}

func platformFaultStatus(value string) (ocbinds.E_OpenconfigPlatform_Components_Component_Healthz_Faults_Fault_State_Status, error) {
	switch value {
	case "UNSPECIFIED":
		return ocbinds.OpenconfigPlatform_Components_Component_Healthz_Faults_Fault_State_Status_UNSPECIFIED, nil
	case "ACTIVE":
		return ocbinds.OpenconfigPlatform_Components_Component_Healthz_Faults_Fault_State_Status_ACTIVE, nil
	case "INACTIVE":
		return ocbinds.OpenconfigPlatform_Components_Component_Healthz_Faults_Fault_State_Status_INACTIVE, nil
	default:
		return ocbinds.OpenconfigPlatform_Components_Component_Healthz_Faults_Fault_State_Status_UNSET,
			fmt.Errorf("unsupported status %q", value)
	}
}

func (app *PlatformApp) translateFaultSubscribe(req translateSubRequest) (translateSubResponse, error) {
	targetPath, err := getYangPathFromUri(req.path)
	if err != nil || !platformPathNeedsFaults(targetPath) {
		return emptySubscribeResponse(req.path)
	}

	pathInfo := NewPathInfo(req.path)
	component := pathInfo.StringVar("name", "*")
	symptom, err := platformFaultIdentityName(pathInfo.StringVar("symptom", "*"))
	if err != nil {
		return translateSubResponse{}, tlerr.InvalidArgs("invalid symptom: %v", err)
	}
	subscribePath, err := ygot.StringToStructuredPath(req.path)
	if err != nil {
		return translateSubResponse{}, err
	}

	info := &notificationAppInfo{
		dbno:                db.StateDB,
		table:               &db.TableSpec{Name: "FAULT_INFO", CompCt: 2},
		key:                 &db.Key{Comp: []string{faultKeyPattern(component), faultKeyPattern(symptom)}},
		path:                subscribePath,
		handlerFunc:         processPlatformFaultOnChange,
		isOnChangeSupported: true,
		pType:               OnChange,
	}
	return translateSubResponse{ntfAppInfoTrgt: []*notificationAppInfo{info}}, nil
}

func (app *PlatformApp) processFaultSubscribe(req processSubRequest) (processSubResponse, error) {
	if req.table == nil || req.table.Name != "FAULT_INFO" || req.key == nil || req.key.Len() != 2 {
		return processSubResponse{}, tlerr.New("unsupported platform subscription")
	}
	resolved := path.Clone(req.path)
	component, err := decodeFaultKeyComponent(req.key.Get(0))
	if err != nil {
		return processSubResponse{}, err
	}
	symptom, err := decodeFaultKeyComponent(req.key.Get(1))
	if err != nil {
		return processSubResponse{}, err
	}
	symptom, err = platformFaultIdentityName(symptom)
	if err != nil {
		return processSubResponse{}, err
	}
	resolvePlatformFaultPath(resolved, component, symptom)
	return processSubResponse{path: resolved}, nil
}

func processPlatformFaultOnChange(ctx *apis.NotificationContext, sender apis.NotificationSender) {
	if ctx == nil || ctx.Key == nil || ctx.Key.Len() != 2 {
		return
	}

	oldFault, oldErr := parsePlatformFault(*ctx.Key, ctx.OldValue)
	newFault, newErr := parsePlatformFault(*ctx.Key, ctx.NewValue)
	if oldErr != nil && newErr != nil {
		return
	}

	var component, symptom string
	if newErr == nil {
		component = newFault.component
		symptom = newFault.symptomName
	} else if oldErr == nil {
		component = oldFault.component
		symptom = oldFault.symptomName
	}

	target := platformFaultNotificationPath(ctx.Path, component, symptom)
	if target == "" {
		return
	}
	parent, relative := path.SplitLastElem(target)
	if newErr == nil {
		sender.Send(&apis.Notification{Path: parent, UpdatePaths: []string{relative}})
		return
	}
	sender.Send(&apis.Notification{Path: parent, Delete: []string{relative}})
}

func platformFaultNotificationPath(subscribePath *gnmi.Path, component, symptom string) string {
	resolved := path.Clone(subscribePath)
	resolvePlatformFaultPath(resolved, component, symptom)

	if path.Len(resolved) < 5 {
		resolved = &gnmi.Path{Elem: []*gnmi.PathElem{
			{Name: "openconfig-platform:components"},
			{Name: "component", Key: map[string]string{"name": component}},
			{Name: "healthz"},
			{Name: "faults"},
			{Name: "fault", Key: map[string]string{"symptom": symptom}},
		}}
	}
	result, err := ygot.PathToString(resolved)
	if err != nil {
		return ""
	}
	return result
}

func resolvePlatformFaultPath(p *gnmi.Path, component, symptom string) {
	for i, elem := range p.Elem {
		switch elem.Name {
		case "component":
			path.SetKeyAt(p, i, "name", component)
		case "fault":
			path.SetKeyAt(p, i, "symptom", symptom)
		}
	}
}
