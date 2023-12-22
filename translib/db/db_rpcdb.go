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
	// "sync"
	"time"

	"github.com/Azure/sonic-mgmt-common/translib/tlerr"
	"github.com/go-redis/redis/v7"
	"github.com/golang/glog"
)

////////////////////////////////////////////////////////////////////////////////
//  Exported Types                                                            //
////////////////////////////////////////////////////////////////////////////////

////////////////////////////////////////////////////////////////////////////////
//  Exported Functions                                                        //
////////////////////////////////////////////////////////////////////////////////

func PubSubRpcDB(opt Options, responseChannel string) (*DB, error) {

	var e error

	// mutexDbRpcDB.Lock()
	// defer mutexDbRpcDB.Unlock()

	if glog.V(3) {
		glog.Info("PubSubRpcDB: Begin: opt: ", opt, " responseChannel: ",
			responseChannel)
	}

	// NewDB
	d, e := NewDB(opt)
	if d.client == nil {
		return nil, e
	}

	// Future TBD: Mark as PubSubRpcDB

	// responseChannel db.Subscribe()
	// db.receive ()
	d.rPubSub = d.client.Subscribe(responseChannel)
	msg, e := d.rPubSub.Receive()
	if e != nil {
		glog.Error("PubSubRpcDB: ", d.Name(), ": ", responseChannel,
			": Receive() Error: ", e.Error())
		d.DeleteDB()
		return nil, e
	}

	switch msg.(type) {
	case *redis.Subscription:
		glog.V(3).Info("PubSubRpcDB: Subscription succeeded")
	case *redis.Message:
		glog.Error("PubSubRpcDB: ", d.Name(), ": ", responseChannel,
			": Message Received")
	case *redis.Pong:
		glog.V(3).Info("PubSubRpcDB: Pong received")
	default:
		glog.Error("PubSubRpcDB: ", d.Name(), ": ", responseChannel,
			": Unknown Received")
	}

	// Error checking

	return d, e
}

func (d *DB) SendRpcRequest(requestChannel string, message string) (int, error) {
	if glog.V(3) {
		glog.Info("SendRpcRequest: Begin: requestChannel: ", requestChannel,
			" message: ", message)
	}

	// Publish on this channel
	listeners, e := d.client.Publish(requestChannel, message).Result()

	if glog.V(3) {
		glog.Info("SendRpcRequest: End: listeners: ", requestChannel,
			" message: ", message)
	}
	return int(listeners), e
}

func (d *DB) GetRpcResponse(numResponses int, timeout int) ([]string, error) {

	var e error

	if glog.V(3) {
		glog.Info("GetRpcResponse: Begin: numResponses: ", numResponses,
			" timeout: ", timeout)
	}

	msg := make([]string, 0, numResponses)

	ch := d.rPubSub.Channel()

	// Get the number of messages for that time (redis client max time in
	// channel is 30 seconds)
	for i := 0; i < numResponses; i++ {
		var timedOut bool
		var toDuration time.Duration = time.Duration(int64(timeout) * int64(time.Second))
		select {
		case message := <-ch:
			if glog.V(5) {
				glog.Info("GetRpcResponse: ", i, ": ", message)
			}
			msg = append(msg, message.Payload)
		case <-time.After(toDuration):
			if glog.V(1) {
				glog.Info("GetRpcResponse: timeout: ", timeout,
					" msgs received: ", i)
			}
			timedOut = true
		}

		if timedOut {
			e = tlerr.TranslibTimeoutError{}
			break
		}
	}

	return msg, e
}

func (d *DB) ClosePubSubRpcDB() error {
	if glog.V(3) {
		glog.Info("ClosePubSubRpcDB: Begin: ")
	}

	if d.rPubSub != nil {
		glog.V(3).Info("ClosePubSubRpcDB: calling rPubSub.Close(): ")
		d.rPubSub.Close()
	}
	d.DeleteDB()
	return nil
}

////////////////////////////////////////////////////////////////////////////////
//  Internal Functions                                                        //
////////////////////////////////////////////////////////////////////////////////

// var mutexDbRpcDB sync.Mutex
