////////////////////////////////////////////////////////////////////////////////
//                                                                            //
//  Copyright 2023 Broadcom. The term Broadcom refers to Broadcom Inc. and/or //
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
	"encoding/json"
	"strconv"
	"strings"

	"github.com/Azure/sonic-mgmt-common/cvl"
	ctypes "github.com/Azure/sonic-mgmt-common/cvl/common"
	"github.com/Azure/sonic-mgmt-common/translib/tlerr"
	"github.com/go-redis/redis/v7"
	log "github.com/golang/glog"
)

type cvlDBAccess struct {
	Db *DB
}

func (d *DB) NewValidationSession() (*cvl.CVL, error) {
	if d == nil || d.Opts.DBNo != ConfigDB {
		return nil, tlerr.TranslibDBNotSupported{}
	}

	if c, status := cvl.ValidationSessOpen(&cvlDBAccess{d}); status != cvl.CVL_SUCCESS {
		return nil, tlerr.TranslibCVLFailure{Code: int(status)}
	} else {
		return c, nil
	}
}

func NewValidationSession() (*cvl.CVL, error) {
	if c, status := cvl.ValidationSessOpen(nil); status != cvl.CVL_SUCCESS {
		return nil, tlerr.TranslibCVLFailure{Code: int(status)}
	} else {
		return c, nil
	}
}

func (c *cvlDBAccess) Exists(key string) ctypes.IntResult {
	keys, err := c.Keys(key).Result()
	switch err.(type) {
	case tlerr.TranslibRedisClientEntryNotExist:
		err = redis.Nil
	}
	if len(keys) > 1 {
		//TODO have an optimized implementation in DBAL for Exists
		return intResult{int64(0), err}
	}
	return intResult{int64(len(keys)), err}
}

func (c *cvlDBAccess) Keys(pattern string) ctypes.StrSliceResult {
	ts, pat := c.Db.redis2ts_key(pattern)
	keys, err := c.Db.GetKeysPattern(&ts, pat)
	switch err.(type) {
	case tlerr.TranslibRedisClientEntryNotExist:
		err = redis.Nil
	}
	if err != nil {
		return strSliceResult{nil, err}
	}
	keyArr := make([]string, len(keys))
	for i, k := range keys {
		keyArr[i] = c.Db.key2redis(&ts, k)
	}
	return strSliceResult{keyArr, nil}
}

func (c *cvlDBAccess) HGet(key, field string) ctypes.StrResult {
	//TODO have an optimized implementation in DBAL
	data, err := c.HGetAll(key).Result()
	if err != nil {
		return strResult{"", err}
	}
	if v, ok := data[field]; ok {
		return strResult{v, nil}
	} else {
		return strResult{"", redis.Nil}
	}
}

func (c *cvlDBAccess) HMGet(key string, fields ...string) ctypes.SliceResult {
	//TODO have an optimized implementation in DBAL
	data, err := c.HGetAll(key).Result()
	if err != nil {
		return sliceResult{nil, err}
	}

	vals := make([]interface{}, len(fields))
	for i, field := range fields {
		if v, ok := data[field]; ok {
			vals[i] = v
		}
	}
	return sliceResult{vals, nil}
}

func (c *cvlDBAccess) HGetAll(key string) ctypes.StrMapResult {
	ts, k := c.Db.redis2ts_key(key)
	v, err := c.Db.GetEntry(&ts, k)
	switch err.(type) {
	case tlerr.TranslibRedisClientEntryNotExist:
		err = nil
	}
	if v.Field == nil {
		v.Field = map[string]string{}
	}
	return mapResult{v.Field, err}
}

func (c *cvlDBAccess) getTxData(pattern string, incRow bool) ([]byte, error) {
	ts, dbKey := c.Db.redis2ts_key(pattern)
	if log.V(5) {
		log.Infof("cvlDBAccess: getTxData: TableSpec: %v, Key: %v", ts, dbKey)
	}

	keyVals := make(map[string]map[string]string)

	for k := range c.Db.txTsEntryMap[ts.Name] {
		if patternMatch(k, 0, pattern, 0) {
			if len(c.Db.txTsEntryMap[ts.Name][k].Field) > 0 {
				if incRow {
					keyVals[k] = c.Db.txTsEntryMap[ts.Name][k].Field
				} else {
					keyVals[k] = map[string]string{}
				}
			} else {
				keyVals[k] = nil
			}
		}
	}

	return json.Marshal(keyVals)
}

func (c *cvlDBAccess) Lookup(s ctypes.Search) ctypes.JsonResult {
	var count string
	if s.Limit > 0 {
		count = strconv.Itoa(s.Limit)
	}

	txEntries, err := c.getTxData(s.Pattern, true)
	if err != nil {
		log.Warningf("cvlDBAccess: Lookup: error in getTxData: %v", err)
		return strResult{"", err}
	}

	v, err := cvl.RunLua(
		"filter_entries",
		s.Pattern,
		strings.Join(s.KeyNames, "|"),
		predicateToReturnStmt(s.Predicate),
		"", // Select fields -- not used by the lua script
		count,
		txEntries,
	)
	if err != nil {
		return strResult{"", err}
	}
	return strResult{v.(string), nil}
}

