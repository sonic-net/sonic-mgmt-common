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
	// "fmt"
	// "strconv"

	// "errors"
	// "strings"

	// "github.com/Azure/sonic-mgmt-common/cvl"
	// "github.com/go-redis/redis/v7"
	"os"
	"os/signal"
	"syscall"
	// "github.com/golang/glog"
	// "github.com/Azure/sonic-mgmt-common/translib/tlerr"
)

////////////////////////////////////////////////////////////////////////////////
//  Exported Types                                                            //
////////////////////////////////////////////////////////////////////////////////

////////////////////////////////////////////////////////////////////////////////
//  Exported Functions                                                        //
////////////////////////////////////////////////////////////////////////////////

func SignalHandler() {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGUSR2)
	go func() {
		for {
			s := <-sigs
			if s == syscall.SIGUSR2 {
				HandleSIGUSR2()
			}
		}
	}()
}

func HandleSIGUSR2() {
	if dbCacheConfig != nil {
		dbCacheConfig.handleReconfigureSignal()
	}

	if dbStatsConfig != nil {
		dbStatsConfig.handleReconfigureSignal()
	}

	if dbRedisOptsConfig != nil {
		dbRedisOptsConfig.handleReconfigureSignal()
	}
}

////////////////////////////////////////////////////////////////////////////////
//  Internal Functions                                                        //
////////////////////////////////////////////////////////////////////////////////

func init() {
	SignalHandler()
}
