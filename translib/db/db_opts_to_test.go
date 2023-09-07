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
	// "context"

	// "fmt"
	// "errors"
	// "flag"
	// "github.com/golang/glog"

	// "github.com/Azure/sonic-mgmt-common/translib/tlerr"
	// "os/exec"
	"os"
	// "reflect"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-redis/redis/v7"
)

func TestDefaultTimeout(t *testing.T) {

	var pid int = os.Getpid()

	t.Logf("TestDefaultTimeout: %s: begin", time.Now().String())

	d, e := NewDB(Options{
		DBNo:               ConfigDB,
		InitIndicator:      "",
		TableNameSeparator: "|",
		KeySeparator:       "|",
		IsWriteDisabled:    true,
		DisableCVLCheck:    true,
	})

	if d == nil {
		t.Errorf("NewDB() fails e = %v", e)
	}

	// Run a blocking LUA script on it for ~ 5 seconds
	var wg sync.WaitGroup
	wg.Add(1)
	go blockLUAScript(&wg, 5, t)

	t.Logf("TestDefaultTimeout: %s: Sleep(1) for LUA...", time.Now().String())
	time.Sleep(time.Second)
	t.Logf("TestDefaultTimeout: %s: call GetEntry()", time.Now().String())

	// Do GetEntry
	ts := TableSpec{Name: DBPAT_TST_PREFIX + strconv.FormatInt(int64(pid), 10)}

	ca := make([]string, 1, 1)
	ca[0] = "MyACL1_ACL_IPVNOTEXIST"
	akey := Key{Comp: ca}

	_, e = d.GetEntry(&ts, akey)

	// Confirm the Network Timeout Error
	if e != nil {
		t.Logf("GetEntry() got error e = %v", e)
		s := e.Error()
		if !(strings.HasPrefix(s, "i/o timeout") ||
			strings.HasPrefix(s, "BUSY")) {
			t.Errorf("GetEntry() Expecting timeout, BUSY... e = %v", e)
		}
	} else {
		t.Errorf("GetEntry() should have failed")
	}

	t.Logf("TestDefaultTimeout: %s: Wait for LUA...", time.Now().String())

	// Wait for the LUA script to return
	wg.Wait()

	t.Logf("TestDefaultTimeout: %s: Sleep(20s)...", time.Now().String())

	// Sleep In case we got a i/o timeout
	time.Sleep(20 * time.Second)

	if e = d.DeleteDB(); e != nil {
		t.Errorf("DeleteDB() fails e = %v", e)
	}

	t.Logf("TestDefaultTimeout: %s: end", time.Now().String())
}

func blockLUAScript(wg *sync.WaitGroup, secs int, t *testing.T) {

	defer wg.Done()

	t.Logf("blockLUAScript: %s: begin: secs: %v", time.Now().String(), secs)

	d, e := NewDB(Options{
		DBNo:               ConfigDB,
		InitIndicator:      "",
		TableNameSeparator: "|",
		KeySeparator:       "|",
		IsWriteDisabled:    true,
		DisableCVLCheck:    true,
	})

	if e != nil {
		t.Errorf("blockLUAScript: NewDB() fails e = %v", e)
	}

	// LUA does not have a sleep(), so empirical calculations.
	luaScript := redis.NewScript(`
local i = tonumber(ARGV[1]) * 5150000
while (i > 0) do
  local res = redis.call('GET', 'RANDOM_KEY')
  i=i-1
end
return i
`)

	if _, e := luaScript.Run(d.client, []string{}, secs).Int(); e != nil {
		t.Logf("blockLUAScript: luaScript.Run() fails e = %v", e)
	}

	t.Logf("blockLUAScript: %s: end: secs: %v", time.Now().String(), secs)

	t.Logf("blockLUAScript: %s: Sleep(2s)...", time.Now().String())
	time.Sleep(2 * time.Second)

	if e = d.DeleteDB(); e != nil {
		t.Errorf("blockLUAScript: DeleteDB() fails e = %v", e)
	}
}
