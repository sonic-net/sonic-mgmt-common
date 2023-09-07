////////////////////////////////////////////////////////////////////////////////
//                                                                            //
//  Copyright 2020 Broadcom. The term Broadcom refers to Broadcom Inc. and/or //
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
	"github.com/Azure/sonic-mgmt-common/translib/tlerr"
	"github.com/golang/glog"
)

////////////////////////////////////////////////////////////////////////////////
//  Exported Types                                                            //
////////////////////////////////////////////////////////////////////////////////

type MAP struct {
	ts       *TableSpec
	mapMap   map[string]string
	complete bool
	db       *DB
}

////////////////////////////////////////////////////////////////////////////////
//  Exported Functions                                                        //
////////////////////////////////////////////////////////////////////////////////

// GetMAP gets the entire Map.
func (d *DB) GetMAP(ts *TableSpec) (MAP, error) {
	if glog.V(3) {
		glog.Info("GetMAP: Begin: ts: ", ts)
	}

	if (d == nil) || (d.client == nil) {
		return MAP{}, tlerr.TranslibDBConnectionReset{}
	}

	var mapObj MAP

	v, e := d.GetMapAll(ts)
	if e == nil {
		mapObj = MAP{
			ts:       ts,
			complete: true,
			mapMap:   v.Field,
			db:       d,
		}
	}

	/*
		v, e := d.client.HGetAll(ts.Name).Result()

		if len(v) != 0 {
			mapObj.mapMap = v
		} else {
			if glog.V(1) {
				glog.Info("GetMAP: HGetAll(): empty map")
			}
			mapObj = MAP{}
			e = tlerr.TranslibRedisClientEntryNotExist { Entry: ts.Name }
		}
	*/

	if glog.V(3) {
		glog.Info("GetMAP: End: MAP: ", mapObj)
	}

	return mapObj, e
}

func (m *MAP) GetMap(mapKey string) (string, error) {
	if glog.V(3) {
		glog.Info("MAP.GetMap: Begin: ", " mapKey: ", mapKey)
	}

	var e error
	res, ok := m.mapMap[mapKey]
	if !ok {
		e = tlerr.TranslibRedisClientEntryNotExist{Entry: m.ts.Name}
	}

	if glog.V(3) {
		glog.Info("MAP.GetMap: End: ", "res: ", res, " e: ", e)
	}

	return res, e
}

func (m *MAP) GetMapAll() (Value, error) {

	if glog.V(3) {
		glog.Info("MAP.GetMapAll: Begin: ")
	}

	v := Value{Field: m.mapMap} // TBD: This is a reference

	if glog.V(3) {
		glog.Info("MAP.GetMapAll: End: ", "v: ", v)
	}

	return v, nil
}
