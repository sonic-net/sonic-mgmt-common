////////////////////////////////////////////////////////////////////////////////
//                                                                            //
//  Copyright 2023 Dell, Inc.                                                //
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

package transformer_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"reflect"
	"testing"

	. "github.com/Azure/sonic-mgmt-common/translib"
	db "github.com/Azure/sonic-mgmt-common/translib/db"
	"github.com/go-redis/redis/v7"
)

type queryParamsUT struct {
	depth   uint
	content string
	fields  []string
}

func checkErr(t *testing.T, err error, expErr error) {
	if err.Error() != expErr.Error() {
		t.Fatalf("Error %v, Expect Err: %v", err, expErr)
	} else if reflect.TypeOf(err) != reflect.TypeOf(expErr) {
		t.Fatalf("Error type %T, Expect Err Type: %T", err, expErr)
	}
}

func processGetRequest(url string, qparams *queryParamsUT, expectedRespJson string, errorCase bool, expErr ...error) func(*testing.T) {
	return func(t *testing.T) {
		var expectedMap map[string]interface{}
		var receivedMap map[string]interface{}
		var qp QueryParameters

		qp.Depth = 0
		qp.Content = ""
		qp.Fields = make([]string, 0)

		if qparams != nil {
			qp.Depth = qparams.depth
			qp.Content = qparams.content
			qp.Fields = qparams.fields
		}
		response, err := Get(GetRequest{Path: url, User: UserRoles{Name: "admin", Roles: []string{"admin"}}, QueryParams: qp})
		if err != nil {
			if !errorCase {
				t.Fatalf("Error %v received for Url: %s", err, url)
			} else if expErr != nil {
				checkErr(t, err, expErr[0])
			}
			return
		} else if errorCase {
			// Testcase expected an error, but no error recvd
			t.Fatalf("Error expected but no error received for Url: %s", url)
			return
		}

		err = json.Unmarshal([]byte(expectedRespJson), &expectedMap)
		if err != nil {
			t.Fatalf("failed to unmarshal %v err: %v", expectedRespJson, err)
		}

		respJson := response.Payload
		err = json.Unmarshal(respJson, &receivedMap)
		if err != nil {
			t.Fatalf("failed to unmarshal %v err: %v", string(respJson), err)
		}

		if reflect.DeepEqual(receivedMap, expectedMap) != true {
			t.Fatalf("Response for Url: %s received is not expected:\n Received: %s\n Expected: %s", url, receivedMap, expectedMap)
		}
	}
}

func processGetRequestWithFile(url string, expectedJsonFile string, errorCase bool, expErr ...error) func(*testing.T) {
	return func(t *testing.T) {
		var expectedMap map[string]interface{}
		var receivedMap map[string]interface{}

		jsonStr, err := ioutil.ReadFile(expectedJsonFile)
		if err != nil {
			t.Fatalf("read file %v err: %v", expectedJsonFile, err)
		}
		err = json.Unmarshal([]byte(jsonStr), &expectedMap)
		if err != nil {
			t.Fatalf("failed to unmarshal %v err: %v", jsonStr, err)
		}

		response, err := Get(GetRequest{Path: url, User: UserRoles{Name: "admin", Roles: []string{"admin"}}})
		if err != nil {
			if !errorCase {
				t.Fatalf("Error %v received for Url: %s", err, url)
			} else if expErr != nil {
				checkErr(t, err, expErr[0])
			}
			return
		} else if errorCase {
			// Testcase expected an error, but no error recvd
			t.Fatalf("Error expected but no error received for Url: %s", url)
			return
		}

		respJson := response.Payload
		err = json.Unmarshal(respJson, &receivedMap)
		if err != nil {
			t.Fatalf("failed to unmarshal %v err: %v", string(respJson), err)
		}

		if reflect.DeepEqual(receivedMap, expectedMap) != true {
			t.Fatalf("Response for Url: %s received is not expected:\n Received: %s\n Expected: %s", url, receivedMap, expectedMap)
		}
	}
}

