////////////////////////////////////////////////////////////////////////////////
//                                                                            //
//  Copyright 2021 Broadcom. The term Broadcom refers to Broadcom Inc. and/or //
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

package db

import (
	"strings"

	"github.com/Azure/sonic-mgmt-common/translib/tlerr"
	"github.com/golang/glog"
)

////////////////////////////////////////////////////////////////////////////////
//  Exported Types                                                            //
////////////////////////////////////////////////////////////////////////////////

////////////////////////////////////////////////////////////////////////////////
//  Exported Functions                                                        //
////////////////////////////////////////////////////////////////////////////////

// Get gets the value of the key
func (d *DB) Get(key string) (string, error) {
	if glog.V(3) {
		glog.Info("Get: Begin: key: ", key)
	}

	if (d == nil) || (d.client == nil) {
		return "", tlerr.TranslibDBConnectionReset{}
	}

	// Only meant to retrieve metadata.
	if !strings.HasPrefix(key, "CONFIG_DB") || strings.Contains(key, "|") {
		return "", UseGetEntry
	}

	glog.Info("Get: RedisCmd: ", d.Name(), ": ", "GET ", key)
	val, e := d.client.Get(key).Result()

	if glog.V(3) {
		glog.Info("Get: End: key: ", key, " val: ", val, " e: ", e)
	}

	return val, e
}

////////////////////////////////////////////////////////////////////////////////
//  Internal Functions                                                        //
////////////////////////////////////////////////////////////////////////////////
