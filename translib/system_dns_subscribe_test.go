////////////////////////////////////////////////////////////////////////////////
//                                                                            //
//  Copyright 2026 Nokia.                                                     //
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

package translib

import (
	"testing"

	"github.com/Azure/sonic-mgmt-common/translib/db"
)

func TestTranslateSubscribe_dns_servers(t *testing.T) {
	const serversPath = "/openconfig-system:system/dns/servers"
	nInfo := func(addr string) *notificationAppInfo {
		return &notificationAppInfo{
			dbno:                db.ConfigDB,
			table:               &db.TableSpec{Name: "DNS_NAMESERVER"},
			key:                 db.NewKey(addr),
			isOnChangeSupported: true,
		}
	}

	// Server adds and deletes must be reported per server, not on the
	// servers container.
	for _, path := range []string{serversPath, serversPath + "/server"} {
		t.Run(path, func(t *testing.T) {
			tv := testTranslateSubscribe(t, path)
			tv.VerifyCount(1, 0)
			tv.VerifyTarget(serversPath+"/server[address=*]", nInfo("*"))
		})
	}

	t.Run("all states", func(t *testing.T) {
		tv := testTranslateSubscribe(t, serversPath+"/server[address=*]/state")
		tv.VerifyCount(1, 0)
		tv.VerifyTarget(serversPath+"/server[address=*]/state", nInfo("*"))
	})

	t.Run("one server", func(t *testing.T) {
		tv := testTranslateSubscribe(t, serversPath+"/server[address=192.0.2.1]/state/address")
		tv.VerifyCount(1, 0)
		tv.VerifyTarget(serversPath+"/server[address=192.0.2.1]/state/address", nInfo("192.0.2.1"))
	})
}