func (c *cvlDBAccess) Count(s ctypes.Search) ctypes.IntResult {
	incRow := len(s.Predicate) > 0 || len(s.WithField) > 0
	txEntries, err := c.getTxData(s.Pattern, incRow)
	if err != nil {
		log.Warningf("cvlDBAccess: Count: error in getTxData: %v", err)
		return intResult{0, err}
	}
	// Advanced key search, with match criteria on has values
	v, err := cvl.RunLua(
		"count_entries",
		s.Pattern,
		strings.Join(s.KeyNames, "|"),
		predicateToReturnStmt(s.Predicate),
		s.WithField,
		txEntries,
	)
	if err != nil {
		return intResult{0, err}
	}
	return intResult{v.(int64), nil}
}

func (c *cvlDBAccess) Pipeline() ctypes.PipeResult {
	pipe := c.Db.client.Pipeline()
	if log.V(5) {
		log.Infof("cvlDBAccess: Pipeline: redis pipeline: %v", pipe)
	}
	return &dbAccessPipe{dbAccess: c, rp: pipe}
}

func predicateToReturnStmt(p string) string {
	if len(p) == 0 || strings.HasPrefix(p, "return") {
		return p
	}
	return "return (" + p + ")"
}

//==================================

type strResult struct {
	val string
	err error
}

func (r strResult) Result() (string, error) {
	return r.val, r.err
}

//==================================

type strSliceResult struct {
	val []string
	err error
}

func (r strSliceResult) Result() ([]string, error) {
	return r.val, r.err
}

//==================================

type sliceResult struct {
	val []interface{}
	err error
}

func (r sliceResult) Result() ([]interface{}, error) {
	return r.val, r.err
}

//==================================

type mapResult struct {
	val map[string]string
	err error
}

func (r mapResult) Result() (map[string]string, error) {
	return r.val, r.err
}

//==================================

type intResult struct {
	val int64
	err error
}

func (ir intResult) Result() (int64, error) {
	return ir.val, ir.err
}

//==================================

type dbAccessPipe struct {
	rp         redis.Pipeliner
	qryResList []pipeQueryResult
	dbAccess   *cvlDBAccess
}

type pipeQueryResult interface {
	update(c *cvlDBAccess) // to update the db results with cache
}

type pipeKeysResult struct {
	pattern string
	sRes    strSliceResult
	rsRes   *redis.StringSliceCmd
}

func (p *dbAccessPipe) Keys(pattern string) ctypes.StrSliceResult {
	if log.V(5) {
		log.Infof("dbAccessPipe: Keys: for the given pattern: %v", pattern)
	}

	pr := &pipeKeysResult{pattern: pattern, rsRes: p.rp.Keys(pattern)}
	p.qryResList = append(p.qryResList, pr)
	return &pr.sRes
}

func (pr *pipeKeysResult) update(c *cvlDBAccess) {
	if log.V(5) {
		log.Infof("pipeQuery: update: key pattern: %v; redis pipe result: %v", pr.pattern, pr.rsRes)
	}

	keys, err := pr.rsRes.Result()
	if err != nil {
		log.Warningf("pipeKeysResult: update: error in Keys pipe query: keys: %v; err: %v", keys, err)
		pr.sRes.err = err
		return
	}

	keyMap := make(map[string]bool)
	for i := 0; i < len(keys); i++ {
		keyMap[keys[i]] = true
	}

	ts, dbKey := c.Db.redis2ts_key(pr.pattern)
	if log.V(5) {
		log.Infof("dbAccessPipe: TableSpec: %v, Key: %v", ts, dbKey)
	}

	for k := range c.Db.txTsEntryMap[ts.Name] {
		if patternMatch(k, 0, pr.pattern, 0) {
			if keyMap[k] {
				if len(c.Db.txTsEntryMap[ts.Name][k].Field) == 0 {
					keyMap[k] = false
				}
			} else if len(c.Db.txTsEntryMap[ts.Name][k].Field) > 0 {
				keyMap[k] = true
			}
		}
	}

	for k, v := range keyMap {
		if v {
			pr.sRes.val = append(pr.sRes.val, k)
		}
	}

	if len(pr.sRes.val) == 0 {
		pr.sRes.val = make([]string, 0)
	}
}

type pipeHMGetResult struct {
	key    string
	fields []string
	sRes   sliceResult
	rsRes  *redis.SliceCmd
	vals   []interface{}
}

func (p *dbAccessPipe) HMGet(key string, fields ...string) ctypes.SliceResult {
	if log.V(5) {
		log.Infof("dbAccessPipe: HMGet for the given key: %v, and the fields: %v", key, fields)
	}

	ts, dbKey := p.dbAccess.Db.redis2ts_key(key)
	if log.V(5) {
		log.Infof("dbAccessPipe: HMGet: TableSpec: %v, Key: %v", ts, dbKey)
	}

	pr := &pipeHMGetResult{key: key, fields: fields}

	if txEntry, ok := p.dbAccess.Db.txTsEntryMap[ts.Name][key]; ok {
		for _, fn := range fields {
			if fv, ok := txEntry.Field[fn]; ok {
				pr.vals = append(pr.vals, fv)
			} else {
				pr.vals = append(pr.vals, nil)
			}
		}
	} else {
		pr.rsRes = p.rp.HMGet(key, fields...)
	}

	p.qryResList = append(p.qryResList, pr)
	return &pr.sRes
}

