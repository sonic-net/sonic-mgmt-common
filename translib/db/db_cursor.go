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
	"fmt"

	"reflect"

	"github.com/Azure/sonic-mgmt-common/translib/tlerr"
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
	scnr         scanner
}

// ScanType type indicates the type of scan (Eg: KeyScanType, FieldScanType).
type ScanType int

const (
	KeyScanType ScanType = iota // 0
	FieldScanType
)

type ScanCursorOpts struct {
	CountHint       int64  // Hint of redis work required
	ReturnFixed     bool   // (TBD) Return exactly CountHint # of keys
	AllowDuplicates bool   // Do not suppress redis duplicate keys
	ScanType               // To mention the type of scan; default is KeyScanType
	FldScanPatt     string // Field pattern to scan
	AllowWritable   bool   // Allow on write enabled DB object; ignores tx cache
}

type scanner interface {
	scan(sc *ScanCursor, countHint int64) ([]string, uint64, error)
}

type keyScanner struct {
}

type fieldScanner struct {
	fldNamePattern string // pattern to match field name
}

func (scnr *keyScanner) scan(sc *ScanCursor, countHint int64) ([]string, uint64, error) {
	return sc.db.client.Scan(sc.cursor,
		sc.db.key2redis(sc.ts, sc.pattern), countHint).Result()
}

func (scnr *fieldScanner) scan(sc *ScanCursor, countHint int64) ([]string, uint64, error) {
	key := sc.ts.Name
	if len(sc.pattern.Comp) > 0 {
		key = sc.db.key2redis(sc.ts, sc.pattern)
	}
	return sc.db.client.HScan(key, sc.cursor, scnr.fldNamePattern, countHint).Result()
}

////////////////////////////////////////////////////////////////////////////////
//  Exported Functions                                                        //
////////////////////////////////////////////////////////////////////////////////

// NewScanCursor Factory method to create ScanCursor; Scan cursor will not be supported for write enabled DB.
func (d *DB) NewScanCursor(ts *TableSpec, pattern Key, scOpts *ScanCursorOpts) (*ScanCursor, error) {
	if glog.V(3) {
		glog.Info("NewScanCursor: Begin: ts: ", ts, " pattern: ", pattern,
			" scOpts: ", scOpts)
	}

	if (d == nil) || (d.client == nil) {
		return nil, tlerr.TranslibDBConnectionReset{}
	}

	if !d.Opts.IsWriteDisabled && (scOpts == nil || !scOpts.AllowWritable) {
		err := fmt.Errorf("ScanCursor is not supported for write enabled DB")
		glog.Error("NewScanCursor: error: ", err)
		return nil, err
	}

	var countHint int64 = 10
	scnType := KeyScanType // default is key scanner

	if scOpts != nil {
		if scOpts.CountHint != 0 {
			countHint = scOpts.CountHint
		}
		scnType = scOpts.ScanType
	}

	var scnr scanner
	if scnType == KeyScanType { // Key Scanner
		scnr = &keyScanner{}
	} else if scnType == FieldScanType { // Field Scanner
		scnr = &fieldScanner{scOpts.FldScanPatt}
	}

	// Create ScanCursor
	scanCursor := ScanCursor{
		ts:      ts,
		pattern: pattern,
		count:   countHint,
		db:      d,
		scnr:    scnr,
	}

	if !scOpts.AllowDuplicates {
		scanCursor.seenKeys = make(map[string]bool, initialSCSeenKeysCacheSize)
	}

	if scOpts.ReturnFixed {
		glog.Info("NewScanCursor: ReturnFixed is not implemented")
		scanCursor.lookAhead = make([]string, 0, initialSCLookAheadBufferSize)
	}

	if glog.V(3) {
		glog.Info("NewScanCursor: End: scanCursor: ", scanCursor)
	}
	return &scanCursor, nil
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
	var keys []Key
	if (sc == nil) || (sc.db == nil) || (sc.db.client == nil) {
		return keys, true, tlerr.TranslibDBConnectionReset{}
	}

	if _, ok := sc.scnr.(*keyScanner); !ok {
		err := fmt.Errorf("Invalid scanner interface %v; exepcted is keyScanner", reflect.TypeOf(sc.scnr))
		glog.Error("ScanCursor: GetNextKeys: error: ", err)
		return keys, false, err
	}
	if scOpts != nil && scOpts.ScanType == FieldScanType {
		err := fmt.Errorf("Invalid scan cursor option: given scan type is %v; exepcted is %v", scOpts.ScanType, KeyScanType)
		glog.Error("ScanCursor: GetNextKeys: error: ", err)
		return keys, false, err
	}
	keys, _, _, scnComplete, err := sc.getNext(scOpts, false)
	return keys, scnComplete, err
}

