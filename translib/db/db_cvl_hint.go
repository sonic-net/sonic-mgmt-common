////////////////////////////////////////////////////////////////////////////////
//                                                                            //
//  Copyright 2022 Broadcom. The term Broadcom refers to Broadcom Inc. and/or //
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
	// TBD Wait for the CVL PR
	// "github.com/Azure/sonic-mgmt-common/cvl"
	"github.com/Azure/sonic-mgmt-common/translib/tlerr"
	"github.com/golang/glog"
)

////////////////////////////////////////////////////////////////////////////////
//  Exported Types                                                            //
////////////////////////////////////////////////////////////////////////////////

////////////////////////////////////////////////////////////////////////////////
//  Exported Functions                                                        //
////////////////////////////////////////////////////////////////////////////////

// StoreCVLHint Stores a key, value hint for CVL Session Custom Validation CBs
func (d *DB) StoreCVLHint(key string, value interface{}) error {
	var err error

	if glog.V(3) {
		glog.Info("StoreCVLHint: Begin: key: ", key, " value: ", value)
		defer glog.Info("StoreCVLHint: End: key: ", key, " err: ", err)
	} else {
		glog.Info("StoreCVLHint: Begin: key: ", key)
	}

	if (d == nil) || (d.client == nil) {
		err = tlerr.TranslibDBConnectionReset{}
	} else if d.Opts.DBNo != ConfigDB {
		err = tlerr.TranslibDBNotSupported{
			Description: "StoreCVLHint: Supports CONFIG_DB only",
		}
	} else if len(key) == 0 {
		err = tlerr.TranslibDBNotSupported{
			Description: "StoreCVLHint: Supports non-zero string key type only",
		}
	} else if d.Opts.DisableCVLCheck {
		glog.Info("StoreCVLHint: CVL Disabled. Skipping CVL")
	} else {
		if d.cv != nil {
			// TBD Wait for the CVL PR
			// if crCode := d.cv.StoreHint(key, value); crCode != cvl.CVL_SUCCESS {
			// 	err = tlerr.TranslibCVLFailure{Code: int(crCode)}
			// } else {
			// 	// Savepoint
			// 	d.doCHintSave(key, value)
			// }
			d.doCHintSave(key, value)
		} else {
			glog.Info("StoreCVLHint: CVL Validation Session not opened.")

			// Save till Session Opened.
			if d.cvlHintsB4Open == nil {
				d.cvlHintsB4Open = make(map[string]interface{})
			}
			d.cvlHintsB4Open[key] = value
		}
	}

	if err != nil {
		glog.Error("StoreCVLHint: error: ", err)
	}

	return err
}

////////////////////////////////////////////////////////////////////////////////
//  Internal Functions                                                        //
////////////////////////////////////////////////////////////////////////////////

func (d *DB) clearCVLHint(key string) {

	if (d == nil) || (d.client == nil) {
		glog.Warningf("clearCVLHint: (d == nil): %t (d.client == nil): %t",
			d == nil, d.client == nil)
	} else if d.cv != nil {
		// TBD Wait for the CVL PR
		// if crCode := d.cv.StoreHint(key, nil); crCode != cvl.CVL_SUCCESS {
		// 	glog.Errorf("clearCVLHint: crCode: %v", crCode)
		// }
		d.doCHintSave(key, nil)
	}
}
