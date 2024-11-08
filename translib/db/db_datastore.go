///////////////////////////////////////////////////////////////////////////////
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

// import (
// "errors"
// "flag"
// "reflect"
// "strconv"
// "strings"
// "sync"
// "time"

// "github.com/Azure/sonic-mgmt-common/cvl"
// "github.com/go-redis/redis/v7"
// "github.com/golang/glog"
// )

////////////////////////////////////////////////////////////////////////////////
//  Exported Types                                                            //
////////////////////////////////////////////////////////////////////////////////

type DBDatastore interface {

	// Eg: Commit-ID, Filename(Future), Snapshot-ID(Future), Writable(Future)
	Attributes() map[string]string
}

// CommitIdDbDs is a Datastore modeled from a saved-to-disk CONFIG_DB of a
// commit-id  which is stored in the CHECKPOINTS_DIR, with a
// CHECKPOINT_EXT (cp.json)
type CommitIdDbDs struct {
	CommitID string
}

func (ds *CommitIdDbDs) Attributes() map[string]string {
	return map[string]string{
		"commit-id": ds.CommitID,
	}
}

// DefaultDbDs is the default Datastore representing the data
// stored in the CONFIG_DB database(/selection) of the redis-server
type DefaultDbDs struct {
}

func (ds *DefaultDbDs) Attributes() map[string]string {
	return map[string]string{}
}
