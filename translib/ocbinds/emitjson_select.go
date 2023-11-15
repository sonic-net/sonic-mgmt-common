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

package ocbinds

import (
	"os"
	"os/signal"
	"path/filepath"
	"sync/atomic"
	"syscall"

	"github.com/golang/glog"
	"github.com/openconfig/ygot/ygot"
)

// emitJSONImpl is a function that implements ygot-to-json translation.
type emitJSONImpl func(s ygot.ValidatedGoStruct, opts *EmitJSONOptions) ([]byte, error)

// useYgotEmitJSON indicates whether ygot.EmitJSON library API is to be used
// instead of the in-house implementation. Any non-zero value is treated as 'true'.
// It is defined as int32 instead of bool for using with atomic APIs.
var useYgotEmitJSON int32

func init() {
	setEmitJSONImpl()
	installSignalHandler()
}

// getEmitJSONImpl returns an emitJSONImpl function that should be used for
// ygot-to-json translation. Picks the function based on the value of useYgotEmitJSON flag.
func getEmitJSONImpl() emitJSONImpl {
	if atomic.LoadInt32(&useYgotEmitJSON) != 0 {
		return ygotEmitJSON
	}
	return newEmitJSON
}

// setEmitJSONImpl configures the emitJSONImpl to be used by setting the useYgotEmitJSON
// flag -- based on presence of a marker file /var/run/{exe_name}/use_ygot_emitjson.
func setEmitJSONImpl() {
	exeName := filepath.Base(os.Args[0])
	markerFile := filepath.Join("/var/run", exeName, "use_ygot_emitjson")
	if root, ok := os.LookupEnv("SYSROOT"); ok {
		markerFile = filepath.Join(root, markerFile)
	}

	if _, err := os.Stat(markerFile); err == nil {
		glog.Infof("Marker file %s exists; will use ygot.EmitJSON()", markerFile)
		atomic.StoreInt32(&useYgotEmitJSON, 1)
	} else {
		glog.Infof("Marker file %s not found (err = %v). Will use internal EmitJSON", markerFile, err)
		atomic.StoreInt32(&useYgotEmitJSON, 0)
	}
}

// installSignalHandler registers a SIGUSR2 handler which calls setEmitJSONImpl
// when signalled.
func installSignalHandler() {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGUSR2)
	go func() {
		for {
			<-sigs
			setEmitJSONImpl()
		}
	}()
}

// ygotEmitJSON implements ygot-to-json translation using the ygot.EmitJSON
// library API.
func ygotEmitJSON(s ygot.ValidatedGoStruct, opts *EmitJSONOptions) ([]byte, error) {
	glog.Infof("Render %T using ygot.EmitJSON", s)
	jsonStr, err := ygot.EmitJSON(s, &ygot.EmitJSONConfig{
		Format:         ygot.RFC7951,
		SkipValidation: true,
		RFC7951Config: &ygot.RFC7951JSONConfig{
			AppendModuleName: (opts == nil || !opts.NoPrefix),
		},
	})
	return []byte(jsonStr), err
}
