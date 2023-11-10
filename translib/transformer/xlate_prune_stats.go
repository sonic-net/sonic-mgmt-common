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

package transformer

import (
	"fmt"
	"time"
)

type PruneQPStats struct {
	Hits uint `json:"hits"`

	Time time.Duration `json:"total-time"`
	Peak time.Duration `json:"peak-time"`
	Last time.Duration `json:"last-time"`

	PeakUri string `json:"peak-uri"`
	LastUri string `json:"last-uri"`
}

var pruneQPStats PruneQPStats
var zeroPruneQPStats = &PruneQPStats{}

func GetPruneQPStats() *PruneQPStats {
	return &pruneQPStats
}

func (pqps *PruneQPStats) ClearPruneQPStats() {
	if pqps != nil {
		*pqps = *zeroPruneQPStats
	}
}

func (pqps *PruneQPStats) String() string {
	return fmt.Sprintf("\tLastTime: %s LastUri: %s Hits: %d TotalTime: %s PeakTime: %s PeakUri: %s\n",
		pqps.Last, pqps.LastUri, pqps.Hits, pqps.Time, pqps.Peak, pqps.PeakUri)
}

func (pqps *PruneQPStats) add(t time.Duration, uri string) {
	pqps.Hits++
	pqps.Last = t
	pqps.LastUri = uri
	pqps.Time += t
	if t > pqps.Peak {
		pqps.Peak = t
		pqps.PeakUri = uri
	}
}