// GetNextRedisKeys retrieves a few redisKeys. bool returns true if the scan is complete
func (sc *ScanCursor) GetNextRedisKeys(scOpts *ScanCursorOpts) ([]string, bool, error) {
	var redisKeys []string
	var scnComplete bool

	if _, ok := sc.scnr.(*keyScanner); !ok {
		err := fmt.Errorf("Invalid scanner interface %v; expected keyScanner",
			reflect.TypeOf(sc.scnr))
		glog.Error("ScanCursor: GetNextRedisKeys: error: ", err)
		return redisKeys, false, err
	}
	if scOpts != nil && scOpts.ScanType == FieldScanType {
		err := fmt.Errorf(
			"Invalid scan cursor option: given scan type is %v; expected is %v",
			scOpts.ScanType, KeyScanType)
		glog.Error("ScanCursor: GetNextRedisKeys: error: ", err)
		return redisKeys, false, err
	}
	_, redisKeys, _, scnComplete, err := sc.getNext(scOpts, true)
	return redisKeys, scnComplete, err
}

// GetNextFields retrieves a few matching fields. bool returns true if the scan is complete.
func (sc *ScanCursor) GetNextFields(scOpts *ScanCursorOpts) (Value, bool, error) {
	var val Value
	if (sc == nil) || (sc.db == nil) || (sc.db.client == nil) {
		return val, true, tlerr.TranslibDBConnectionReset{}
	}

	if _, ok := sc.scnr.(*fieldScanner); !ok {
		err := fmt.Errorf("Invalid scanner interface %v; exepcted is fieldScanner", reflect.TypeOf(sc.scnr))
		glog.Error("ScanCursor: GetNextFields: error: ", err)
		return val, false, err
	}
	if scOpts != nil && scOpts.ScanType == KeyScanType {
		err := fmt.Errorf("Invalid scan cursor option: given scan type is %v; exepcted is %v", scOpts.ScanType, FieldScanType)
		glog.Error("ScanCursor: GetNextFields: error: ", err)
		return val, false, err
	}
	_, _, fldNameVals, scnComplete, err := sc.getNext(scOpts, false)
	val = Value{Field: make(map[string]string)}
	for i := 0; i < len(fldNameVals); i = i + 2 {
		val.Field[fldNameVals[i]] = fldNameVals[i+1]
	}
	return val, scnComplete, err
}

// getNext retrieves next entry (either keys or fields based on the given scan type in the ScanCursorOpts,
// default is KeyScanType), bool returns true if the scan is complete.
func (sc *ScanCursor) getNext(scOpts *ScanCursorOpts, returnRedisKeys bool) ([]Key, []string, []string, bool, error) {
	if glog.V(6) {
		glog.Info("ScanCursor.getNext: Begin: sc: ", sc, "scOpts: ", scOpts)
	} else if glog.V(3) {
		glog.Info("ScanCursor.getNext: Begin: sc.pattern: ", sc.pattern)
	}

	var e error
	var entries []string
	var cursor uint64

	countHint := sc.count
	scanType := KeyScanType

	if scOpts != nil {
		if scOpts.CountHint != 0 {
			countHint = scOpts.CountHint
		}
		scanType = scOpts.ScanType
	}

	for (!sc.scanComplete) && (len(entries) == 0) && (e == nil) {
		entries, cursor, e = sc.scnr.scan(sc, countHint)
		if glog.V(4) {
			glog.Info("ScanCursor.getNext: entries: ", entries,
				" cursor: ", cursor, " e: ", e)
		}
		sc.cursor = cursor
		if sc.cursor == 0 {
			sc.scanComplete = true
		}
	}

	if e != nil {
		glog.Error("ScanCursor.getNext: pattern: ", sc.pattern, ": error: ", e)
	}

	var keys []Key
	var redisKeys []string
	var fldNameVals []string

	if scanType == KeyScanType {

		if returnRedisKeys {
			redisKeys = make([]string, 0, len(entries))
		} else {
			keys = make([]Key, 0, len(entries))
		}

		for i := 0; i < len(entries); i++ {
			if sc.seenKeys != nil {
				if _, present := sc.seenKeys[entries[i]]; present {
					continue
				}
				sc.seenKeys[entries[i]] = true
			}
			if returnRedisKeys {
				redisKeys = append(redisKeys, entries[i])
			} else {
				keys = append(keys, sc.db.redis2key(sc.ts, entries[i]))
			}
		}

	} else {
		fldNameVals = entries
	}

	if glog.V(6) {
		glog.Info("ScanCursor.getNext: End: entries: ", entries,
			" scanComplete: ", sc.scanComplete, " e: ", e)
	} else if glog.V(3) {
		glog.Info("ScanCursor.getNext: End: #entries: ", len(entries),
			" scanComplete: ", sc.scanComplete, " e: ", e)
	}
	return keys, redisKeys, fldNameVals, sc.scanComplete, e
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
