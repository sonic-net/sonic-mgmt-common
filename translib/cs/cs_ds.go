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

package cs

// Config Session Data Store

type DSType int

const (
	DSRunning DSType = iota

	DSCandidate  // Label (Session Token, except in Start Request)
	DSCheckpoint // Label (* in /etc/sonic/checkpoints/*.cp.json) (Future)
	DSFile       // Label (Arbitrary Filename *.json in the /etc dir) (Future)
)

type DataStore struct {
	Type  DSType
	Label string
}

// Token returns the Label if Type is DSCandidate. Otherwise ""
func (ds *DataStore) Token() string {
	var token string
	switch ds.Type {
	case DSCandidate:
		token = ds.Label
	default:
	}
	return token
}
