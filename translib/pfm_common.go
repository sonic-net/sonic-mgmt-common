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
	"strings"

	"github.com/Azure/sonic-mgmt-common/translib/db"
	"github.com/Azure/sonic-mgmt-common/translib/tlerr"
	"github.com/Azure/sonic-mgmt-common/translib/transformer"
	"github.com/openconfig/ygot/ygot"
)

const platformHealthzPath = "/openconfig-platform:components/component/healthz"

func platformHealthzKeys(stateDB *db.DB, table *db.TableSpec, key db.Key) ([]db.Key, error) {
	if !stateDB.IsOpen() {
		return nil, tlerr.TranslibDBConnectionReset{}
	}
	if key.IsPattern() {
		return stateDB.GetKeysPattern(table, key)
	}
	return []db.Key{key}, nil
}

func isPlatformHealthzURI(uri string) bool {
	schemaPath, err := getYangPathFromUri(uri)
	if err != nil {
		return false
	}
	schemaPath = normalizePlatformSchemaPath(schemaPath)
	return schemaPath == platformHealthzPath || strings.HasPrefix(schemaPath, platformHealthzPath+"/")
}

func isPlatformHealthzParentURI(uri string) bool {
	schemaPath, err := getYangPathFromUri(uri)
	if err != nil {
		return false
	}
	switch normalizePlatformSchemaPath(schemaPath) {
	case "/openconfig-platform:components", "/openconfig-platform:components/component":
		return true
	}
	return false
}

func (app *CommonApp) platformHealthzApp(root *ygot.GoStruct) *PlatformApp {
	return &PlatformApp{path: app.pathInfo, ygotRoot: root, ygotTarget: app.ygotTarget}
}

// composePlatformHealthz keeps transformer-owned component data and adds DLDD
// state to parent requests. A projected component may have no inventory row in
// the upstream component tables, so its valid state can satisfy a NotFound.
func (app *CommonApp) composePlatformHealthz(root *ygot.GoStruct, dbs [db.MaxDB]*db.DB, qParams transformer.QueryParams, empty bool, err error) (bool, error) {
	if !isPlatformHealthzParentURI(app.pathInfo.Path) || (err != nil && !tlerr.IsNotFound(err)) {
		return empty, err
	}
	platform := app.platformHealthzApp(root)
	faults, getErr := platform.doGetFaults(dbs[db.StateDB])
	if getErr != nil {
		return empty, getErr
	}
	health, getErr := platform.doGetHealthzState(dbs[db.StateDB])
	if getErr != nil {
		return empty, getErr
	}
	if faults+health == 0 {
		return empty, err
	}
	if pruneErr := transformer.PruneQueryParams(root, qParams, app.pathInfo.Path, app.ctxt); pruneErr != nil {
		return empty, pruneErr
	}
	return false, nil
}

func (app *CommonApp) translateSubscribe(req translateSubRequest) (translateSubResponse, error) {
	if isPlatformHealthzURI(req.path) {
		return (&PlatformApp{}).translateSubscribe(req)
	}
	response, err := app.translateTransformerSubscribe(req)
	if !isPlatformHealthzParentURI(req.path) || (err != nil && !tlerr.IsNotFound(err)) {
		return response, err
	}
	healthz, healthErr := (&PlatformApp{}).translateSubscribe(req)
	if healthErr != nil {
		return translateSubResponse{}, healthErr
	}
	if err != nil {
		return healthz, nil
	}
	if NewPathInfo(req.path).Var("name") == "*" && len(response.ntfAppInfoTrgt) > 0 {
		if req.mode == Sample {
			// Sample reads only targets. Include components absent from inventory.
			response.ntfAppInfoTrgt = append(response.ntfAppInfoTrgt, healthz.ntfAppInfoTrgtChlds...)
			return response, nil
		}
		scansInventory := true
		for _, target := range response.ntfAppInfoTrgt {
			if target.isNonDB() || target.isDataSrcDynamic || !target.isOnChangeSupported || target.pType == Sample {
				scansInventory = false
				break
			}
		}
		if scansInventory {
			// These tables can contain components absent from the inventory scan.
			response.ntfAppInfoTrgt = append(response.ntfAppInfoTrgt, healthz.ntfAppInfoTrgtChlds...)
			return response, nil
		}
	}
	if len(response.ntfAppInfoTrgt) == 0 {
		response.ntfAppInfoTrgt = healthz.ntfAppInfoTrgt
	}
	response.ntfAppInfoTrgtChlds = append(response.ntfAppInfoTrgtChlds, healthz.ntfAppInfoTrgtChlds...)
	return response, nil
}
