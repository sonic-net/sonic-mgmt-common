////////////////////////////////////////////////////////////////////////////////
//                                                                            //
//  Copyright 2019 Broadcom. The term Broadcom refers to Broadcom Inc. and/or //
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
//go:build radius_prune_xfmrtest
// +build radius_prune_xfmrtest

package transformer

import (
	"reflect"

	"github.com/Azure/sonic-mgmt-common/translib/ocbinds"
)

func init() {
	xp_tests = append(xp_tests, radius_xp_tests...)
}

var radius_xp_tests = []xpTests{
	{ // RADIUS Identity Test Case (i.e. no QP)
		tid:        "RADIUS Identity",
		uri:        "/openconfig-system:system/aaa/server-groups/server-group[name=RADIUS]/servers/server[address=10.10.10.10]",
		requestUri: "/openconfig-system:system/aaa/server-groups/server-group[name=RADIUS]/servers/server[address=10.10.10.10]",
		payload: []byte(`
            {
               "openconfig-system:server" : [
                  {
                     "tacacs" : {
                        "config" : {
                           "encrypted" : true,
                           "secret-key" : ""
                        }
                     },
                     "radius" : {
                        "config" : {
                           "openconfig-aaa-radius-ext:encrypted" : true,
                           "secret-key" : "",
                           "retransmit-attempts" : 1
                        },
                        "state" : {
                           "retransmit-attempts" : 1,
                           "counters" : {
                              "access-rejects" : "1",
                              "access-accepts" : "2",
                              "openconfig-aaa-radius-ext:access-requests" : "3"
                           }
                        }
                     },
                     "config" : {
                        "priority" : 1,
                        "auth-type" : "chap",
                        "address" : "10.10.10.10"
                     },
                     "address" : "10.10.10.10",
                     "state" : {
                        "priority" : 1,
                        "auth-type" : "chap",
                        "address" : "10.10.10.10"
                     }
                  }
               ]
            }
            `),
		appRootType: reflect.TypeOf(ocbinds.OpenconfigSystem_System{}),
		queryParams: QueryParams{
			depthEnabled:  false,
			curDepth:      1,
			content:       QUERY_CONTENT_ALL,
			fields:        []string{},
			fieldsFillAll: false,
		},
		prunedPayload: []byte(`
            {
               "openconfig-system:server" : [
                  {
                     "tacacs" : {
                        "config" : {
                           "encrypted" : true,
                           "secret-key" : ""
                        }
                     },
                     "radius" : {
                        "config" : {
                           "openconfig-aaa-radius-ext:encrypted" : true,
                           "secret-key" : "",
                           "retransmit-attempts" : 1
                        },
                        "state" : {
                           "retransmit-attempts" : 1,
                           "counters" : {
                              "access-rejects" : "1",
                              "access-accepts" : "2",
                              "openconfig-aaa-radius-ext:access-requests" : "3"
                           }
                        }
                     },
                     "config" : {
                        "priority" : 1,
                        "auth-type" : "chap",
                        "address" : "10.10.10.10"
                     },
                     "address" : "10.10.10.10",
                     "state" : {
                        "priority" : 1,
                        "auth-type" : "chap",
                        "address" : "10.10.10.10"
                     }
                  }
               ]
            }
            `),
	},
	{ // RADIUS depth = 3
		tid:        "RADIUS Depth",
		uri:        "/openconfig-system:system/aaa/server-groups/server-group[name=RADIUS]/servers/server[address=10.10.10.10]",
		requestUri: "/openconfig-system:system/aaa/server-groups/server-group[name=RADIUS]/servers/server[address=10.10.10.10]",
		payload: []byte(`
            {
               "openconfig-system:server" : [
                  {
                     "tacacs" : {
                        "config" : {
                           "encrypted" : true,
                           "secret-key" : ""
                        }
                     },
                     "radius" : {
                        "config" : {
                           "openconfig-aaa-radius-ext:encrypted" : true,
                           "secret-key" : "",
                           "retransmit-attempts" : 1
                        },
                        "state" : {
                           "retransmit-attempts" : 1,
                           "counters" : {
                              "access-rejects" : "1",
                              "access-accepts" : "2",
                              "openconfig-aaa-radius-ext:access-requests" : "3"
                           }
                        }
                     },
                     "config" : {
                        "priority" : 1,
                        "auth-type" : "chap",
                        "address" : "10.10.10.10"
                     },
                     "address" : "10.10.10.10",
                     "state" : {
                        "priority" : 1,
                        "auth-type" : "chap",
                        "address" : "10.10.10.10"
                     }
                  }
               ]
            }
            `),
		appRootType: reflect.TypeOf(ocbinds.OpenconfigSystem_System{}),
		queryParams: QueryParams{
			depthEnabled:  true,
			curDepth:      3,
			content:       QUERY_CONTENT_ALL,
			fields:        []string{},
			fieldsFillAll: false,
		},
		prunedPayload: []byte(`
            {
               "openconfig-system:server" : [
                  {
                     "tacacs" : {
                        "config" : {
                        }
                     },
                     "radius" : {
                        "config" : {
                        },
                        "state" : {
                        }
                     },
                     "config" : {
                        "priority" : 1,
                        "auth-type" : "chap",
                        "address" : "10.10.10.10"
                     },
                     "address" : "10.10.10.10",
                     "state" : {
                        "priority" : 1,
                        "auth-type" : "chap",
                        "address" : "10.10.10.10"
                     }
                  }
               ]
            }
            `),
	},
	{ // RADIUS content = QUERY_CONTENT_CONFIG
		tid:        "RADIUS Content Config",
		uri:        "/openconfig-system:system/aaa/server-groups/server-group[name=RADIUS]/servers/server[address=10.10.10.10]",
		requestUri: "/openconfig-system:system/aaa/server-groups/server-group[name=RADIUS]/servers/server[address=10.10.10.10]",
		payload: []byte(`
            {
               "openconfig-system:server" : [
                  {
                     "tacacs" : {
                        "config" : {
                           "encrypted" : true,
                           "secret-key" : ""
                        }
                     },
                     "radius" : {
                        "config" : {
                           "openconfig-aaa-radius-ext:encrypted" : true,
                           "secret-key" : "",
                           "retransmit-attempts" : 1
                        },
                        "state" : {
                           "retransmit-attempts" : 1,
                           "counters" : {
                              "access-rejects" : "1",
                              "access-accepts" : "2",
                              "openconfig-aaa-radius-ext:access-requests" : "3"
                           }
                        }
                     },
                     "config" : {
                        "priority" : 1,
                        "auth-type" : "chap",
                        "address" : "10.10.10.10"
                     },
                     "address" : "10.10.10.10",
                     "state" : {
                        "priority" : 1,
                        "auth-type" : "chap",
                        "address" : "10.10.10.10"
                     }
                  }
               ]
            }
            `),
		appRootType: reflect.TypeOf(ocbinds.OpenconfigSystem_System{}),
		queryParams: QueryParams{
			depthEnabled:  false,
			curDepth:      1,
			content:       QUERY_CONTENT_CONFIG,
			fields:        []string{},
			fieldsFillAll: false,
		},
		prunedPayload: []byte(`
            {
               "openconfig-system:server" : [
                  {
                     "tacacs" : {
                        "config" : {
                           "encrypted" : true,
                           "secret-key" : ""
                        }
                     },
                     "radius" : {
                        "config" : {
                           "openconfig-aaa-radius-ext:encrypted" : true,
                           "secret-key" : "",
                           "retransmit-attempts" : 1
                        }
                     },
                     "config" : {
                        "priority" : 1,
                        "auth-type" : "chap",
                        "address" : "10.10.10.10"
                     },
                     "address" : "10.10.10.10"
                  }
               ]
            }
            `),
	},
	{ // RADIUS content = QUERY_CONTENT_NONCONFIG
		tid:        "RADIUS Content Nonconfig",
		uri:        "/openconfig-system:system/aaa/server-groups/server-group[name=RADIUS]/servers/server[address=10.10.10.10]",
		requestUri: "/openconfig-system:system/aaa/server-groups/server-group[name=RADIUS]/servers/server[address=10.10.10.10]",
		payload: []byte(`
            {
               "openconfig-system:server" : [
                  {
                     "tacacs" : {
                        "config" : {
                           "encrypted" : true,
                           "secret-key" : ""
                        }
                     },
                     "radius" : {
                        "config" : {
                           "openconfig-aaa-radius-ext:encrypted" : true,
                           "secret-key" : "",
                           "retransmit-attempts" : 1
                        },
                        "state" : {
                           "retransmit-attempts" : 1,
                           "counters" : {
                              "access-rejects" : "1",
                              "access-accepts" : "2",
                              "openconfig-aaa-radius-ext:access-requests" : "3"
                           }
                        }
                     },
                     "config" : {
                        "priority" : 1,
                        "auth-type" : "chap",
                        "address" : "10.10.10.10"
                     },
                     "address" : "10.10.10.10",
                     "state" : {
                        "priority" : 1,
                        "auth-type" : "chap",
                        "address" : "10.10.10.10"
                     }
                  }
               ]
            }
            `),
		appRootType: reflect.TypeOf(ocbinds.OpenconfigSystem_System{}),
		queryParams: QueryParams{
			depthEnabled:  false,
			curDepth:      1,
			content:       QUERY_CONTENT_NONCONFIG,
			fields:        []string{},
			fieldsFillAll: false,
		},
		prunedPayload: []byte(`
            {
               "openconfig-system:server" : [
                  {
                     "tacacs" : {
                     },
                     "radius" : {
                        "state" : {
                           "retransmit-attempts" : 1,
                           "counters" : {
                              "access-rejects" : "1",
                              "access-accepts" : "2",
                              "openconfig-aaa-radius-ext:access-requests" : "3"
                           }
                        }
                     },
                     "address" : "10.10.10.10",
                     "state" : {
                        "priority" : 1,
                        "auth-type" : "chap",
                        "address" : "10.10.10.10"
                     }
                  }
               ]
            }
            `),
	},
	{ // RADIUS content = QUERY_CONTENT_OPERATIONAL
		tid:        "RADIUS Content Operational",
		uri:        "/openconfig-system:system/aaa/server-groups/server-group[name=RADIUS]/servers/server[address=10.10.10.10]",
		requestUri: "/openconfig-system:system/aaa/server-groups/server-group[name=RADIUS]/servers/server[address=10.10.10.10]",
		payload: []byte(`
            {
               "openconfig-system:server" : [
                  {
                     "tacacs" : {
                        "config" : {
                           "encrypted" : true,
                           "secret-key" : ""
                        }
                     },
                     "radius" : {
                        "config" : {
                           "openconfig-aaa-radius-ext:encrypted" : true,
                           "secret-key" : "",
                           "retransmit-attempts" : 1
                        },
                        "state" : {
                           "retransmit-attempts" : 1,
                           "counters" : {
                              "access-rejects" : "1",
                              "access-accepts" : "2",
                              "openconfig-aaa-radius-ext:access-requests" : "3"
                           }
                        }
                     },
                     "config" : {
                        "priority" : 1,
                        "auth-type" : "chap",
                        "address" : "10.10.10.10"
                     },
                     "address" : "10.10.10.10",
                     "state" : {
                        "priority" : 1,
                        "auth-type" : "chap",
                        "address" : "10.10.10.10"
                     }
                  }
               ]
            }
            `),
		appRootType: reflect.TypeOf(ocbinds.OpenconfigSystem_System{}),
		queryParams: QueryParams{
			depthEnabled:  false,
			curDepth:      1,
			content:       QUERY_CONTENT_OPERATIONAL,
			fields:        []string{},
			fieldsFillAll: false,
		},
		prunedPayload: []byte(`
            {
               "openconfig-system:server" : [
                  {
                     "tacacs" : {
                     },
                     "radius" : {
                        "state" : {
                           "counters" : {
                              "access-rejects" : "1",
                              "access-accepts" : "2",
                              "openconfig-aaa-radius-ext:access-requests" : "3"
                           }
                        }
                     },
                     "address" : "10.10.10.10",
                     "state" : {
                     }
                  }
               ]
            }
            `),
	},
	{ // RADIUS fields
		tid:        "RADIUS Fields",
		uri:        "/openconfig-system:system/aaa/server-groups/server-group[name=RADIUS]/servers/server[address=10.10.10.10]",
		requestUri: "/openconfig-system:system/aaa/server-groups/server-group[name=RADIUS]/servers/server[address=10.10.10.10]",
		payload: []byte(`
            {
               "openconfig-system:server" : [
                  {
                     "tacacs" : {
                        "config" : {
                           "encrypted" : true,
                           "secret-key" : ""
                        }
                     },
                     "radius" : {
                        "config" : {
                           "openconfig-aaa-radius-ext:encrypted" : true,
                           "secret-key" : "",
                           "retransmit-attempts" : 1
                        },
                        "state" : {
                           "retransmit-attempts" : 1,
                           "counters" : {
                              "access-rejects" : "1",
                              "access-accepts" : "2",
                              "openconfig-aaa-radius-ext:access-requests" : "3"
                           }
                        }
                     },
                     "config" : {
                        "priority" : 1,
                        "auth-type" : "chap",
                        "address" : "10.10.10.10"
                     },
                     "address" : "10.10.10.10",
                     "state" : {
                        "priority" : 1,
                        "auth-type" : "chap",
                        "address" : "10.10.10.10"
                     }
                  }
               ]
            }
            `),
		appRootType: reflect.TypeOf(ocbinds.OpenconfigSystem_System{}),
		queryParams: QueryParams{
			depthEnabled:  false,
			curDepth:      1,
			content:       QUERY_CONTENT_ALL,
			fields:        []string{"radius/state/counters/access-rejects", "address"},
			fieldsFillAll: false,
		},
		prunedPayload: []byte(`
            {
               "openconfig-system:server" : [
                  {
                     "radius" : {
                        "state" : {
                           "counters" : {
                              "access-rejects" : "1"
                           }
                        }
                     },
                     "address" : "10.10.10.10"
                  }
               ]
            }
            `),
	},
}
