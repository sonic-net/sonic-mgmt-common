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
	"github.com/golang/glog"
)

// Work Done vs Entries Returned (Need to return a fixed number of
//  entries for every call? Is variable # of entries returned ok ?
//  If fixed, it will partially negate the Time complexity advantage of
//  pagination. [ Variable number of keys returned for every call to
//  the ScanCursor API is ok. If fixed is needed, it may be implemented later.]

// Duplicate Suppression? Redis can return duplicates. To supress them
//  we need a cache. This will negate the Space complexity advantage of
//  pagination. [ Duplicates to be suppressed. Currently the OnChange is
//  getting all the keys first, so there is effectively a store of all the
//  keys at some point. This will now be moved to a map/dictionary inside the
//  ScanCursor to avoid duplicates.]

////////////////////////////////////////////////////////////////////////////////
//  Exported Types                                                            //
////////////////////////////////////////////////////////////////////////////////

// ScanCursor iterates over a set of keys

type ScanCursor struct {
	ts           *TableSpec
	cursor       uint64
	pattern      Key
	count        int64
	scanComplete bool
	seenKeys     map[string]bool
	lookAhead    []string // (TBD) For exactly CountHint # of keys
	db           *DB
}

type ScanCursorOpts struct {
	CountHint       int64 // Hint of redis work required
	ReturnFixed     bool  // (TBD) Return exactly CountHint # of keys
	AllowDuplicates bool  // Do not suppress redis duplicate keys
}

////////////////////////////////////////////////////////////////////////////////
//  Exported Functions                                                        //
////////////////////////////////////////////////////////////////////////////////

// NewScanCursor Factory method to create ScanCursor
func (d *DB) NewScanCursor(ts *TableSpec, pattern Key, scOpts *ScanCursorOpts) (*ScanCursor, error) {
	if glog.V(3) {
		glog.Info("NewScanCursor: Begin: ts: ", ts, " pattern: ", pattern,
			" scOpts: ", scOpts)
	}

	var e error
	var countHint int64 = 10

	if scOpts != nil && scOpts.CountHint != 0 {
		countHint = scOpts.CountHint
	}

	// Create ScanCursor
	scanCursor := ScanCursor{
		ts:      ts,
		pattern: pattern,
		count:   countHint,
		db:      d,
	}

	if !scOpts.AllowDuplicates {
		scanCursor.seenKeys = make(map[string]bool, initialSCSeenKeysCacheSize)
	}

	if scOpts.ReturnFixed {
		glog.Info("NewScanCursor: ReturnFixed is not implemented")
		scanCursor.lookAhead = make([]string, 0, initialSCLookAheadBufferSize)
	}

	if glog.V(3) {
		glog.Info("NewScanCursor: End: scanCursor: ", scanCursor, e)
	}
	return &scanCursor, e
}

// DeleteScanCursor Gently release state/cache of ScanCursor
func (sc *ScanCursor) DeleteScanCursor() error {
	if glog.V(6) {
		glog.Info("DeleteScanCursor: Begin: sc: ", sc)
	} else if glog.V(3) {
		glog.Info("DeleteScanCursor: Begin: sc.pattern: ", sc.pattern,
			" #keys: ", len(sc.seenKeys))
	}

	// Release any state/cache
	sc.scanComplete = true
	sc.seenKeys = nil
	sc.lookAhead = nil

	return nil
}

// GetNextKeys retrieves a few keys. bool returns true if the scan is complete.
func (sc *ScanCursor) GetNextKeys(scOpts *ScanCursorOpts) ([]Key, bool, error) {
	if glog.V(6) {
		glog.Info("ScanCursor.GetNextKeys: Begin: sc: ", sc, "scOpts: ", scOpts)
	} else if glog.V(3) {
		glog.Info("ScanCursor.GetNextKeys: Begin: sc.pattern: ", sc.pattern)
	}

	var e error
	var redisKeys []string
	var cursor uint64

	countHint := sc.count

	if scOpts != nil && scOpts.CountHint != 0 {
		countHint = scOpts.CountHint
	}

	for (!sc.scanComplete) && (len(redisKeys) == 0) && (e == nil) {
		redisKeys, cursor, e = sc.db.client.Scan(sc.cursor,
			sc.db.key2redis(sc.ts, sc.pattern), countHint).Result()
		if glog.V(4) {
			glog.Info("ScanCursor.GetNextKeys: redisKeys: ", redisKeys,
				" cursor: ", cursor, " e: ", e)
		}
		sc.cursor = cursor
		if sc.cursor == 0 {
			sc.scanComplete = true
		}
	}

	if e != nil {
		glog.Error("ScanCursor.GetNextKeys: error: ", e)
	}

	keys := make([]Key, 0, len(redisKeys))
	for i := 0; i < len(redisKeys); i++ {
		if sc.seenKeys != nil {
			if _, present := sc.seenKeys[redisKeys[i]]; present {
				continue
			}
			sc.seenKeys[redisKeys[i]] = true
		}
		keys = append(keys, sc.db.redis2key(sc.ts, redisKeys[i]))
	}

	if glog.V(6) {
		glog.Info("ScanCursor.GetNextKeys: End: keys: ", keys,
			" scanComplete: ", sc.scanComplete, " e: ", e)
	} else if glog.V(3) {
		glog.Info("ScanCursor.GetNextKeys: End: #keys: ", len(keys),
			" scanComplete: ", sc.scanComplete, " e: ", e)
	}
	return keys, sc.scanComplete, e
}

////////////////////////////////////////////////////////////////////////////////
//  Internal Constants                                                        //
////////////////////////////////////////////////////////////////////////////////

const (
	initialSCSeenKeysCacheSize   = 100
	initialSCLookAheadBufferSize = 100
)

////////////////////////////////////////////////////////////////////////////////
//  Internal Functions                                                        //
////////////////////////////////////////////////////////////////////////////////