func processSetRequest(url string, jsonPayload string, oper string, errorCase bool, expErr ...error) func(*testing.T) {
	return func(t *testing.T) {
		var err error
		switch oper {
		case "POST":
			_, err = Create(SetRequest{Path: url, Payload: []byte(jsonPayload)})
		case "PATCH":
			_, err = Update(SetRequest{Path: url, Payload: []byte(jsonPayload)})
		case "PUT":
			_, err = Replace(SetRequest{Path: url, Payload: []byte(jsonPayload)})
		default:
			t.Fatalf("Operation not supported")
		}
		if err != nil {
			if !errorCase {
				t.Fatalf("Error %v received for Url: %s", err, url)
			} else if expErr != nil {
				checkErr(t, err, expErr[0])
			}
		} else if errorCase {
			// Testcase expected an error, but no error recvd
			t.Fatalf("Error expected but no error received for Url: %s", url)
		}
	}
}

func processSetRequestFromFile(url string, jsonFile string, oper string, errorCase bool, expErr ...error) func(*testing.T) {
	return func(t *testing.T) {
		jsonPayload, err := ioutil.ReadFile(jsonFile)
		if err != nil {
			t.Fatalf("read file %v err: %v", jsonFile, err)
		}
		switch oper {
		case "POST":
			_, err = Create(SetRequest{Path: url, Payload: []byte(jsonPayload)})
		case "PATCH":
			_, err = Update(SetRequest{Path: url, Payload: []byte(jsonPayload)})
		case "PUT":
			_, err = Replace(SetRequest{Path: url, Payload: []byte(jsonPayload)})
		default:
			t.Fatalf("Operation not supported")
		}
		if err != nil {
			if !errorCase {
				t.Fatalf("Error %v received for Url: %s", err, url)
			} else if expErr != nil {
				checkErr(t, err, expErr[0])
			}
		} else if errorCase {
			// Testcase expected an error, but no error recvd
			t.Fatalf("Error expected but no error received for Url: %s", url)
		}
	}
}

func processDeleteRequest(url string, errorCase bool, expErr ...error) func(*testing.T) {
	return func(t *testing.T) {
		_, err := Delete(SetRequest{Path: url})
		if err != nil {
			if !errorCase {
				t.Fatalf("Error %v received for Url: %s", err, url)
			} else if expErr != nil {
				checkErr(t, err, expErr[0])
			}
		} else if errorCase {
			// Testcase expected an error, but no error recvd
			t.Fatalf("Error expected but no error received for Url: %s", url)
		}
	}
}

func processActionRequest(url string, jsonPayload string, oper string, user string, role string, auth bool, errorCase bool, expErr ...error) func(*testing.T) {
	return func(t *testing.T) {
		var err error
		switch oper {
		case "POST":
			ur := UserRoles{Name: user, Roles: []string{role}}
			_, err = Action(ActionRequest{Path: url, Payload: []byte(jsonPayload), User: ur, AuthEnabled: auth})
		default:
			t.Fatalf("Operation not supported")
		}
		if err != nil {
			if !errorCase {
				t.Fatalf("Error %v received for Url: %s", err, url)
			} else if expErr != nil {
				checkErr(t, err, expErr[0])
			}
		} else if errorCase {
			// Testcase expected an error, but no error recvd
			t.Fatalf("Error expected but no error received for Url: %s", url)
		}
	}
}

func getConfigDb() *db.DB {
	configDb, _ := db.NewDB(db.Options{
		DBNo:               db.ConfigDB,
		TableNameSeparator: "|",
		KeySeparator:       "|",
	})

	return configDb
}

func verifyDbResult(client *redis.Client, key string, expectedResult map[string]interface{}, errorCase bool) func(*testing.T) {
	return func(t *testing.T) {
		result, err := client.HGetAll(key).Result()
		if err != nil {
			t.Fatalf("Error %v hgetall for key: %s", err, key)
		}

		expect := make(map[string]string)
		for ts := range expectedResult {
			for _, k := range expectedResult[ts].(map[string]interface{}) {
				for f, v := range k.(map[string]interface{}) {
					strKey := fmt.Sprintf("%v", f)
					var strVal string
					strVal = fmt.Sprintf("%v", v)
					expect[strKey] = strVal
				}
			}
		}

		if reflect.DeepEqual(result, expect) != true {
			t.Fatalf("Response for Key: %v received is not expected: Received %v Expected %v\n", key, result, expect)
		}
	}
}

var emptyJson string = "{}"
