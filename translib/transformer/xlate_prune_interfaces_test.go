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
//go:build interfaces_prune_xfmrtest
// +build interfaces_prune_xfmrtest

package transformer

import (
	"reflect"

	"github.com/Azure/sonic-mgmt-common/translib/ocbinds"
)

func init() {
	xp_tests = append(xp_tests, interfaces_xp_tests...)
}

var interfaces_xp_tests = []xpTests{
	{ // Interface Identity Test Case (i.e. no QP)
		tid:        "Interface Identity",
		uri:        "/openconfig-interfaces:interfaces/interface[name=Ethernet0]",
		requestUri: "/openconfig-interfaces:interfaces/interface[name=Ethernet0]",
		payload: []byte(`
            {
               "openconfig-interfaces:interface" : [
                  {
                     "openconfig-if-ethernet:ethernet" : {
                        "config" : {
                           "port-speed" : "openconfig-if-ethernet:SPEED_100GB"
                        }
                     },
                     "subinterfaces" : {
                        "subinterface" : [
                           {
                              "openconfig-if-ip:ipv6" : {
                                 "config" : {
                                    "enabled" : false
                                 },
                                 "state" : {
                                    "enabled" : false
                                 }
                              },
                              "index" : 0,
                              "config" : {
                                 "index" : 0
                              },
                              "state" : {
                                 "index" : 0
                              }
                           }
                        ]
                     },
                     "name" : "Ethernet0",
                     "config" : {
                        "name" : "Ethernet0",
                        "type" : "iana-if-type:ethernetCsmacd",
                        "enabled" : true
                     },
                     "state" : {
                        "oper-status" : "DOWN",
                        "mac-address" : "50:6b:8d:16:6e:8b",
                        "name" : "Ethernet0"
                     }
                  }
               ]
            }
            `),
		appRootType: reflect.TypeOf(ocbinds.OpenconfigInterfaces_Interfaces{}),
		queryParams: QueryParams{
			depthEnabled:  false,
			curDepth:      1,
			content:       QUERY_CONTENT_ALL,
			fields:        []string{},
			fieldsFillAll: false,
		},
		prunedPayload: []byte(`
            {
               "openconfig-interfaces:interface" : [
                  {
                     "openconfig-if-ethernet:ethernet" : {
                        "config" : {
                           "port-speed" : "openconfig-if-ethernet:SPEED_100GB"
                        }
                     },
                     "subinterfaces" : {
                        "subinterface" : [
                           {
                              "openconfig-if-ip:ipv6" : {
                                 "config" : {
                                    "enabled" : false
                                 },
                                 "state" : {
                                    "enabled" : false
                                 }
                              },
                              "index" : 0,
                              "config" : {
                                 "index" : 0
                              },
                              "state" : {
                                 "index" : 0
                              }
                           }
                        ]
                     },
                     "name" : "Ethernet0",
                     "config" : {
                        "name" : "Ethernet0",
                        "type" : "iana-if-type:ethernetCsmacd",
                        "enabled" : true
                     },
                     "state" : {
                        "oper-status" : "DOWN",
                        "mac-address" : "50:6b:8d:16:6e:8b",
                        "name" : "Ethernet0"
                     }
                  }
               ]
            }
            `),
	},
	{ // Interface depth = 3
		tid:        "Interface Depth",
		uri:        "/openconfig-interfaces:interfaces/interface[name=Ethernet0]",
		requestUri: "/openconfig-interfaces:interfaces/interface[name=Ethernet0]",
		payload: []byte(`
            {
               "openconfig-interfaces:interface" : [
                  {
                     "openconfig-if-ethernet:ethernet" : {
                        "config" : {
                           "port-speed" : "openconfig-if-ethernet:SPEED_100GB"
                        }
                     },
                     "subinterfaces" : {
                        "subinterface" : [
                           {
                              "openconfig-if-ip:ipv6" : {
                                 "config" : {
                                    "enabled" : false
                                 },
                                 "state" : {
                                    "enabled" : false
                                 }
                              },
                              "index" : 0,
                              "config" : {
                                 "index" : 0
                              },
                              "state" : {
                                 "index" : 0
                              }
                           }
                        ]
                     },
                     "name" : "Ethernet0",
                     "config" : {
                        "name" : "Ethernet0",
                        "type" : "iana-if-type:ethernetCsmacd",
                        "enabled" : true
                     },
                     "state" : {
                        "oper-status" : "DOWN",
                        "mac-address" : "50:6b:8d:16:6e:8b",
                        "name" : "Ethernet0"
                     }
                  }
               ]
            }
            `),
		appRootType: reflect.TypeOf(ocbinds.OpenconfigInterfaces_Interfaces{}),
		queryParams: QueryParams{
			depthEnabled:  true,
			curDepth:      3,
			content:       QUERY_CONTENT_ALL,
			fields:        []string{},
			fieldsFillAll: false,
		},
		prunedPayload: []byte(`
            {
               "openconfig-interfaces:interface" : [
                  {
                     "openconfig-if-ethernet:ethernet" : {
                        "config" : {
                        }
                     },
                     "subinterfaces" : {
                        "subinterface" : [
                        ]
                     },
                     "name" : "Ethernet0",
                     "config" : {
                        "name" : "Ethernet0",
                        "type" : "iana-if-type:ethernetCsmacd",
                        "enabled" : true
                     },
                     "state" : {
                        "oper-status" : "DOWN",
                        "mac-address" : "50:6b:8d:16:6e:8b",
                        "name" : "Ethernet0"
                     }
                  }
               ]
            }
            `),
	},
	{ // Interface Content QUERY_CONTENT_CONFIG
		tid:        "Interface Content Config",
		uri:        "/openconfig-interfaces:interfaces/interface[name=Ethernet0]",
		requestUri: "/openconfig-interfaces:interfaces/interface[name=Ethernet0]",
		payload: []byte(`
            {
               "openconfig-interfaces:interface" : [
                  {
                     "openconfig-if-ethernet:ethernet" : {
                        "config" : {
                           "port-speed" : "openconfig-if-ethernet:SPEED_100GB"
                        }
                     },
                     "subinterfaces" : {
                        "subinterface" : [
                           {
                              "openconfig-if-ip:ipv6" : {
                                 "config" : {
                                    "enabled" : false
                                 },
                                 "state" : {
                                    "enabled" : false
                                 }
                              },
                              "index" : 0,
                              "config" : {
                                 "index" : 0
                              },
                              "state" : {
                                 "index" : 0
                              }
                           }
                        ]
                     },
                     "name" : "Ethernet0",
                     "config" : {
                        "name" : "Ethernet0",
                        "type" : "iana-if-type:ethernetCsmacd",
                        "enabled" : true
                     },
                     "state" : {
                        "oper-status" : "DOWN",
                        "mac-address" : "50:6b:8d:16:6e:8b",
                        "name" : "Ethernet0"
                     }
                  }
               ]
            }
            `),
		appRootType: reflect.TypeOf(ocbinds.OpenconfigInterfaces_Interfaces{}),
		queryParams: QueryParams{
			depthEnabled:  false,
			curDepth:      1,
			content:       QUERY_CONTENT_CONFIG,
			fields:        []string{},
			fieldsFillAll: false,
		},
		prunedPayload: []byte(`
            {
               "openconfig-interfaces:interface" : [
                  {
                     "openconfig-if-ethernet:ethernet" : {
                        "config" : {
                           "port-speed" : "openconfig-if-ethernet:SPEED_100GB"
                        }
                     },
                     "subinterfaces" : {
                        "subinterface" : [
                           {
                              "openconfig-if-ip:ipv6" : {
                                 "config" : {
                                    "enabled" : false
                                 }
                              },
                              "index" : 0,
                              "config" : {
                                 "index" : 0
                              }
                           }
                        ]
                     },
                     "name" : "Ethernet0",
                     "config" : {
                        "name" : "Ethernet0",
                        "type" : "iana-if-type:ethernetCsmacd",
                        "enabled" : true
                     }
                  }
               ]
            }
            `),
	},
	{ // Interface Content QUERY_CONTENT_NONCONFIG
		tid:        "Interface Content Nonconfig",
		uri:        "/openconfig-interfaces:interfaces/interface[name=Ethernet0]",
		requestUri: "/openconfig-interfaces:interfaces/interface[name=Ethernet0]",
		payload: []byte(`
            {
               "openconfig-interfaces:interface" : [
                  {
                     "openconfig-if-ethernet:ethernet" : {
                        "config" : {
                           "port-speed" : "openconfig-if-ethernet:SPEED_100GB"
                        }
                     },
                     "subinterfaces" : {
                        "subinterface" : [
                           {
                              "openconfig-if-ip:ipv6" : {
                                 "config" : {
                                    "enabled" : false
                                 },
                                 "state" : {
                                    "enabled" : false
                                 }
                              },
                              "index" : 0,
                              "config" : {
                                 "index" : 0
                              },
                              "state" : {
                                 "index" : 0
                              }
                           }
                        ]
                     },
                     "name" : "Ethernet0",
                     "config" : {
                        "name" : "Ethernet0",
                        "type" : "iana-if-type:ethernetCsmacd",
                        "enabled" : true
                     },
                     "state" : {
                        "oper-status" : "DOWN",
                        "mac-address" : "50:6b:8d:16:6e:8b",
                        "name" : "Ethernet0"
                     }
                  }
               ]
            }
            `),
		appRootType: reflect.TypeOf(ocbinds.OpenconfigInterfaces_Interfaces{}),
		queryParams: QueryParams{
			depthEnabled:  false,
			curDepth:      1,
			content:       QUERY_CONTENT_NONCONFIG,
			fields:        []string{},
			fieldsFillAll: false,
		},
		prunedPayload: []byte(`
            {
               "openconfig-interfaces:interface" : [
                  {
                     "openconfig-if-ethernet:ethernet" : {
                     },
                     "subinterfaces" : {
                        "subinterface" : [
                           {
                              "openconfig-if-ip:ipv6" : {
                                 "state" : {
                                    "enabled" : false
                                 }
                              },
                              "index" : 0,
                              "state" : {
                                 "index" : 0
                              }
                           }
                        ]
                     },
                     "name" : "Ethernet0",
                     "state" : {
                        "oper-status" : "DOWN",
                        "mac-address" : "50:6b:8d:16:6e:8b",
                        "name" : "Ethernet0"
                     }
                  }
               ]
            }
            `),
	},
	{ // Interface fields
		tid:        "Interface fields",
		uri:        "/openconfig-interfaces:interfaces/interface[name=Ethernet0]",
		requestUri: "/openconfig-interfaces:interfaces/interface[name=Ethernet0]",
		payload: []byte(`
            {
               "openconfig-interfaces:interface" : [
                  {
                     "openconfig-if-ethernet:ethernet" : {
                        "config" : {
                           "port-speed" : "openconfig-if-ethernet:SPEED_100GB"
                        }
                     },
                     "subinterfaces" : {
                        "subinterface" : [
                           {
                              "openconfig-if-ip:ipv6" : {
                                 "config" : {
                                    "enabled" : false
                                 },
                                 "state" : {
                                    "enabled" : false
                                 }
                              },
                              "index" : 0,
                              "config" : {
                                 "index" : 0
                              },
                              "state" : {
                                 "index" : 0
                              }
                           }
                        ]
                     },
                     "name" : "Ethernet0",
                     "config" : {
                        "name" : "Ethernet0",
                        "type" : "iana-if-type:ethernetCsmacd",
                        "enabled" : true
                     },
                     "state" : {
                        "oper-status" : "DOWN",
                        "mac-address" : "50:6b:8d:16:6e:8b",
                        "name" : "Ethernet0"
                     }
                  }
               ]
            }
            `),
		appRootType: reflect.TypeOf(ocbinds.OpenconfigInterfaces_Interfaces{}),
		queryParams: QueryParams{
			depthEnabled:  false,
			curDepth:      1,
			content:       QUERY_CONTENT_ALL,
			fields:        []string{"ethernet/config/port-speed", "name"},
			fieldsFillAll: false,
		},
		prunedPayload: []byte(`
            {
               "openconfig-interfaces:interface" : [
                  {
                     "openconfig-if-ethernet:ethernet" : {
                        "config" : {
                           "port-speed" : "openconfig-if-ethernet:SPEED_100GB"
                        }
                     },
                     "name" : "Ethernet0"
                  }
               ]
            }
            `),
	},
	{ // Interface depth = 3, content = CONFIG
		tid:        "Interface Depth and Content",
		uri:        "/openconfig-interfaces:interfaces/interface[name=Ethernet0]",
		requestUri: "/openconfig-interfaces:interfaces/interface[name=Ethernet0]",
		payload: []byte(`
            {
               "openconfig-interfaces:interface" : [
                  {
                     "openconfig-if-ethernet:ethernet" : {
                        "config" : {
                           "port-speed" : "openconfig-if-ethernet:SPEED_100GB"
                        }
                     },
                     "subinterfaces" : {
                        "subinterface" : [
                           {
                              "openconfig-if-ip:ipv6" : {
                                 "config" : {
                                    "enabled" : false
                                 },
                                 "state" : {
                                    "enabled" : false
                                 }
                              },
                              "index" : 0,
                              "config" : {
                                 "index" : 0
                              },
                              "state" : {
                                 "index" : 0
                              }
                           }
                        ]
                     },
                     "name" : "Ethernet0",
                     "config" : {
                        "name" : "Ethernet0",
                        "type" : "iana-if-type:ethernetCsmacd",
                        "enabled" : true
                     },
                     "state" : {
                        "oper-status" : "DOWN",
                        "mac-address" : "50:6b:8d:16:6e:8b",
                        "name" : "Ethernet0"
                     }
                  }
               ]
            }
            `),
		appRootType: reflect.TypeOf(ocbinds.OpenconfigInterfaces_Interfaces{}),
		queryParams: QueryParams{
			depthEnabled:  true,
			curDepth:      3,
			content:       QUERY_CONTENT_CONFIG,
			fields:        []string{},
			fieldsFillAll: false,
		},
		prunedPayload: []byte(`
            {
               "openconfig-interfaces:interface" : [
                  {
                     "openconfig-if-ethernet:ethernet" : {
                        "config" : {
                        }
                     },
                     "subinterfaces" : {
                        "subinterface" : [
                        ]
                     },
                     "name" : "Ethernet0",
                     "config" : {
                        "name" : "Ethernet0",
                        "type" : "iana-if-type:ethernetCsmacd",
                        "enabled" : true
                     }
                  }
               ]
            }
            `),
	},
}
