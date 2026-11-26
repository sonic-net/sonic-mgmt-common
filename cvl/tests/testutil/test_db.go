/*
Copyright 2026 Google LLC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package testutil

import (
	"context"
	"encoding/json"
	"strings"

	cmn "github.com/Azure/sonic-mgmt-common/cvl/common"
	"github.com/redis/go-redis/v9"
)

type TestJsonResult struct {
	Val string
	Err error
}

func (r TestJsonResult) Result() (string, error) {
	return r.Val, r.Err
}

type TestIntResult struct {
	Val int64
	Err error
}

func (r TestIntResult) Result() (int64, error) {
	return r.Val, r.Err
}

type TestPipeResult struct {
	pipe redis.Pipeliner
}

func (p *TestPipeResult) Keys(pattern string) cmn.StrSliceResult {
	return p.pipe.Keys(context.Background(), pattern)
}

func (p *TestPipeResult) HGet(key, field string) cmn.StrResult {
	return p.pipe.HGet(context.Background(), key, field)
}

func (p *TestPipeResult) HMGet(key string, fields ...string) cmn.SliceResult {
	return p.pipe.HMGet(context.Background(), key, fields...)
}

func (p *TestPipeResult) HGetAll(key string) cmn.StrMapResult {
	return p.pipe.HGetAll(context.Background(), key)
}

func (p *TestPipeResult) Exec() error {
	_, err := p.pipe.Exec(context.Background())
	return err
}

func (p *TestPipeResult) Close() {
}

type TestDbAccess struct {
	rclient *redis.Client
}

func NewTestDbAccess(rclient *redis.Client) *TestDbAccess {
	return &TestDbAccess{rclient: rclient}
}

func (t *TestDbAccess) Exists(key string) cmn.IntResult {
	return t.rclient.Exists(context.Background(), key)
}

func (t *TestDbAccess) Keys(pattern string) cmn.StrSliceResult {
	return t.rclient.Keys(context.Background(), pattern)
}

func (t *TestDbAccess) HGet(key, field string) cmn.StrResult {
	return t.rclient.HGet(context.Background(), key, field)
}

func (t *TestDbAccess) HMGet(key string, fields ...string) cmn.SliceResult {
	return t.rclient.HMGet(context.Background(), key, fields...)
}

func (t *TestDbAccess) HGetAll(key string) cmn.StrMapResult {
	return t.rclient.HGetAll(context.Background(), key)
}

func (t *TestDbAccess) Pipeline() cmn.PipeResult {
	return &TestPipeResult{pipe: t.rclient.Pipeline()}
}

func (t *TestDbAccess) Lookup(s cmn.Search) cmn.JsonResult {
	ctx := context.Background()
	keys, err := t.rclient.Keys(ctx, s.Pattern).Result()
	if err != nil {
		return TestJsonResult{Val: "", Err: err}
	}
	if len(keys) == 0 {
		return TestJsonResult{Val: "{}", Err: nil}
	}

	dbMap := make(map[string]map[string]map[string]string)
	for _, k := range keys {
		parts := strings.SplitN(k, "|", 2)
		if len(parts) < 2 {
			continue
		}
		tbl, entryKey := parts[0], parts[1]
		fields, err := t.rclient.HGetAll(ctx, k).Result()
		if err != nil || len(fields) == 0 {
			continue
		}
		if _, ok := dbMap[tbl]; !ok {
			dbMap[tbl] = make(map[string]map[string]string)
		}
		dbMap[tbl][entryKey] = fields
	}

	b, err := json.Marshal(dbMap)
	if err != nil {
		return TestJsonResult{Val: "", Err: err}
	}
	return TestJsonResult{Val: string(b), Err: nil}
}

func (t *TestDbAccess) Count(s cmn.Search) cmn.IntResult {
	keys, err := t.rclient.Keys(context.Background(), s.Pattern).Result()
	if err != nil {
		return TestIntResult{Val: 0, Err: err}
	}
	return TestIntResult{Val: int64(len(keys)), Err: nil}
}