func (pr *pipeHMGetResult) update(c *cvlDBAccess) {
	if log.V(5) {
		log.Infof("pipeHMGetResult: update: key: %v; fields: %v; "+
			"redis result: %v; cache result: %v", pr.key, pr.fields, pr.rsRes, pr.vals)
	}
	if pr.rsRes != nil {
		pr.sRes.val = pr.rsRes.Val()
		pr.sRes.err = pr.rsRes.Err()
	} else {
		pr.sRes.val = pr.vals
		pr.sRes.err = nil
	}
}

type pipeHGetResult struct {
	key      string
	field    string
	sRes     strResult
	rsRes    *redis.StringCmd
	val      string
	fldExist bool
}

func (p *dbAccessPipe) HGet(key, field string) ctypes.StrResult {
	if log.V(5) {
		log.Infof("dbAccessPipe: HGet for the given key: %v, and the field: %v", key, field)
	}

	pr := &pipeHGetResult{key: key, field: field}

	ts, dbKey := p.dbAccess.Db.redis2ts_key(key)
	if log.V(5) {
		log.Infof("dbAccessPipe: HGet: TableSpec: %v, Key: %v", ts, dbKey)
	}

	if txEntry, ok := p.dbAccess.Db.txTsEntryMap[ts.Name][key]; ok {
		pr.val, pr.fldExist = txEntry.Field[field]
	} else {
		pr.rsRes = p.rp.HGet(key, field)
	}

	p.qryResList = append(p.qryResList, pr)
	return &pr.sRes
}

func (pr *pipeHGetResult) update(c *cvlDBAccess) {
	if log.V(5) {
		log.Infof("pipeHGetResult: update: key: %v; field: %v; "+
			"redis result: %v; cache result: %v", pr.key, pr.field, pr.rsRes, pr.val)
	}
	if pr.rsRes != nil {
		pr.sRes.val = pr.rsRes.Val()
		pr.sRes.err = pr.rsRes.Err()
	} else if !pr.fldExist {
		pr.sRes.err = redis.Nil
	} else {
		pr.sRes.val = pr.val
	}
}

type pipeHGetAllResult struct {
	key    string
	sRes   mapResult
	rsRes  *redis.StringStringMapCmd
	fnvMap map[string]string
}

func (p *dbAccessPipe) HGetAll(key string) ctypes.StrMapResult {
	if log.V(5) {
		log.Infof("dbAccessPipe: HGetAll for the given key: %v", key)
	}

	pr := &pipeHGetAllResult{key: key, fnvMap: make(map[string]string)}

	ts, dbKey := p.dbAccess.Db.redis2ts_key(key)
	if log.V(5) {
		log.Infof("dbAccessPipe: HGetAll: TableSpec: %v, Key: %v", ts, dbKey)
	}

	if txEntry, ok := p.dbAccess.Db.txTsEntryMap[ts.Name][key]; ok {
		for k, v := range txEntry.Field {
			pr.fnvMap[k] = v
		}
	} else {
		pr.rsRes = p.rp.HGetAll(key)
	}

	p.qryResList = append(p.qryResList, pr)
	return &pr.sRes
}

func (pr *pipeHGetAllResult) update(c *cvlDBAccess) {
	if log.V(5) {
		log.Infof("pipeHGetAllResult: update: key: %v; "+
			"redis result: %v; cache result: %v", pr.key, pr.rsRes, pr.fnvMap)
	}
	if pr.rsRes != nil {
		pr.sRes.val = pr.rsRes.Val()
		pr.sRes.err = pr.rsRes.Err()
	} else {
		pr.sRes.val = pr.fnvMap
		pr.sRes.err = nil
	}
}

func (p *dbAccessPipe) Exec() error {
	if log.V(5) {
		log.Infof("dbAccessPipe: Exec: query list: %v", p.qryResList)
	}

	cmder, err := p.rp.Exec()
	if err != nil && err != redis.Nil {
		log.Warningf("dbAccessPipe: Exec: error in pipeline.Exec; error: %v; "+
			"cmder: %v; pw.qryMap: %v", err, cmder, p.qryResList)
	}

	// update the pipe query results with db cache
	for _, pr := range p.qryResList {
		pr.update(p.dbAccess)
	}

	if log.V(5) {
		log.Infof("dbAccessPipe: updated: pipe query list: %v; cmder: %v; error: %v", p.qryResList, cmder, err)
	}
	return err
}

func (p *dbAccessPipe) Close() {
	if log.V(5) {
		log.Infof("dbAccessPipe: Close: redis pipeliner: %v", p.rp)
	}
	p.rp.Close()
}

//==================================
