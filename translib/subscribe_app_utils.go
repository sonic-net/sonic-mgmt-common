////////////////////////////////////////////////////////////////////////////////
//                                                                            //
//  Copyright 2023 Broadcom. The term Broadcom refers to Broadcom Inc. and/or //
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

package translib

import (
	"fmt"

	"github.com/Azure/sonic-mgmt-common/translib/db"
	"github.com/openconfig/ygot/ygot"
)

// emptySubscribeResponse returns a translateSubResponse containing a non-db mapping
// for the given path
func emptySubscribeResponse(reqPath string) (*translateSubResponse, error) {
	p, err := ygot.StringToStructuredPath(reqPath)
	if err != nil {
		return nil, err
	}
	resp := new(translateSubResponse)
	resp.ntfAppInfoTrgt = append(resp.ntfAppInfoTrgt, &notificationAppInfo{
		path:                p,
		dbno:                db.MaxDB, // non-DB
		isOnChangeSupported: false,
	})
	return resp, nil
}

// translateSubscribeBridge calls the new translateSubscribe() on an app and returns the
// responses as per old signature. Will be removed after enhancing translib.Subscribe() API
func translateSubscribeBridge(path string, app appInterface, dbs [db.MaxDB]*db.DB) (*notificationOpts, *notificationInfo, error) {
	var nAppInfo *notificationAppInfo
	resp, err := app.translateSubscribe(&translateSubRequest{path: path, dbs: dbs})
	if err == nil && resp != nil && len(resp.ntfAppInfoTrgt) != 0 {
		nAppInfo = resp.ntfAppInfoTrgt[0]
	}
	if nAppInfo == nil {
		return nil, nil, fmt.Errorf("subscribe not supported (%w)", err)
	}

	nOpts := &notificationOpts{
		isOnChangeSupported: nAppInfo.isOnChangeSupported,
		pType:               nAppInfo.pType,
		mInterval:           nAppInfo.mInterval,
	}
	nInfo := &notificationInfo{dbno: nAppInfo.dbno}
	if !nAppInfo.isNonDB() {
		nInfo.table, nInfo.key = *nAppInfo.table, *nAppInfo.key
	}

	return nOpts, nInfo, nil
}
