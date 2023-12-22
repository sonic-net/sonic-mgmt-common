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

package db

import (
	"fmt"
	"os"
	"testing"
	"time"

	// "reflect"
	"strconv"
	"strings"
	"sync"
)

func init() {
	fmt.Println("+++++  Init db_rpcdb_test  +++++")
}

var chNm string
var backendChNm string
var request = `["REQUEST", "COMMAND", "SEQUENCE", "3", "REPLY_TO", "DEBUGSH_SERVER", "INPUT", "show system internal logger vxlanmgrd"]`
var response = []string{`["RESPONSE","OUTPUT","SEQUENCE","3","LINE","================================================================================\n"]`,
	`["RESPONSE","OUTPUT","SEQUENCE","3","LINE","Component                           Level                     Output    \n"]`,
	`["RESPONSE","OUTPUT","SEQUENCE","3","LINE","================================================================================\n"]`,
	`["RESPONSE","OUTPUT","SEQUENCE","3","LINE","vxlanmgrd                           NOTICE                    SYSLOG    \n"]`,
	`["RESPONSE","END","SEQUENCE","3"]`,
}

func getChannelName() {
	var once sync.Once
	onceBody := func() {
		var pid int = os.Getpid()
		backendChNm = "DEBUGSH_CLIENT_" +
			strconv.FormatInt(int64(pid), 10) + "_CHANNEL"
		chNm = "DEBUGSH_SERVER_" + strconv.FormatInt(int64(pid), 10)
		request = strings.Replace(request, "DEBUGSH_SERVER", chNm, 1)
		for pos, _ := range response {
			response[pos] = strings.Replace(response[pos], `\n`, "\n", -1)
		}
	}
	once.Do(onceBody)
}

func openPubSubRpcDB(t *testing.T, nm string) *DB {
	d, e := PubSubRpcDB(Options{DBNo: LogLevelDB}, nm)

	if (d == nil) || (e != nil) {
		t.Fatalf("PubSubRpcDB() fails e = %v", e)
		return nil
	}
	return d
}

func startBackend(t *testing.T) {
	d := openPubSubRpcDB(t, backendChNm)
	defer d.ClosePubSubRpcDB()

	var msg []string
	var e error

	msg, e = d.GetRpcResponse(1, 5)

	if e != nil {
		t.Fatalf("startBackend: GetRpcResponse() fails e = %v", e)
	}

	if msg[0] != request {
		t.Fatalf("startBackend: GetRpcResponse() %s != %s", msg[0], request)
	}

	for pos, r := range response {
		var listeners int
		listeners, e = d.SendRpcRequest(chNm, r)

		if listeners != 1 || e != nil {
			t.Fatalf("startBackend: SendRpcRequest() listeners %d, response[%d]:(%s), fails e = %v", listeners, pos, r, e)
		}
	}
}

func TestPubSubRpcDB(t *testing.T) {
	getChannelName()
	d := openPubSubRpcDB(t, chNm)
	defer d.ClosePubSubRpcDB()
}

func TestPubSubRpcSend(t *testing.T) {
	getChannelName()
	go startBackend(t)

	time.Sleep(2 * time.Second)

	d := openPubSubRpcDB(t, chNm)
	defer d.ClosePubSubRpcDB()

	listeners, e := d.SendRpcRequest(backendChNm, request)
	if listeners != 1 || e != nil {
		t.Fatalf("SendRpcRequest() listeners %d, fails e = %v", listeners, e)
	}

	result, e2 := d.GetRpcResponse(len(response), 5)
	if e2 != nil {
		t.Fatalf("GetRpcResponse() fails e2 = %v", e2)
	}

	// Verify correct responses
	if len(result) != len(response) {
		t.Fatalf("GetRpcResponse() len(result) != len(response) -- %d != %d", len(result), len(response))
	}

	for pos, _ := range response {
		if result[pos] != response[pos] {
			t.Fatalf("GetRpcResponse(): %d: %s != %s", pos, result[pos], response[pos])
		}
	}
}
