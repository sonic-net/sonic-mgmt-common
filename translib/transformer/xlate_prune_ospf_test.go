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
//go:build ospf_prune_xfmrtest
// +build ospf_prune_xfmrtest

package transformer

import (
	"reflect"

	"github.com/Azure/sonic-mgmt-common/translib/ocbinds"
)

func init() {
	xp_tests = append(xp_tests, ni_ospf_tests...)
}

var ni_ospf_tests = []xpTests{
	{ // OSPF Identity Test Case (i.e. no QP)
		tid:        "OSPF Identity uri == requestUri",
		uri:        "/openconfig-network-instance:network-instances/network-instance[name=default]/protocols/protocol[identifier=OSPF][name=ospfv2]",
		requestUri: "/openconfig-network-instance:network-instances/network-instance[name=default]/protocols/protocol[identifier=OSPF][name=ospfv2]",
		payload: []byte(`
{
   "protocol" : [
      {
         "identifier" : "openconfig-policy-types:OSPF",
         "ospfv2" : {
            "openconfig-ospfv2-ext:route-tables" : {
               "route-table" : [
                  {
                     "routes" : {
                        "route" : [
                           {
                              "next-hops" : {
                                 "next-hop" : [
                                    {
                                       "area-id" : "0.0.0.0",
                                       "address" : "32.1.0.1",
                                       "state" : {
                                          "area-id" : "0.0.0.0",
                                          "address" : "32.1.0.1",
                                          "out-interface" : "Vlan3719"
                                       },
                                       "out-interface" : "Vlan3719"
                                    }
                                 ]
                              },
                              "state" : {
                                 "cost" : 4,
                                 "sub-type" : "openconfig-ospfv2-ext:EXTERNAL_ROUTE_TYPE_2",
                                 "type" : "openconfig-ospfv2-ext:EXTERNAL_ROUTE",
                                 "prefix" : "10.193.80.0/20"
                              },
                              "prefix" : "10.193.80.0/20"
                           }
                        ]
                     },
                     "type" : "openconfig-ospfv2-ext:EXTERNAL_ROUTE_TABLE",
                     "state" : {
                        "type" : "openconfig-ospfv2-ext:EXTERNAL_ROUTE_TABLE"
                     }
                  },
                  {
                     "routes" : {
                        "route" : [
                           {
                              "next-hops" : {
                                 "next-hop" : [
                                    {
                                       "area-id" : "0.0.0.0",
                                       "address" : "32.1.0.1",
                                       "state" : {
                                          "area-id" : "0.0.0.0",
                                          "address" : "32.1.0.1",
                                          "out-interface" : "Vlan3719"
                                       },
                                       "out-interface" : "Vlan3719"
                                    }
                                 ]
                              },
                              "state" : {
                                 "cost" : 14,
                                 "type" : "openconfig-ospfv2-ext:NETWORK_ROUTE",
                                 "inter-area" : true,
                                 "prefix" : "10.193.80.0/20"
                              },
                              "prefix" : "21.1.0.0/24"
                           },
                           {
                              "next-hops" : {
                                 "next-hop" : [
                                    {
                                       "area-id" : "0.0.0.0",
                                       "address" : "0.0.0.0",
                                       "state" : {
                                          "area-id" : "0.0.0.0",
                                          "address" : "32.1.0.1",
                                          "out-interface" : "Vlan3719"
                                       },
                                       "out-interface" : "Vlan3719"
                                    }
                                 ]
                              },
                              "state" : {
                                 "cost" : 4,
                                 "type" : "openconfig-ospfv2-ext:NETWORK_ROUTE",
                                 "prefix" : "10.193.80.0/20"
                              },
                              "prefix" : "32.1.0.0/24"
                           },
                           {
                              "next-hops" : {
                                 "next-hop" : [
                                    {
                                       "area-id" : "0.0.0.2",
                                       "address" : "0.0.0.0",
                                       "state" : {
                                          "area-id" : "0.0.0.0",
                                          "address" : "32.1.0.1",
                                          "out-interface" : "Vlan3719"
                                       },
                                       "out-interface" : "Ethernet120"
                                    }
                                 ]
                              },
                              "state" : {
                                 "cost" : 10,
                                 "type" : "openconfig-ospfv2-ext:NETWORK_ROUTE",
                                 "prefix" : "10.193.80.0/20"
                              },
                              "prefix" : "41.1.0.0/24"
                           }
                        ]
                     },
                     "type" : "openconfig-ospfv2-ext:NETWORK_ROUTE_TABLE",
                     "state" : {
                        "type" : "openconfig-ospfv2-ext:NETWORK_ROUTE_TABLE"
                     }
                  },
                  {
                     "routes" : {
                        "route" : [
                           {
                              "next-hops" : {
                                 "next-hop" : [
                                    {
                                       "area-id" : "0.0.0.0",
                                       "address" : "32.1.0.1",
                                       "state" : {
                                          "area-id" : "0.0.0.0",
                                          "address" : "32.1.0.1",
                                          "out-interface" : "Vlan3719"
                                       },
                                       "out-interface" : "Vlan3719"
                                    }
                                 ]
                              },
                              "state" : {
                                 "router-type" : "openconfig-ospfv2-ext:ABRASBR",
                                 "cost" : 4,
                                 "type" : "openconfig-ospfv2-ext:ROUTER_ROUTE",
                                 "prefix" : "10.193.80.0/20"
                              },
                              "prefix" : "6.6.6.5"
                           }
                        ]
                     },
                     "type" : "openconfig-ospfv2-ext:ROUTER_ROUTE_TABLE",
                     "state" : {
                        "type" : "openconfig-ospfv2-ext:ROUTER_ROUTE_TABLE"
                     }
                  }
               ]
            },
            "global" : {
               "graceful-restart" : {
                  "openconfig-ospfv2-ext:helpers" : {
                     "helper" : [
                        {
                           "neighbour-id" : "5.5.5.4",
                           "config" : {
                              "vrf-name" : "default",
                              "neighbour-id" : "5.5.5.4"
                           },
                           "state" : {
                              "neighbour-id" : "5.5.5.4"
                           }
                        }
                     ]
                  },
                  "config" : {
                     "openconfig-ospfv2-ext:planned-only" : true,
                     "helper-only" : true,
                     "openconfig-ospfv2-ext:supported-grace-time" : 800,
                     "openconfig-ospfv2-ext:strict-lsa-checking" : true
                  },
                  "state" : {
                     "openconfig-ospfv2-ext:planned-only" : true,
                     "helper-only" : true,
                     "openconfig-ospfv2-ext:grace-period" : 0,
                     "openconfig-ospfv2-ext:supported-grace-time" : 800,
                     "openconfig-ospfv2-ext:strict-lsa-checking" : true
                  }
               },
               "inter-area-propagation-policies" : {
                  "openconfig-ospfv2-ext:inter-area-policy" : [
                     {
                        "config" : {
                           "src-area" : "0.0.0.0"
                        },
                        "src-area" : "0.0.0.0",
                        "state" : {
                           "src-area" : "0.0.0.0"
                        }
                     },
                     {
                        "config" : {
                           "src-area" : "0.0.0.2"
                        },
                        "src-area" : "0.0.0.2",
                        "state" : {
                           "src-area" : "0.0.0.2"
                        }
                     }
                  ]
               },
               "config" : {
                  "router-id" : "5.5.5.4",
                  "openconfig-ospfv2-ext:opaque-lsa-capability" : true,
                  "openconfig-ospfv2-ext:description" : "OSPFv2 Router",
                  "openconfig-ospfv2-ext:enable" : true
               },
               "timers" : {
                  "lsa-generation" : {
                     "state" : {
                        "openconfig-ospfv2-ext:refresh-timer" : 10000,
                        "openconfig-ospfv2-ext:lsa-min-interval-timer" : 5000,
                        "openconfig-ospfv2-ext:lsa-min-arrival-timer" : 1000
                     }
                  },
                  "spf" : {
                     "state" : {
                        "maximum-delay" : 5000,
                        "initial-delay" : 50,
                        "openconfig-ospfv2-ext:throttle-delay" : 0,
                        "openconfig-ospfv2-ext:spf-timer-due" : 0
                     }
                  }
               },
               "state" : {
                  "openconfig-ospfv2-ext:area-count" : 2,
                  "openconfig-ospfv2-ext:last-spf-duration" : 358,
                  "openconfig-ospfv2-ext:hold-time-multiplier" : 1,
                  "openconfig-ospfv2-ext:last-spf-execution-time" : "1187616",
                  "openconfig-ospfv2-ext:opaque-lsa-count" : 0,
                  "router-id" : "5.5.5.4",
                  "openconfig-ospfv2-ext:abr-type" : "openconfig-ospfv2-ext:CISCO",
                  "openconfig-ospfv2-ext:external-lsa-checksum" : "0x00016bcc",
                  "openconfig-ospfv2-ext:opaque-lsa-checksum" : "0x00000000",
                  "openconfig-ospfv2-ext:write-multiplier" : 20,
                  "openconfig-ospfv2-ext:opaque-lsa-capability" : true,
                  "openconfig-ospfv2-ext:external-lsa-count" : 2
               }
            },
            "areas" : {
               "area" : [
                  {
                     "lsdb" : {
                        "lsa-types" : {
                           "lsa-type" : [
                              {
                                 "type" : "openconfig-ospf-types:AS_EXTERNAL_LSA",
                                 "state" : {
                                    "type" : "openconfig-ospf-types:AS_EXTERNAL_LSA"
                                 },
                                 "lsas" : {
                                    "openconfig-ospfv2-ext:lsa-ext" : [
                                       {
                                          "advertising-router" : "5.5.5.4",
                                          "as-external-lsa" : {
                                             "types-of-service" : {
                                                "type-of-service" : [
                                                   {
                                                      "tos" : 0,
                                                      "state" : {
                                                         "forwarding-address" : "0.0.0.0",
                                                         "external-route-tag" : 0,
                                                         "metric" : 20
                                                      }
                                                   }
                                                ]
                                             },
                                             "state" : {
                                                "metric-type" : "TYPE_2",
                                                "mask" : 20
                                             }
                                          },
                                          "link-state-id" : "10.193.80.0",
                                          "state" : {
                                             "checksum" : 50393,
                                             "length" : 36,
                                             "flags" : 11,
                                             "display-sequence-number" : "80000001",
                                             "option" : 2,
                                             "option-expanded" : "*||||||E|",
                                             "age" : 1227
                                          }
                                       },
                                       {
                                          "advertising-router" : "6.6.6.5",
                                          "as-external-lsa" : {
                                             "types-of-service" : {
                                                "type-of-service" : [
                                                   {
                                                      "tos" : 0,
                                                      "state" : {
                                                         "forwarding-address" : "0.0.0.0",
                                                         "external-route-tag" : 0,
                                                         "metric" : 20
                                                      }
                                                   }
                                                ]
                                             },
                                             "state" : {
                                                "metric-type" : "TYPE_2",
                                                "mask" : 20
                                             }
                                          },
                                          "link-state-id" : "10.193.80.0",
                                          "state" : {
                                             "checksum" : 42739,
                                             "length" : 36,
                                             "flags" : 6,
                                             "display-sequence-number" : "80000001",
                                             "option" : 2,
                                             "option-expanded" : "*||||||E|",
                                             "age" : 1228
                                          }
                                       }
                                    ]
                                 }
                              },
                              {
                                 "type" : "openconfig-ospf-types:NETWORK_LSA",
                                 "state" : {
                                    "type" : "openconfig-ospf-types:NETWORK_LSA"
                                 },
                                 "lsas" : {
                                    "openconfig-ospfv2-ext:lsa-ext" : [
                                       {
                                          "advertising-router" : "6.6.6.5",
                                          "link-state-id" : "32.1.0.1",
                                          "state" : {
                                             "checksum" : 51756,
                                             "length" : 32,
                                             "flags" : 6,
                                             "display-sequence-number" : "80000001",
                                             "option" : 2,
                                             "option-expanded" : "*|-|-|-|-|-|E|-",
                                             "age" : 1198
                                          },
                                          "network-lsa" : {
                                             "state" : {
                                                "network-mask" : 24
                                             }
                                          }
                                       }
                                    ]
                                 }
                              },
                              {
                                 "type" : "openconfig-ospf-types:OSPFV2_LINK_SCOPE_OPAQUE_LSA"
                              },
                              {
                                 "type" : "openconfig-ospf-types:ROUTER_LSA",
                                 "state" : {
                                    "type" : "openconfig-ospf-types:ROUTER_LSA"
                                 },
                                 "lsas" : {
                                    "openconfig-ospfv2-ext:lsa-ext" : [
                                       {
                                          "advertising-router" : "5.5.5.4",
                                          "link-state-id" : "5.5.5.4",
                                          "state" : {
                                             "checksum" : 5321,
                                             "length" : 36,
                                             "flags" : 3,
                                             "display-sequence-number" : "80000005",
                                             "option" : 2,
                                             "option-expanded" : "*||||||E|",
                                             "age" : 1197
                                          },
                                          "router-lsa" : {
                                             "link-informations" : {
                                                "link-information" : [
                                                   {
                                                      "state" : {
                                                         "metric" : 4,
                                                         "type" : "openconfig-ospf-types:ROUTER_LSA_TRANSIT_NETWORK",
                                                         "number-tos-metrics" : 0
                                                      }
                                                   }
                                                ]
                                             },
                                             "types-of-service" : {
                                                "type-of-service" : [
                                                   {
                                                      "tos" : 0,
                                                      "state" : {
                                                         "metric" : 4
                                                      }
                                                   }
                                                ]
                                             },
                                             "state" : {
                                                "flags-description" : " : ABR ASBR",
                                                "flags" : 3,
                                                "number-links" : 1
                                             }
                                          }
                                       },
                                       {
                                          "advertising-router" : "6.6.6.5",
                                          "link-state-id" : "6.6.6.5",
                                          "state" : {
                                             "checksum" : 47900,
                                             "length" : 36,
                                             "flags" : 6,
                                             "display-sequence-number" : "80000004",
                                             "option" : 2,
                                             "option-expanded" : "*||||||E|",
                                             "age" : 1197
                                          },
                                          "router-lsa" : {
                                             "link-informations" : {
                                                "link-information" : [
                                                   {
                                                      "state" : {
                                                         "metric" : 4,
                                                         "type" : "openconfig-ospf-types:ROUTER_LSA_TRANSIT_NETWORK",
                                                         "number-tos-metrics" : 0
                                                      }
                                                   }
                                                ]
                                             },
                                             "types-of-service" : {
                                                "type-of-service" : [
                                                   {
                                                      "tos" : 0,
                                                      "state" : {
                                                         "metric" : 4
                                                      }
                                                   }
                                                ]
                                             },
                                             "state" : {
                                                "flags-description" : " : ABR ASBR",
                                                "flags" : 3,
                                                "number-links" : 1
                                             }
                                          }
                                       }
                                    ]
                                 }
                              },
                              {
                                 "type" : "openconfig-ospf-types:SUMMARY_ASBR_LSA",
                                 "state" : {
                                    "type" : "openconfig-ospf-types:SUMMARY_ASBR_LSA"
                                 }
                              },
                              {
                                 "type" : "openconfig-ospf-types:SUMMARY_IP_NETWORK_LSA",
                                 "state" : {
                                    "type" : "openconfig-ospf-types:SUMMARY_IP_NETWORK_LSA"
                                 },
                                 "lsas" : {
                                    "openconfig-ospfv2-ext:lsa-ext" : [
                                       {
                                          "advertising-router" : "6.6.6.5",
                                          "link-state-id" : "21.1.0.0",
                                          "summary-lsa" : {
                                             "types-of-service" : {
                                                "type-of-service" : [
                                                   {
                                                      "tos" : 0,
                                                      "state" : {
                                                         "metric" : 10
                                                      }
                                                   }
                                                ]
                                             },
                                             "state" : {
                                                "network-mask" : 24
                                             }
                                          },
                                          "state" : {
                                             "checksum" : 803,
                                             "length" : 28,
                                             "flags" : 6,
                                             "display-sequence-number" : "80000001",
                                             "option" : 2,
                                             "option-expanded" : "*|-|-|-|-|-|E|-",
                                             "age" : 1228
                                          }
                                       },
                                       {
                                          "advertising-router" : "5.5.5.4",
                                          "link-state-id" : "41.1.0.0",
                                          "summary-lsa" : {
                                             "types-of-service" : {
                                                "type-of-service" : [
                                                   {
                                                      "tos" : 0,
                                                      "state" : {
                                                         "metric" : 10
                                                      }
                                                   }
                                                ]
                                             },
                                             "state" : {
                                                "network-mask" : 24
                                             }
                                          },
                                          "state" : {
                                             "checksum" : 7417,
                                             "length" : 28,
                                             "flags" : 11,
                                             "display-sequence-number" : "80000001",
                                             "option" : 2,
                                             "option-expanded" : "*|-|-|-|-|-|E|-",
                                             "age" : 1227
                                          }
                                       }
                                    ]
                                 }
                              }
                           ]
                        },
                        "state" : {
                           "identifier" : "0.0.0.0"
                        }
                     },
                     "identifier" : "0.0.0.0",
                     "interfaces" : {
                        "interface" : [
                           {
                              "openconfig-ospfv2-ext:message-statistics" : {
                                 "state" : {
                                    "ls-update-transmit" : 4,
                                    "ls-request-receive" : 1,
                                    "ls-acknowledge-receive" : 2,
                                    "ls-acknowledge-transmit" : 2,
                                    "db-description-transmit" : 3,
                                    "db-description-receive" : 2,
                                    "hello-transmit" : 124,
                                    "hello-receive" : 124,
                                    "ls-request-transmit" : 1,
                                    "ls-update-receive" : 3
                                 }
                              },
                              "openconfig-ospfv2-ext:neighbours" : {
                                 "neighbour" : [
                                    {
                                       "neighbor-id" : "6.6.6.5",
                                       "neighbor-address" : "32.1.0.1",
                                       "state" : {
                                          "priority" : 1,
                                          "option-value" : 66,
                                          "gr-helper-status" : "None",
                                          "state-changes" : 6,
                                          "backup-designated-router" : "32.1.0.2",
                                          "adjacency-state" : "openconfig-ospf-types:FULL",
                                          "neighbor-id" : "6.6.6.5",
                                          "optional-capabilities" : "*|O|-|-|-|-|E|-",
                                          "thread-inactivity-timer" : true,
                                          "thread-ls-request-retransmission" : true,
                                          "thread-ls-update-retransmission" : true,
                                          "retransmit-summary-queue-length" : 0,
                                          "area-id" : "0.0.0.0",
                                          "gr-active-helper" : false,
                                          "interface-address" : "32.1.0.2",
                                          "neighbor-address" : "32.1.0.1",
                                          "last-established-time" : "1197595",
                                          "designated-router" : "32.1.0.1",
                                          "database-summary-queue-length" : 0,
                                          "interface-name" : "Vlan3719",
                                          "link-state-request-queue-length" : 0,
                                          "dead-time" : "32410"
                                       }
                                    }
                                 ]
                              },
                              "id" : "Vlan3719",
                              "timers" : {
                                 "state" : {
                                    "retransmission-interval" : 5,
                                    "dead-interval" : 40,
                                    "openconfig-ospfv2-ext:wait-time" : 40,
                                    "hello-interval" : 10000,
                                    "openconfig-ospfv2-ext:hello-due" : 2409
                                 }
                              },
                              "state" : {
                                 "priority" : 1,
                                 "openconfig-ospfv2-ext:member-of-ospf-all-routers" : true,
                                 "openconfig-ospfv2-ext:index" : 310,
                                 "openconfig-ospfv2-ext:mtu" : 9100,
                                 "openconfig-ospfv2-ext:if-flags" : "<UP,BROADCAST,RUNNING,MULTICAST>",
                                 "openconfig-ospfv2-ext:adjacency-count" : 1,
                                 "openconfig-ospfv2-ext:address-len" : 24,
                                 "openconfig-ospfv2-ext:ospf-enable" : true,
                                 "openconfig-ospfv2-ext:cost" : 4,
                                 "openconfig-ospfv2-ext:address" : "32.1.0.2",
                                 "id" : "Vlan3719",
                                 "openconfig-ospfv2-ext:adjacency-status" : "Backup",
                                 "openconfig-ospfv2-ext:neighbor-count" : 1,
                                 "openconfig-ospfv2-ext:backup-designated-router-id" : "5.5.5.4",
                                 "openconfig-ospfv2-ext:member-of-ospf-designated-routers" : true,
                                 "openconfig-ospfv2-ext:router-id" : "5.5.5.4",
                                 "openconfig-ospfv2-ext:broadcast-address" : "32.1.0.255",
                                 "openconfig-ospfv2-ext:bandwidth" : 25000,
                                 "openconfig-ospfv2-ext:area-id" : "0.0.0.0",
                                 "openconfig-ospfv2-ext:ospf-interface-type" : "Broadcast",
                                 "openconfig-ospfv2-ext:backup-designated-router-address" : "32.1.0.2",
                                 "network-type" : "openconfig-ospf-types:BROADCAST_NETWORK",
                                 "openconfig-ospfv2-ext:transmit-delay" : 1,
                                 "openconfig-ospfv2-ext:operational-state" : "Up"
                              }
                           }
                        ]
                     },
                     "config" : {
                        "identifier" : "0.0.0.0"
                     },
                     "openconfig-ospfv2-ext:networks" : {
                        "network" : [
                           {
                              "address-prefix" : "31.1.0.0/24",
                              "config" : {
                                 "address-prefix" : "31.1.0.0/24",
                                 "description" : "Network prefix"
                              },
                              "state" : {
                                 "address-prefix" : "31.1.0.0/24",
                                 "description" : "Network prefix"
                              }
                           },
                           {
                              "address-prefix" : "32.1.0.0/24",
                              "config" : {
                                 "address-prefix" : "32.1.0.0/24",
                                 "description" : "Network prefix"
                              },
                              "state" : {
                                 "address-prefix" : "32.1.0.0/24",
                                 "description" : "Network prefix"
                              }
                           }
                        ]
                     },
                     "state" : {
                        "openconfig-ospfv2-ext:summary-lsa-count" : 2,
                        "openconfig-ospfv2-ext:nssa-lsa-checksum" : "0x00000000",
                        "openconfig-ospfv2-ext:authentication-type" : "no",
                        "openconfig-ospfv2-ext:asbr-summary-lsa-count" : 0,
                        "openconfig-ospfv2-ext:adjacency-count" : 1,
                        "openconfig-ospfv2-ext:opaque-area-lsa-count" : 0,
                        "openconfig-ospfv2-ext:opaque-area-lsa-checksum" : "0x00000000",
                        "openconfig-ospfv2-ext:lsa-count" : 5,
                        "openconfig-ospfv2-ext:opaque-link-lsa-checksum" : "0x00000000",
                        "openconfig-ospfv2-ext:spf-execution-count" : 6,
                        "openconfig-ospfv2-ext:interface-count" : 1,
                        "openconfig-ospfv2-ext:network-lsa-count" : 1,
                        "openconfig-ospfv2-ext:asbr-summary-lsa-checksum" : "0x00000000",
                        "openconfig-ospfv2-ext:router-lsa-checksum" : "0x0000cfe5",
                        "openconfig-ospfv2-ext:router-lsa-count" : 2,
                        "openconfig-ospfv2-ext:nssa-lsa-count" : 0,
                        "openconfig-ospfv2-ext:active-interface-count" : 1,
                        "openconfig-ospfv2-ext:summary-lsa-checksum" : "0x0000201c",
                        "openconfig-ospfv2-ext:network-lsa-checksum" : "0x0000ca2c",
                        "openconfig-ospfv2-ext:opaque-link-lsa-count" : 0
                     }
                  },
                  {
                     "lsdb" : {
                        "lsa-types" : {
                           "lsa-type" : [
                              {
                                 "type" : "openconfig-ospf-types:AS_EXTERNAL_LSA",
                                 "state" : {
                                    "type" : "openconfig-ospf-types:AS_EXTERNAL_LSA"
                                 },
                                 "lsas" : {
                                    "openconfig-ospfv2-ext:lsa-ext" : [
                                       {
                                          "advertising-router" : "5.5.5.4",
                                          "as-external-lsa" : {
                                             "types-of-service" : {
                                                "type-of-service" : [
                                                   {
                                                      "tos" : 0,
                                                      "state" : {
                                                         "forwarding-address" : "0.0.0.0",
                                                         "external-route-tag" : 0,
                                                         "metric" : 20
                                                      }
                                                   }
                                                ]
                                             },
                                             "state" : {
                                                "metric-type" : "TYPE_2",
                                                "mask" : 20
                                             }
                                          },
                                          "link-state-id" : "10.193.80.0",
                                          "state" : {
                                             "checksum" : 50393,
                                             "length" : 36,
                                             "flags" : 11,
                                             "display-sequence-number" : "80000001",
                                             "option" : 2,
                                             "option-expanded" : "*||||||E|",
                                             "age" : 1227
                                          }
                                       },
                                       {
                                          "advertising-router" : "6.6.6.5",
                                          "as-external-lsa" : {
                                             "types-of-service" : {
                                                "type-of-service" : [
                                                   {
                                                      "tos" : 0,
                                                      "state" : {
                                                         "forwarding-address" : "0.0.0.0",
                                                         "external-route-tag" : 0,
                                                         "metric" : 20
                                                      }
                                                   }
                                                ]
                                             },
                                             "state" : {
                                                "metric-type" : "TYPE_2",
                                                "mask" : 20
                                             }
                                          },
                                          "link-state-id" : "10.193.80.0",
                                          "state" : {
                                             "checksum" : 42739,
                                             "length" : 36,
                                             "flags" : 6,
                                             "display-sequence-number" : "80000001",
                                             "option" : 2,
                                             "option-expanded" : "*||||||E|",
                                             "age" : 1228
                                          }
                                       }
                                    ]
                                 }
                              },
                              {
                                 "type" : "openconfig-ospf-types:NETWORK_LSA",
                                 "state" : {
                                    "type" : "openconfig-ospf-types:NETWORK_LSA"
                                 },
                                 "lsas" : {
                                    "openconfig-ospfv2-ext:lsa-ext" : [
                                       {
                                          "advertising-router" : "5.5.5.4",
                                          "link-state-id" : "41.1.0.2",
                                          "state" : {
                                             "checksum" : 1017,
                                             "length" : 32,
                                             "flags" : 3,
                                             "display-sequence-number" : "80000001",
                                             "option" : 2,
                                             "option-expanded" : "*|-|-|-|-|-|E|-",
                                             "age" : 1198
                                          },
                                          "network-lsa" : {
                                             "state" : {
                                                "network-mask" : 24
                                             }
                                          }
                                       }
                                    ]
                                 }
                              },
                              {
                                 "type" : "openconfig-ospf-types:OSPFV2_LINK_SCOPE_OPAQUE_LSA"
                              },
                              {
                                 "type" : "openconfig-ospf-types:ROUTER_LSA",
                                 "state" : {
                                    "type" : "openconfig-ospf-types:ROUTER_LSA"
                                 },
                                 "lsas" : {
                                    "openconfig-ospfv2-ext:lsa-ext" : [
                                       {
                                          "advertising-router" : "3.3.3.2",
                                          "link-state-id" : "3.3.3.2",
                                          "state" : {
                                             "checksum" : 53771,
                                             "length" : 36,
                                             "flags" : 6,
                                             "display-sequence-number" : "80000002",
                                             "option" : 0,
                                             "option-expanded" : "*|||||||-",
                                             "age" : 1189
                                          },
                                          "router-lsa" : {
                                             "link-informations" : {
                                                "link-information" : [
                                                   {
                                                      "state" : {
                                                         "metric" : 10,
                                                         "type" : "openconfig-ospf-types:ROUTER_LSA_TRANSIT_NETWORK",
                                                         "number-tos-metrics" : 0
                                                      }
                                                   }
                                                ]
                                             },
                                             "types-of-service" : {
                                                "type-of-service" : [
                                                   {
                                                      "tos" : 0,
                                                      "state" : {
                                                         "metric" : 10
                                                      }
                                                   }
                                                ]
                                             },
                                             "state" : {
                                                "flags-description" : " :",
                                                "flags" : 0,
                                                "number-links" : 1
                                             }
                                          }
                                       },
                                       {
                                          "advertising-router" : "5.5.5.4",
                                          "link-state-id" : "5.5.5.4",
                                          "state" : {
                                             "checksum" : 12693,
                                             "length" : 36,
                                             "flags" : 3,
                                             "display-sequence-number" : "80000003",
                                             "option" : 2,
                                             "option-expanded" : "*||||||E|",
                                             "age" : 1198
                                          },
                                          "router-lsa" : {
                                             "link-informations" : {
                                                "link-information" : [
                                                   {
                                                      "state" : {
                                                         "metric" : 10,
                                                         "type" : "openconfig-ospf-types:ROUTER_LSA_TRANSIT_NETWORK",
                                                         "number-tos-metrics" : 0
                                                      }
                                                   }
                                                ]
                                             },
                                             "types-of-service" : {
                                                "type-of-service" : [
                                                   {
                                                      "tos" : 0,
                                                      "state" : {
                                                         "metric" : 10
                                                      }
                                                   }
                                                ]
                                             },
                                             "state" : {
                                                "flags-description" : " : ABR ASBR",
                                                "flags" : 3,
                                                "number-links" : 1
                                             }
                                          }
                                       }
                                    ]
                                 }
                              },
                              {
                                 "type" : "openconfig-ospf-types:SUMMARY_ASBR_LSA",
                                 "state" : {
                                    "type" : "openconfig-ospf-types:SUMMARY_ASBR_LSA"
                                 },
                                 "lsas" : {
                                    "openconfig-ospfv2-ext:lsa-ext" : [
                                       {
                                          "advertising-router" : "5.5.5.4",
                                          "link-state-id" : "6.6.6.5",
                                          "summary-lsa" : {
                                             "types-of-service" : {
                                                "type-of-service" : [
                                                   {
                                                      "tos" : 0,
                                                      "state" : {
                                                         "metric" : 4
                                                      }
                                                   }
                                                ]
                                             },
                                             "state" : {
                                                "network-mask" : 0
                                             }
                                          },
                                          "state" : {
                                             "checksum" : 59716,
                                             "length" : 28,
                                             "flags" : 11,
                                             "display-sequence-number" : "80000001",
                                             "option" : 2,
                                             "option-expanded" : "*|-|-|-|-|-|E|-",
                                             "age" : 1187
                                          }
                                       }
                                    ]
                                 }
                              },
                              {
                                 "type" : "openconfig-ospf-types:SUMMARY_IP_NETWORK_LSA",
                                 "state" : {
                                    "type" : "openconfig-ospf-types:SUMMARY_IP_NETWORK_LSA"
                                 },
                                 "lsas" : {
                                    "openconfig-ospfv2-ext:lsa-ext" : [
                                       {
                                          "advertising-router" : "5.5.5.4",
                                          "link-state-id" : "21.1.0.0",
                                          "summary-lsa" : {
                                             "types-of-service" : {
                                                "type-of-service" : [
                                                   {
                                                      "tos" : 0,
                                                      "state" : {
                                                         "metric" : 14
                                                      }
                                                   }
                                                ]
                                             },
                                             "state" : {
                                                "network-mask" : 24
                                             }
                                          },
                                          "state" : {
                                             "checksum" : 18908,
                                             "length" : 28,
                                             "flags" : 11,
                                             "display-sequence-number" : "80000001",
                                             "option" : 2,
                                             "option-expanded" : "*|-|-|-|-|-|E|-",
                                             "age" : 1187
                                          }
                                       },
                                       {
                                          "advertising-router" : "5.5.5.4",
                                          "link-state-id" : "32.1.0.0",
                                          "summary-lsa" : {
                                             "types-of-service" : {
                                                "type-of-service" : [
                                                   {
                                                      "tos" : 0,
                                                      "state" : {
                                                         "metric" : 4
                                                      }
                                                   }
                                                ]
                                             },
                                             "state" : {
                                                "network-mask" : 24
                                             }
                                          },
                                          "state" : {
                                             "checksum" : 21967,
                                             "length" : 28,
                                             "flags" : 11,
                                             "display-sequence-number" : "80000001",
                                             "option" : 2,
                                             "option-expanded" : "*|-|-|-|-|-|E|-",
                                             "age" : 1229
                                          }
                                       }
                                    ]
                                 }
                              }
                           ]
                        },
                        "state" : {
                           "identifier" : "0.0.0.2"
                        }
                     },
                     "identifier" : "0.0.0.2",
                     "interfaces" : {
                        "interface" : [
                           {
                              "openconfig-ospfv2-ext:message-statistics" : {
                                 "state" : {
                                    "ls-update-transmit" : 3,
                                    "ls-request-receive" : 0,
                                    "ls-acknowledge-receive" : 3,
                                    "ls-acknowledge-transmit" : 2,
                                    "db-description-transmit" : 2,
                                    "db-description-receive" : 6,
                                    "hello-transmit" : 124,
                                    "hello-receive" : 125,
                                    "ls-request-transmit" : 1,
                                    "ls-update-receive" : 2
                                 }
                              },
                              "openconfig-ospfv2-ext:neighbours" : {
                                 "neighbour" : [
                                    {
                                       "neighbor-id" : "3.3.3.2",
                                       "neighbor-address" : "41.1.0.1",
                                       "state" : {
                                          "priority" : 0,
                                          "option-value" : 66,
                                          "gr-helper-status" : "None",
                                          "state-changes" : 6,
                                          "backup-designated-router" : "0.0.0.0",
                                          "adjacency-state" : "openconfig-ospf-types:FULL",
                                          "neighbor-id" : "6.6.6.5",
                                          "optional-capabilities" : "*|O|-|-|-|-|E|-",
                                          "thread-inactivity-timer" : true,
                                          "thread-ls-request-retransmission" : true,
                                          "thread-ls-update-retransmission" : true,
                                          "retransmit-summary-queue-length" : 0,
                                          "area-id" : "0.0.0.2",
                                          "gr-active-helper" : false,
                                          "interface-address" : "41.1.0.2",
                                          "neighbor-address" : "41.1.0.1",
                                          "last-established-time" : "1198302",
                                          "designated-router" : "41.1.0.2",
                                          "database-summary-queue-length" : 0,
                                          "interface-name" : "Ethernet120",
                                          "link-state-request-queue-length" : 0,
                                          "dead-time" : "34872"
                                       }
                                    }
                                 ]
                              },
                              "id" : "Ethernet120",
                              "timers" : {
                                 "state" : {
                                    "retransmission-interval" : 5,
                                    "dead-interval" : 40,
                                    "openconfig-ospfv2-ext:wait-time" : 40,
                                    "hello-interval" : 10000,
                                    "openconfig-ospfv2-ext:hello-due" : 1711
                                 }
                              },
                              "state" : {
                                 "priority" : 1,
                                 "openconfig-ospfv2-ext:member-of-ospf-all-routers" : true,
                                 "openconfig-ospfv2-ext:index" : 299,
                                 "openconfig-ospfv2-ext:mtu" : 9100,
                                 "openconfig-ospfv2-ext:if-flags" : "<UP,BROADCAST,RUNNING,MULTICAST>",
                                 "openconfig-ospfv2-ext:adjacency-count" : 1,
                                 "openconfig-ospfv2-ext:address-len" : 24,
                                 "openconfig-ospfv2-ext:ospf-enable" : true,
                                 "openconfig-ospfv2-ext:cost" : 10,
                                 "openconfig-ospfv2-ext:address" : "41.1.0.2",
                                 "id" : "Ethernet120",
                                 "openconfig-ospfv2-ext:adjacency-status" : "DR",
                                 "openconfig-ospfv2-ext:member-of-ospf-designated-routers" : true,
                                 "openconfig-ospfv2-ext:neighbor-count" : 1,
                                 "openconfig-ospfv2-ext:router-id" : "5.5.5.4",
                                 "openconfig-ospfv2-ext:broadcast-address" : "41.1.0.255",
                                 "openconfig-ospfv2-ext:bandwidth" : 10000,
                                 "openconfig-ospfv2-ext:area-id" : "0.0.0.2",
                                 "openconfig-ospfv2-ext:ospf-interface-type" : "Broadcast",
                                 "network-type" : "openconfig-ospf-types:BROADCAST_NETWORK",
                                 "openconfig-ospfv2-ext:transmit-delay" : 1,
                                 "openconfig-ospfv2-ext:operational-state" : "Up"
                              }
                           }
                        ]
                     },
                     "config" : {
                        "identifier" : "0.0.0.2"
                     },
                     "openconfig-ospfv2-ext:networks" : {
                        "network" : [
                           {
                              "address-prefix" : "41.1.0.0/24",
                              "config" : {
                                 "address-prefix" : "41.1.0.0/24",
                                 "description" : "Network prefix"
                              },
                              "state" : {
                                 "address-prefix" : "41.1.0.0/24",
                                 "description" : "Network prefix"
                              }
                           }
                        ]
                     },
                     "state" : {
                        "openconfig-ospfv2-ext:summary-lsa-count" : 2,
                        "openconfig-ospfv2-ext:nssa-lsa-checksum" : "0x00000000",
                        "openconfig-ospfv2-ext:authentication-type" : "no",
                        "openconfig-ospfv2-ext:asbr-summary-lsa-count" : 1,
                        "openconfig-ospfv2-ext:adjacency-count" : 1,
                        "openconfig-ospfv2-ext:opaque-area-lsa-count" : 0,
                        "openconfig-ospfv2-ext:opaque-area-lsa-checksum" : "0x00000000",
                        "openconfig-ospfv2-ext:lsa-count" : 6,
                        "openconfig-ospfv2-ext:opaque-link-lsa-checksum" : "0x00000000",
                        "openconfig-ospfv2-ext:virtual-link-adjacency-count" : 0,
                        "openconfig-ospfv2-ext:spf-execution-count" : 6,
                        "openconfig-ospfv2-ext:interface-count" : 1,
                        "openconfig-ospfv2-ext:network-lsa-count" : 1,
                        "openconfig-ospfv2-ext:asbr-summary-lsa-checksum" : "0x0000e944",
                        "openconfig-ospfv2-ext:router-lsa-checksum" : "0x000103a0",
                        "openconfig-ospfv2-ext:router-lsa-count" : 2,
                        "openconfig-ospfv2-ext:nssa-lsa-count" : 0,
                        "openconfig-ospfv2-ext:active-interface-count" : 1,
                        "openconfig-ospfv2-ext:shortcut" : "openconfig-ospfv2-ext:DEFAULT",
                        "openconfig-ospfv2-ext:summary-lsa-checksum" : "0x00009fab",
                        "openconfig-ospfv2-ext:network-lsa-checksum" : "0x000003f9",
                        "openconfig-ospfv2-ext:opaque-link-lsa-count" : 0
                     }
                  }
               ]
            }
         },
         "name" : "ospfv2",
         "config" : {
            "identifier" : "openconfig-policy-types:OSPF",
            "name" : "ospfv2"
         },
         "state" : {
            "identifier" : "openconfig-policy-types:OSPF",
            "name" : "ospfv2"
         }
      }
   ]
}
            `),
		appRootType: reflect.TypeOf(ocbinds.OpenconfigNetworkInstance_NetworkInstances{}),
		queryParams: QueryParams{
			depthEnabled:  false,
			curDepth:      1,
			content:       QUERY_CONTENT_ALL,
			fields:        []string{},
			fieldsFillAll: false,
		},
		prunedPayload: []byte(`
{
   "protocol" : [
      {
         "identifier" : "openconfig-policy-types:OSPF",
         "ospfv2" : {
            "openconfig-ospfv2-ext:route-tables" : {
               "route-table" : [
                  {
                     "routes" : {
                        "route" : [
                           {
                              "next-hops" : {
                                 "next-hop" : [
                                    {
                                       "area-id" : "0.0.0.0",
                                       "address" : "32.1.0.1",
                                       "state" : {
                                          "area-id" : "0.0.0.0",
                                          "address" : "32.1.0.1",
                                          "out-interface" : "Vlan3719"
                                       },
                                       "out-interface" : "Vlan3719"
                                    }
                                 ]
                              },
                              "state" : {
                                 "cost" : 4,
                                 "sub-type" : "openconfig-ospfv2-ext:EXTERNAL_ROUTE_TYPE_2",
                                 "type" : "openconfig-ospfv2-ext:EXTERNAL_ROUTE",
                                 "prefix" : "10.193.80.0/20"
                              },
                              "prefix" : "10.193.80.0/20"
                           }
                        ]
                     },
                     "type" : "openconfig-ospfv2-ext:EXTERNAL_ROUTE_TABLE",
                     "state" : {
                        "type" : "openconfig-ospfv2-ext:EXTERNAL_ROUTE_TABLE"
                     }
                  },
                  {
                     "routes" : {
                        "route" : [
                           {
                              "next-hops" : {
                                 "next-hop" : [
                                    {
                                       "area-id" : "0.0.0.0",
                                       "address" : "32.1.0.1",
                                       "state" : {
                                          "area-id" : "0.0.0.0",
                                          "address" : "32.1.0.1",
                                          "out-interface" : "Vlan3719"
                                       },
                                       "out-interface" : "Vlan3719"
                                    }
                                 ]
                              },
                              "state" : {
                                 "cost" : 14,
                                 "type" : "openconfig-ospfv2-ext:NETWORK_ROUTE",
                                 "inter-area" : true,
                                 "prefix" : "10.193.80.0/20"
                              },
                              "prefix" : "21.1.0.0/24"
                           },
                           {
                              "next-hops" : {
                                 "next-hop" : [
                                    {
                                       "area-id" : "0.0.0.0",
                                       "address" : "0.0.0.0",
                                       "state" : {
                                          "area-id" : "0.0.0.0",
                                          "address" : "32.1.0.1",
                                          "out-interface" : "Vlan3719"
                                       },
                                       "out-interface" : "Vlan3719"
                                    }
                                 ]
                              },
                              "state" : {
                                 "cost" : 4,
                                 "type" : "openconfig-ospfv2-ext:NETWORK_ROUTE",
                                 "prefix" : "10.193.80.0/20"
                              },
                              "prefix" : "32.1.0.0/24"
                           },
                           {
                              "next-hops" : {
                                 "next-hop" : [
                                    {
                                       "area-id" : "0.0.0.2",
                                       "address" : "0.0.0.0",
                                       "state" : {
                                          "area-id" : "0.0.0.0",
                                          "address" : "32.1.0.1",
                                          "out-interface" : "Vlan3719"
                                       },
                                       "out-interface" : "Ethernet120"
                                    }
                                 ]
                              },
                              "state" : {
                                 "cost" : 10,
                                 "type" : "openconfig-ospfv2-ext:NETWORK_ROUTE",
                                 "prefix" : "10.193.80.0/20"
                              },
                              "prefix" : "41.1.0.0/24"
                           }
                        ]
                     },
                     "type" : "openconfig-ospfv2-ext:NETWORK_ROUTE_TABLE",
                     "state" : {
                        "type" : "openconfig-ospfv2-ext:NETWORK_ROUTE_TABLE"
                     }
                  },
                  {
                     "routes" : {
                        "route" : [
                           {
                              "next-hops" : {
                                 "next-hop" : [
                                    {
                                       "area-id" : "0.0.0.0",
                                       "address" : "32.1.0.1",
                                       "state" : {
                                          "area-id" : "0.0.0.0",
                                          "address" : "32.1.0.1",
                                          "out-interface" : "Vlan3719"
                                       },
                                       "out-interface" : "Vlan3719"
                                    }
                                 ]
                              },
                              "state" : {
                                 "router-type" : "openconfig-ospfv2-ext:ABRASBR",
                                 "cost" : 4,
                                 "type" : "openconfig-ospfv2-ext:ROUTER_ROUTE",
                                 "prefix" : "10.193.80.0/20"
                              },
                              "prefix" : "6.6.6.5"
                           }
                        ]
                     },
                     "type" : "openconfig-ospfv2-ext:ROUTER_ROUTE_TABLE",
                     "state" : {
                        "type" : "openconfig-ospfv2-ext:ROUTER_ROUTE_TABLE"
                     }
                  }
               ]
            },
            "global" : {
               "graceful-restart" : {
                  "openconfig-ospfv2-ext:helpers" : {
                     "helper" : [
                        {
                           "neighbour-id" : "5.5.5.4",
                           "config" : {
                              "vrf-name" : "default",
                              "neighbour-id" : "5.5.5.4"
                           },
                           "state" : {
                              "neighbour-id" : "5.5.5.4"
                           }
                        }
                     ]
                  },
                  "config" : {
                     "openconfig-ospfv2-ext:planned-only" : true,
                     "helper-only" : true,
                     "openconfig-ospfv2-ext:supported-grace-time" : 800,
                     "openconfig-ospfv2-ext:strict-lsa-checking" : true
                  },
                  "state" : {
                     "openconfig-ospfv2-ext:planned-only" : true,
                     "helper-only" : true,
                     "openconfig-ospfv2-ext:grace-period" : 0,
                     "openconfig-ospfv2-ext:supported-grace-time" : 800,
                     "openconfig-ospfv2-ext:strict-lsa-checking" : true
                  }
               },
               "inter-area-propagation-policies" : {
                  "openconfig-ospfv2-ext:inter-area-policy" : [
                     {
                        "config" : {
                           "src-area" : "0.0.0.0"
                        },
                        "src-area" : "0.0.0.0",
                        "state" : {
                           "src-area" : "0.0.0.0"
                        }
                     },
                     {
                        "config" : {
                           "src-area" : "0.0.0.2"
                        },
                        "src-area" : "0.0.0.2",
                        "state" : {
                           "src-area" : "0.0.0.2"
                        }
                     }
                  ]
               },
               "config" : {
                  "router-id" : "5.5.5.4",
                  "openconfig-ospfv2-ext:opaque-lsa-capability" : true,
                  "openconfig-ospfv2-ext:description" : "OSPFv2 Router",
                  "openconfig-ospfv2-ext:enable" : true
               },
               "timers" : {
                  "lsa-generation" : {
                     "state" : {
                        "openconfig-ospfv2-ext:refresh-timer" : 10000,
                        "openconfig-ospfv2-ext:lsa-min-interval-timer" : 5000,
                        "openconfig-ospfv2-ext:lsa-min-arrival-timer" : 1000
                     }
                  },
                  "spf" : {
                     "state" : {
                        "maximum-delay" : 5000,
                        "initial-delay" : 50,
                        "openconfig-ospfv2-ext:throttle-delay" : 0,
                        "openconfig-ospfv2-ext:spf-timer-due" : 0
                     }
                  }
               },
               "state" : {
                  "openconfig-ospfv2-ext:area-count" : 2,
                  "openconfig-ospfv2-ext:last-spf-duration" : 358,
                  "openconfig-ospfv2-ext:hold-time-multiplier" : 1,
                  "openconfig-ospfv2-ext:last-spf-execution-time" : "1187616",
                  "openconfig-ospfv2-ext:opaque-lsa-count" : 0,
                  "router-id" : "5.5.5.4",
                  "openconfig-ospfv2-ext:abr-type" : "openconfig-ospfv2-ext:CISCO",
                  "openconfig-ospfv2-ext:external-lsa-checksum" : "0x00016bcc",
                  "openconfig-ospfv2-ext:opaque-lsa-checksum" : "0x00000000",
                  "openconfig-ospfv2-ext:write-multiplier" : 20,
                  "openconfig-ospfv2-ext:opaque-lsa-capability" : true,
                  "openconfig-ospfv2-ext:external-lsa-count" : 2
               }
            },
            "areas" : {
               "area" : [
                  {
                     "lsdb" : {
                        "lsa-types" : {
                           "lsa-type" : [
                              {
                                 "type" : "openconfig-ospf-types:AS_EXTERNAL_LSA",
                                 "state" : {
                                    "type" : "openconfig-ospf-types:AS_EXTERNAL_LSA"
                                 },
                                 "lsas" : {
                                    "openconfig-ospfv2-ext:lsa-ext" : [
                                       {
                                          "advertising-router" : "5.5.5.4",
                                          "as-external-lsa" : {
                                             "types-of-service" : {
                                                "type-of-service" : [
                                                   {
                                                      "tos" : 0,
                                                      "state" : {
                                                         "forwarding-address" : "0.0.0.0",
                                                         "external-route-tag" : 0,
                                                         "metric" : 20
                                                      }
                                                   }
                                                ]
                                             },
                                             "state" : {
                                                "metric-type" : "TYPE_2",
                                                "mask" : 20
                                             }
                                          },
                                          "link-state-id" : "10.193.80.0",
                                          "state" : {
                                             "checksum" : 50393,
                                             "length" : 36,
                                             "flags" : 11,
                                             "display-sequence-number" : "80000001",
                                             "option" : 2,
                                             "option-expanded" : "*||||||E|",
                                             "age" : 1227
                                          }
                                       },
                                       {
                                          "advertising-router" : "6.6.6.5",
                                          "as-external-lsa" : {
                                             "types-of-service" : {
                                                "type-of-service" : [
                                                   {
                                                      "tos" : 0,
                                                      "state" : {
                                                         "forwarding-address" : "0.0.0.0",
                                                         "external-route-tag" : 0,
                                                         "metric" : 20
                                                      }
                                                   }
                                                ]
                                             },
                                             "state" : {
                                                "metric-type" : "TYPE_2",
                                                "mask" : 20
                                             }
                                          },
                                          "link-state-id" : "10.193.80.0",
                                          "state" : {
                                             "checksum" : 42739,
                                             "length" : 36,
                                             "flags" : 6,
                                             "display-sequence-number" : "80000001",
                                             "option" : 2,
                                             "option-expanded" : "*||||||E|",
                                             "age" : 1228
                                          }
                                       }
                                    ]
                                 }
                              },
                              {
                                 "type" : "openconfig-ospf-types:NETWORK_LSA",
                                 "state" : {
                                    "type" : "openconfig-ospf-types:NETWORK_LSA"
                                 },
                                 "lsas" : {
                                    "openconfig-ospfv2-ext:lsa-ext" : [
                                       {
                                          "advertising-router" : "6.6.6.5",
                                          "link-state-id" : "32.1.0.1",
                                          "state" : {
                                             "checksum" : 51756,
                                             "length" : 32,
                                             "flags" : 6,
                                             "display-sequence-number" : "80000001",
                                             "option" : 2,
                                             "option-expanded" : "*|-|-|-|-|-|E|-",
                                             "age" : 1198
                                          },
                                          "network-lsa" : {
                                             "state" : {
                                                "network-mask" : 24
                                             }
                                          }
                                       }
                                    ]
                                 }
                              },
                              {
                                 "type" : "openconfig-ospf-types:OSPFV2_LINK_SCOPE_OPAQUE_LSA"
                              },
                              {
                                 "type" : "openconfig-ospf-types:ROUTER_LSA",
                                 "state" : {
                                    "type" : "openconfig-ospf-types:ROUTER_LSA"
                                 },
                                 "lsas" : {
                                    "openconfig-ospfv2-ext:lsa-ext" : [
                                       {
                                          "advertising-router" : "5.5.5.4",
                                          "link-state-id" : "5.5.5.4",
                                          "state" : {
                                             "checksum" : 5321,
                                             "length" : 36,
                                             "flags" : 3,
                                             "display-sequence-number" : "80000005",
                                             "option" : 2,
                                             "option-expanded" : "*||||||E|",
                                             "age" : 1197
                                          },
                                          "router-lsa" : {
                                             "link-informations" : {
                                                "link-information" : [
                                                   {
                                                      "state" : {
                                                         "metric" : 4,
                                                         "type" : "openconfig-ospf-types:ROUTER_LSA_TRANSIT_NETWORK",
                                                         "number-tos-metrics" : 0
                                                      }
                                                   }
                                                ]
                                             },
                                             "types-of-service" : {
                                                "type-of-service" : [
                                                   {
                                                      "tos" : 0,
                                                      "state" : {
                                                         "metric" : 4
                                                      }
                                                   }
                                                ]
                                             },
                                             "state" : {
                                                "flags-description" : " : ABR ASBR",
                                                "flags" : 3,
                                                "number-links" : 1
                                             }
                                          }
                                       },
                                       {
                                          "advertising-router" : "6.6.6.5",
                                          "link-state-id" : "6.6.6.5",
                                          "state" : {
                                             "checksum" : 47900,
                                             "length" : 36,
                                             "flags" : 6,
                                             "display-sequence-number" : "80000004",
                                             "option" : 2,
                                             "option-expanded" : "*||||||E|",
                                             "age" : 1197
                                          },
                                          "router-lsa" : {
                                             "link-informations" : {
                                                "link-information" : [
                                                   {
                                                      "state" : {
                                                         "metric" : 4,
                                                         "type" : "openconfig-ospf-types:ROUTER_LSA_TRANSIT_NETWORK",
                                                         "number-tos-metrics" : 0
                                                      }
                                                   }
                                                ]
                                             },
                                             "types-of-service" : {
                                                "type-of-service" : [
                                                   {
                                                      "tos" : 0,
                                                      "state" : {
                                                         "metric" : 4
                                                      }
                                                   }
                                                ]
                                             },
                                             "state" : {
                                                "flags-description" : " : ABR ASBR",
                                                "flags" : 3,
                                                "number-links" : 1
                                             }
                                          }
                                       }
                                    ]
                                 }
                              },
                              {
                                 "type" : "openconfig-ospf-types:SUMMARY_ASBR_LSA",
                                 "state" : {
                                    "type" : "openconfig-ospf-types:SUMMARY_ASBR_LSA"
                                 }
                              },
                              {
                                 "type" : "openconfig-ospf-types:SUMMARY_IP_NETWORK_LSA",
                                 "state" : {
                                    "type" : "openconfig-ospf-types:SUMMARY_IP_NETWORK_LSA"
                                 },
                                 "lsas" : {
                                    "openconfig-ospfv2-ext:lsa-ext" : [
                                       {
                                          "advertising-router" : "6.6.6.5",
                                          "link-state-id" : "21.1.0.0",
                                          "summary-lsa" : {
                                             "types-of-service" : {
                                                "type-of-service" : [
                                                   {
                                                      "tos" : 0,
                                                      "state" : {
                                                         "metric" : 10
                                                      }
                                                   }
                                                ]
                                             },
                                             "state" : {
                                                "network-mask" : 24
                                             }
                                          },
                                          "state" : {
                                             "checksum" : 803,
                                             "length" : 28,
                                             "flags" : 6,
                                             "display-sequence-number" : "80000001",
                                             "option" : 2,
                                             "option-expanded" : "*|-|-|-|-|-|E|-",
                                             "age" : 1228
                                          }
                                       },
                                       {
                                          "advertising-router" : "5.5.5.4",
                                          "link-state-id" : "41.1.0.0",
                                          "summary-lsa" : {
                                             "types-of-service" : {
                                                "type-of-service" : [
                                                   {
                                                      "tos" : 0,
                                                      "state" : {
                                                         "metric" : 10
                                                      }
                                                   }
                                                ]
                                             },
                                             "state" : {
                                                "network-mask" : 24
                                             }
                                          },
                                          "state" : {
                                             "checksum" : 7417,
                                             "length" : 28,
                                             "flags" : 11,
                                             "display-sequence-number" : "80000001",
                                             "option" : 2,
                                             "option-expanded" : "*|-|-|-|-|-|E|-",
                                             "age" : 1227
                                          }
                                       }
                                    ]
                                 }
                              }
                           ]
                        },
                        "state" : {
                           "identifier" : "0.0.0.0"
                        }
                     },
                     "identifier" : "0.0.0.0",
                     "interfaces" : {
                        "interface" : [
                           {
                              "openconfig-ospfv2-ext:message-statistics" : {
                                 "state" : {
                                    "ls-update-transmit" : 4,
                                    "ls-request-receive" : 1,
                                    "ls-acknowledge-receive" : 2,
                                    "ls-acknowledge-transmit" : 2,
                                    "db-description-transmit" : 3,
                                    "db-description-receive" : 2,
                                    "hello-transmit" : 124,
                                    "hello-receive" : 124,
                                    "ls-request-transmit" : 1,
                                    "ls-update-receive" : 3
                                 }
                              },
                              "openconfig-ospfv2-ext:neighbours" : {
                                 "neighbour" : [
                                    {
                                       "neighbor-id" : "6.6.6.5",
                                       "neighbor-address" : "32.1.0.1",
                                       "state" : {
                                          "priority" : 1,
                                          "option-value" : 66,
                                          "gr-helper-status" : "None",
                                          "state-changes" : 6,
                                          "backup-designated-router" : "32.1.0.2",
                                          "adjacency-state" : "openconfig-ospf-types:FULL",
                                          "neighbor-id" : "6.6.6.5",
                                          "optional-capabilities" : "*|O|-|-|-|-|E|-",
                                          "thread-inactivity-timer" : true,
                                          "thread-ls-request-retransmission" : true,
                                          "thread-ls-update-retransmission" : true,
                                          "retransmit-summary-queue-length" : 0,
                                          "area-id" : "0.0.0.0",
                                          "gr-active-helper" : false,
                                          "interface-address" : "32.1.0.2",
                                          "neighbor-address" : "32.1.0.1",
                                          "last-established-time" : "1197595",
                                          "designated-router" : "32.1.0.1",
                                          "database-summary-queue-length" : 0,
                                          "interface-name" : "Vlan3719",
                                          "link-state-request-queue-length" : 0,
                                          "dead-time" : "32410"
                                       }
                                    }
                                 ]
                              },
                              "id" : "Vlan3719",
                              "timers" : {
                                 "state" : {
                                    "retransmission-interval" : 5,
                                    "dead-interval" : 40,
                                    "openconfig-ospfv2-ext:wait-time" : 40,
                                    "hello-interval" : 10000,
                                    "openconfig-ospfv2-ext:hello-due" : 2409
                                 }
                              },
                              "state" : {
                                 "priority" : 1,
                                 "openconfig-ospfv2-ext:member-of-ospf-all-routers" : true,
                                 "openconfig-ospfv2-ext:index" : 310,
                                 "openconfig-ospfv2-ext:mtu" : 9100,
                                 "openconfig-ospfv2-ext:if-flags" : "<UP,BROADCAST,RUNNING,MULTICAST>",
                                 "openconfig-ospfv2-ext:adjacency-count" : 1,
                                 "openconfig-ospfv2-ext:address-len" : 24,
                                 "openconfig-ospfv2-ext:ospf-enable" : true,
                                 "openconfig-ospfv2-ext:cost" : 4,
                                 "openconfig-ospfv2-ext:address" : "32.1.0.2",
                                 "id" : "Vlan3719",
                                 "openconfig-ospfv2-ext:adjacency-status" : "Backup",
                                 "openconfig-ospfv2-ext:neighbor-count" : 1,
                                 "openconfig-ospfv2-ext:backup-designated-router-id" : "5.5.5.4",
                                 "openconfig-ospfv2-ext:member-of-ospf-designated-routers" : true,
                                 "openconfig-ospfv2-ext:router-id" : "5.5.5.4",
                                 "openconfig-ospfv2-ext:broadcast-address" : "32.1.0.255",
                                 "openconfig-ospfv2-ext:bandwidth" : 25000,
                                 "openconfig-ospfv2-ext:area-id" : "0.0.0.0",
                                 "openconfig-ospfv2-ext:ospf-interface-type" : "Broadcast",
                                 "openconfig-ospfv2-ext:backup-designated-router-address" : "32.1.0.2",
                                 "network-type" : "openconfig-ospf-types:BROADCAST_NETWORK",
                                 "openconfig-ospfv2-ext:transmit-delay" : 1,
                                 "openconfig-ospfv2-ext:operational-state" : "Up"
                              }
                           }
                        ]
                     },
                     "config" : {
                        "identifier" : "0.0.0.0"
                     },
                     "openconfig-ospfv2-ext:networks" : {
                        "network" : [
                           {
                              "address-prefix" : "31.1.0.0/24",
                              "config" : {
                                 "address-prefix" : "31.1.0.0/24",
                                 "description" : "Network prefix"
                              },
                              "state" : {
                                 "address-prefix" : "31.1.0.0/24",
                                 "description" : "Network prefix"
                              }
                           },
                           {
                              "address-prefix" : "32.1.0.0/24",
                              "config" : {
                                 "address-prefix" : "32.1.0.0/24",
                                 "description" : "Network prefix"
                              },
                              "state" : {
                                 "address-prefix" : "32.1.0.0/24",
                                 "description" : "Network prefix"
                              }
                           }
                        ]
                     },
                     "state" : {
                        "openconfig-ospfv2-ext:summary-lsa-count" : 2,
                        "openconfig-ospfv2-ext:nssa-lsa-checksum" : "0x00000000",
                        "openconfig-ospfv2-ext:authentication-type" : "no",
                        "openconfig-ospfv2-ext:asbr-summary-lsa-count" : 0,
                        "openconfig-ospfv2-ext:adjacency-count" : 1,
                        "openconfig-ospfv2-ext:opaque-area-lsa-count" : 0,
                        "openconfig-ospfv2-ext:opaque-area-lsa-checksum" : "0x00000000",
                        "openconfig-ospfv2-ext:lsa-count" : 5,
                        "openconfig-ospfv2-ext:opaque-link-lsa-checksum" : "0x00000000",
                        "openconfig-ospfv2-ext:spf-execution-count" : 6,
                        "openconfig-ospfv2-ext:interface-count" : 1,
                        "openconfig-ospfv2-ext:network-lsa-count" : 1,
                        "openconfig-ospfv2-ext:asbr-summary-lsa-checksum" : "0x00000000",
                        "openconfig-ospfv2-ext:router-lsa-checksum" : "0x0000cfe5",
                        "openconfig-ospfv2-ext:router-lsa-count" : 2,
                        "openconfig-ospfv2-ext:nssa-lsa-count" : 0,
                        "openconfig-ospfv2-ext:active-interface-count" : 1,
                        "openconfig-ospfv2-ext:summary-lsa-checksum" : "0x0000201c",
                        "openconfig-ospfv2-ext:network-lsa-checksum" : "0x0000ca2c",
                        "openconfig-ospfv2-ext:opaque-link-lsa-count" : 0
                     }
                  },
                  {
                     "lsdb" : {
                        "lsa-types" : {
                           "lsa-type" : [
                              {
                                 "type" : "openconfig-ospf-types:AS_EXTERNAL_LSA",
                                 "state" : {
                                    "type" : "openconfig-ospf-types:AS_EXTERNAL_LSA"
                                 },
                                 "lsas" : {
                                    "openconfig-ospfv2-ext:lsa-ext" : [
                                       {
                                          "advertising-router" : "5.5.5.4",
                                          "as-external-lsa" : {
                                             "types-of-service" : {
                                                "type-of-service" : [
                                                   {
                                                      "tos" : 0,
                                                      "state" : {
                                                         "forwarding-address" : "0.0.0.0",
                                                         "external-route-tag" : 0,
                                                         "metric" : 20
                                                      }
                                                   }
                                                ]
                                             },
                                             "state" : {
                                                "metric-type" : "TYPE_2",
                                                "mask" : 20
                                             }
                                          },
                                          "link-state-id" : "10.193.80.0",
                                          "state" : {
                                             "checksum" : 50393,
                                             "length" : 36,
                                             "flags" : 11,
                                             "display-sequence-number" : "80000001",
                                             "option" : 2,
                                             "option-expanded" : "*||||||E|",
                                             "age" : 1227
                                          }
                                       },
                                       {
                                          "advertising-router" : "6.6.6.5",
                                          "as-external-lsa" : {
                                             "types-of-service" : {
                                                "type-of-service" : [
                                                   {
                                                      "tos" : 0,
                                                      "state" : {
                                                         "forwarding-address" : "0.0.0.0",
                                                         "external-route-tag" : 0,
                                                         "metric" : 20
                                                      }
                                                   }
                                                ]
                                             },
                                             "state" : {
                                                "metric-type" : "TYPE_2",
                                                "mask" : 20
                                             }
                                          },
                                          "link-state-id" : "10.193.80.0",
                                          "state" : {
                                             "checksum" : 42739,
                                             "length" : 36,
                                             "flags" : 6,
                                             "display-sequence-number" : "80000001",
                                             "option" : 2,
                                             "option-expanded" : "*||||||E|",
                                             "age" : 1228
                                          }
                                       }
                                    ]
                                 }
                              },
                              {
                                 "type" : "openconfig-ospf-types:NETWORK_LSA",
                                 "state" : {
                                    "type" : "openconfig-ospf-types:NETWORK_LSA"
                                 },
                                 "lsas" : {
                                    "openconfig-ospfv2-ext:lsa-ext" : [
                                       {
                                          "advertising-router" : "5.5.5.4",
                                          "link-state-id" : "41.1.0.2",
                                          "state" : {
                                             "checksum" : 1017,
                                             "length" : 32,
                                             "flags" : 3,
                                             "display-sequence-number" : "80000001",
                                             "option" : 2,
                                             "option-expanded" : "*|-|-|-|-|-|E|-",
                                             "age" : 1198
                                          },
                                          "network-lsa" : {
                                             "state" : {
                                                "network-mask" : 24
                                             }
                                          }
                                       }
                                    ]
                                 }
                              },
                              {
                                 "type" : "openconfig-ospf-types:OSPFV2_LINK_SCOPE_OPAQUE_LSA"
                              },
                              {
                                 "type" : "openconfig-ospf-types:ROUTER_LSA",
                                 "state" : {
                                    "type" : "openconfig-ospf-types:ROUTER_LSA"
                                 },
                                 "lsas" : {
                                    "openconfig-ospfv2-ext:lsa-ext" : [
                                       {
                                          "advertising-router" : "3.3.3.2",
                                          "link-state-id" : "3.3.3.2",
                                          "state" : {
                                             "checksum" : 53771,
                                             "length" : 36,
                                             "flags" : 6,
                                             "display-sequence-number" : "80000002",
                                             "option" : 0,
                                             "option-expanded" : "*|||||||-",
                                             "age" : 1189
                                          },
                                          "router-lsa" : {
                                             "link-informations" : {
                                                "link-information" : [
                                                   {
                                                      "state" : {
                                                         "metric" : 10,
                                                         "type" : "openconfig-ospf-types:ROUTER_LSA_TRANSIT_NETWORK",
                                                         "number-tos-metrics" : 0
                                                      }
                                                   }
                                                ]
                                             },
                                             "types-of-service" : {
                                                "type-of-service" : [
                                                   {
                                                      "tos" : 0,
                                                      "state" : {
                                                         "metric" : 10
                                                      }
                                                   }
                                                ]
                                             },
                                             "state" : {
                                                "flags-description" : " :",
                                                "flags" : 0,
                                                "number-links" : 1
                                             }
                                          }
                                       },
                                       {
                                          "advertising-router" : "5.5.5.4",
                                          "link-state-id" : "5.5.5.4",
                                          "state" : {
                                             "checksum" : 12693,
                                             "length" : 36,
                                             "flags" : 3,
                                             "display-sequence-number" : "80000003",
                                             "option" : 2,
                                             "option-expanded" : "*||||||E|",
                                             "age" : 1198
                                          },
                                          "router-lsa" : {
                                             "link-informations" : {
                                                "link-information" : [
                                                   {
                                                      "state" : {
                                                         "metric" : 10,
                                                         "type" : "openconfig-ospf-types:ROUTER_LSA_TRANSIT_NETWORK",
                                                         "number-tos-metrics" : 0
                                                      }
                                                   }
                                                ]
                                             },
                                             "types-of-service" : {
                                                "type-of-service" : [
                                                   {
                                                      "tos" : 0,
                                                      "state" : {
                                                         "metric" : 10
                                                      }
                                                   }
                                                ]
                                             },
                                             "state" : {
                                                "flags-description" : " : ABR ASBR",
                                                "flags" : 3,
                                                "number-links" : 1
                                             }
                                          }
                                       }
                                    ]
                                 }
                              },
                              {
                                 "type" : "openconfig-ospf-types:SUMMARY_ASBR_LSA",
                                 "state" : {
                                    "type" : "openconfig-ospf-types:SUMMARY_ASBR_LSA"
                                 },
                                 "lsas" : {
                                    "openconfig-ospfv2-ext:lsa-ext" : [
                                       {
                                          "advertising-router" : "5.5.5.4",
                                          "link-state-id" : "6.6.6.5",
                                          "summary-lsa" : {
                                             "types-of-service" : {
                                                "type-of-service" : [
                                                   {
                                                      "tos" : 0,
                                                      "state" : {
                                                         "metric" : 4
                                                      }
                                                   }
                                                ]
                                             },
                                             "state" : {
                                                "network-mask" : 0
                                             }
                                          },
                                          "state" : {
                                             "checksum" : 59716,
                                             "length" : 28,
                                             "flags" : 11,
                                             "display-sequence-number" : "80000001",
                                             "option" : 2,
                                             "option-expanded" : "*|-|-|-|-|-|E|-",
                                             "age" : 1187
                                          }
                                       }
                                    ]
                                 }
                              },
                              {
                                 "type" : "openconfig-ospf-types:SUMMARY_IP_NETWORK_LSA",
                                 "state" : {
                                    "type" : "openconfig-ospf-types:SUMMARY_IP_NETWORK_LSA"
                                 },
                                 "lsas" : {
                                    "openconfig-ospfv2-ext:lsa-ext" : [
                                       {
                                          "advertising-router" : "5.5.5.4",
                                          "link-state-id" : "21.1.0.0",
                                          "summary-lsa" : {
                                             "types-of-service" : {
                                                "type-of-service" : [
                                                   {
                                                      "tos" : 0,
                                                      "state" : {
                                                         "metric" : 14
                                                      }
                                                   }
                                                ]
                                             },
                                             "state" : {
                                                "network-mask" : 24
                                             }
                                          },
                                          "state" : {
                                             "checksum" : 18908,
                                             "length" : 28,
                                             "flags" : 11,
                                             "display-sequence-number" : "80000001",
                                             "option" : 2,
                                             "option-expanded" : "*|-|-|-|-|-|E|-",
                                             "age" : 1187
                                          }
                                       },
                                       {
                                          "advertising-router" : "5.5.5.4",
                                          "link-state-id" : "32.1.0.0",
                                          "summary-lsa" : {
                                             "types-of-service" : {
                                                "type-of-service" : [
                                                   {
                                                      "tos" : 0,
                                                      "state" : {
                                                         "metric" : 4
                                                      }
                                                   }
                                                ]
                                             },
                                             "state" : {
                                                "network-mask" : 24
                                             }
                                          },
                                          "state" : {
                                             "checksum" : 21967,
                                             "length" : 28,
                                             "flags" : 11,
                                             "display-sequence-number" : "80000001",
                                             "option" : 2,
                                             "option-expanded" : "*|-|-|-|-|-|E|-",
                                             "age" : 1229
                                          }
                                       }
                                    ]
                                 }
                              }
                           ]
                        },
                        "state" : {
                           "identifier" : "0.0.0.2"
                        }
                     },
                     "identifier" : "0.0.0.2",
                     "interfaces" : {
                        "interface" : [
                           {
                              "openconfig-ospfv2-ext:message-statistics" : {
                                 "state" : {
                                    "ls-update-transmit" : 3,
                                    "ls-request-receive" : 0,
                                    "ls-acknowledge-receive" : 3,
                                    "ls-acknowledge-transmit" : 2,
                                    "db-description-transmit" : 2,
                                    "db-description-receive" : 6,
                                    "hello-transmit" : 124,
                                    "hello-receive" : 125,
                                    "ls-request-transmit" : 1,
                                    "ls-update-receive" : 2
                                 }
                              },
                              "openconfig-ospfv2-ext:neighbours" : {
                                 "neighbour" : [
                                    {
                                       "neighbor-id" : "3.3.3.2",
                                       "neighbor-address" : "41.1.0.1",
                                       "state" : {
                                          "priority" : 0,
                                          "option-value" : 66,
                                          "gr-helper-status" : "None",
                                          "state-changes" : 6,
                                          "backup-designated-router" : "0.0.0.0",
                                          "adjacency-state" : "openconfig-ospf-types:FULL",
                                          "neighbor-id" : "6.6.6.5",
                                          "optional-capabilities" : "*|O|-|-|-|-|E|-",
                                          "thread-inactivity-timer" : true,
                                          "thread-ls-request-retransmission" : true,
                                          "thread-ls-update-retransmission" : true,
                                          "retransmit-summary-queue-length" : 0,
                                          "area-id" : "0.0.0.2",
                                          "gr-active-helper" : false,
                                          "interface-address" : "41.1.0.2",
                                          "neighbor-address" : "41.1.0.1",
                                          "last-established-time" : "1198302",
                                          "designated-router" : "41.1.0.2",
                                          "database-summary-queue-length" : 0,
                                          "interface-name" : "Ethernet120",
                                          "link-state-request-queue-length" : 0,
                                          "dead-time" : "34872"
                                       }
                                    }
                                 ]
                              },
                              "id" : "Ethernet120",
                              "timers" : {
                                 "state" : {
                                    "retransmission-interval" : 5,
                                    "dead-interval" : 40,
                                    "openconfig-ospfv2-ext:wait-time" : 40,
                                    "hello-interval" : 10000,
                                    "openconfig-ospfv2-ext:hello-due" : 1711
                                 }
                              },
                              "state" : {
                                 "priority" : 1,
                                 "openconfig-ospfv2-ext:member-of-ospf-all-routers" : true,
                                 "openconfig-ospfv2-ext:index" : 299,
                                 "openconfig-ospfv2-ext:mtu" : 9100,
                                 "openconfig-ospfv2-ext:if-flags" : "<UP,BROADCAST,RUNNING,MULTICAST>",
                                 "openconfig-ospfv2-ext:adjacency-count" : 1,
                                 "openconfig-ospfv2-ext:address-len" : 24,
                                 "openconfig-ospfv2-ext:ospf-enable" : true,
                                 "openconfig-ospfv2-ext:cost" : 10,
                                 "openconfig-ospfv2-ext:address" : "41.1.0.2",
                                 "id" : "Ethernet120",
                                 "openconfig-ospfv2-ext:adjacency-status" : "DR",
                                 "openconfig-ospfv2-ext:member-of-ospf-designated-routers" : true,
                                 "openconfig-ospfv2-ext:neighbor-count" : 1,
                                 "openconfig-ospfv2-ext:router-id" : "5.5.5.4",
                                 "openconfig-ospfv2-ext:broadcast-address" : "41.1.0.255",
                                 "openconfig-ospfv2-ext:bandwidth" : 10000,
                                 "openconfig-ospfv2-ext:area-id" : "0.0.0.2",
                                 "openconfig-ospfv2-ext:ospf-interface-type" : "Broadcast",
                                 "network-type" : "openconfig-ospf-types:BROADCAST_NETWORK",
                                 "openconfig-ospfv2-ext:transmit-delay" : 1,
                                 "openconfig-ospfv2-ext:operational-state" : "Up"
                              }
                           }
                        ]
                     },
                     "config" : {
                        "identifier" : "0.0.0.2"
                     },
                     "openconfig-ospfv2-ext:networks" : {
                        "network" : [
                           {
                              "address-prefix" : "41.1.0.0/24",
                              "config" : {
                                 "address-prefix" : "41.1.0.0/24",
                                 "description" : "Network prefix"
                              },
                              "state" : {
                                 "address-prefix" : "41.1.0.0/24",
                                 "description" : "Network prefix"
                              }
                           }
                        ]
                     },
                     "state" : {
                        "openconfig-ospfv2-ext:summary-lsa-count" : 2,
                        "openconfig-ospfv2-ext:nssa-lsa-checksum" : "0x00000000",
                        "openconfig-ospfv2-ext:authentication-type" : "no",
                        "openconfig-ospfv2-ext:asbr-summary-lsa-count" : 1,
                        "openconfig-ospfv2-ext:adjacency-count" : 1,
                        "openconfig-ospfv2-ext:opaque-area-lsa-count" : 0,
                        "openconfig-ospfv2-ext:opaque-area-lsa-checksum" : "0x00000000",
                        "openconfig-ospfv2-ext:lsa-count" : 6,
                        "openconfig-ospfv2-ext:opaque-link-lsa-checksum" : "0x00000000",
                        "openconfig-ospfv2-ext:virtual-link-adjacency-count" : 0,
                        "openconfig-ospfv2-ext:spf-execution-count" : 6,
                        "openconfig-ospfv2-ext:interface-count" : 1,
                        "openconfig-ospfv2-ext:network-lsa-count" : 1,
                        "openconfig-ospfv2-ext:asbr-summary-lsa-checksum" : "0x0000e944",
                        "openconfig-ospfv2-ext:router-lsa-checksum" : "0x000103a0",
                        "openconfig-ospfv2-ext:router-lsa-count" : 2,
                        "openconfig-ospfv2-ext:nssa-lsa-count" : 0,
                        "openconfig-ospfv2-ext:active-interface-count" : 1,
                        "openconfig-ospfv2-ext:shortcut" : "openconfig-ospfv2-ext:DEFAULT",
                        "openconfig-ospfv2-ext:summary-lsa-checksum" : "0x00009fab",
                        "openconfig-ospfv2-ext:network-lsa-checksum" : "0x000003f9",
                        "openconfig-ospfv2-ext:opaque-link-lsa-count" : 0
                     }
                  }
               ]
            }
         },
         "name" : "ospfv2",
         "config" : {
            "identifier" : "openconfig-policy-types:OSPF",
            "name" : "ospfv2"
         },
         "state" : {
            "identifier" : "openconfig-policy-types:OSPF",
            "name" : "ospfv2"
         }
      }
   ]
}
            `),
	},
	{ // OSPF Identity Test Case (i.e. no QP)
		tid:        "OSPF Identity",
		uri:        "/openconfig-network-instance:network-instances/network-instance[name=default]/protocols/protocol[identifier=OSPF][name=ospfv2]/ospfv2/areas/area[identifier=0.0.0.0]/lsdb",
		requestUri: "/openconfig-network-instance:network-instances/network-instance[name=default]/protocols/protocol[identifier=OSPF][name=ospfv2]",
		payload: []byte(`
{
   "protocol" : [
      {
         "identifier" : "openconfig-policy-types:OSPF",
         "ospfv2" : {
            "openconfig-ospfv2-ext:route-tables" : {
               "route-table" : [
                  {
                     "routes" : {
                        "route" : [
                           {
                              "next-hops" : {
                                 "next-hop" : [
                                    {
                                       "area-id" : "0.0.0.0",
                                       "address" : "32.1.0.1",
                                       "state" : {
                                          "area-id" : "0.0.0.0",
                                          "address" : "32.1.0.1",
                                          "out-interface" : "Vlan3719"
                                       },
                                       "out-interface" : "Vlan3719"
                                    }
                                 ]
                              },
                              "state" : {
                                 "cost" : 4,
                                 "sub-type" : "openconfig-ospfv2-ext:EXTERNAL_ROUTE_TYPE_2",
                                 "type" : "openconfig-ospfv2-ext:EXTERNAL_ROUTE",
                                 "prefix" : "10.193.80.0/20"
                              },
                              "prefix" : "10.193.80.0/20"
                           }
                        ]
                     },
                     "type" : "openconfig-ospfv2-ext:EXTERNAL_ROUTE_TABLE",
                     "state" : {
                        "type" : "openconfig-ospfv2-ext:EXTERNAL_ROUTE_TABLE"
                     }
                  },
                  {
                     "routes" : {
                        "route" : [
                           {
                              "next-hops" : {
                                 "next-hop" : [
                                    {
                                       "area-id" : "0.0.0.0",
                                       "address" : "32.1.0.1",
                                       "state" : {
                                          "area-id" : "0.0.0.0",
                                          "address" : "32.1.0.1",
                                          "out-interface" : "Vlan3719"
                                       },
                                       "out-interface" : "Vlan3719"
                                    }
                                 ]
                              },
                              "state" : {
                                 "cost" : 14,
                                 "type" : "openconfig-ospfv2-ext:NETWORK_ROUTE",
                                 "inter-area" : true,
                                 "prefix" : "10.193.80.0/20"
                              },
                              "prefix" : "21.1.0.0/24"
                           },
                           {
                              "next-hops" : {
                                 "next-hop" : [
                                    {
                                       "area-id" : "0.0.0.0",
                                       "address" : "0.0.0.0",
                                       "state" : {
                                          "area-id" : "0.0.0.0",
                                          "address" : "32.1.0.1",
                                          "out-interface" : "Vlan3719"
                                       },
                                       "out-interface" : "Vlan3719"
                                    }
                                 ]
                              },
                              "state" : {
                                 "cost" : 4,
                                 "type" : "openconfig-ospfv2-ext:NETWORK_ROUTE",
                                 "prefix" : "10.193.80.0/20"
                              },
                              "prefix" : "32.1.0.0/24"
                           },
                           {
                              "next-hops" : {
                                 "next-hop" : [
                                    {
                                       "area-id" : "0.0.0.2",
                                       "address" : "0.0.0.0",
                                       "state" : {
                                          "area-id" : "0.0.0.0",
                                          "address" : "32.1.0.1",
                                          "out-interface" : "Vlan3719"
                                       },
                                       "out-interface" : "Ethernet120"
                                    }
                                 ]
                              },
                              "state" : {
                                 "cost" : 10,
                                 "type" : "openconfig-ospfv2-ext:NETWORK_ROUTE",
                                 "prefix" : "10.193.80.0/20"
                              },
                              "prefix" : "41.1.0.0/24"
                           }
                        ]
                     },
                     "type" : "openconfig-ospfv2-ext:NETWORK_ROUTE_TABLE",
                     "state" : {
                        "type" : "openconfig-ospfv2-ext:NETWORK_ROUTE_TABLE"
                     }
                  },
                  {
                     "routes" : {
                        "route" : [
                           {
                              "next-hops" : {
                                 "next-hop" : [
                                    {
                                       "area-id" : "0.0.0.0",
                                       "address" : "32.1.0.1",
                                       "state" : {
                                          "area-id" : "0.0.0.0",
                                          "address" : "32.1.0.1",
                                          "out-interface" : "Vlan3719"
                                       },
                                       "out-interface" : "Vlan3719"
                                    }
                                 ]
                              },
                              "state" : {
                                 "router-type" : "openconfig-ospfv2-ext:ABRASBR",
                                 "cost" : 4,
                                 "type" : "openconfig-ospfv2-ext:ROUTER_ROUTE",
                                 "prefix" : "10.193.80.0/20"
                              },
                              "prefix" : "6.6.6.5"
                           }
                        ]
                     },
                     "type" : "openconfig-ospfv2-ext:ROUTER_ROUTE_TABLE",
                     "state" : {
                        "type" : "openconfig-ospfv2-ext:ROUTER_ROUTE_TABLE"
                     }
                  }
               ]
            },
            "global" : {
               "graceful-restart" : {
                  "openconfig-ospfv2-ext:helpers" : {
                     "helper" : [
                        {
                           "neighbour-id" : "5.5.5.4",
                           "config" : {
                              "vrf-name" : "default",
                              "neighbour-id" : "5.5.5.4"
                           },
                           "state" : {
                              "neighbour-id" : "5.5.5.4"
                           }
                        }
                     ]
                  },
                  "config" : {
                     "openconfig-ospfv2-ext:planned-only" : true,
                     "helper-only" : true,
                     "openconfig-ospfv2-ext:supported-grace-time" : 800,
                     "openconfig-ospfv2-ext:strict-lsa-checking" : true
                  },
                  "state" : {
                     "openconfig-ospfv2-ext:planned-only" : true,
                     "helper-only" : true,
                     "openconfig-ospfv2-ext:grace-period" : 0,
                     "openconfig-ospfv2-ext:supported-grace-time" : 800,
                     "openconfig-ospfv2-ext:strict-lsa-checking" : true
                  }
               },
               "inter-area-propagation-policies" : {
                  "openconfig-ospfv2-ext:inter-area-policy" : [
                     {
                        "config" : {
                           "src-area" : "0.0.0.0"
                        },
                        "src-area" : "0.0.0.0",
                        "state" : {
                           "src-area" : "0.0.0.0"
                        }
                     },
                     {
                        "config" : {
                           "src-area" : "0.0.0.2"
                        },
                        "src-area" : "0.0.0.2",
                        "state" : {
                           "src-area" : "0.0.0.2"
                        }
                     }
                  ]
               },
               "config" : {
                  "router-id" : "5.5.5.4",
                  "openconfig-ospfv2-ext:opaque-lsa-capability" : true,
                  "openconfig-ospfv2-ext:description" : "OSPFv2 Router",
                  "openconfig-ospfv2-ext:enable" : true
               },
               "timers" : {
                  "lsa-generation" : {
                     "state" : {
                        "openconfig-ospfv2-ext:refresh-timer" : 10000,
                        "openconfig-ospfv2-ext:lsa-min-interval-timer" : 5000,
                        "openconfig-ospfv2-ext:lsa-min-arrival-timer" : 1000
                     }
                  },
                  "spf" : {
                     "state" : {
                        "maximum-delay" : 5000,
                        "initial-delay" : 50,
                        "openconfig-ospfv2-ext:throttle-delay" : 0,
                        "openconfig-ospfv2-ext:spf-timer-due" : 0
                     }
                  }
               },
               "state" : {
                  "openconfig-ospfv2-ext:area-count" : 2,
                  "openconfig-ospfv2-ext:last-spf-duration" : 358,
                  "openconfig-ospfv2-ext:hold-time-multiplier" : 1,
                  "openconfig-ospfv2-ext:last-spf-execution-time" : "1187616",
                  "openconfig-ospfv2-ext:opaque-lsa-count" : 0,
                  "router-id" : "5.5.5.4",
                  "openconfig-ospfv2-ext:abr-type" : "openconfig-ospfv2-ext:CISCO",
                  "openconfig-ospfv2-ext:external-lsa-checksum" : "0x00016bcc",
                  "openconfig-ospfv2-ext:opaque-lsa-checksum" : "0x00000000",
                  "openconfig-ospfv2-ext:write-multiplier" : 20,
                  "openconfig-ospfv2-ext:opaque-lsa-capability" : true,
                  "openconfig-ospfv2-ext:external-lsa-count" : 2
               }
            },
            "areas" : {
               "area" : [
                  {
                     "lsdb" : {
                        "lsa-types" : {
                           "lsa-type" : [
                              {
                                 "type" : "openconfig-ospf-types:AS_EXTERNAL_LSA",
                                 "state" : {
                                    "type" : "openconfig-ospf-types:AS_EXTERNAL_LSA"
                                 },
                                 "lsas" : {
                                    "openconfig-ospfv2-ext:lsa-ext" : [
                                       {
                                          "advertising-router" : "5.5.5.4",
                                          "as-external-lsa" : {
                                             "types-of-service" : {
                                                "type-of-service" : [
                                                   {
                                                      "tos" : 0,
                                                      "state" : {
                                                         "forwarding-address" : "0.0.0.0",
                                                         "external-route-tag" : 0,
                                                         "metric" : 20
                                                      }
                                                   }
                                                ]
                                             },
                                             "state" : {
                                                "metric-type" : "TYPE_2",
                                                "mask" : 20
                                             }
                                          },
                                          "link-state-id" : "10.193.80.0",
                                          "state" : {
                                             "checksum" : 50393,
                                             "length" : 36,
                                             "flags" : 11,
                                             "display-sequence-number" : "80000001",
                                             "option" : 2,
                                             "option-expanded" : "*||||||E|",
                                             "age" : 1227
                                          }
                                       },
                                       {
                                          "advertising-router" : "6.6.6.5",
                                          "as-external-lsa" : {
                                             "types-of-service" : {
                                                "type-of-service" : [
                                                   {
                                                      "tos" : 0,
                                                      "state" : {
                                                         "forwarding-address" : "0.0.0.0",
                                                         "external-route-tag" : 0,
                                                         "metric" : 20
                                                      }
                                                   }
                                                ]
                                             },
                                             "state" : {
                                                "metric-type" : "TYPE_2",
                                                "mask" : 20
                                             }
                                          },
                                          "link-state-id" : "10.193.80.0",
                                          "state" : {
                                             "checksum" : 42739,
                                             "length" : 36,
                                             "flags" : 6,
                                             "display-sequence-number" : "80000001",
                                             "option" : 2,
                                             "option-expanded" : "*||||||E|",
                                             "age" : 1228
                                          }
                                       }
                                    ]
                                 }
                              },
                              {
                                 "type" : "openconfig-ospf-types:NETWORK_LSA",
                                 "state" : {
                                    "type" : "openconfig-ospf-types:NETWORK_LSA"
                                 },
                                 "lsas" : {
                                    "openconfig-ospfv2-ext:lsa-ext" : [
                                       {
                                          "advertising-router" : "6.6.6.5",
                                          "link-state-id" : "32.1.0.1",
                                          "state" : {
                                             "checksum" : 51756,
                                             "length" : 32,
                                             "flags" : 6,
                                             "display-sequence-number" : "80000001",
                                             "option" : 2,
                                             "option-expanded" : "*|-|-|-|-|-|E|-",
                                             "age" : 1198
                                          },
                                          "network-lsa" : {
                                             "state" : {
                                                "network-mask" : 24
                                             }
                                          }
                                       }
                                    ]
                                 }
                              },
                              {
                                 "type" : "openconfig-ospf-types:OSPFV2_LINK_SCOPE_OPAQUE_LSA"
                              },
                              {
                                 "type" : "openconfig-ospf-types:ROUTER_LSA",
                                 "state" : {
                                    "type" : "openconfig-ospf-types:ROUTER_LSA"
                                 },
                                 "lsas" : {
                                    "openconfig-ospfv2-ext:lsa-ext" : [
                                       {
                                          "advertising-router" : "5.5.5.4",
                                          "link-state-id" : "5.5.5.4",
                                          "state" : {
                                             "checksum" : 5321,
                                             "length" : 36,
                                             "flags" : 3,
                                             "display-sequence-number" : "80000005",
                                             "option" : 2,
                                             "option-expanded" : "*||||||E|",
                                             "age" : 1197
                                          },
                                          "router-lsa" : {
                                             "link-informations" : {
                                                "link-information" : [
                                                   {
                                                      "state" : {
                                                         "metric" : 4,
                                                         "type" : "openconfig-ospf-types:ROUTER_LSA_TRANSIT_NETWORK",
                                                         "number-tos-metrics" : 0
                                                      }
                                                   }
                                                ]
                                             },
                                             "types-of-service" : {
                                                "type-of-service" : [
                                                   {
                                                      "tos" : 0,
                                                      "state" : {
                                                         "metric" : 4
                                                      }
                                                   }
                                                ]
                                             },
                                             "state" : {
                                                "flags-description" : " : ABR ASBR",
                                                "flags" : 3,
                                                "number-links" : 1
                                             }
                                          }
                                       },
                                       {
                                          "advertising-router" : "6.6.6.5",
                                          "link-state-id" : "6.6.6.5",
                                          "state" : {
                                             "checksum" : 47900,
                                             "length" : 36,
                                             "flags" : 6,
                                             "display-sequence-number" : "80000004",
                                             "option" : 2,
                                             "option-expanded" : "*||||||E|",
                                             "age" : 1197
                                          },
                                          "router-lsa" : {
                                             "link-informations" : {
                                                "link-information" : [
                                                   {
                                                      "state" : {
                                                         "metric" : 4,
                                                         "type" : "openconfig-ospf-types:ROUTER_LSA_TRANSIT_NETWORK",
                                                         "number-tos-metrics" : 0
                                                      }
                                                   }
                                                ]
                                             },
                                             "types-of-service" : {
                                                "type-of-service" : [
                                                   {
                                                      "tos" : 0,
                                                      "state" : {
                                                         "metric" : 4
                                                      }
                                                   }
                                                ]
                                             },
                                             "state" : {
                                                "flags-description" : " : ABR ASBR",
                                                "flags" : 3,
                                                "number-links" : 1
                                             }
                                          }
                                       }
                                    ]
                                 }
                              },
                              {
                                 "type" : "openconfig-ospf-types:SUMMARY_ASBR_LSA",
                                 "state" : {
                                    "type" : "openconfig-ospf-types:SUMMARY_ASBR_LSA"
                                 }
                              },
                              {
                                 "type" : "openconfig-ospf-types:SUMMARY_IP_NETWORK_LSA",
                                 "state" : {
                                    "type" : "openconfig-ospf-types:SUMMARY_IP_NETWORK_LSA"
                                 },
                                 "lsas" : {
                                    "openconfig-ospfv2-ext:lsa-ext" : [
                                       {
                                          "advertising-router" : "6.6.6.5",
                                          "link-state-id" : "21.1.0.0",
                                          "summary-lsa" : {
                                             "types-of-service" : {
                                                "type-of-service" : [
                                                   {
                                                      "tos" : 0,
                                                      "state" : {
                                                         "metric" : 10
                                                      }
                                                   }
                                                ]
                                             },
                                             "state" : {
                                                "network-mask" : 24
                                             }
                                          },
                                          "state" : {
                                             "checksum" : 803,
                                             "length" : 28,
                                             "flags" : 6,
                                             "display-sequence-number" : "80000001",
                                             "option" : 2,
                                             "option-expanded" : "*|-|-|-|-|-|E|-",
                                             "age" : 1228
                                          }
                                       },
                                       {
                                          "advertising-router" : "5.5.5.4",
                                          "link-state-id" : "41.1.0.0",
                                          "summary-lsa" : {
                                             "types-of-service" : {
                                                "type-of-service" : [
                                                   {
                                                      "tos" : 0,
                                                      "state" : {
                                                         "metric" : 10
                                                      }
                                                   }
                                                ]
                                             },
                                             "state" : {
                                                "network-mask" : 24
                                             }
                                          },
                                          "state" : {
                                             "checksum" : 7417,
                                             "length" : 28,
                                             "flags" : 11,
                                             "display-sequence-number" : "80000001",
                                             "option" : 2,
                                             "option-expanded" : "*|-|-|-|-|-|E|-",
                                             "age" : 1227
                                          }
                                       }
                                    ]
                                 }
                              }
                           ]
                        },
                        "state" : {
                           "identifier" : "0.0.0.0"
                        }
                     },
                     "identifier" : "0.0.0.0",
                     "interfaces" : {
                        "interface" : [
                           {
                              "openconfig-ospfv2-ext:message-statistics" : {
                                 "state" : {
                                    "ls-update-transmit" : 4,
                                    "ls-request-receive" : 1,
                                    "ls-acknowledge-receive" : 2,
                                    "ls-acknowledge-transmit" : 2,
                                    "db-description-transmit" : 3,
                                    "db-description-receive" : 2,
                                    "hello-transmit" : 124,
                                    "hello-receive" : 124,
                                    "ls-request-transmit" : 1,
                                    "ls-update-receive" : 3
                                 }
                              },
                              "openconfig-ospfv2-ext:neighbours" : {
                                 "neighbour" : [
                                    {
                                       "neighbor-id" : "6.6.6.5",
                                       "neighbor-address" : "32.1.0.1",
                                       "state" : {
                                          "priority" : 1,
                                          "option-value" : 66,
                                          "gr-helper-status" : "None",
                                          "state-changes" : 6,
                                          "backup-designated-router" : "32.1.0.2",
                                          "adjacency-state" : "openconfig-ospf-types:FULL",
                                          "neighbor-id" : "6.6.6.5",
                                          "optional-capabilities" : "*|O|-|-|-|-|E|-",
                                          "thread-inactivity-timer" : true,
                                          "thread-ls-request-retransmission" : true,
                                          "thread-ls-update-retransmission" : true,
                                          "retransmit-summary-queue-length" : 0,
                                          "area-id" : "0.0.0.0",
                                          "gr-active-helper" : false,
                                          "interface-address" : "32.1.0.2",
                                          "neighbor-address" : "32.1.0.1",
                                          "last-established-time" : "1197595",
                                          "designated-router" : "32.1.0.1",
                                          "database-summary-queue-length" : 0,
                                          "interface-name" : "Vlan3719",
                                          "link-state-request-queue-length" : 0,
                                          "dead-time" : "32410"
                                       }
                                    }
                                 ]
                              },
                              "id" : "Vlan3719",
                              "timers" : {
                                 "state" : {
                                    "retransmission-interval" : 5,
                                    "dead-interval" : 40,
                                    "openconfig-ospfv2-ext:wait-time" : 40,
                                    "hello-interval" : 10000,
                                    "openconfig-ospfv2-ext:hello-due" : 2409
                                 }
                              },
                              "state" : {
                                 "priority" : 1,
                                 "openconfig-ospfv2-ext:member-of-ospf-all-routers" : true,
                                 "openconfig-ospfv2-ext:index" : 310,
                                 "openconfig-ospfv2-ext:mtu" : 9100,
                                 "openconfig-ospfv2-ext:if-flags" : "<UP,BROADCAST,RUNNING,MULTICAST>",
                                 "openconfig-ospfv2-ext:adjacency-count" : 1,
                                 "openconfig-ospfv2-ext:address-len" : 24,
                                 "openconfig-ospfv2-ext:ospf-enable" : true,
                                 "openconfig-ospfv2-ext:cost" : 4,
                                 "openconfig-ospfv2-ext:address" : "32.1.0.2",
                                 "id" : "Vlan3719",
                                 "openconfig-ospfv2-ext:adjacency-status" : "Backup",
                                 "openconfig-ospfv2-ext:neighbor-count" : 1,
                                 "openconfig-ospfv2-ext:backup-designated-router-id" : "5.5.5.4",
                                 "openconfig-ospfv2-ext:member-of-ospf-designated-routers" : true,
                                 "openconfig-ospfv2-ext:router-id" : "5.5.5.4",
                                 "openconfig-ospfv2-ext:broadcast-address" : "32.1.0.255",
                                 "openconfig-ospfv2-ext:bandwidth" : 25000,
                                 "openconfig-ospfv2-ext:area-id" : "0.0.0.0",
                                 "openconfig-ospfv2-ext:ospf-interface-type" : "Broadcast",
                                 "openconfig-ospfv2-ext:backup-designated-router-address" : "32.1.0.2",
                                 "network-type" : "openconfig-ospf-types:BROADCAST_NETWORK",
                                 "openconfig-ospfv2-ext:transmit-delay" : 1,
                                 "openconfig-ospfv2-ext:operational-state" : "Up"
                              }
                           }
                        ]
                     },
                     "config" : {
                        "identifier" : "0.0.0.0"
                     },
                     "openconfig-ospfv2-ext:networks" : {
                        "network" : [
                           {
                              "address-prefix" : "31.1.0.0/24",
                              "config" : {
                                 "address-prefix" : "31.1.0.0/24",
                                 "description" : "Network prefix"
                              },
                              "state" : {
                                 "address-prefix" : "31.1.0.0/24",
                                 "description" : "Network prefix"
                              }
                           },
                           {
                              "address-prefix" : "32.1.0.0/24",
                              "config" : {
                                 "address-prefix" : "32.1.0.0/24",
                                 "description" : "Network prefix"
                              },
                              "state" : {
                                 "address-prefix" : "32.1.0.0/24",
                                 "description" : "Network prefix"
                              }
                           }
                        ]
                     },
                     "state" : {
                        "openconfig-ospfv2-ext:summary-lsa-count" : 2,
                        "openconfig-ospfv2-ext:nssa-lsa-checksum" : "0x00000000",
                        "openconfig-ospfv2-ext:authentication-type" : "no",
                        "openconfig-ospfv2-ext:asbr-summary-lsa-count" : 0,
                        "openconfig-ospfv2-ext:adjacency-count" : 1,
                        "openconfig-ospfv2-ext:opaque-area-lsa-count" : 0,
                        "openconfig-ospfv2-ext:opaque-area-lsa-checksum" : "0x00000000",
                        "openconfig-ospfv2-ext:lsa-count" : 5,
                        "openconfig-ospfv2-ext:opaque-link-lsa-checksum" : "0x00000000",
                        "openconfig-ospfv2-ext:spf-execution-count" : 6,
                        "openconfig-ospfv2-ext:interface-count" : 1,
                        "openconfig-ospfv2-ext:network-lsa-count" : 1,
                        "openconfig-ospfv2-ext:asbr-summary-lsa-checksum" : "0x00000000",
                        "openconfig-ospfv2-ext:router-lsa-checksum" : "0x0000cfe5",
                        "openconfig-ospfv2-ext:router-lsa-count" : 2,
                        "openconfig-ospfv2-ext:nssa-lsa-count" : 0,
                        "openconfig-ospfv2-ext:active-interface-count" : 1,
                        "openconfig-ospfv2-ext:summary-lsa-checksum" : "0x0000201c",
                        "openconfig-ospfv2-ext:network-lsa-checksum" : "0x0000ca2c",
                        "openconfig-ospfv2-ext:opaque-link-lsa-count" : 0
                     }
                  },
                  {
                     "lsdb" : {
                        "lsa-types" : {
                           "lsa-type" : [
                              {
                                 "type" : "openconfig-ospf-types:AS_EXTERNAL_LSA",
                                 "state" : {
                                    "type" : "openconfig-ospf-types:AS_EXTERNAL_LSA"
                                 },
                                 "lsas" : {
                                    "openconfig-ospfv2-ext:lsa-ext" : [
                                       {
                                          "advertising-router" : "5.5.5.4",
                                          "as-external-lsa" : {
                                             "types-of-service" : {
                                                "type-of-service" : [
                                                   {
                                                      "tos" : 0,
                                                      "state" : {
                                                         "forwarding-address" : "0.0.0.0",
                                                         "external-route-tag" : 0,
                                                         "metric" : 20
                                                      }
                                                   }
                                                ]
                                             },
                                             "state" : {
                                                "metric-type" : "TYPE_2",
                                                "mask" : 20
                                             }
                                          },
                                          "link-state-id" : "10.193.80.0",
                                          "state" : {
                                             "checksum" : 50393,
                                             "length" : 36,
                                             "flags" : 11,
                                             "display-sequence-number" : "80000001",
                                             "option" : 2,
                                             "option-expanded" : "*||||||E|",
                                             "age" : 1227
                                          }
                                       },
                                       {
                                          "advertising-router" : "6.6.6.5",
                                          "as-external-lsa" : {
                                             "types-of-service" : {
                                                "type-of-service" : [
                                                   {
                                                      "tos" : 0,
                                                      "state" : {
                                                         "forwarding-address" : "0.0.0.0",
                                                         "external-route-tag" : 0,
                                                         "metric" : 20
                                                      }
                                                   }
                                                ]
                                             },
                                             "state" : {
                                                "metric-type" : "TYPE_2",
                                                "mask" : 20
                                             }
                                          },
                                          "link-state-id" : "10.193.80.0",
                                          "state" : {
                                             "checksum" : 42739,
                                             "length" : 36,
                                             "flags" : 6,
                                             "display-sequence-number" : "80000001",
                                             "option" : 2,
                                             "option-expanded" : "*||||||E|",
                                             "age" : 1228
                                          }
                                       }
                                    ]
                                 }
                              },
                              {
                                 "type" : "openconfig-ospf-types:NETWORK_LSA",
                                 "state" : {
                                    "type" : "openconfig-ospf-types:NETWORK_LSA"
                                 },
                                 "lsas" : {
                                    "openconfig-ospfv2-ext:lsa-ext" : [
                                       {
                                          "advertising-router" : "5.5.5.4",
                                          "link-state-id" : "41.1.0.2",
                                          "state" : {
                                             "checksum" : 1017,
                                             "length" : 32,
                                             "flags" : 3,
                                             "display-sequence-number" : "80000001",
                                             "option" : 2,
                                             "option-expanded" : "*|-|-|-|-|-|E|-",
                                             "age" : 1198
                                          },
                                          "network-lsa" : {
                                             "state" : {
                                                "network-mask" : 24
                                             }
                                          }
                                       }
                                    ]
                                 }
                              },
                              {
                                 "type" : "openconfig-ospf-types:OSPFV2_LINK_SCOPE_OPAQUE_LSA"
                              },
                              {
                                 "type" : "openconfig-ospf-types:ROUTER_LSA",
                                 "state" : {
                                    "type" : "openconfig-ospf-types:ROUTER_LSA"
                                 },
                                 "lsas" : {
                                    "openconfig-ospfv2-ext:lsa-ext" : [
                                       {
                                          "advertising-router" : "3.3.3.2",
                                          "link-state-id" : "3.3.3.2",
                                          "state" : {
                                             "checksum" : 53771,
                                             "length" : 36,
                                             "flags" : 6,
                                             "display-sequence-number" : "80000002",
                                             "option" : 0,
                                             "option-expanded" : "*|||||||-",
                                             "age" : 1189
                                          },
                                          "router-lsa" : {
                                             "link-informations" : {
                                                "link-information" : [
                                                   {
                                                      "state" : {
                                                         "metric" : 10,
                                                         "type" : "openconfig-ospf-types:ROUTER_LSA_TRANSIT_NETWORK",
                                                         "number-tos-metrics" : 0
                                                      }
                                                   }
                                                ]
                                             },
                                             "types-of-service" : {
                                                "type-of-service" : [
                                                   {
                                                      "tos" : 0,
                                                      "state" : {
                                                         "metric" : 10
                                                      }
                                                   }
                                                ]
                                             },
                                             "state" : {
                                                "flags-description" : " :",
                                                "flags" : 0,
                                                "number-links" : 1
                                             }
                                          }
                                       },
                                       {
                                          "advertising-router" : "5.5.5.4",
                                          "link-state-id" : "5.5.5.4",
                                          "state" : {
                                             "checksum" : 12693,
                                             "length" : 36,
                                             "flags" : 3,
                                             "display-sequence-number" : "80000003",
                                             "option" : 2,
                                             "option-expanded" : "*||||||E|",
                                             "age" : 1198
                                          },
                                          "router-lsa" : {
                                             "link-informations" : {
                                                "link-information" : [
                                                   {
                                                      "state" : {
                                                         "metric" : 10,
                                                         "type" : "openconfig-ospf-types:ROUTER_LSA_TRANSIT_NETWORK",
                                                         "number-tos-metrics" : 0
                                                      }
                                                   }
                                                ]
                                             },
                                             "types-of-service" : {
                                                "type-of-service" : [
                                                   {
                                                      "tos" : 0,
                                                      "state" : {
                                                         "metric" : 10
                                                      }
                                                   }
                                                ]
                                             },
                                             "state" : {
                                                "flags-description" : " : ABR ASBR",
                                                "flags" : 3,
                                                "number-links" : 1
                                             }
                                          }
                                       }
                                    ]
                                 }
                              },
                              {
                                 "type" : "openconfig-ospf-types:SUMMARY_ASBR_LSA",
                                 "state" : {
                                    "type" : "openconfig-ospf-types:SUMMARY_ASBR_LSA"
                                 },
                                 "lsas" : {
                                    "openconfig-ospfv2-ext:lsa-ext" : [
                                       {
                                          "advertising-router" : "5.5.5.4",
                                          "link-state-id" : "6.6.6.5",
                                          "summary-lsa" : {
                                             "types-of-service" : {
                                                "type-of-service" : [
                                                   {
                                                      "tos" : 0,
                                                      "state" : {
                                                         "metric" : 4
                                                      }
                                                   }
                                                ]
                                             },
                                             "state" : {
                                                "network-mask" : 0
                                             }
                                          },
                                          "state" : {
                                             "checksum" : 59716,
                                             "length" : 28,
                                             "flags" : 11,
                                             "display-sequence-number" : "80000001",
                                             "option" : 2,
                                             "option-expanded" : "*|-|-|-|-|-|E|-",
                                             "age" : 1187
                                          }
                                       }
                                    ]
                                 }
                              },
                              {
                                 "type" : "openconfig-ospf-types:SUMMARY_IP_NETWORK_LSA",
                                 "state" : {
                                    "type" : "openconfig-ospf-types:SUMMARY_IP_NETWORK_LSA"
                                 },
                                 "lsas" : {
                                    "openconfig-ospfv2-ext:lsa-ext" : [
                                       {
                                          "advertising-router" : "5.5.5.4",
                                          "link-state-id" : "21.1.0.0",
                                          "summary-lsa" : {
                                             "types-of-service" : {
                                                "type-of-service" : [
                                                   {
                                                      "tos" : 0,
                                                      "state" : {
                                                         "metric" : 14
                                                      }
                                                   }
                                                ]
                                             },
                                             "state" : {
                                                "network-mask" : 24
                                             }
                                          },
                                          "state" : {
                                             "checksum" : 18908,
                                             "length" : 28,
                                             "flags" : 11,
                                             "display-sequence-number" : "80000001",
                                             "option" : 2,
                                             "option-expanded" : "*|-|-|-|-|-|E|-",
                                             "age" : 1187
                                          }
                                       },
                                       {
                                          "advertising-router" : "5.5.5.4",
                                          "link-state-id" : "32.1.0.0",
                                          "summary-lsa" : {
                                             "types-of-service" : {
                                                "type-of-service" : [
                                                   {
                                                      "tos" : 0,
                                                      "state" : {
                                                         "metric" : 4
                                                      }
                                                   }
                                                ]
                                             },
                                             "state" : {
                                                "network-mask" : 24
                                             }
                                          },
                                          "state" : {
                                             "checksum" : 21967,
                                             "length" : 28,
                                             "flags" : 11,
                                             "display-sequence-number" : "80000001",
                                             "option" : 2,
                                             "option-expanded" : "*|-|-|-|-|-|E|-",
                                             "age" : 1229
                                          }
                                       }
                                    ]
                                 }
                              }
                           ]
                        },
                        "state" : {
                           "identifier" : "0.0.0.2"
                        }
                     },
                     "identifier" : "0.0.0.2",
                     "interfaces" : {
                        "interface" : [
                           {
                              "openconfig-ospfv2-ext:message-statistics" : {
                                 "state" : {
                                    "ls-update-transmit" : 3,
                                    "ls-request-receive" : 0,
                                    "ls-acknowledge-receive" : 3,
                                    "ls-acknowledge-transmit" : 2,
                                    "db-description-transmit" : 2,
                                    "db-description-receive" : 6,
                                    "hello-transmit" : 124,
                                    "hello-receive" : 125,
                                    "ls-request-transmit" : 1,
                                    "ls-update-receive" : 2
                                 }
                              },
                              "openconfig-ospfv2-ext:neighbours" : {
                                 "neighbour" : [
                                    {
                                       "neighbor-id" : "3.3.3.2",
                                       "neighbor-address" : "41.1.0.1",
                                       "state" : {
                                          "priority" : 0,
                                          "option-value" : 66,
                                          "gr-helper-status" : "None",
                                          "state-changes" : 6,
                                          "backup-designated-router" : "0.0.0.0",
                                          "adjacency-state" : "openconfig-ospf-types:FULL",
                                          "neighbor-id" : "6.6.6.5",
                                          "optional-capabilities" : "*|O|-|-|-|-|E|-",
                                          "thread-inactivity-timer" : true,
                                          "thread-ls-request-retransmission" : true,
                                          "thread-ls-update-retransmission" : true,
                                          "retransmit-summary-queue-length" : 0,
                                          "area-id" : "0.0.0.2",
                                          "gr-active-helper" : false,
                                          "interface-address" : "41.1.0.2",
                                          "neighbor-address" : "41.1.0.1",
                                          "last-established-time" : "1198302",
                                          "designated-router" : "41.1.0.2",
                                          "database-summary-queue-length" : 0,
                                          "interface-name" : "Ethernet120",
                                          "link-state-request-queue-length" : 0,
                                          "dead-time" : "34872"
                                       }
                                    }
                                 ]
                              },
                              "id" : "Ethernet120",
                              "timers" : {
                                 "state" : {
                                    "retransmission-interval" : 5,
                                    "dead-interval" : 40,
                                    "openconfig-ospfv2-ext:wait-time" : 40,
                                    "hello-interval" : 10000,
                                    "openconfig-ospfv2-ext:hello-due" : 1711
                                 }
                              },
                              "state" : {
                                 "priority" : 1,
                                 "openconfig-ospfv2-ext:member-of-ospf-all-routers" : true,
                                 "openconfig-ospfv2-ext:index" : 299,
                                 "openconfig-ospfv2-ext:mtu" : 9100,
                                 "openconfig-ospfv2-ext:if-flags" : "<UP,BROADCAST,RUNNING,MULTICAST>",
                                 "openconfig-ospfv2-ext:adjacency-count" : 1,
                                 "openconfig-ospfv2-ext:address-len" : 24,
                                 "openconfig-ospfv2-ext:ospf-enable" : true,
                                 "openconfig-ospfv2-ext:cost" : 10,
                                 "openconfig-ospfv2-ext:address" : "41.1.0.2",
                                 "id" : "Ethernet120",
                                 "openconfig-ospfv2-ext:adjacency-status" : "DR",
                                 "openconfig-ospfv2-ext:member-of-ospf-designated-routers" : true,
                                 "openconfig-ospfv2-ext:neighbor-count" : 1,
                                 "openconfig-ospfv2-ext:router-id" : "5.5.5.4",
                                 "openconfig-ospfv2-ext:broadcast-address" : "41.1.0.255",
                                 "openconfig-ospfv2-ext:bandwidth" : 10000,
                                 "openconfig-ospfv2-ext:area-id" : "0.0.0.2",
                                 "openconfig-ospfv2-ext:ospf-interface-type" : "Broadcast",
                                 "network-type" : "openconfig-ospf-types:BROADCAST_NETWORK",
                                 "openconfig-ospfv2-ext:transmit-delay" : 1,
                                 "openconfig-ospfv2-ext:operational-state" : "Up"
                              }
                           }
                        ]
                     },
                     "config" : {
                        "identifier" : "0.0.0.2"
                     },
                     "openconfig-ospfv2-ext:networks" : {
                        "network" : [
                           {
                              "address-prefix" : "41.1.0.0/24",
                              "config" : {
                                 "address-prefix" : "41.1.0.0/24",
                                 "description" : "Network prefix"
                              },
                              "state" : {
                                 "address-prefix" : "41.1.0.0/24",
                                 "description" : "Network prefix"
                              }
                           }
                        ]
                     },
                     "state" : {
                        "openconfig-ospfv2-ext:summary-lsa-count" : 2,
                        "openconfig-ospfv2-ext:nssa-lsa-checksum" : "0x00000000",
                        "openconfig-ospfv2-ext:authentication-type" : "no",
                        "openconfig-ospfv2-ext:asbr-summary-lsa-count" : 1,
                        "openconfig-ospfv2-ext:adjacency-count" : 1,
                        "openconfig-ospfv2-ext:opaque-area-lsa-count" : 0,
                        "openconfig-ospfv2-ext:opaque-area-lsa-checksum" : "0x00000000",
                        "openconfig-ospfv2-ext:lsa-count" : 6,
                        "openconfig-ospfv2-ext:opaque-link-lsa-checksum" : "0x00000000",
                        "openconfig-ospfv2-ext:virtual-link-adjacency-count" : 0,
                        "openconfig-ospfv2-ext:spf-execution-count" : 6,
                        "openconfig-ospfv2-ext:interface-count" : 1,
                        "openconfig-ospfv2-ext:network-lsa-count" : 1,
                        "openconfig-ospfv2-ext:asbr-summary-lsa-checksum" : "0x0000e944",
                        "openconfig-ospfv2-ext:router-lsa-checksum" : "0x000103a0",
                        "openconfig-ospfv2-ext:router-lsa-count" : 2,
                        "openconfig-ospfv2-ext:nssa-lsa-count" : 0,
                        "openconfig-ospfv2-ext:active-interface-count" : 1,
                        "openconfig-ospfv2-ext:shortcut" : "openconfig-ospfv2-ext:DEFAULT",
                        "openconfig-ospfv2-ext:summary-lsa-checksum" : "0x00009fab",
                        "openconfig-ospfv2-ext:network-lsa-checksum" : "0x000003f9",
                        "openconfig-ospfv2-ext:opaque-link-lsa-count" : 0
                     }
                  }
               ]
            }
         },
         "name" : "ospfv2",
         "config" : {
            "identifier" : "openconfig-policy-types:OSPF",
            "name" : "ospfv2"
         },
         "state" : {
            "identifier" : "openconfig-policy-types:OSPF",
            "name" : "ospfv2"
         }
      }
   ]
}
            `),
		appRootType: reflect.TypeOf(ocbinds.OpenconfigNetworkInstance_NetworkInstances{}),
		queryParams: QueryParams{
			depthEnabled:  false,
			curDepth:      1,
			content:       QUERY_CONTENT_ALL,
			fields:        []string{},
			fieldsFillAll: false,
		},
		prunedPayload: []byte(`
{
   "protocol" : [
      {
         "identifier" : "openconfig-policy-types:OSPF",
         "ospfv2" : {
            "openconfig-ospfv2-ext:route-tables" : {
               "route-table" : [
                  {
                     "routes" : {
                        "route" : [
                           {
                              "next-hops" : {
                                 "next-hop" : [
                                    {
                                       "area-id" : "0.0.0.0",
                                       "address" : "32.1.0.1",
                                       "state" : {
                                          "area-id" : "0.0.0.0",
                                          "address" : "32.1.0.1",
                                          "out-interface" : "Vlan3719"
                                       },
                                       "out-interface" : "Vlan3719"
                                    }
                                 ]
                              },
                              "state" : {
                                 "cost" : 4,
                                 "sub-type" : "openconfig-ospfv2-ext:EXTERNAL_ROUTE_TYPE_2",
                                 "type" : "openconfig-ospfv2-ext:EXTERNAL_ROUTE",
                                 "prefix" : "10.193.80.0/20"
                              },
                              "prefix" : "10.193.80.0/20"
                           }
                        ]
                     },
                     "type" : "openconfig-ospfv2-ext:EXTERNAL_ROUTE_TABLE",
                     "state" : {
                        "type" : "openconfig-ospfv2-ext:EXTERNAL_ROUTE_TABLE"
                     }
                  },
                  {
                     "routes" : {
                        "route" : [
                           {
                              "next-hops" : {
                                 "next-hop" : [
                                    {
                                       "area-id" : "0.0.0.0",
                                       "address" : "32.1.0.1",
                                       "state" : {
                                          "area-id" : "0.0.0.0",
                                          "address" : "32.1.0.1",
                                          "out-interface" : "Vlan3719"
                                       },
                                       "out-interface" : "Vlan3719"
                                    }
                                 ]
                              },
                              "state" : {
                                 "cost" : 14,
                                 "type" : "openconfig-ospfv2-ext:NETWORK_ROUTE",
                                 "inter-area" : true,
                                 "prefix" : "10.193.80.0/20"
                              },
                              "prefix" : "21.1.0.0/24"
                           },
                           {
                              "next-hops" : {
                                 "next-hop" : [
                                    {
                                       "area-id" : "0.0.0.0",
                                       "address" : "0.0.0.0",
                                       "state" : {
                                          "area-id" : "0.0.0.0",
                                          "address" : "32.1.0.1",
                                          "out-interface" : "Vlan3719"
                                       },
                                       "out-interface" : "Vlan3719"
                                    }
                                 ]
                              },
                              "state" : {
                                 "cost" : 4,
                                 "type" : "openconfig-ospfv2-ext:NETWORK_ROUTE",
                                 "prefix" : "10.193.80.0/20"
                              },
                              "prefix" : "32.1.0.0/24"
                           },
                           {
                              "next-hops" : {
                                 "next-hop" : [
                                    {
                                       "area-id" : "0.0.0.2",
                                       "address" : "0.0.0.0",
                                       "state" : {
                                          "area-id" : "0.0.0.0",
                                          "address" : "32.1.0.1",
                                          "out-interface" : "Vlan3719"
                                       },
                                       "out-interface" : "Ethernet120"
                                    }
                                 ]
                              },
                              "state" : {
                                 "cost" : 10,
                                 "type" : "openconfig-ospfv2-ext:NETWORK_ROUTE",
                                 "prefix" : "10.193.80.0/20"
                              },
                              "prefix" : "41.1.0.0/24"
                           }
                        ]
                     },
                     "type" : "openconfig-ospfv2-ext:NETWORK_ROUTE_TABLE",
                     "state" : {
                        "type" : "openconfig-ospfv2-ext:NETWORK_ROUTE_TABLE"
                     }
                  },
                  {
                     "routes" : {
                        "route" : [
                           {
                              "next-hops" : {
                                 "next-hop" : [
                                    {
                                       "area-id" : "0.0.0.0",
                                       "address" : "32.1.0.1",
                                       "state" : {
                                          "area-id" : "0.0.0.0",
                                          "address" : "32.1.0.1",
                                          "out-interface" : "Vlan3719"
                                       },
                                       "out-interface" : "Vlan3719"
                                    }
                                 ]
                              },
                              "state" : {
                                 "router-type" : "openconfig-ospfv2-ext:ABRASBR",
                                 "cost" : 4,
                                 "type" : "openconfig-ospfv2-ext:ROUTER_ROUTE",
                                 "prefix" : "10.193.80.0/20"
                              },
                              "prefix" : "6.6.6.5"
                           }
                        ]
                     },
                     "type" : "openconfig-ospfv2-ext:ROUTER_ROUTE_TABLE",
                     "state" : {
                        "type" : "openconfig-ospfv2-ext:ROUTER_ROUTE_TABLE"
                     }
                  }
               ]
            },
            "global" : {
               "graceful-restart" : {
                  "openconfig-ospfv2-ext:helpers" : {
                     "helper" : [
                        {
                           "neighbour-id" : "5.5.5.4",
                           "config" : {
                              "vrf-name" : "default",
                              "neighbour-id" : "5.5.5.4"
                           },
                           "state" : {
                              "neighbour-id" : "5.5.5.4"
                           }
                        }
                     ]
                  },
                  "config" : {
                     "openconfig-ospfv2-ext:planned-only" : true,
                     "helper-only" : true,
                     "openconfig-ospfv2-ext:supported-grace-time" : 800,
                     "openconfig-ospfv2-ext:strict-lsa-checking" : true
                  },
                  "state" : {
                     "openconfig-ospfv2-ext:planned-only" : true,
                     "helper-only" : true,
                     "openconfig-ospfv2-ext:grace-period" : 0,
                     "openconfig-ospfv2-ext:supported-grace-time" : 800,
                     "openconfig-ospfv2-ext:strict-lsa-checking" : true
                  }
               },
               "inter-area-propagation-policies" : {
                  "openconfig-ospfv2-ext:inter-area-policy" : [
                     {
                        "config" : {
                           "src-area" : "0.0.0.0"
                        },
                        "src-area" : "0.0.0.0",
                        "state" : {
                           "src-area" : "0.0.0.0"
                        }
                     },
                     {
                        "config" : {
                           "src-area" : "0.0.0.2"
                        },
                        "src-area" : "0.0.0.2",
                        "state" : {
                           "src-area" : "0.0.0.2"
                        }
                     }
                  ]
               },
               "config" : {
                  "router-id" : "5.5.5.4",
                  "openconfig-ospfv2-ext:opaque-lsa-capability" : true,
                  "openconfig-ospfv2-ext:description" : "OSPFv2 Router",
                  "openconfig-ospfv2-ext:enable" : true
               },
               "timers" : {
                  "lsa-generation" : {
                     "state" : {
                        "openconfig-ospfv2-ext:refresh-timer" : 10000,
                        "openconfig-ospfv2-ext:lsa-min-interval-timer" : 5000,
                        "openconfig-ospfv2-ext:lsa-min-arrival-timer" : 1000
                     }
                  },
                  "spf" : {
                     "state" : {
                        "maximum-delay" : 5000,
                        "initial-delay" : 50,
                        "openconfig-ospfv2-ext:throttle-delay" : 0,
                        "openconfig-ospfv2-ext:spf-timer-due" : 0
                     }
                  }
               },
               "state" : {
                  "openconfig-ospfv2-ext:area-count" : 2,
                  "openconfig-ospfv2-ext:last-spf-duration" : 358,
                  "openconfig-ospfv2-ext:hold-time-multiplier" : 1,
                  "openconfig-ospfv2-ext:last-spf-execution-time" : "1187616",
                  "openconfig-ospfv2-ext:opaque-lsa-count" : 0,
                  "router-id" : "5.5.5.4",
                  "openconfig-ospfv2-ext:abr-type" : "openconfig-ospfv2-ext:CISCO",
                  "openconfig-ospfv2-ext:external-lsa-checksum" : "0x00016bcc",
                  "openconfig-ospfv2-ext:opaque-lsa-checksum" : "0x00000000",
                  "openconfig-ospfv2-ext:write-multiplier" : 20,
                  "openconfig-ospfv2-ext:opaque-lsa-capability" : true,
                  "openconfig-ospfv2-ext:external-lsa-count" : 2
               }
            },
            "areas" : {
               "area" : [
                  {
                     "lsdb" : {
                        "lsa-types" : {
                           "lsa-type" : [
                              {
                                 "type" : "openconfig-ospf-types:AS_EXTERNAL_LSA",
                                 "state" : {
                                    "type" : "openconfig-ospf-types:AS_EXTERNAL_LSA"
                                 },
                                 "lsas" : {
                                    "openconfig-ospfv2-ext:lsa-ext" : [
                                       {
                                          "advertising-router" : "5.5.5.4",
                                          "as-external-lsa" : {
                                             "types-of-service" : {
                                                "type-of-service" : [
                                                   {
                                                      "tos" : 0,
                                                      "state" : {
                                                         "forwarding-address" : "0.0.0.0",
                                                         "external-route-tag" : 0,
                                                         "metric" : 20
                                                      }
                                                   }
                                                ]
                                             },
                                             "state" : {
                                                "metric-type" : "TYPE_2",
                                                "mask" : 20
                                             }
                                          },
                                          "link-state-id" : "10.193.80.0",
                                          "state" : {
                                             "checksum" : 50393,
                                             "length" : 36,
                                             "flags" : 11,
                                             "display-sequence-number" : "80000001",
                                             "option" : 2,
                                             "option-expanded" : "*||||||E|",
                                             "age" : 1227
                                          }
                                       },
                                       {
                                          "advertising-router" : "6.6.6.5",
                                          "as-external-lsa" : {
                                             "types-of-service" : {
                                                "type-of-service" : [
                                                   {
                                                      "tos" : 0,
                                                      "state" : {
                                                         "forwarding-address" : "0.0.0.0",
                                                         "external-route-tag" : 0,
                                                         "metric" : 20
                                                      }
                                                   }
                                                ]
                                             },
                                             "state" : {
                                                "metric-type" : "TYPE_2",
                                                "mask" : 20
                                             }
                                          },
                                          "link-state-id" : "10.193.80.0",
                                          "state" : {
                                             "checksum" : 42739,
                                             "length" : 36,
                                             "flags" : 6,
                                             "display-sequence-number" : "80000001",
                                             "option" : 2,
                                             "option-expanded" : "*||||||E|",
                                             "age" : 1228
                                          }
                                       }
                                    ]
                                 }
                              },
                              {
                                 "type" : "openconfig-ospf-types:NETWORK_LSA",
                                 "state" : {
                                    "type" : "openconfig-ospf-types:NETWORK_LSA"
                                 },
                                 "lsas" : {
                                    "openconfig-ospfv2-ext:lsa-ext" : [
                                       {
                                          "advertising-router" : "6.6.6.5",
                                          "link-state-id" : "32.1.0.1",
                                          "state" : {
                                             "checksum" : 51756,
                                             "length" : 32,
                                             "flags" : 6,
                                             "display-sequence-number" : "80000001",
                                             "option" : 2,
                                             "option-expanded" : "*|-|-|-|-|-|E|-",
                                             "age" : 1198
                                          },
                                          "network-lsa" : {
                                             "state" : {
                                                "network-mask" : 24
                                             }
                                          }
                                       }
                                    ]
                                 }
                              },
                              {
                                 "type" : "openconfig-ospf-types:OSPFV2_LINK_SCOPE_OPAQUE_LSA"
                              },
                              {
                                 "type" : "openconfig-ospf-types:ROUTER_LSA",
                                 "state" : {
                                    "type" : "openconfig-ospf-types:ROUTER_LSA"
                                 },
                                 "lsas" : {
                                    "openconfig-ospfv2-ext:lsa-ext" : [
                                       {
                                          "advertising-router" : "5.5.5.4",
                                          "link-state-id" : "5.5.5.4",
                                          "state" : {
                                             "checksum" : 5321,
                                             "length" : 36,
                                             "flags" : 3,
                                             "display-sequence-number" : "80000005",
                                             "option" : 2,
                                             "option-expanded" : "*||||||E|",
                                             "age" : 1197
                                          },
                                          "router-lsa" : {
                                             "link-informations" : {
                                                "link-information" : [
                                                   {
                                                      "state" : {
                                                         "metric" : 4,
                                                         "type" : "openconfig-ospf-types:ROUTER_LSA_TRANSIT_NETWORK",
                                                         "number-tos-metrics" : 0
                                                      }
                                                   }
                                                ]
                                             },
                                             "types-of-service" : {
                                                "type-of-service" : [
                                                   {
                                                      "tos" : 0,
                                                      "state" : {
                                                         "metric" : 4
                                                      }
                                                   }
                                                ]
                                             },
                                             "state" : {
                                                "flags-description" : " : ABR ASBR",
                                                "flags" : 3,
                                                "number-links" : 1
                                             }
                                          }
                                       },
                                       {
                                          "advertising-router" : "6.6.6.5",
                                          "link-state-id" : "6.6.6.5",
                                          "state" : {
                                             "checksum" : 47900,
                                             "length" : 36,
                                             "flags" : 6,
                                             "display-sequence-number" : "80000004",
                                             "option" : 2,
                                             "option-expanded" : "*||||||E|",
                                             "age" : 1197
                                          },
                                          "router-lsa" : {
                                             "link-informations" : {
                                                "link-information" : [
                                                   {
                                                      "state" : {
                                                         "metric" : 4,
                                                         "type" : "openconfig-ospf-types:ROUTER_LSA_TRANSIT_NETWORK",
                                                         "number-tos-metrics" : 0
                                                      }
                                                   }
                                                ]
                                             },
                                             "types-of-service" : {
                                                "type-of-service" : [
                                                   {
                                                      "tos" : 0,
                                                      "state" : {
                                                         "metric" : 4
                                                      }
                                                   }
                                                ]
                                             },
                                             "state" : {
                                                "flags-description" : " : ABR ASBR",
                                                "flags" : 3,
                                                "number-links" : 1
                                             }
                                          }
                                       }
                                    ]
                                 }
                              },
                              {
                                 "type" : "openconfig-ospf-types:SUMMARY_ASBR_LSA",
                                 "state" : {
                                    "type" : "openconfig-ospf-types:SUMMARY_ASBR_LSA"
                                 }
                              },
                              {
                                 "type" : "openconfig-ospf-types:SUMMARY_IP_NETWORK_LSA",
                                 "state" : {
                                    "type" : "openconfig-ospf-types:SUMMARY_IP_NETWORK_LSA"
                                 },
                                 "lsas" : {
                                    "openconfig-ospfv2-ext:lsa-ext" : [
                                       {
                                          "advertising-router" : "6.6.6.5",
                                          "link-state-id" : "21.1.0.0",
                                          "summary-lsa" : {
                                             "types-of-service" : {
                                                "type-of-service" : [
                                                   {
                                                      "tos" : 0,
                                                      "state" : {
                                                         "metric" : 10
                                                      }
                                                   }
                                                ]
                                             },
                                             "state" : {
                                                "network-mask" : 24
                                             }
                                          },
                                          "state" : {
                                             "checksum" : 803,
                                             "length" : 28,
                                             "flags" : 6,
                                             "display-sequence-number" : "80000001",
                                             "option" : 2,
                                             "option-expanded" : "*|-|-|-|-|-|E|-",
                                             "age" : 1228
                                          }
                                       },
                                       {
                                          "advertising-router" : "5.5.5.4",
                                          "link-state-id" : "41.1.0.0",
                                          "summary-lsa" : {
                                             "types-of-service" : {
                                                "type-of-service" : [
                                                   {
                                                      "tos" : 0,
                                                      "state" : {
                                                         "metric" : 10
                                                      }
                                                   }
                                                ]
                                             },
                                             "state" : {
                                                "network-mask" : 24
                                             }
                                          },
                                          "state" : {
                                             "checksum" : 7417,
                                             "length" : 28,
                                             "flags" : 11,
                                             "display-sequence-number" : "80000001",
                                             "option" : 2,
                                             "option-expanded" : "*|-|-|-|-|-|E|-",
                                             "age" : 1227
                                          }
                                       }
                                    ]
                                 }
                              }
                           ]
                        },
                        "state" : {
                           "identifier" : "0.0.0.0"
                        }
                     },
                     "identifier" : "0.0.0.0",
                     "interfaces" : {
                        "interface" : [
                           {
                              "openconfig-ospfv2-ext:message-statistics" : {
                                 "state" : {
                                    "ls-update-transmit" : 4,
                                    "ls-request-receive" : 1,
                                    "ls-acknowledge-receive" : 2,
                                    "ls-acknowledge-transmit" : 2,
                                    "db-description-transmit" : 3,
                                    "db-description-receive" : 2,
                                    "hello-transmit" : 124,
                                    "hello-receive" : 124,
                                    "ls-request-transmit" : 1,
                                    "ls-update-receive" : 3
                                 }
                              },
                              "openconfig-ospfv2-ext:neighbours" : {
                                 "neighbour" : [
                                    {
                                       "neighbor-id" : "6.6.6.5",
                                       "neighbor-address" : "32.1.0.1",
                                       "state" : {
                                          "priority" : 1,
                                          "option-value" : 66,
                                          "gr-helper-status" : "None",
                                          "state-changes" : 6,
                                          "backup-designated-router" : "32.1.0.2",
                                          "adjacency-state" : "openconfig-ospf-types:FULL",
                                          "neighbor-id" : "6.6.6.5",
                                          "optional-capabilities" : "*|O|-|-|-|-|E|-",
                                          "thread-inactivity-timer" : true,
                                          "thread-ls-request-retransmission" : true,
                                          "thread-ls-update-retransmission" : true,
                                          "retransmit-summary-queue-length" : 0,
                                          "area-id" : "0.0.0.0",
                                          "gr-active-helper" : false,
                                          "interface-address" : "32.1.0.2",
                                          "neighbor-address" : "32.1.0.1",
                                          "last-established-time" : "1197595",
                                          "designated-router" : "32.1.0.1",
                                          "database-summary-queue-length" : 0,
                                          "interface-name" : "Vlan3719",
                                          "link-state-request-queue-length" : 0,
                                          "dead-time" : "32410"
                                       }
                                    }
                                 ]
                              },
                              "id" : "Vlan3719",
                              "timers" : {
                                 "state" : {
                                    "retransmission-interval" : 5,
                                    "dead-interval" : 40,
                                    "openconfig-ospfv2-ext:wait-time" : 40,
                                    "hello-interval" : 10000,
                                    "openconfig-ospfv2-ext:hello-due" : 2409
                                 }
                              },
                              "state" : {
                                 "priority" : 1,
                                 "openconfig-ospfv2-ext:member-of-ospf-all-routers" : true,
                                 "openconfig-ospfv2-ext:index" : 310,
                                 "openconfig-ospfv2-ext:mtu" : 9100,
                                 "openconfig-ospfv2-ext:if-flags" : "<UP,BROADCAST,RUNNING,MULTICAST>",
                                 "openconfig-ospfv2-ext:adjacency-count" : 1,
                                 "openconfig-ospfv2-ext:address-len" : 24,
                                 "openconfig-ospfv2-ext:ospf-enable" : true,
                                 "openconfig-ospfv2-ext:cost" : 4,
                                 "openconfig-ospfv2-ext:address" : "32.1.0.2",
                                 "id" : "Vlan3719",
                                 "openconfig-ospfv2-ext:adjacency-status" : "Backup",
                                 "openconfig-ospfv2-ext:neighbor-count" : 1,
                                 "openconfig-ospfv2-ext:backup-designated-router-id" : "5.5.5.4",
                                 "openconfig-ospfv2-ext:member-of-ospf-designated-routers" : true,
                                 "openconfig-ospfv2-ext:router-id" : "5.5.5.4",
                                 "openconfig-ospfv2-ext:broadcast-address" : "32.1.0.255",
                                 "openconfig-ospfv2-ext:bandwidth" : 25000,
                                 "openconfig-ospfv2-ext:area-id" : "0.0.0.0",
                                 "openconfig-ospfv2-ext:ospf-interface-type" : "Broadcast",
                                 "openconfig-ospfv2-ext:backup-designated-router-address" : "32.1.0.2",
                                 "network-type" : "openconfig-ospf-types:BROADCAST_NETWORK",
                                 "openconfig-ospfv2-ext:transmit-delay" : 1,
                                 "openconfig-ospfv2-ext:operational-state" : "Up"
                              }
                           }
                        ]
                     },
                     "config" : {
                        "identifier" : "0.0.0.0"
                     },
                     "openconfig-ospfv2-ext:networks" : {
                        "network" : [
                           {
                              "address-prefix" : "31.1.0.0/24",
                              "config" : {
                                 "address-prefix" : "31.1.0.0/24",
                                 "description" : "Network prefix"
                              },
                              "state" : {
                                 "address-prefix" : "31.1.0.0/24",
                                 "description" : "Network prefix"
                              }
                           },
                           {
                              "address-prefix" : "32.1.0.0/24",
                              "config" : {
                                 "address-prefix" : "32.1.0.0/24",
                                 "description" : "Network prefix"
                              },
                              "state" : {
                                 "address-prefix" : "32.1.0.0/24",
                                 "description" : "Network prefix"
                              }
                           }
                        ]
                     },
                     "state" : {
                        "openconfig-ospfv2-ext:summary-lsa-count" : 2,
                        "openconfig-ospfv2-ext:nssa-lsa-checksum" : "0x00000000",
                        "openconfig-ospfv2-ext:authentication-type" : "no",
                        "openconfig-ospfv2-ext:asbr-summary-lsa-count" : 0,
                        "openconfig-ospfv2-ext:adjacency-count" : 1,
                        "openconfig-ospfv2-ext:opaque-area-lsa-count" : 0,
                        "openconfig-ospfv2-ext:opaque-area-lsa-checksum" : "0x00000000",
                        "openconfig-ospfv2-ext:lsa-count" : 5,
                        "openconfig-ospfv2-ext:opaque-link-lsa-checksum" : "0x00000000",
                        "openconfig-ospfv2-ext:spf-execution-count" : 6,
                        "openconfig-ospfv2-ext:interface-count" : 1,
                        "openconfig-ospfv2-ext:network-lsa-count" : 1,
                        "openconfig-ospfv2-ext:asbr-summary-lsa-checksum" : "0x00000000",
                        "openconfig-ospfv2-ext:router-lsa-checksum" : "0x0000cfe5",
                        "openconfig-ospfv2-ext:router-lsa-count" : 2,
                        "openconfig-ospfv2-ext:nssa-lsa-count" : 0,
                        "openconfig-ospfv2-ext:active-interface-count" : 1,
                        "openconfig-ospfv2-ext:summary-lsa-checksum" : "0x0000201c",
                        "openconfig-ospfv2-ext:network-lsa-checksum" : "0x0000ca2c",
                        "openconfig-ospfv2-ext:opaque-link-lsa-count" : 0
                     }
                  },
                  {
                     "lsdb" : {
                        "lsa-types" : {
                           "lsa-type" : [
                              {
                                 "type" : "openconfig-ospf-types:AS_EXTERNAL_LSA",
                                 "state" : {
                                    "type" : "openconfig-ospf-types:AS_EXTERNAL_LSA"
                                 },
                                 "lsas" : {
                                    "openconfig-ospfv2-ext:lsa-ext" : [
                                       {
                                          "advertising-router" : "5.5.5.4",
                                          "as-external-lsa" : {
                                             "types-of-service" : {
                                                "type-of-service" : [
                                                   {
                                                      "tos" : 0,
                                                      "state" : {
                                                         "forwarding-address" : "0.0.0.0",
                                                         "external-route-tag" : 0,
                                                         "metric" : 20
                                                      }
                                                   }
                                                ]
                                             },
                                             "state" : {
                                                "metric-type" : "TYPE_2",
                                                "mask" : 20
                                             }
                                          },
                                          "link-state-id" : "10.193.80.0",
                                          "state" : {
                                             "checksum" : 50393,
                                             "length" : 36,
                                             "flags" : 11,
                                             "display-sequence-number" : "80000001",
                                             "option" : 2,
                                             "option-expanded" : "*||||||E|",
                                             "age" : 1227
                                          }
                                       },
                                       {
                                          "advertising-router" : "6.6.6.5",
                                          "as-external-lsa" : {
                                             "types-of-service" : {
                                                "type-of-service" : [
                                                   {
                                                      "tos" : 0,
                                                      "state" : {
                                                         "forwarding-address" : "0.0.0.0",
                                                         "external-route-tag" : 0,
                                                         "metric" : 20
                                                      }
                                                   }
                                                ]
                                             },
                                             "state" : {
                                                "metric-type" : "TYPE_2",
                                                "mask" : 20
                                             }
                                          },
                                          "link-state-id" : "10.193.80.0",
                                          "state" : {
                                             "checksum" : 42739,
                                             "length" : 36,
                                             "flags" : 6,
                                             "display-sequence-number" : "80000001",
                                             "option" : 2,
                                             "option-expanded" : "*||||||E|",
                                             "age" : 1228
                                          }
                                       }
                                    ]
                                 }
                              },
                              {
                                 "type" : "openconfig-ospf-types:NETWORK_LSA",
                                 "state" : {
                                    "type" : "openconfig-ospf-types:NETWORK_LSA"
                                 },
                                 "lsas" : {
                                    "openconfig-ospfv2-ext:lsa-ext" : [
                                       {
                                          "advertising-router" : "5.5.5.4",
                                          "link-state-id" : "41.1.0.2",
                                          "state" : {
                                             "checksum" : 1017,
                                             "length" : 32,
                                             "flags" : 3,
                                             "display-sequence-number" : "80000001",
                                             "option" : 2,
                                             "option-expanded" : "*|-|-|-|-|-|E|-",
                                             "age" : 1198
                                          },
                                          "network-lsa" : {
                                             "state" : {
                                                "network-mask" : 24
                                             }
                                          }
                                       }
                                    ]
                                 }
                              },
                              {
                                 "type" : "openconfig-ospf-types:OSPFV2_LINK_SCOPE_OPAQUE_LSA"
                              },
                              {
                                 "type" : "openconfig-ospf-types:ROUTER_LSA",
                                 "state" : {
                                    "type" : "openconfig-ospf-types:ROUTER_LSA"
                                 },
                                 "lsas" : {
                                    "openconfig-ospfv2-ext:lsa-ext" : [
                                       {
                                          "advertising-router" : "3.3.3.2",
                                          "link-state-id" : "3.3.3.2",
                                          "state" : {
                                             "checksum" : 53771,
                                             "length" : 36,
                                             "flags" : 6,
                                             "display-sequence-number" : "80000002",
                                             "option" : 0,
                                             "option-expanded" : "*|||||||-",
                                             "age" : 1189
                                          },
                                          "router-lsa" : {
                                             "link-informations" : {
                                                "link-information" : [
                                                   {
                                                      "state" : {
                                                         "metric" : 10,
                                                         "type" : "openconfig-ospf-types:ROUTER_LSA_TRANSIT_NETWORK",
                                                         "number-tos-metrics" : 0
                                                      }
                                                   }
                                                ]
                                             },
                                             "types-of-service" : {
                                                "type-of-service" : [
                                                   {
                                                      "tos" : 0,
                                                      "state" : {
                                                         "metric" : 10
                                                      }
                                                   }
                                                ]
                                             },
                                             "state" : {
                                                "flags-description" : " :",
                                                "flags" : 0,
                                                "number-links" : 1
                                             }
                                          }
                                       },
                                       {
                                          "advertising-router" : "5.5.5.4",
                                          "link-state-id" : "5.5.5.4",
                                          "state" : {
                                             "checksum" : 12693,
                                             "length" : 36,
                                             "flags" : 3,
                                             "display-sequence-number" : "80000003",
                                             "option" : 2,
                                             "option-expanded" : "*||||||E|",
                                             "age" : 1198
                                          },
                                          "router-lsa" : {
                                             "link-informations" : {
                                                "link-information" : [
                                                   {
                                                      "state" : {
                                                         "metric" : 10,
                                                         "type" : "openconfig-ospf-types:ROUTER_LSA_TRANSIT_NETWORK",
                                                         "number-tos-metrics" : 0
                                                      }
                                                   }
                                                ]
                                             },
                                             "types-of-service" : {
                                                "type-of-service" : [
                                                   {
                                                      "tos" : 0,
                                                      "state" : {
                                                         "metric" : 10
                                                      }
                                                   }
                                                ]
                                             },
                                             "state" : {
                                                "flags-description" : " : ABR ASBR",
                                                "flags" : 3,
                                                "number-links" : 1
                                             }
                                          }
                                       }
                                    ]
                                 }
                              },
                              {
                                 "type" : "openconfig-ospf-types:SUMMARY_ASBR_LSA",
                                 "state" : {
                                    "type" : "openconfig-ospf-types:SUMMARY_ASBR_LSA"
                                 },
                                 "lsas" : {
                                    "openconfig-ospfv2-ext:lsa-ext" : [
                                       {
                                          "advertising-router" : "5.5.5.4",
                                          "link-state-id" : "6.6.6.5",
                                          "summary-lsa" : {
                                             "types-of-service" : {
                                                "type-of-service" : [
                                                   {
                                                      "tos" : 0,
                                                      "state" : {
                                                         "metric" : 4
                                                      }
                                                   }
                                                ]
                                             },
                                             "state" : {
                                                "network-mask" : 0
                                             }
                                          },
                                          "state" : {
                                             "checksum" : 59716,
                                             "length" : 28,
                                             "flags" : 11,
                                             "display-sequence-number" : "80000001",
                                             "option" : 2,
                                             "option-expanded" : "*|-|-|-|-|-|E|-",
                                             "age" : 1187
                                          }
                                       }
                                    ]
                                 }
                              },
                              {
                                 "type" : "openconfig-ospf-types:SUMMARY_IP_NETWORK_LSA",
                                 "state" : {
                                    "type" : "openconfig-ospf-types:SUMMARY_IP_NETWORK_LSA"
                                 },
                                 "lsas" : {
                                    "openconfig-ospfv2-ext:lsa-ext" : [
                                       {
                                          "advertising-router" : "5.5.5.4",
                                          "link-state-id" : "21.1.0.0",
                                          "summary-lsa" : {
                                             "types-of-service" : {
                                                "type-of-service" : [
                                                   {
                                                      "tos" : 0,
                                                      "state" : {
                                                         "metric" : 14
                                                      }
                                                   }
                                                ]
                                             },
                                             "state" : {
                                                "network-mask" : 24
                                             }
                                          },
                                          "state" : {
                                             "checksum" : 18908,
                                             "length" : 28,
                                             "flags" : 11,
                                             "display-sequence-number" : "80000001",
                                             "option" : 2,
                                             "option-expanded" : "*|-|-|-|-|-|E|-",
                                             "age" : 1187
                                          }
                                       },
                                       {
                                          "advertising-router" : "5.5.5.4",
                                          "link-state-id" : "32.1.0.0",
                                          "summary-lsa" : {
                                             "types-of-service" : {
                                                "type-of-service" : [
                                                   {
                                                      "tos" : 0,
                                                      "state" : {
                                                         "metric" : 4
                                                      }
                                                   }
                                                ]
                                             },
                                             "state" : {
                                                "network-mask" : 24
                                             }
                                          },
                                          "state" : {
                                             "checksum" : 21967,
                                             "length" : 28,
                                             "flags" : 11,
                                             "display-sequence-number" : "80000001",
                                             "option" : 2,
                                             "option-expanded" : "*|-|-|-|-|-|E|-",
                                             "age" : 1229
                                          }
                                       }
                                    ]
                                 }
                              }
                           ]
                        },
                        "state" : {
                           "identifier" : "0.0.0.2"
                        }
                     },
                     "identifier" : "0.0.0.2",
                     "interfaces" : {
                        "interface" : [
                           {
                              "openconfig-ospfv2-ext:message-statistics" : {
                                 "state" : {
                                    "ls-update-transmit" : 3,
                                    "ls-request-receive" : 0,
                                    "ls-acknowledge-receive" : 3,
                                    "ls-acknowledge-transmit" : 2,
                                    "db-description-transmit" : 2,
                                    "db-description-receive" : 6,
                                    "hello-transmit" : 124,
                                    "hello-receive" : 125,
                                    "ls-request-transmit" : 1,
                                    "ls-update-receive" : 2
                                 }
                              },
                              "openconfig-ospfv2-ext:neighbours" : {
                                 "neighbour" : [
                                    {
                                       "neighbor-id" : "3.3.3.2",
                                       "neighbor-address" : "41.1.0.1",
                                       "state" : {
                                          "priority" : 0,
                                          "option-value" : 66,
                                          "gr-helper-status" : "None",
                                          "state-changes" : 6,
                                          "backup-designated-router" : "0.0.0.0",
                                          "adjacency-state" : "openconfig-ospf-types:FULL",
                                          "neighbor-id" : "6.6.6.5",
                                          "optional-capabilities" : "*|O|-|-|-|-|E|-",
                                          "thread-inactivity-timer" : true,
                                          "thread-ls-request-retransmission" : true,
                                          "thread-ls-update-retransmission" : true,
                                          "retransmit-summary-queue-length" : 0,
                                          "area-id" : "0.0.0.2",
                                          "gr-active-helper" : false,
                                          "interface-address" : "41.1.0.2",
                                          "neighbor-address" : "41.1.0.1",
                                          "last-established-time" : "1198302",
                                          "designated-router" : "41.1.0.2",
                                          "database-summary-queue-length" : 0,
                                          "interface-name" : "Ethernet120",
                                          "link-state-request-queue-length" : 0,
                                          "dead-time" : "34872"
                                       }
                                    }
                                 ]
                              },
                              "id" : "Ethernet120",
                              "timers" : {
                                 "state" : {
                                    "retransmission-interval" : 5,
                                    "dead-interval" : 40,
                                    "openconfig-ospfv2-ext:wait-time" : 40,
                                    "hello-interval" : 10000,
                                    "openconfig-ospfv2-ext:hello-due" : 1711
                                 }
                              },
                              "state" : {
                                 "priority" : 1,
                                 "openconfig-ospfv2-ext:member-of-ospf-all-routers" : true,
                                 "openconfig-ospfv2-ext:index" : 299,
                                 "openconfig-ospfv2-ext:mtu" : 9100,
                                 "openconfig-ospfv2-ext:if-flags" : "<UP,BROADCAST,RUNNING,MULTICAST>",
                                 "openconfig-ospfv2-ext:adjacency-count" : 1,
                                 "openconfig-ospfv2-ext:address-len" : 24,
                                 "openconfig-ospfv2-ext:ospf-enable" : true,
                                 "openconfig-ospfv2-ext:cost" : 10,
                                 "openconfig-ospfv2-ext:address" : "41.1.0.2",
                                 "id" : "Ethernet120",
                                 "openconfig-ospfv2-ext:adjacency-status" : "DR",
                                 "openconfig-ospfv2-ext:member-of-ospf-designated-routers" : true,
                                 "openconfig-ospfv2-ext:neighbor-count" : 1,
                                 "openconfig-ospfv2-ext:router-id" : "5.5.5.4",
                                 "openconfig-ospfv2-ext:broadcast-address" : "41.1.0.255",
                                 "openconfig-ospfv2-ext:bandwidth" : 10000,
                                 "openconfig-ospfv2-ext:area-id" : "0.0.0.2",
                                 "openconfig-ospfv2-ext:ospf-interface-type" : "Broadcast",
                                 "network-type" : "openconfig-ospf-types:BROADCAST_NETWORK",
                                 "openconfig-ospfv2-ext:transmit-delay" : 1,
                                 "openconfig-ospfv2-ext:operational-state" : "Up"
                              }
                           }
                        ]
                     },
                     "config" : {
                        "identifier" : "0.0.0.2"
                     },
                     "openconfig-ospfv2-ext:networks" : {
                        "network" : [
                           {
                              "address-prefix" : "41.1.0.0/24",
                              "config" : {
                                 "address-prefix" : "41.1.0.0/24",
                                 "description" : "Network prefix"
                              },
                              "state" : {
                                 "address-prefix" : "41.1.0.0/24",
                                 "description" : "Network prefix"
                              }
                           }
                        ]
                     },
                     "state" : {
                        "openconfig-ospfv2-ext:summary-lsa-count" : 2,
                        "openconfig-ospfv2-ext:nssa-lsa-checksum" : "0x00000000",
                        "openconfig-ospfv2-ext:authentication-type" : "no",
                        "openconfig-ospfv2-ext:asbr-summary-lsa-count" : 1,
                        "openconfig-ospfv2-ext:adjacency-count" : 1,
                        "openconfig-ospfv2-ext:opaque-area-lsa-count" : 0,
                        "openconfig-ospfv2-ext:opaque-area-lsa-checksum" : "0x00000000",
                        "openconfig-ospfv2-ext:lsa-count" : 6,
                        "openconfig-ospfv2-ext:opaque-link-lsa-checksum" : "0x00000000",
                        "openconfig-ospfv2-ext:virtual-link-adjacency-count" : 0,
                        "openconfig-ospfv2-ext:spf-execution-count" : 6,
                        "openconfig-ospfv2-ext:interface-count" : 1,
                        "openconfig-ospfv2-ext:network-lsa-count" : 1,
                        "openconfig-ospfv2-ext:asbr-summary-lsa-checksum" : "0x0000e944",
                        "openconfig-ospfv2-ext:router-lsa-checksum" : "0x000103a0",
                        "openconfig-ospfv2-ext:router-lsa-count" : 2,
                        "openconfig-ospfv2-ext:nssa-lsa-count" : 0,
                        "openconfig-ospfv2-ext:active-interface-count" : 1,
                        "openconfig-ospfv2-ext:shortcut" : "openconfig-ospfv2-ext:DEFAULT",
                        "openconfig-ospfv2-ext:summary-lsa-checksum" : "0x00009fab",
                        "openconfig-ospfv2-ext:network-lsa-checksum" : "0x000003f9",
                        "openconfig-ospfv2-ext:opaque-link-lsa-count" : 0
                     }
                  }
               ]
            }
         },
         "name" : "ospfv2",
         "config" : {
            "identifier" : "openconfig-policy-types:OSPF",
            "name" : "ospfv2"
         },
         "state" : {
            "identifier" : "openconfig-policy-types:OSPF",
            "name" : "ospfv2"
         }
      }
   ]
}
            `),
	},
	{ // OSPF TroubleShooting Test Case (i.e. no QP)
		tid:        "OSPF Troubleshoot BareBones",
		uri:        "/openconfig-network-instance:network-instances/network-instance[name=default]/protocols/protocol[identifier=OSPF][name=ospfv2]",
		requestUri: "/openconfig-network-instance:network-instances/network-instance[name=default]/protocols/protocol[identifier=OSPF][name=ospfv2]",
		payload: []byte(`
{
   "protocol" : [
      {
         "identifier" : "openconfig-policy-types:OSPF",
         "name" : "ospfv2",
         "config" : {
            "identifier" : "openconfig-policy-types:OSPF",
            "name" : "ospfv2"
         },
         "state" : {
            "identifier" : "openconfig-policy-types:OSPF",
            "name" : "ospfv2"
         }
      }
   ]
}
            `),
		appRootType: reflect.TypeOf(ocbinds.OpenconfigNetworkInstance_NetworkInstances{}),
		queryParams: QueryParams{
			depthEnabled:  false,
			curDepth:      1,
			content:       QUERY_CONTENT_ALL,
			fields:        []string{},
			fieldsFillAll: false,
		},
		prunedPayload: []byte(`
{
   "protocol" : [
      {
         "identifier" : "openconfig-policy-types:OSPF",
         "name" : "ospfv2",
         "config" : {
            "identifier" : "openconfig-policy-types:OSPF",
            "name" : "ospfv2"
         },
         "state" : {
            "identifier" : "openconfig-policy-types:OSPF",
            "name" : "ospfv2"
         }
      }
   ]
}
            `),
	},
	{ // OSPF Identity Test Case (i.e. no QP)
		tid:        "OSPF Troubleshoot No Areas",
		uri:        "/openconfig-network-instance:network-instances/network-instance[name=default]/protocols/protocol[identifier=OSPF][name=ospfv2]",
		requestUri: "/openconfig-network-instance:network-instances/network-instance[name=default]/protocols/protocol[identifier=OSPF][name=ospfv2]",
		payload: []byte(`
{
   "protocol" : [
      {
         "identifier" : "openconfig-policy-types:OSPF",
         "ospfv2" : {
            "openconfig-ospfv2-ext:route-tables" : {
               "route-table" : [
                  {
                     "routes" : {
                        "route" : [
                           {
                              "next-hops" : {
                                 "next-hop" : [
                                    {
                                       "area-id" : "0.0.0.0",
                                       "address" : "32.1.0.1",
                                       "state" : {
                                          "area-id" : "0.0.0.0",
                                          "address" : "32.1.0.1",
                                          "out-interface" : "Vlan3719"
                                       },
                                       "out-interface" : "Vlan3719"
                                    }
                                 ]
                              },
                              "state" : {
                                 "cost" : 4,
                                 "sub-type" : "openconfig-ospfv2-ext:EXTERNAL_ROUTE_TYPE_2",
                                 "type" : "openconfig-ospfv2-ext:EXTERNAL_ROUTE",
                                 "prefix" : "10.193.80.0/20"
                              },
                              "prefix" : "10.193.80.0/20"
                           }
                        ]
                     },
                     "type" : "openconfig-ospfv2-ext:EXTERNAL_ROUTE_TABLE",
                     "state" : {
                        "type" : "openconfig-ospfv2-ext:EXTERNAL_ROUTE_TABLE"
                     }
                  },
                  {
                     "routes" : {
                        "route" : [
                           {
                              "next-hops" : {
                                 "next-hop" : [
                                    {
                                       "area-id" : "0.0.0.0",
                                       "address" : "32.1.0.1",
                                       "state" : {
                                          "area-id" : "0.0.0.0",
                                          "address" : "32.1.0.1",
                                          "out-interface" : "Vlan3719"
                                       },
                                       "out-interface" : "Vlan3719"
                                    }
                                 ]
                              },
                              "state" : {
                                 "cost" : 14,
                                 "type" : "openconfig-ospfv2-ext:NETWORK_ROUTE",
                                 "inter-area" : true,
                                 "prefix" : "10.193.80.0/20"
                              },
                              "prefix" : "21.1.0.0/24"
                           },
                           {
                              "next-hops" : {
                                 "next-hop" : [
                                    {
                                       "area-id" : "0.0.0.0",
                                       "address" : "0.0.0.0",
                                       "state" : {
                                          "area-id" : "0.0.0.0",
                                          "address" : "32.1.0.1",
                                          "out-interface" : "Vlan3719"
                                       },
                                       "out-interface" : "Vlan3719"
                                    }
                                 ]
                              },
                              "state" : {
                                 "cost" : 4,
                                 "type" : "openconfig-ospfv2-ext:NETWORK_ROUTE",
                                 "prefix" : "10.193.80.0/20"
                              },
                              "prefix" : "32.1.0.0/24"
                           },
                           {
                              "next-hops" : {
                                 "next-hop" : [
                                    {
                                       "area-id" : "0.0.0.2",
                                       "address" : "0.0.0.0",
                                       "state" : {
                                          "area-id" : "0.0.0.0",
                                          "address" : "32.1.0.1",
                                          "out-interface" : "Vlan3719"
                                       },
                                       "out-interface" : "Ethernet120"
                                    }
                                 ]
                              },
                              "state" : {
                                 "cost" : 10,
                                 "type" : "openconfig-ospfv2-ext:NETWORK_ROUTE",
                                 "prefix" : "10.193.80.0/20"
                              },
                              "prefix" : "41.1.0.0/24"
                           }
                        ]
                     },
                     "type" : "openconfig-ospfv2-ext:NETWORK_ROUTE_TABLE",
                     "state" : {
                        "type" : "openconfig-ospfv2-ext:NETWORK_ROUTE_TABLE"
                     }
                  },
                  {
                     "routes" : {
                        "route" : [
                           {
                              "next-hops" : {
                                 "next-hop" : [
                                    {
                                       "area-id" : "0.0.0.0",
                                       "address" : "32.1.0.1",
                                       "state" : {
                                          "area-id" : "0.0.0.0",
                                          "address" : "32.1.0.1",
                                          "out-interface" : "Vlan3719"
                                       },
                                       "out-interface" : "Vlan3719"
                                    }
                                 ]
                              },
                              "state" : {
                                 "router-type" : "openconfig-ospfv2-ext:ABRASBR",
                                 "cost" : 4,
                                 "type" : "openconfig-ospfv2-ext:ROUTER_ROUTE",
                                 "prefix" : "10.193.80.0/20"
                              },
                              "prefix" : "6.6.6.5"
                           }
                        ]
                     },
                     "type" : "openconfig-ospfv2-ext:ROUTER_ROUTE_TABLE",
                     "state" : {
                        "type" : "openconfig-ospfv2-ext:ROUTER_ROUTE_TABLE"
                     }
                  }
               ]
            },
            "global" : {
               "graceful-restart" : {
                  "openconfig-ospfv2-ext:helpers" : {
                     "helper" : [
                        {
                           "neighbour-id" : "5.5.5.4",
                           "config" : {
                              "vrf-name" : "default",
                              "neighbour-id" : "5.5.5.4"
                           },
                           "state" : {
                              "neighbour-id" : "5.5.5.4"
                           }
                        }
                     ]
                  },
                  "config" : {
                     "openconfig-ospfv2-ext:planned-only" : true,
                     "helper-only" : true,
                     "openconfig-ospfv2-ext:supported-grace-time" : 800,
                     "openconfig-ospfv2-ext:strict-lsa-checking" : true
                  },
                  "state" : {
                     "openconfig-ospfv2-ext:planned-only" : true,
                     "helper-only" : true,
                     "openconfig-ospfv2-ext:grace-period" : 0,
                     "openconfig-ospfv2-ext:supported-grace-time" : 800,
                     "openconfig-ospfv2-ext:strict-lsa-checking" : true
                  }
               },
               "inter-area-propagation-policies" : {
                  "openconfig-ospfv2-ext:inter-area-policy" : [
                     {
                        "config" : {
                           "src-area" : "0.0.0.0"
                        },
                        "src-area" : "0.0.0.0",
                        "state" : {
                           "src-area" : "0.0.0.0"
                        }
                     },
                     {
                        "config" : {
                           "src-area" : "0.0.0.2"
                        },
                        "src-area" : "0.0.0.2",
                        "state" : {
                           "src-area" : "0.0.0.2"
                        }
                     }
                  ]
               },
               "config" : {
                  "router-id" : "5.5.5.4",
                  "openconfig-ospfv2-ext:opaque-lsa-capability" : true,
                  "openconfig-ospfv2-ext:description" : "OSPFv2 Router",
                  "openconfig-ospfv2-ext:enable" : true
               },
               "timers" : {
                  "lsa-generation" : {
                     "state" : {
                        "openconfig-ospfv2-ext:refresh-timer" : 10000,
                        "openconfig-ospfv2-ext:lsa-min-interval-timer" : 5000,
                        "openconfig-ospfv2-ext:lsa-min-arrival-timer" : 1000
                     }
                  },
                  "spf" : {
                     "state" : {
                        "maximum-delay" : 5000,
                        "initial-delay" : 50,
                        "openconfig-ospfv2-ext:throttle-delay" : 0,
                        "openconfig-ospfv2-ext:spf-timer-due" : 0
                     }
                  }
               },
               "state" : {
                  "openconfig-ospfv2-ext:area-count" : 2,
                  "openconfig-ospfv2-ext:last-spf-duration" : 358,
                  "openconfig-ospfv2-ext:hold-time-multiplier" : 1,
                  "openconfig-ospfv2-ext:last-spf-execution-time" : "1187616",
                  "openconfig-ospfv2-ext:opaque-lsa-count" : 0,
                  "router-id" : "5.5.5.4",
                  "openconfig-ospfv2-ext:abr-type" : "openconfig-ospfv2-ext:CISCO",
                  "openconfig-ospfv2-ext:external-lsa-checksum" : "0x00016bcc",
                  "openconfig-ospfv2-ext:opaque-lsa-checksum" : "0x00000000",
                  "openconfig-ospfv2-ext:write-multiplier" : 20,
                  "openconfig-ospfv2-ext:opaque-lsa-capability" : true,
                  "openconfig-ospfv2-ext:external-lsa-count" : 2
               }
            }
         },
         "name" : "ospfv2",
         "config" : {
            "identifier" : "openconfig-policy-types:OSPF",
            "name" : "ospfv2"
         },
         "state" : {
            "identifier" : "openconfig-policy-types:OSPF",
            "name" : "ospfv2"
         }
      }
   ]
}
            `),
		appRootType: reflect.TypeOf(ocbinds.OpenconfigNetworkInstance_NetworkInstances{}),
		queryParams: QueryParams{
			depthEnabled:  false,
			curDepth:      1,
			content:       QUERY_CONTENT_ALL,
			fields:        []string{},
			fieldsFillAll: false,
		},
		prunedPayload: []byte(`
{
   "protocol" : [
      {
         "identifier" : "openconfig-policy-types:OSPF",
         "ospfv2" : {
            "openconfig-ospfv2-ext:route-tables" : {
               "route-table" : [
                  {
                     "routes" : {
                        "route" : [
                           {
                              "next-hops" : {
                                 "next-hop" : [
                                    {
                                       "area-id" : "0.0.0.0",
                                       "address" : "32.1.0.1",
                                       "state" : {
                                          "area-id" : "0.0.0.0",
                                          "address" : "32.1.0.1",
                                          "out-interface" : "Vlan3719"
                                       },
                                       "out-interface" : "Vlan3719"
                                    }
                                 ]
                              },
                              "state" : {
                                 "cost" : 4,
                                 "sub-type" : "openconfig-ospfv2-ext:EXTERNAL_ROUTE_TYPE_2",
                                 "type" : "openconfig-ospfv2-ext:EXTERNAL_ROUTE",
                                 "prefix" : "10.193.80.0/20"
                              },
                              "prefix" : "10.193.80.0/20"
                           }
                        ]
                     },
                     "type" : "openconfig-ospfv2-ext:EXTERNAL_ROUTE_TABLE",
                     "state" : {
                        "type" : "openconfig-ospfv2-ext:EXTERNAL_ROUTE_TABLE"
                     }
                  },
                  {
                     "routes" : {
                        "route" : [
                           {
                              "next-hops" : {
                                 "next-hop" : [
                                    {
                                       "area-id" : "0.0.0.0",
                                       "address" : "32.1.0.1",
                                       "state" : {
                                          "area-id" : "0.0.0.0",
                                          "address" : "32.1.0.1",
                                          "out-interface" : "Vlan3719"
                                       },
                                       "out-interface" : "Vlan3719"
                                    }
                                 ]
                              },
                              "state" : {
                                 "cost" : 14,
                                 "type" : "openconfig-ospfv2-ext:NETWORK_ROUTE",
                                 "inter-area" : true,
                                 "prefix" : "10.193.80.0/20"
                              },
                              "prefix" : "21.1.0.0/24"
                           },
                           {
                              "next-hops" : {
                                 "next-hop" : [
                                    {
                                       "area-id" : "0.0.0.0",
                                       "address" : "0.0.0.0",
                                       "state" : {
                                          "area-id" : "0.0.0.0",
                                          "address" : "32.1.0.1",
                                          "out-interface" : "Vlan3719"
                                       },
                                       "out-interface" : "Vlan3719"
                                    }
                                 ]
                              },
                              "state" : {
                                 "cost" : 4,
                                 "type" : "openconfig-ospfv2-ext:NETWORK_ROUTE",
                                 "prefix" : "10.193.80.0/20"
                              },
                              "prefix" : "32.1.0.0/24"
                           },
                           {
                              "next-hops" : {
                                 "next-hop" : [
                                    {
                                       "area-id" : "0.0.0.2",
                                       "address" : "0.0.0.0",
                                       "state" : {
                                          "area-id" : "0.0.0.0",
                                          "address" : "32.1.0.1",
                                          "out-interface" : "Vlan3719"
                                       },
                                       "out-interface" : "Ethernet120"
                                    }
                                 ]
                              },
                              "state" : {
                                 "cost" : 10,
                                 "type" : "openconfig-ospfv2-ext:NETWORK_ROUTE",
                                 "prefix" : "10.193.80.0/20"
                              },
                              "prefix" : "41.1.0.0/24"
                           }
                        ]
                     },
                     "type" : "openconfig-ospfv2-ext:NETWORK_ROUTE_TABLE",
                     "state" : {
                        "type" : "openconfig-ospfv2-ext:NETWORK_ROUTE_TABLE"
                     }
                  },
                  {
                     "routes" : {
                        "route" : [
                           {
                              "next-hops" : {
                                 "next-hop" : [
                                    {
                                       "area-id" : "0.0.0.0",
                                       "address" : "32.1.0.1",
                                       "state" : {
                                          "area-id" : "0.0.0.0",
                                          "address" : "32.1.0.1",
                                          "out-interface" : "Vlan3719"
                                       },
                                       "out-interface" : "Vlan3719"
                                    }
                                 ]
                              },
                              "state" : {
                                 "router-type" : "openconfig-ospfv2-ext:ABRASBR",
                                 "cost" : 4,
                                 "type" : "openconfig-ospfv2-ext:ROUTER_ROUTE",
                                 "prefix" : "10.193.80.0/20"
                              },
                              "prefix" : "6.6.6.5"
                           }
                        ]
                     },
                     "type" : "openconfig-ospfv2-ext:ROUTER_ROUTE_TABLE",
                     "state" : {
                        "type" : "openconfig-ospfv2-ext:ROUTER_ROUTE_TABLE"
                     }
                  }
               ]
            },
            "global" : {
               "graceful-restart" : {
                  "openconfig-ospfv2-ext:helpers" : {
                     "helper" : [
                        {
                           "neighbour-id" : "5.5.5.4",
                           "config" : {
                              "vrf-name" : "default",
                              "neighbour-id" : "5.5.5.4"
                           },
                           "state" : {
                              "neighbour-id" : "5.5.5.4"
                           }
                        }
                     ]
                  },
                  "config" : {
                     "openconfig-ospfv2-ext:planned-only" : true,
                     "helper-only" : true,
                     "openconfig-ospfv2-ext:supported-grace-time" : 800,
                     "openconfig-ospfv2-ext:strict-lsa-checking" : true
                  },
                  "state" : {
                     "openconfig-ospfv2-ext:planned-only" : true,
                     "helper-only" : true,
                     "openconfig-ospfv2-ext:grace-period" : 0,
                     "openconfig-ospfv2-ext:supported-grace-time" : 800,
                     "openconfig-ospfv2-ext:strict-lsa-checking" : true
                  }
               },
               "inter-area-propagation-policies" : {
                  "openconfig-ospfv2-ext:inter-area-policy" : [
                     {
                        "config" : {
                           "src-area" : "0.0.0.0"
                        },
                        "src-area" : "0.0.0.0",
                        "state" : {
                           "src-area" : "0.0.0.0"
                        }
                     },
                     {
                        "config" : {
                           "src-area" : "0.0.0.2"
                        },
                        "src-area" : "0.0.0.2",
                        "state" : {
                           "src-area" : "0.0.0.2"
                        }
                     }
                  ]
               },
               "config" : {
                  "router-id" : "5.5.5.4",
                  "openconfig-ospfv2-ext:opaque-lsa-capability" : true,
                  "openconfig-ospfv2-ext:description" : "OSPFv2 Router",
                  "openconfig-ospfv2-ext:enable" : true
               },
               "timers" : {
                  "lsa-generation" : {
                     "state" : {
                        "openconfig-ospfv2-ext:refresh-timer" : 10000,
                        "openconfig-ospfv2-ext:lsa-min-interval-timer" : 5000,
                        "openconfig-ospfv2-ext:lsa-min-arrival-timer" : 1000
                     }
                  },
                  "spf" : {
                     "state" : {
                        "maximum-delay" : 5000,
                        "initial-delay" : 50,
                        "openconfig-ospfv2-ext:throttle-delay" : 0,
                        "openconfig-ospfv2-ext:spf-timer-due" : 0
                     }
                  }
               },
               "state" : {
                  "openconfig-ospfv2-ext:area-count" : 2,
                  "openconfig-ospfv2-ext:last-spf-duration" : 358,
                  "openconfig-ospfv2-ext:hold-time-multiplier" : 1,
                  "openconfig-ospfv2-ext:last-spf-execution-time" : "1187616",
                  "openconfig-ospfv2-ext:opaque-lsa-count" : 0,
                  "router-id" : "5.5.5.4",
                  "openconfig-ospfv2-ext:abr-type" : "openconfig-ospfv2-ext:CISCO",
                  "openconfig-ospfv2-ext:external-lsa-checksum" : "0x00016bcc",
                  "openconfig-ospfv2-ext:opaque-lsa-checksum" : "0x00000000",
                  "openconfig-ospfv2-ext:write-multiplier" : 20,
                  "openconfig-ospfv2-ext:opaque-lsa-capability" : true,
                  "openconfig-ospfv2-ext:external-lsa-count" : 2
               }
            }
         },
         "name" : "ospfv2",
         "config" : {
            "identifier" : "openconfig-policy-types:OSPF",
            "name" : "ospfv2"
         },
         "state" : {
            "identifier" : "openconfig-policy-types:OSPF",
            "name" : "ospfv2"
         }
      }
   ]
}
            `),
	},
	{ // OSPF Identity Test Case (i.e. no QP)
		tid:        "OSPF Troubleshoot No Areas No Propagation Policies",
		uri:        "/openconfig-network-instance:network-instances/network-instance[name=default]/protocols/protocol[identifier=OSPF][name=ospfv2]",
		requestUri: "/openconfig-network-instance:network-instances/network-instance[name=default]/protocols/protocol[identifier=OSPF][name=ospfv2]",
		payload: []byte(`
{
   "protocol" : [
      {
         "identifier" : "openconfig-policy-types:OSPF",
         "ospfv2" : {
            "global" : {
               "graceful-restart" : {
                  "openconfig-ospfv2-ext:helpers" : {
                     "helper" : [
                        {
                           "neighbour-id" : "5.5.5.4",
                           "config" : {
                              "vrf-name" : "default",
                              "neighbour-id" : "5.5.5.4"
                           },
                           "state" : {
                              "neighbour-id" : "5.5.5.4"
                           }
                        }
                     ]
                  },
                  "config" : {
                     "openconfig-ospfv2-ext:planned-only" : true,
                     "helper-only" : true,
                     "openconfig-ospfv2-ext:supported-grace-time" : 800,
                     "openconfig-ospfv2-ext:strict-lsa-checking" : true
                  },
                  "state" : {
                     "openconfig-ospfv2-ext:planned-only" : true,
                     "helper-only" : true,
                     "openconfig-ospfv2-ext:grace-period" : 0,
                     "openconfig-ospfv2-ext:supported-grace-time" : 800,
                     "openconfig-ospfv2-ext:strict-lsa-checking" : true
                  }
               },
               "config" : {
                  "router-id" : "5.5.5.4",
                  "openconfig-ospfv2-ext:opaque-lsa-capability" : true,
                  "openconfig-ospfv2-ext:description" : "OSPFv2 Router",
                  "openconfig-ospfv2-ext:enable" : true
               },
               "timers" : {
                  "lsa-generation" : {
                     "state" : {
                        "openconfig-ospfv2-ext:refresh-timer" : 10000,
                        "openconfig-ospfv2-ext:lsa-min-interval-timer" : 5000,
                        "openconfig-ospfv2-ext:lsa-min-arrival-timer" : 1000
                     }
                  },
                  "spf" : {
                     "state" : {
                        "maximum-delay" : 5000,
                        "initial-delay" : 50,
                        "openconfig-ospfv2-ext:throttle-delay" : 0,
                        "openconfig-ospfv2-ext:spf-timer-due" : 0
                     }
                  }
               },
               "state" : {
                  "openconfig-ospfv2-ext:area-count" : 2,
                  "openconfig-ospfv2-ext:last-spf-duration" : 358,
                  "openconfig-ospfv2-ext:hold-time-multiplier" : 1,
                  "openconfig-ospfv2-ext:last-spf-execution-time" : "1187616",
                  "openconfig-ospfv2-ext:opaque-lsa-count" : 0,
                  "router-id" : "5.5.5.4",
                  "openconfig-ospfv2-ext:abr-type" : "openconfig-ospfv2-ext:CISCO",
                  "openconfig-ospfv2-ext:external-lsa-checksum" : "0x00016bcc",
                  "openconfig-ospfv2-ext:opaque-lsa-checksum" : "0x00000000",
                  "openconfig-ospfv2-ext:write-multiplier" : 20,
                  "openconfig-ospfv2-ext:opaque-lsa-capability" : true,
                  "openconfig-ospfv2-ext:external-lsa-count" : 2
               }
            }
         },
         "name" : "ospfv2",
         "config" : {
            "identifier" : "openconfig-policy-types:OSPF",
            "name" : "ospfv2"
         },
         "state" : {
            "identifier" : "openconfig-policy-types:OSPF",
            "name" : "ospfv2"
         }
      }
   ]
}
            `),
		appRootType: reflect.TypeOf(ocbinds.OpenconfigNetworkInstance_NetworkInstances{}),
		queryParams: QueryParams{
			depthEnabled:  false,
			curDepth:      1,
			content:       QUERY_CONTENT_ALL,
			fields:        []string{},
			fieldsFillAll: false,
		},
		prunedPayload: []byte(`
{
   "protocol" : [
      {
         "identifier" : "openconfig-policy-types:OSPF",
         "ospfv2" : {
            "global" : {
               "graceful-restart" : {
                  "openconfig-ospfv2-ext:helpers" : {
                     "helper" : [
                        {
                           "neighbour-id" : "5.5.5.4",
                           "config" : {
                              "vrf-name" : "default",
                              "neighbour-id" : "5.5.5.4"
                           },
                           "state" : {
                              "neighbour-id" : "5.5.5.4"
                           }
                        }
                     ]
                  },
                  "config" : {
                     "openconfig-ospfv2-ext:planned-only" : true,
                     "helper-only" : true,
                     "openconfig-ospfv2-ext:supported-grace-time" : 800,
                     "openconfig-ospfv2-ext:strict-lsa-checking" : true
                  },
                  "state" : {
                     "openconfig-ospfv2-ext:planned-only" : true,
                     "helper-only" : true,
                     "openconfig-ospfv2-ext:grace-period" : 0,
                     "openconfig-ospfv2-ext:supported-grace-time" : 800,
                     "openconfig-ospfv2-ext:strict-lsa-checking" : true
                  }
               },
               "config" : {
                  "router-id" : "5.5.5.4",
                  "openconfig-ospfv2-ext:opaque-lsa-capability" : true,
                  "openconfig-ospfv2-ext:description" : "OSPFv2 Router",
                  "openconfig-ospfv2-ext:enable" : true
               },
               "timers" : {
                  "lsa-generation" : {
                     "state" : {
                        "openconfig-ospfv2-ext:refresh-timer" : 10000,
                        "openconfig-ospfv2-ext:lsa-min-interval-timer" : 5000,
                        "openconfig-ospfv2-ext:lsa-min-arrival-timer" : 1000
                     }
                  },
                  "spf" : {
                     "state" : {
                        "maximum-delay" : 5000,
                        "initial-delay" : 50,
                        "openconfig-ospfv2-ext:throttle-delay" : 0,
                        "openconfig-ospfv2-ext:spf-timer-due" : 0
                     }
                  }
               },
               "state" : {
                  "openconfig-ospfv2-ext:area-count" : 2,
                  "openconfig-ospfv2-ext:last-spf-duration" : 358,
                  "openconfig-ospfv2-ext:hold-time-multiplier" : 1,
                  "openconfig-ospfv2-ext:last-spf-execution-time" : "1187616",
                  "openconfig-ospfv2-ext:opaque-lsa-count" : 0,
                  "router-id" : "5.5.5.4",
                  "openconfig-ospfv2-ext:abr-type" : "openconfig-ospfv2-ext:CISCO",
                  "openconfig-ospfv2-ext:external-lsa-checksum" : "0x00016bcc",
                  "openconfig-ospfv2-ext:opaque-lsa-checksum" : "0x00000000",
                  "openconfig-ospfv2-ext:write-multiplier" : 20,
                  "openconfig-ospfv2-ext:opaque-lsa-capability" : true,
                  "openconfig-ospfv2-ext:external-lsa-count" : 2
               }
            }
         },
         "name" : "ospfv2",
         "config" : {
            "identifier" : "openconfig-policy-types:OSPF",
            "name" : "ospfv2"
         },
         "state" : {
            "identifier" : "openconfig-policy-types:OSPF",
            "name" : "ospfv2"
         }
      }
   ]
}
            `),
	},
	{ // OSPF Identity Test Case (i.e. no QP)
		tid:        "OSPF Identity Content CONFIG",
		uri:        "/openconfig-network-instance:network-instances/network-instance[name=default]/protocols/protocol[identifier=OSPF][name=ospfv2]/ospfv2/areas/area[identifier=0.0.0.0]/lsdb",
		requestUri: "/openconfig-network-instance:network-instances/network-instance[name=default]/protocols/protocol[identifier=OSPF][name=ospfv2]",
		payload: []byte(`
{
   "protocol" : [
      {
         "identifier" : "openconfig-policy-types:OSPF",
         "ospfv2" : {
            "openconfig-ospfv2-ext:route-tables" : {
               "route-table" : [
                  {
                     "routes" : {
                        "route" : [
                           {
                              "next-hops" : {
                                 "next-hop" : [
                                    {
                                       "area-id" : "0.0.0.0",
                                       "address" : "32.1.0.1",
                                       "state" : {
                                          "area-id" : "0.0.0.0",
                                          "address" : "32.1.0.1",
                                          "out-interface" : "Vlan3719"
                                       },
                                       "out-interface" : "Vlan3719"
                                    }
                                 ]
                              },
                              "state" : {
                                 "cost" : 4,
                                 "sub-type" : "openconfig-ospfv2-ext:EXTERNAL_ROUTE_TYPE_2",
                                 "type" : "openconfig-ospfv2-ext:EXTERNAL_ROUTE",
                                 "prefix" : "10.193.80.0/20"
                              },
                              "prefix" : "10.193.80.0/20"
                           }
                        ]
                     },
                     "type" : "openconfig-ospfv2-ext:EXTERNAL_ROUTE_TABLE",
                     "state" : {
                        "type" : "openconfig-ospfv2-ext:EXTERNAL_ROUTE_TABLE"
                     }
                  },
                  {
                     "routes" : {
                        "route" : [
                           {
                              "next-hops" : {
                                 "next-hop" : [
                                    {
                                       "area-id" : "0.0.0.0",
                                       "address" : "32.1.0.1",
                                       "state" : {
                                          "area-id" : "0.0.0.0",
                                          "address" : "32.1.0.1",
                                          "out-interface" : "Vlan3719"
                                       },
                                       "out-interface" : "Vlan3719"
                                    }
                                 ]
                              },
                              "state" : {
                                 "cost" : 14,
                                 "type" : "openconfig-ospfv2-ext:NETWORK_ROUTE",
                                 "inter-area" : true,
                                 "prefix" : "10.193.80.0/20"
                              },
                              "prefix" : "21.1.0.0/24"
                           },
                           {
                              "next-hops" : {
                                 "next-hop" : [
                                    {
                                       "area-id" : "0.0.0.0",
                                       "address" : "0.0.0.0",
                                       "state" : {
                                          "area-id" : "0.0.0.0",
                                          "address" : "32.1.0.1",
                                          "out-interface" : "Vlan3719"
                                       },
                                       "out-interface" : "Vlan3719"
                                    }
                                 ]
                              },
                              "state" : {
                                 "cost" : 4,
                                 "type" : "openconfig-ospfv2-ext:NETWORK_ROUTE",
                                 "prefix" : "10.193.80.0/20"
                              },
                              "prefix" : "32.1.0.0/24"
                           },
                           {
                              "next-hops" : {
                                 "next-hop" : [
                                    {
                                       "area-id" : "0.0.0.2",
                                       "address" : "0.0.0.0",
                                       "state" : {
                                          "area-id" : "0.0.0.0",
                                          "address" : "32.1.0.1",
                                          "out-interface" : "Vlan3719"
                                       },
                                       "out-interface" : "Ethernet120"
                                    }
                                 ]
                              },
                              "state" : {
                                 "cost" : 10,
                                 "type" : "openconfig-ospfv2-ext:NETWORK_ROUTE",
                                 "prefix" : "10.193.80.0/20"
                              },
                              "prefix" : "41.1.0.0/24"
                           }
                        ]
                     },
                     "type" : "openconfig-ospfv2-ext:NETWORK_ROUTE_TABLE",
                     "state" : {
                        "type" : "openconfig-ospfv2-ext:NETWORK_ROUTE_TABLE"
                     }
                  },
                  {
                     "routes" : {
                        "route" : [
                           {
                              "next-hops" : {
                                 "next-hop" : [
                                    {
                                       "area-id" : "0.0.0.0",
                                       "address" : "32.1.0.1",
                                       "state" : {
                                          "area-id" : "0.0.0.0",
                                          "address" : "32.1.0.1",
                                          "out-interface" : "Vlan3719"
                                       },
                                       "out-interface" : "Vlan3719"
                                    }
                                 ]
                              },
                              "state" : {
                                 "router-type" : "openconfig-ospfv2-ext:ABRASBR",
                                 "cost" : 4,
                                 "type" : "openconfig-ospfv2-ext:ROUTER_ROUTE",
                                 "prefix" : "10.193.80.0/20"
                              },
                              "prefix" : "6.6.6.5"
                           }
                        ]
                     },
                     "type" : "openconfig-ospfv2-ext:ROUTER_ROUTE_TABLE",
                     "state" : {
                        "type" : "openconfig-ospfv2-ext:ROUTER_ROUTE_TABLE"
                     }
                  }
               ]
            },
            "global" : {
               "graceful-restart" : {
                  "openconfig-ospfv2-ext:helpers" : {
                     "helper" : [
                        {
                           "neighbour-id" : "5.5.5.4",
                           "config" : {
                              "vrf-name" : "default",
                              "neighbour-id" : "5.5.5.4"
                           },
                           "state" : {
                              "neighbour-id" : "5.5.5.4"
                           }
                        }
                     ]
                  },
                  "config" : {
                     "openconfig-ospfv2-ext:planned-only" : true,
                     "helper-only" : true,
                     "openconfig-ospfv2-ext:supported-grace-time" : 800,
                     "openconfig-ospfv2-ext:strict-lsa-checking" : true
                  },
                  "state" : {
                     "openconfig-ospfv2-ext:planned-only" : true,
                     "helper-only" : true,
                     "openconfig-ospfv2-ext:grace-period" : 0,
                     "openconfig-ospfv2-ext:supported-grace-time" : 800,
                     "openconfig-ospfv2-ext:strict-lsa-checking" : true
                  }
               },
               "inter-area-propagation-policies" : {
                  "openconfig-ospfv2-ext:inter-area-policy" : [
                     {
                        "config" : {
                           "src-area" : "0.0.0.0"
                        },
                        "src-area" : "0.0.0.0",
                        "state" : {
                           "src-area" : "0.0.0.0"
                        }
                     },
                     {
                        "config" : {
                           "src-area" : "0.0.0.2"
                        },
                        "src-area" : "0.0.0.2",
                        "state" : {
                           "src-area" : "0.0.0.2"
                        }
                     }
                  ]
               },
               "config" : {
                  "router-id" : "5.5.5.4",
                  "openconfig-ospfv2-ext:opaque-lsa-capability" : true,
                  "openconfig-ospfv2-ext:description" : "OSPFv2 Router",
                  "openconfig-ospfv2-ext:enable" : true
               },
               "timers" : {
                  "lsa-generation" : {
                     "state" : {
                        "openconfig-ospfv2-ext:refresh-timer" : 10000,
                        "openconfig-ospfv2-ext:lsa-min-interval-timer" : 5000,
                        "openconfig-ospfv2-ext:lsa-min-arrival-timer" : 1000
                     }
                  },
                  "spf" : {
                     "state" : {
                        "maximum-delay" : 5000,
                        "initial-delay" : 50,
                        "openconfig-ospfv2-ext:throttle-delay" : 0,
                        "openconfig-ospfv2-ext:spf-timer-due" : 0
                     }
                  }
               },
               "state" : {
                  "openconfig-ospfv2-ext:area-count" : 2,
                  "openconfig-ospfv2-ext:last-spf-duration" : 358,
                  "openconfig-ospfv2-ext:hold-time-multiplier" : 1,
                  "openconfig-ospfv2-ext:last-spf-execution-time" : "1187616",
                  "openconfig-ospfv2-ext:opaque-lsa-count" : 0,
                  "router-id" : "5.5.5.4",
                  "openconfig-ospfv2-ext:abr-type" : "openconfig-ospfv2-ext:CISCO",
                  "openconfig-ospfv2-ext:external-lsa-checksum" : "0x00016bcc",
                  "openconfig-ospfv2-ext:opaque-lsa-checksum" : "0x00000000",
                  "openconfig-ospfv2-ext:write-multiplier" : 20,
                  "openconfig-ospfv2-ext:opaque-lsa-capability" : true,
                  "openconfig-ospfv2-ext:external-lsa-count" : 2
               }
            },
            "areas" : {
               "area" : [
                  {
                     "lsdb" : {
                        "lsa-types" : {
                           "lsa-type" : [
                              {
                                 "type" : "openconfig-ospf-types:AS_EXTERNAL_LSA",
                                 "state" : {
                                    "type" : "openconfig-ospf-types:AS_EXTERNAL_LSA"
                                 },
                                 "lsas" : {
                                    "openconfig-ospfv2-ext:lsa-ext" : [
                                       {
                                          "advertising-router" : "5.5.5.4",
                                          "as-external-lsa" : {
                                             "types-of-service" : {
                                                "type-of-service" : [
                                                   {
                                                      "tos" : 0,
                                                      "state" : {
                                                         "forwarding-address" : "0.0.0.0",
                                                         "external-route-tag" : 0,
                                                         "metric" : 20
                                                      }
                                                   }
                                                ]
                                             },
                                             "state" : {
                                                "metric-type" : "TYPE_2",
                                                "mask" : 20
                                             }
                                          },
                                          "link-state-id" : "10.193.80.0",
                                          "state" : {
                                             "checksum" : 50393,
                                             "length" : 36,
                                             "flags" : 11,
                                             "display-sequence-number" : "80000001",
                                             "option" : 2,
                                             "option-expanded" : "*||||||E|",
                                             "age" : 1227
                                          }
                                       },
                                       {
                                          "advertising-router" : "6.6.6.5",
                                          "as-external-lsa" : {
                                             "types-of-service" : {
                                                "type-of-service" : [
                                                   {
                                                      "tos" : 0,
                                                      "state" : {
                                                         "forwarding-address" : "0.0.0.0",
                                                         "external-route-tag" : 0,
                                                         "metric" : 20
                                                      }
                                                   }
                                                ]
                                             },
                                             "state" : {
                                                "metric-type" : "TYPE_2",
                                                "mask" : 20
                                             }
                                          },
                                          "link-state-id" : "10.193.80.0",
                                          "state" : {
                                             "checksum" : 42739,
                                             "length" : 36,
                                             "flags" : 6,
                                             "display-sequence-number" : "80000001",
                                             "option" : 2,
                                             "option-expanded" : "*||||||E|",
                                             "age" : 1228
                                          }
                                       }
                                    ]
                                 }
                              },
                              {
                                 "type" : "openconfig-ospf-types:NETWORK_LSA",
                                 "state" : {
                                    "type" : "openconfig-ospf-types:NETWORK_LSA"
                                 },
                                 "lsas" : {
                                    "openconfig-ospfv2-ext:lsa-ext" : [
                                       {
                                          "advertising-router" : "6.6.6.5",
                                          "link-state-id" : "32.1.0.1",
                                          "state" : {
                                             "checksum" : 51756,
                                             "length" : 32,
                                             "flags" : 6,
                                             "display-sequence-number" : "80000001",
                                             "option" : 2,
                                             "option-expanded" : "*|-|-|-|-|-|E|-",
                                             "age" : 1198
                                          },
                                          "network-lsa" : {
                                             "state" : {
                                                "network-mask" : 24
                                             }
                                          }
                                       }
                                    ]
                                 }
                              },
                              {
                                 "type" : "openconfig-ospf-types:OSPFV2_LINK_SCOPE_OPAQUE_LSA"
                              },
                              {
                                 "type" : "openconfig-ospf-types:ROUTER_LSA",
                                 "state" : {
                                    "type" : "openconfig-ospf-types:ROUTER_LSA"
                                 },
                                 "lsas" : {
                                    "openconfig-ospfv2-ext:lsa-ext" : [
                                       {
                                          "advertising-router" : "5.5.5.4",
                                          "link-state-id" : "5.5.5.4",
                                          "state" : {
                                             "checksum" : 5321,
                                             "length" : 36,
                                             "flags" : 3,
                                             "display-sequence-number" : "80000005",
                                             "option" : 2,
                                             "option-expanded" : "*||||||E|",
                                             "age" : 1197
                                          },
                                          "router-lsa" : {
                                             "link-informations" : {
                                                "link-information" : [
                                                   {
                                                      "state" : {
                                                         "metric" : 4,
                                                         "type" : "openconfig-ospf-types:ROUTER_LSA_TRANSIT_NETWORK",
                                                         "number-tos-metrics" : 0
                                                      }
                                                   }
                                                ]
                                             },
                                             "types-of-service" : {
                                                "type-of-service" : [
                                                   {
                                                      "tos" : 0,
                                                      "state" : {
                                                         "metric" : 4
                                                      }
                                                   }
                                                ]
                                             },
                                             "state" : {
                                                "flags-description" : " : ABR ASBR",
                                                "flags" : 3,
                                                "number-links" : 1
                                             }
                                          }
                                       },
                                       {
                                          "advertising-router" : "6.6.6.5",
                                          "link-state-id" : "6.6.6.5",
                                          "state" : {
                                             "checksum" : 47900,
                                             "length" : 36,
                                             "flags" : 6,
                                             "display-sequence-number" : "80000004",
                                             "option" : 2,
                                             "option-expanded" : "*||||||E|",
                                             "age" : 1197
                                          },
                                          "router-lsa" : {
                                             "link-informations" : {
                                                "link-information" : [
                                                   {
                                                      "state" : {
                                                         "metric" : 4,
                                                         "type" : "openconfig-ospf-types:ROUTER_LSA_TRANSIT_NETWORK",
                                                         "number-tos-metrics" : 0
                                                      }
                                                   }
                                                ]
                                             },
                                             "types-of-service" : {
                                                "type-of-service" : [
                                                   {
                                                      "tos" : 0,
                                                      "state" : {
                                                         "metric" : 4
                                                      }
                                                   }
                                                ]
                                             },
                                             "state" : {
                                                "flags-description" : " : ABR ASBR",
                                                "flags" : 3,
                                                "number-links" : 1
                                             }
                                          }
                                       }
                                    ]
                                 }
                              },
                              {
                                 "type" : "openconfig-ospf-types:SUMMARY_ASBR_LSA",
                                 "state" : {
                                    "type" : "openconfig-ospf-types:SUMMARY_ASBR_LSA"
                                 }
                              },
                              {
                                 "type" : "openconfig-ospf-types:SUMMARY_IP_NETWORK_LSA",
                                 "state" : {
                                    "type" : "openconfig-ospf-types:SUMMARY_IP_NETWORK_LSA"
                                 },
                                 "lsas" : {
                                    "openconfig-ospfv2-ext:lsa-ext" : [
                                       {
                                          "advertising-router" : "6.6.6.5",
                                          "link-state-id" : "21.1.0.0",
                                          "summary-lsa" : {
                                             "types-of-service" : {
                                                "type-of-service" : [
                                                   {
                                                      "tos" : 0,
                                                      "state" : {
                                                         "metric" : 10
                                                      }
                                                   }
                                                ]
                                             },
                                             "state" : {
                                                "network-mask" : 24
                                             }
                                          },
                                          "state" : {
                                             "checksum" : 803,
                                             "length" : 28,
                                             "flags" : 6,
                                             "display-sequence-number" : "80000001",
                                             "option" : 2,
                                             "option-expanded" : "*|-|-|-|-|-|E|-",
                                             "age" : 1228
                                          }
                                       },
                                       {
                                          "advertising-router" : "5.5.5.4",
                                          "link-state-id" : "41.1.0.0",
                                          "summary-lsa" : {
                                             "types-of-service" : {
                                                "type-of-service" : [
                                                   {
                                                      "tos" : 0,
                                                      "state" : {
                                                         "metric" : 10
                                                      }
                                                   }
                                                ]
                                             },
                                             "state" : {
                                                "network-mask" : 24
                                             }
                                          },
                                          "state" : {
                                             "checksum" : 7417,
                                             "length" : 28,
                                             "flags" : 11,
                                             "display-sequence-number" : "80000001",
                                             "option" : 2,
                                             "option-expanded" : "*|-|-|-|-|-|E|-",
                                             "age" : 1227
                                          }
                                       }
                                    ]
                                 }
                              }
                           ]
                        },
                        "state" : {
                           "identifier" : "0.0.0.0"
                        }
                     },
                     "identifier" : "0.0.0.0",
                     "interfaces" : {
                        "interface" : [
                           {
                              "openconfig-ospfv2-ext:message-statistics" : {
                                 "state" : {
                                    "ls-update-transmit" : 4,
                                    "ls-request-receive" : 1,
                                    "ls-acknowledge-receive" : 2,
                                    "ls-acknowledge-transmit" : 2,
                                    "db-description-transmit" : 3,
                                    "db-description-receive" : 2,
                                    "hello-transmit" : 124,
                                    "hello-receive" : 124,
                                    "ls-request-transmit" : 1,
                                    "ls-update-receive" : 3
                                 }
                              },
                              "openconfig-ospfv2-ext:neighbours" : {
                                 "neighbour" : [
                                    {
                                       "neighbor-id" : "6.6.6.5",
                                       "neighbor-address" : "32.1.0.1",
                                       "state" : {
                                          "priority" : 1,
                                          "option-value" : 66,
                                          "gr-helper-status" : "None",
                                          "state-changes" : 6,
                                          "backup-designated-router" : "32.1.0.2",
                                          "adjacency-state" : "openconfig-ospf-types:FULL",
                                          "neighbor-id" : "6.6.6.5",
                                          "optional-capabilities" : "*|O|-|-|-|-|E|-",
                                          "thread-inactivity-timer" : true,
                                          "thread-ls-request-retransmission" : true,
                                          "thread-ls-update-retransmission" : true,
                                          "retransmit-summary-queue-length" : 0,
                                          "area-id" : "0.0.0.0",
                                          "gr-active-helper" : false,
                                          "interface-address" : "32.1.0.2",
                                          "neighbor-address" : "32.1.0.1",
                                          "last-established-time" : "1197595",
                                          "designated-router" : "32.1.0.1",
                                          "database-summary-queue-length" : 0,
                                          "interface-name" : "Vlan3719",
                                          "link-state-request-queue-length" : 0,
                                          "dead-time" : "32410"
                                       }
                                    }
                                 ]
                              },
                              "id" : "Vlan3719",
                              "timers" : {
                                 "state" : {
                                    "retransmission-interval" : 5,
                                    "dead-interval" : 40,
                                    "openconfig-ospfv2-ext:wait-time" : 40,
                                    "hello-interval" : 10000,
                                    "openconfig-ospfv2-ext:hello-due" : 2409
                                 }
                              },
                              "state" : {
                                 "priority" : 1,
                                 "openconfig-ospfv2-ext:member-of-ospf-all-routers" : true,
                                 "openconfig-ospfv2-ext:index" : 310,
                                 "openconfig-ospfv2-ext:mtu" : 9100,
                                 "openconfig-ospfv2-ext:if-flags" : "<UP,BROADCAST,RUNNING,MULTICAST>",
                                 "openconfig-ospfv2-ext:adjacency-count" : 1,
                                 "openconfig-ospfv2-ext:address-len" : 24,
                                 "openconfig-ospfv2-ext:ospf-enable" : true,
                                 "openconfig-ospfv2-ext:cost" : 4,
                                 "openconfig-ospfv2-ext:address" : "32.1.0.2",
                                 "id" : "Vlan3719",
                                 "openconfig-ospfv2-ext:adjacency-status" : "Backup",
                                 "openconfig-ospfv2-ext:neighbor-count" : 1,
                                 "openconfig-ospfv2-ext:backup-designated-router-id" : "5.5.5.4",
                                 "openconfig-ospfv2-ext:member-of-ospf-designated-routers" : true,
                                 "openconfig-ospfv2-ext:router-id" : "5.5.5.4",
                                 "openconfig-ospfv2-ext:broadcast-address" : "32.1.0.255",
                                 "openconfig-ospfv2-ext:bandwidth" : 25000,
                                 "openconfig-ospfv2-ext:area-id" : "0.0.0.0",
                                 "openconfig-ospfv2-ext:ospf-interface-type" : "Broadcast",
                                 "openconfig-ospfv2-ext:backup-designated-router-address" : "32.1.0.2",
                                 "network-type" : "openconfig-ospf-types:BROADCAST_NETWORK",
                                 "openconfig-ospfv2-ext:transmit-delay" : 1,
                                 "openconfig-ospfv2-ext:operational-state" : "Up"
                              }
                           }
                        ]
                     },
                     "config" : {
                        "identifier" : "0.0.0.0"
                     },
                     "openconfig-ospfv2-ext:networks" : {
                        "network" : [
                           {
                              "address-prefix" : "31.1.0.0/24",
                              "config" : {
                                 "address-prefix" : "31.1.0.0/24",
                                 "description" : "Network prefix"
                              },
                              "state" : {
                                 "address-prefix" : "31.1.0.0/24",
                                 "description" : "Network prefix"
                              }
                           },
                           {
                              "address-prefix" : "32.1.0.0/24",
                              "config" : {
                                 "address-prefix" : "32.1.0.0/24",
                                 "description" : "Network prefix"
                              },
                              "state" : {
                                 "address-prefix" : "32.1.0.0/24",
                                 "description" : "Network prefix"
                              }
                           }
                        ]
                     },
                     "state" : {
                        "openconfig-ospfv2-ext:summary-lsa-count" : 2,
                        "openconfig-ospfv2-ext:nssa-lsa-checksum" : "0x00000000",
                        "openconfig-ospfv2-ext:authentication-type" : "no",
                        "openconfig-ospfv2-ext:asbr-summary-lsa-count" : 0,
                        "openconfig-ospfv2-ext:adjacency-count" : 1,
                        "openconfig-ospfv2-ext:opaque-area-lsa-count" : 0,
                        "openconfig-ospfv2-ext:opaque-area-lsa-checksum" : "0x00000000",
                        "openconfig-ospfv2-ext:lsa-count" : 5,
                        "openconfig-ospfv2-ext:opaque-link-lsa-checksum" : "0x00000000",
                        "openconfig-ospfv2-ext:spf-execution-count" : 6,
                        "openconfig-ospfv2-ext:interface-count" : 1,
                        "openconfig-ospfv2-ext:network-lsa-count" : 1,
                        "openconfig-ospfv2-ext:asbr-summary-lsa-checksum" : "0x00000000",
                        "openconfig-ospfv2-ext:router-lsa-checksum" : "0x0000cfe5",
                        "openconfig-ospfv2-ext:router-lsa-count" : 2,
                        "openconfig-ospfv2-ext:nssa-lsa-count" : 0,
                        "openconfig-ospfv2-ext:active-interface-count" : 1,
                        "openconfig-ospfv2-ext:summary-lsa-checksum" : "0x0000201c",
                        "openconfig-ospfv2-ext:network-lsa-checksum" : "0x0000ca2c",
                        "openconfig-ospfv2-ext:opaque-link-lsa-count" : 0
                     }
                  },
                  {
                     "lsdb" : {
                        "lsa-types" : {
                           "lsa-type" : [
                              {
                                 "type" : "openconfig-ospf-types:AS_EXTERNAL_LSA",
                                 "state" : {
                                    "type" : "openconfig-ospf-types:AS_EXTERNAL_LSA"
                                 },
                                 "lsas" : {
                                    "openconfig-ospfv2-ext:lsa-ext" : [
                                       {
                                          "advertising-router" : "5.5.5.4",
                                          "as-external-lsa" : {
                                             "types-of-service" : {
                                                "type-of-service" : [
                                                   {
                                                      "tos" : 0,
                                                      "state" : {
                                                         "forwarding-address" : "0.0.0.0",
                                                         "external-route-tag" : 0,
                                                         "metric" : 20
                                                      }
                                                   }
                                                ]
                                             },
                                             "state" : {
                                                "metric-type" : "TYPE_2",
                                                "mask" : 20
                                             }
                                          },
                                          "link-state-id" : "10.193.80.0",
                                          "state" : {
                                             "checksum" : 50393,
                                             "length" : 36,
                                             "flags" : 11,
                                             "display-sequence-number" : "80000001",
                                             "option" : 2,
                                             "option-expanded" : "*||||||E|",
                                             "age" : 1227
                                          }
                                       },
                                       {
                                          "advertising-router" : "6.6.6.5",
                                          "as-external-lsa" : {
                                             "types-of-service" : {
                                                "type-of-service" : [
                                                   {
                                                      "tos" : 0,
                                                      "state" : {
                                                         "forwarding-address" : "0.0.0.0",
                                                         "external-route-tag" : 0,
                                                         "metric" : 20
                                                      }
                                                   }
                                                ]
                                             },
                                             "state" : {
                                                "metric-type" : "TYPE_2",
                                                "mask" : 20
                                             }
                                          },
                                          "link-state-id" : "10.193.80.0",
                                          "state" : {
                                             "checksum" : 42739,
                                             "length" : 36,
                                             "flags" : 6,
                                             "display-sequence-number" : "80000001",
                                             "option" : 2,
                                             "option-expanded" : "*||||||E|",
                                             "age" : 1228
                                          }
                                       }
                                    ]
                                 }
                              },
                              {
                                 "type" : "openconfig-ospf-types:NETWORK_LSA",
                                 "state" : {
                                    "type" : "openconfig-ospf-types:NETWORK_LSA"
                                 },
                                 "lsas" : {
                                    "openconfig-ospfv2-ext:lsa-ext" : [
                                       {
                                          "advertising-router" : "5.5.5.4",
                                          "link-state-id" : "41.1.0.2",
                                          "state" : {
                                             "checksum" : 1017,
                                             "length" : 32,
                                             "flags" : 3,
                                             "display-sequence-number" : "80000001",
                                             "option" : 2,
                                             "option-expanded" : "*|-|-|-|-|-|E|-",
                                             "age" : 1198
                                          },
                                          "network-lsa" : {
                                             "state" : {
                                                "network-mask" : 24
                                             }
                                          }
                                       }
                                    ]
                                 }
                              },
                              {
                                 "type" : "openconfig-ospf-types:OSPFV2_LINK_SCOPE_OPAQUE_LSA"
                              },
                              {
                                 "type" : "openconfig-ospf-types:ROUTER_LSA",
                                 "state" : {
                                    "type" : "openconfig-ospf-types:ROUTER_LSA"
                                 },
                                 "lsas" : {
                                    "openconfig-ospfv2-ext:lsa-ext" : [
                                       {
                                          "advertising-router" : "3.3.3.2",
                                          "link-state-id" : "3.3.3.2",
                                          "state" : {
                                             "checksum" : 53771,
                                             "length" : 36,
                                             "flags" : 6,
                                             "display-sequence-number" : "80000002",
                                             "option" : 0,
                                             "option-expanded" : "*|||||||-",
                                             "age" : 1189
                                          },
                                          "router-lsa" : {
                                             "link-informations" : {
                                                "link-information" : [
                                                   {
                                                      "state" : {
                                                         "metric" : 10,
                                                         "type" : "openconfig-ospf-types:ROUTER_LSA_TRANSIT_NETWORK",
                                                         "number-tos-metrics" : 0
                                                      }
                                                   }
                                                ]
                                             },
                                             "types-of-service" : {
                                                "type-of-service" : [
                                                   {
                                                      "tos" : 0,
                                                      "state" : {
                                                         "metric" : 10
                                                      }
                                                   }
                                                ]
                                             },
                                             "state" : {
                                                "flags-description" : " :",
                                                "flags" : 0,
                                                "number-links" : 1
                                             }
                                          }
                                       },
                                       {
                                          "advertising-router" : "5.5.5.4",
                                          "link-state-id" : "5.5.5.4",
                                          "state" : {
                                             "checksum" : 12693,
                                             "length" : 36,
                                             "flags" : 3,
                                             "display-sequence-number" : "80000003",
                                             "option" : 2,
                                             "option-expanded" : "*||||||E|",
                                             "age" : 1198
                                          },
                                          "router-lsa" : {
                                             "link-informations" : {
                                                "link-information" : [
                                                   {
                                                      "state" : {
                                                         "metric" : 10,
                                                         "type" : "openconfig-ospf-types:ROUTER_LSA_TRANSIT_NETWORK",
                                                         "number-tos-metrics" : 0
                                                      }
                                                   }
                                                ]
                                             },
                                             "types-of-service" : {
                                                "type-of-service" : [
                                                   {
                                                      "tos" : 0,
                                                      "state" : {
                                                         "metric" : 10
                                                      }
                                                   }
                                                ]
                                             },
                                             "state" : {
                                                "flags-description" : " : ABR ASBR",
                                                "flags" : 3,
                                                "number-links" : 1
                                             }
                                          }
                                       }
                                    ]
                                 }
                              },
                              {
                                 "type" : "openconfig-ospf-types:SUMMARY_ASBR_LSA",
                                 "state" : {
                                    "type" : "openconfig-ospf-types:SUMMARY_ASBR_LSA"
                                 },
                                 "lsas" : {
                                    "openconfig-ospfv2-ext:lsa-ext" : [
                                       {
                                          "advertising-router" : "5.5.5.4",
                                          "link-state-id" : "6.6.6.5",
                                          "summary-lsa" : {
                                             "types-of-service" : {
                                                "type-of-service" : [
                                                   {
                                                      "tos" : 0,
                                                      "state" : {
                                                         "metric" : 4
                                                      }
                                                   }
                                                ]
                                             },
                                             "state" : {
                                                "network-mask" : 0
                                             }
                                          },
                                          "state" : {
                                             "checksum" : 59716,
                                             "length" : 28,
                                             "flags" : 11,
                                             "display-sequence-number" : "80000001",
                                             "option" : 2,
                                             "option-expanded" : "*|-|-|-|-|-|E|-",
                                             "age" : 1187
                                          }
                                       }
                                    ]
                                 }
                              },
                              {
                                 "type" : "openconfig-ospf-types:SUMMARY_IP_NETWORK_LSA",
                                 "state" : {
                                    "type" : "openconfig-ospf-types:SUMMARY_IP_NETWORK_LSA"
                                 },
                                 "lsas" : {
                                    "openconfig-ospfv2-ext:lsa-ext" : [
                                       {
                                          "advertising-router" : "5.5.5.4",
                                          "link-state-id" : "21.1.0.0",
                                          "summary-lsa" : {
                                             "types-of-service" : {
                                                "type-of-service" : [
                                                   {
                                                      "tos" : 0,
                                                      "state" : {
                                                         "metric" : 14
                                                      }
                                                   }
                                                ]
                                             },
                                             "state" : {
                                                "network-mask" : 24
                                             }
                                          },
                                          "state" : {
                                             "checksum" : 18908,
                                             "length" : 28,
                                             "flags" : 11,
                                             "display-sequence-number" : "80000001",
                                             "option" : 2,
                                             "option-expanded" : "*|-|-|-|-|-|E|-",
                                             "age" : 1187
                                          }
                                       },
                                       {
                                          "advertising-router" : "5.5.5.4",
                                          "link-state-id" : "32.1.0.0",
                                          "summary-lsa" : {
                                             "types-of-service" : {
                                                "type-of-service" : [
                                                   {
                                                      "tos" : 0,
                                                      "state" : {
                                                         "metric" : 4
                                                      }
                                                   }
                                                ]
                                             },
                                             "state" : {
                                                "network-mask" : 24
                                             }
                                          },
                                          "state" : {
                                             "checksum" : 21967,
                                             "length" : 28,
                                             "flags" : 11,
                                             "display-sequence-number" : "80000001",
                                             "option" : 2,
                                             "option-expanded" : "*|-|-|-|-|-|E|-",
                                             "age" : 1229
                                          }
                                       }
                                    ]
                                 }
                              }
                           ]
                        },
                        "state" : {
                           "identifier" : "0.0.0.2"
                        }
                     },
                     "identifier" : "0.0.0.2",
                     "interfaces" : {
                        "interface" : [
                           {
                              "openconfig-ospfv2-ext:message-statistics" : {
                                 "state" : {
                                    "ls-update-transmit" : 3,
                                    "ls-request-receive" : 0,
                                    "ls-acknowledge-receive" : 3,
                                    "ls-acknowledge-transmit" : 2,
                                    "db-description-transmit" : 2,
                                    "db-description-receive" : 6,
                                    "hello-transmit" : 124,
                                    "hello-receive" : 125,
                                    "ls-request-transmit" : 1,
                                    "ls-update-receive" : 2
                                 }
                              },
                              "openconfig-ospfv2-ext:neighbours" : {
                                 "neighbour" : [
                                    {
                                       "neighbor-id" : "3.3.3.2",
                                       "neighbor-address" : "41.1.0.1",
                                       "state" : {
                                          "priority" : 0,
                                          "option-value" : 66,
                                          "gr-helper-status" : "None",
                                          "state-changes" : 6,
                                          "backup-designated-router" : "0.0.0.0",
                                          "adjacency-state" : "openconfig-ospf-types:FULL",
                                          "neighbor-id" : "6.6.6.5",
                                          "optional-capabilities" : "*|O|-|-|-|-|E|-",
                                          "thread-inactivity-timer" : true,
                                          "thread-ls-request-retransmission" : true,
                                          "thread-ls-update-retransmission" : true,
                                          "retransmit-summary-queue-length" : 0,
                                          "area-id" : "0.0.0.2",
                                          "gr-active-helper" : false,
                                          "interface-address" : "41.1.0.2",
                                          "neighbor-address" : "41.1.0.1",
                                          "last-established-time" : "1198302",
                                          "designated-router" : "41.1.0.2",
                                          "database-summary-queue-length" : 0,
                                          "interface-name" : "Ethernet120",
                                          "link-state-request-queue-length" : 0,
                                          "dead-time" : "34872"
                                       }
                                    }
                                 ]
                              },
                              "id" : "Ethernet120",
                              "timers" : {
                                 "state" : {
                                    "retransmission-interval" : 5,
                                    "dead-interval" : 40,
                                    "openconfig-ospfv2-ext:wait-time" : 40,
                                    "hello-interval" : 10000,
                                    "openconfig-ospfv2-ext:hello-due" : 1711
                                 }
                              },
                              "state" : {
                                 "priority" : 1,
                                 "openconfig-ospfv2-ext:member-of-ospf-all-routers" : true,
                                 "openconfig-ospfv2-ext:index" : 299,
                                 "openconfig-ospfv2-ext:mtu" : 9100,
                                 "openconfig-ospfv2-ext:if-flags" : "<UP,BROADCAST,RUNNING,MULTICAST>",
                                 "openconfig-ospfv2-ext:adjacency-count" : 1,
                                 "openconfig-ospfv2-ext:address-len" : 24,
                                 "openconfig-ospfv2-ext:ospf-enable" : true,
                                 "openconfig-ospfv2-ext:cost" : 10,
                                 "openconfig-ospfv2-ext:address" : "41.1.0.2",
                                 "id" : "Ethernet120",
                                 "openconfig-ospfv2-ext:adjacency-status" : "DR",
                                 "openconfig-ospfv2-ext:member-of-ospf-designated-routers" : true,
                                 "openconfig-ospfv2-ext:neighbor-count" : 1,
                                 "openconfig-ospfv2-ext:router-id" : "5.5.5.4",
                                 "openconfig-ospfv2-ext:broadcast-address" : "41.1.0.255",
                                 "openconfig-ospfv2-ext:bandwidth" : 10000,
                                 "openconfig-ospfv2-ext:area-id" : "0.0.0.2",
                                 "openconfig-ospfv2-ext:ospf-interface-type" : "Broadcast",
                                 "network-type" : "openconfig-ospf-types:BROADCAST_NETWORK",
                                 "openconfig-ospfv2-ext:transmit-delay" : 1,
                                 "openconfig-ospfv2-ext:operational-state" : "Up"
                              }
                           }
                        ]
                     },
                     "config" : {
                        "identifier" : "0.0.0.2"
                     },
                     "openconfig-ospfv2-ext:networks" : {
                        "network" : [
                           {
                              "address-prefix" : "41.1.0.0/24",
                              "config" : {
                                 "address-prefix" : "41.1.0.0/24",
                                 "description" : "Network prefix"
                              },
                              "state" : {
                                 "address-prefix" : "41.1.0.0/24",
                                 "description" : "Network prefix"
                              }
                           }
                        ]
                     },
                     "state" : {
                        "openconfig-ospfv2-ext:summary-lsa-count" : 2,
                        "openconfig-ospfv2-ext:nssa-lsa-checksum" : "0x00000000",
                        "openconfig-ospfv2-ext:authentication-type" : "no",
                        "openconfig-ospfv2-ext:asbr-summary-lsa-count" : 1,
                        "openconfig-ospfv2-ext:adjacency-count" : 1,
                        "openconfig-ospfv2-ext:opaque-area-lsa-count" : 0,
                        "openconfig-ospfv2-ext:opaque-area-lsa-checksum" : "0x00000000",
                        "openconfig-ospfv2-ext:lsa-count" : 6,
                        "openconfig-ospfv2-ext:opaque-link-lsa-checksum" : "0x00000000",
                        "openconfig-ospfv2-ext:virtual-link-adjacency-count" : 0,
                        "openconfig-ospfv2-ext:spf-execution-count" : 6,
                        "openconfig-ospfv2-ext:interface-count" : 1,
                        "openconfig-ospfv2-ext:network-lsa-count" : 1,
                        "openconfig-ospfv2-ext:asbr-summary-lsa-checksum" : "0x0000e944",
                        "openconfig-ospfv2-ext:router-lsa-checksum" : "0x000103a0",
                        "openconfig-ospfv2-ext:router-lsa-count" : 2,
                        "openconfig-ospfv2-ext:nssa-lsa-count" : 0,
                        "openconfig-ospfv2-ext:active-interface-count" : 1,
                        "openconfig-ospfv2-ext:shortcut" : "openconfig-ospfv2-ext:DEFAULT",
                        "openconfig-ospfv2-ext:summary-lsa-checksum" : "0x00009fab",
                        "openconfig-ospfv2-ext:network-lsa-checksum" : "0x000003f9",
                        "openconfig-ospfv2-ext:opaque-link-lsa-count" : 0
                     }
                  }
               ]
            }
         },
         "name" : "ospfv2",
         "config" : {
            "identifier" : "openconfig-policy-types:OSPF",
            "name" : "ospfv2"
         },
         "state" : {
            "identifier" : "openconfig-policy-types:OSPF",
            "name" : "ospfv2"
         }
      }
   ]
}
            `),
		appRootType: reflect.TypeOf(ocbinds.OpenconfigNetworkInstance_NetworkInstances{}),
		queryParams: QueryParams{
			depthEnabled:  false,
			curDepth:      1,
			content:       QUERY_CONTENT_CONFIG,
			fields:        []string{},
			fieldsFillAll: false,
		},
		prunedPayload: []byte(`
{
   "protocol" : [
      {
         "identifier" : "openconfig-policy-types:OSPF",
         "ospfv2" : {
            "openconfig-ospfv2-ext:route-tables" : {
               "route-table" : [
                  {
                     "routes" : {
                        "route" : [
                           {
                              "next-hops" : {
                                 "next-hop" : [
                                    {
                                       "area-id" : "0.0.0.0",
                                       "address" : "32.1.0.1",
                                       "state" : {
                                          "area-id" : "0.0.0.0",
                                          "address" : "32.1.0.1",
                                          "out-interface" : "Vlan3719"
                                       },
                                       "out-interface" : "Vlan3719"
                                    }
                                 ]
                              },
                              "state" : {
                                 "cost" : 4,
                                 "sub-type" : "openconfig-ospfv2-ext:EXTERNAL_ROUTE_TYPE_2",
                                 "type" : "openconfig-ospfv2-ext:EXTERNAL_ROUTE",
                                 "prefix" : "10.193.80.0/20"
                              },
                              "prefix" : "10.193.80.0/20"
                           }
                        ]
                     },
                     "type" : "openconfig-ospfv2-ext:EXTERNAL_ROUTE_TABLE",
                     "state" : {
                        "type" : "openconfig-ospfv2-ext:EXTERNAL_ROUTE_TABLE"
                     }
                  },
                  {
                     "routes" : {
                        "route" : [
                           {
                              "next-hops" : {
                                 "next-hop" : [
                                    {
                                       "area-id" : "0.0.0.0",
                                       "address" : "32.1.0.1",
                                       "state" : {
                                          "area-id" : "0.0.0.0",
                                          "address" : "32.1.0.1",
                                          "out-interface" : "Vlan3719"
                                       },
                                       "out-interface" : "Vlan3719"
                                    }
                                 ]
                              },
                              "state" : {
                                 "cost" : 14,
                                 "type" : "openconfig-ospfv2-ext:NETWORK_ROUTE",
                                 "inter-area" : true,
                                 "prefix" : "10.193.80.0/20"
                              },
                              "prefix" : "21.1.0.0/24"
                           },
                           {
                              "next-hops" : {
                                 "next-hop" : [
                                    {
                                       "area-id" : "0.0.0.0",
                                       "address" : "0.0.0.0",
                                       "state" : {
                                          "area-id" : "0.0.0.0",
                                          "address" : "32.1.0.1",
                                          "out-interface" : "Vlan3719"
                                       },
                                       "out-interface" : "Vlan3719"
                                    }
                                 ]
                              },
                              "state" : {
                                 "cost" : 4,
                                 "type" : "openconfig-ospfv2-ext:NETWORK_ROUTE",
                                 "prefix" : "10.193.80.0/20"
                              },
                              "prefix" : "32.1.0.0/24"
                           },
                           {
                              "next-hops" : {
                                 "next-hop" : [
                                    {
                                       "area-id" : "0.0.0.2",
                                       "address" : "0.0.0.0",
                                       "state" : {
                                          "area-id" : "0.0.0.0",
                                          "address" : "32.1.0.1",
                                          "out-interface" : "Vlan3719"
                                       },
                                       "out-interface" : "Ethernet120"
                                    }
                                 ]
                              },
                              "state" : {
                                 "cost" : 10,
                                 "type" : "openconfig-ospfv2-ext:NETWORK_ROUTE",
                                 "prefix" : "10.193.80.0/20"
                              },
                              "prefix" : "41.1.0.0/24"
                           }
                        ]
                     },
                     "type" : "openconfig-ospfv2-ext:NETWORK_ROUTE_TABLE",
                     "state" : {
                        "type" : "openconfig-ospfv2-ext:NETWORK_ROUTE_TABLE"
                     }
                  },
                  {
                     "routes" : {
                        "route" : [
                           {
                              "next-hops" : {
                                 "next-hop" : [
                                    {
                                       "area-id" : "0.0.0.0",
                                       "address" : "32.1.0.1",
                                       "state" : {
                                          "area-id" : "0.0.0.0",
                                          "address" : "32.1.0.1",
                                          "out-interface" : "Vlan3719"
                                       },
                                       "out-interface" : "Vlan3719"
                                    }
                                 ]
                              },
                              "state" : {
                                 "router-type" : "openconfig-ospfv2-ext:ABRASBR",
                                 "cost" : 4,
                                 "type" : "openconfig-ospfv2-ext:ROUTER_ROUTE",
                                 "prefix" : "10.193.80.0/20"
                              },
                              "prefix" : "6.6.6.5"
                           }
                        ]
                     },
                     "type" : "openconfig-ospfv2-ext:ROUTER_ROUTE_TABLE",
                     "state" : {
                        "type" : "openconfig-ospfv2-ext:ROUTER_ROUTE_TABLE"
                     }
                  }
               ]
            },
            "global" : {
               "graceful-restart" : {
                  "openconfig-ospfv2-ext:helpers" : {
                     "helper" : [
                        {
                           "neighbour-id" : "5.5.5.4",
                           "config" : {
                              "vrf-name" : "default",
                              "neighbour-id" : "5.5.5.4"
                           },
                           "state" : {
                              "neighbour-id" : "5.5.5.4"
                           }
                        }
                     ]
                  },
                  "config" : {
                     "openconfig-ospfv2-ext:planned-only" : true,
                     "helper-only" : true,
                     "openconfig-ospfv2-ext:supported-grace-time" : 800,
                     "openconfig-ospfv2-ext:strict-lsa-checking" : true
                  },
                  "state" : {
                     "openconfig-ospfv2-ext:planned-only" : true,
                     "helper-only" : true,
                     "openconfig-ospfv2-ext:grace-period" : 0,
                     "openconfig-ospfv2-ext:supported-grace-time" : 800,
                     "openconfig-ospfv2-ext:strict-lsa-checking" : true
                  }
               },
               "inter-area-propagation-policies" : {
                  "openconfig-ospfv2-ext:inter-area-policy" : [
                     {
                        "config" : {
                           "src-area" : "0.0.0.0"
                        },
                        "src-area" : "0.0.0.0",
                        "state" : {
                           "src-area" : "0.0.0.0"
                        }
                     },
                     {
                        "config" : {
                           "src-area" : "0.0.0.2"
                        },
                        "src-area" : "0.0.0.2",
                        "state" : {
                           "src-area" : "0.0.0.2"
                        }
                     }
                  ]
               },
               "config" : {
                  "router-id" : "5.5.5.4",
                  "openconfig-ospfv2-ext:opaque-lsa-capability" : true,
                  "openconfig-ospfv2-ext:description" : "OSPFv2 Router",
                  "openconfig-ospfv2-ext:enable" : true
               },
               "timers" : {
                  "lsa-generation" : {
                     "state" : {
                        "openconfig-ospfv2-ext:refresh-timer" : 10000,
                        "openconfig-ospfv2-ext:lsa-min-interval-timer" : 5000,
                        "openconfig-ospfv2-ext:lsa-min-arrival-timer" : 1000
                     }
                  },
                  "spf" : {
                     "state" : {
                        "maximum-delay" : 5000,
                        "initial-delay" : 50,
                        "openconfig-ospfv2-ext:throttle-delay" : 0,
                        "openconfig-ospfv2-ext:spf-timer-due" : 0
                     }
                  }
               },
               "state" : {
                  "openconfig-ospfv2-ext:area-count" : 2,
                  "openconfig-ospfv2-ext:last-spf-duration" : 358,
                  "openconfig-ospfv2-ext:hold-time-multiplier" : 1,
                  "openconfig-ospfv2-ext:last-spf-execution-time" : "1187616",
                  "openconfig-ospfv2-ext:opaque-lsa-count" : 0,
                  "router-id" : "5.5.5.4",
                  "openconfig-ospfv2-ext:abr-type" : "openconfig-ospfv2-ext:CISCO",
                  "openconfig-ospfv2-ext:external-lsa-checksum" : "0x00016bcc",
                  "openconfig-ospfv2-ext:opaque-lsa-checksum" : "0x00000000",
                  "openconfig-ospfv2-ext:write-multiplier" : 20,
                  "openconfig-ospfv2-ext:opaque-lsa-capability" : true,
                  "openconfig-ospfv2-ext:external-lsa-count" : 2
               }
            },
            "areas" : {
               "area" : [
                  {
                     "lsdb" : {
                     },
                     "identifier" : "0.0.0.0",
                     "interfaces" : {
                        "interface" : [
                           {
                              "openconfig-ospfv2-ext:message-statistics" : {
                                 "state" : {
                                    "ls-update-transmit" : 4,
                                    "ls-request-receive" : 1,
                                    "ls-acknowledge-receive" : 2,
                                    "ls-acknowledge-transmit" : 2,
                                    "db-description-transmit" : 3,
                                    "db-description-receive" : 2,
                                    "hello-transmit" : 124,
                                    "hello-receive" : 124,
                                    "ls-request-transmit" : 1,
                                    "ls-update-receive" : 3
                                 }
                              },
                              "openconfig-ospfv2-ext:neighbours" : {
                                 "neighbour" : [
                                    {
                                       "neighbor-id" : "6.6.6.5",
                                       "neighbor-address" : "32.1.0.1",
                                       "state" : {
                                          "priority" : 1,
                                          "option-value" : 66,
                                          "gr-helper-status" : "None",
                                          "state-changes" : 6,
                                          "backup-designated-router" : "32.1.0.2",
                                          "adjacency-state" : "openconfig-ospf-types:FULL",
                                          "neighbor-id" : "6.6.6.5",
                                          "optional-capabilities" : "*|O|-|-|-|-|E|-",
                                          "thread-inactivity-timer" : true,
                                          "thread-ls-request-retransmission" : true,
                                          "thread-ls-update-retransmission" : true,
                                          "retransmit-summary-queue-length" : 0,
                                          "area-id" : "0.0.0.0",
                                          "gr-active-helper" : false,
                                          "interface-address" : "32.1.0.2",
                                          "neighbor-address" : "32.1.0.1",
                                          "last-established-time" : "1197595",
                                          "designated-router" : "32.1.0.1",
                                          "database-summary-queue-length" : 0,
                                          "interface-name" : "Vlan3719",
                                          "link-state-request-queue-length" : 0,
                                          "dead-time" : "32410"
                                       }
                                    }
                                 ]
                              },
                              "id" : "Vlan3719",
                              "timers" : {
                                 "state" : {
                                    "retransmission-interval" : 5,
                                    "dead-interval" : 40,
                                    "openconfig-ospfv2-ext:wait-time" : 40,
                                    "hello-interval" : 10000,
                                    "openconfig-ospfv2-ext:hello-due" : 2409
                                 }
                              },
                              "state" : {
                                 "priority" : 1,
                                 "openconfig-ospfv2-ext:member-of-ospf-all-routers" : true,
                                 "openconfig-ospfv2-ext:index" : 310,
                                 "openconfig-ospfv2-ext:mtu" : 9100,
                                 "openconfig-ospfv2-ext:if-flags" : "<UP,BROADCAST,RUNNING,MULTICAST>",
                                 "openconfig-ospfv2-ext:adjacency-count" : 1,
                                 "openconfig-ospfv2-ext:address-len" : 24,
                                 "openconfig-ospfv2-ext:ospf-enable" : true,
                                 "openconfig-ospfv2-ext:cost" : 4,
                                 "openconfig-ospfv2-ext:address" : "32.1.0.2",
                                 "id" : "Vlan3719",
                                 "openconfig-ospfv2-ext:adjacency-status" : "Backup",
                                 "openconfig-ospfv2-ext:neighbor-count" : 1,
                                 "openconfig-ospfv2-ext:backup-designated-router-id" : "5.5.5.4",
                                 "openconfig-ospfv2-ext:member-of-ospf-designated-routers" : true,
                                 "openconfig-ospfv2-ext:router-id" : "5.5.5.4",
                                 "openconfig-ospfv2-ext:broadcast-address" : "32.1.0.255",
                                 "openconfig-ospfv2-ext:bandwidth" : 25000,
                                 "openconfig-ospfv2-ext:area-id" : "0.0.0.0",
                                 "openconfig-ospfv2-ext:ospf-interface-type" : "Broadcast",
                                 "openconfig-ospfv2-ext:backup-designated-router-address" : "32.1.0.2",
                                 "network-type" : "openconfig-ospf-types:BROADCAST_NETWORK",
                                 "openconfig-ospfv2-ext:transmit-delay" : 1,
                                 "openconfig-ospfv2-ext:operational-state" : "Up"
                              }
                           }
                        ]
                     },
                     "config" : {
                        "identifier" : "0.0.0.0"
                     },
                     "openconfig-ospfv2-ext:networks" : {
                        "network" : [
                           {
                              "address-prefix" : "31.1.0.0/24",
                              "config" : {
                                 "address-prefix" : "31.1.0.0/24",
                                 "description" : "Network prefix"
                              },
                              "state" : {
                                 "address-prefix" : "31.1.0.0/24",
                                 "description" : "Network prefix"
                              }
                           },
                           {
                              "address-prefix" : "32.1.0.0/24",
                              "config" : {
                                 "address-prefix" : "32.1.0.0/24",
                                 "description" : "Network prefix"
                              },
                              "state" : {
                                 "address-prefix" : "32.1.0.0/24",
                                 "description" : "Network prefix"
                              }
                           }
                        ]
                     },
                     "state" : {
                        "openconfig-ospfv2-ext:summary-lsa-count" : 2,
                        "openconfig-ospfv2-ext:nssa-lsa-checksum" : "0x00000000",
                        "openconfig-ospfv2-ext:authentication-type" : "no",
                        "openconfig-ospfv2-ext:asbr-summary-lsa-count" : 0,
                        "openconfig-ospfv2-ext:adjacency-count" : 1,
                        "openconfig-ospfv2-ext:opaque-area-lsa-count" : 0,
                        "openconfig-ospfv2-ext:opaque-area-lsa-checksum" : "0x00000000",
                        "openconfig-ospfv2-ext:lsa-count" : 5,
                        "openconfig-ospfv2-ext:opaque-link-lsa-checksum" : "0x00000000",
                        "openconfig-ospfv2-ext:spf-execution-count" : 6,
                        "openconfig-ospfv2-ext:interface-count" : 1,
                        "openconfig-ospfv2-ext:network-lsa-count" : 1,
                        "openconfig-ospfv2-ext:asbr-summary-lsa-checksum" : "0x00000000",
                        "openconfig-ospfv2-ext:router-lsa-checksum" : "0x0000cfe5",
                        "openconfig-ospfv2-ext:router-lsa-count" : 2,
                        "openconfig-ospfv2-ext:nssa-lsa-count" : 0,
                        "openconfig-ospfv2-ext:active-interface-count" : 1,
                        "openconfig-ospfv2-ext:summary-lsa-checksum" : "0x0000201c",
                        "openconfig-ospfv2-ext:network-lsa-checksum" : "0x0000ca2c",
                        "openconfig-ospfv2-ext:opaque-link-lsa-count" : 0
                     }
                  },
                  {
                     "lsdb" : {
                        "lsa-types" : {
                           "lsa-type" : [
                              {
                                 "type" : "openconfig-ospf-types:AS_EXTERNAL_LSA",
                                 "state" : {
                                    "type" : "openconfig-ospf-types:AS_EXTERNAL_LSA"
                                 },
                                 "lsas" : {
                                    "openconfig-ospfv2-ext:lsa-ext" : [
                                       {
                                          "advertising-router" : "5.5.5.4",
                                          "as-external-lsa" : {
                                             "types-of-service" : {
                                                "type-of-service" : [
                                                   {
                                                      "tos" : 0,
                                                      "state" : {
                                                         "forwarding-address" : "0.0.0.0",
                                                         "external-route-tag" : 0,
                                                         "metric" : 20
                                                      }
                                                   }
                                                ]
                                             },
                                             "state" : {
                                                "metric-type" : "TYPE_2",
                                                "mask" : 20
                                             }
                                          },
                                          "link-state-id" : "10.193.80.0",
                                          "state" : {
                                             "checksum" : 50393,
                                             "length" : 36,
                                             "flags" : 11,
                                             "display-sequence-number" : "80000001",
                                             "option" : 2,
                                             "option-expanded" : "*||||||E|",
                                             "age" : 1227
                                          }
                                       },
                                       {
                                          "advertising-router" : "6.6.6.5",
                                          "as-external-lsa" : {
                                             "types-of-service" : {
                                                "type-of-service" : [
                                                   {
                                                      "tos" : 0,
                                                      "state" : {
                                                         "forwarding-address" : "0.0.0.0",
                                                         "external-route-tag" : 0,
                                                         "metric" : 20
                                                      }
                                                   }
                                                ]
                                             },
                                             "state" : {
                                                "metric-type" : "TYPE_2",
                                                "mask" : 20
                                             }
                                          },
                                          "link-state-id" : "10.193.80.0",
                                          "state" : {
                                             "checksum" : 42739,
                                             "length" : 36,
                                             "flags" : 6,
                                             "display-sequence-number" : "80000001",
                                             "option" : 2,
                                             "option-expanded" : "*||||||E|",
                                             "age" : 1228
                                          }
                                       }
                                    ]
                                 }
                              },
                              {
                                 "type" : "openconfig-ospf-types:NETWORK_LSA",
                                 "state" : {
                                    "type" : "openconfig-ospf-types:NETWORK_LSA"
                                 },
                                 "lsas" : {
                                    "openconfig-ospfv2-ext:lsa-ext" : [
                                       {
                                          "advertising-router" : "5.5.5.4",
                                          "link-state-id" : "41.1.0.2",
                                          "state" : {
                                             "checksum" : 1017,
                                             "length" : 32,
                                             "flags" : 3,
                                             "display-sequence-number" : "80000001",
                                             "option" : 2,
                                             "option-expanded" : "*|-|-|-|-|-|E|-",
                                             "age" : 1198
                                          },
                                          "network-lsa" : {
                                             "state" : {
                                                "network-mask" : 24
                                             }
                                          }
                                       }
                                    ]
                                 }
                              },
                              {
                                 "type" : "openconfig-ospf-types:OSPFV2_LINK_SCOPE_OPAQUE_LSA"
                              },
                              {
                                 "type" : "openconfig-ospf-types:ROUTER_LSA",
                                 "state" : {
                                    "type" : "openconfig-ospf-types:ROUTER_LSA"
                                 },
                                 "lsas" : {
                                    "openconfig-ospfv2-ext:lsa-ext" : [
                                       {
                                          "advertising-router" : "3.3.3.2",
                                          "link-state-id" : "3.3.3.2",
                                          "state" : {
                                             "checksum" : 53771,
                                             "length" : 36,
                                             "flags" : 6,
                                             "display-sequence-number" : "80000002",
                                             "option" : 0,
                                             "option-expanded" : "*|||||||-",
                                             "age" : 1189
                                          },
                                          "router-lsa" : {
                                             "link-informations" : {
                                                "link-information" : [
                                                   {
                                                      "state" : {
                                                         "metric" : 10,
                                                         "type" : "openconfig-ospf-types:ROUTER_LSA_TRANSIT_NETWORK",
                                                         "number-tos-metrics" : 0
                                                      }
                                                   }
                                                ]
                                             },
                                             "types-of-service" : {
                                                "type-of-service" : [
                                                   {
                                                      "tos" : 0,
                                                      "state" : {
                                                         "metric" : 10
                                                      }
                                                   }
                                                ]
                                             },
                                             "state" : {
                                                "flags-description" : " :",
                                                "flags" : 0,
                                                "number-links" : 1
                                             }
                                          }
                                       },
                                       {
                                          "advertising-router" : "5.5.5.4",
                                          "link-state-id" : "5.5.5.4",
                                          "state" : {
                                             "checksum" : 12693,
                                             "length" : 36,
                                             "flags" : 3,
                                             "display-sequence-number" : "80000003",
                                             "option" : 2,
                                             "option-expanded" : "*||||||E|",
                                             "age" : 1198
                                          },
                                          "router-lsa" : {
                                             "link-informations" : {
                                                "link-information" : [
                                                   {
                                                      "state" : {
                                                         "metric" : 10,
                                                         "type" : "openconfig-ospf-types:ROUTER_LSA_TRANSIT_NETWORK",
                                                         "number-tos-metrics" : 0
                                                      }
                                                   }
                                                ]
                                             },
                                             "types-of-service" : {
                                                "type-of-service" : [
                                                   {
                                                      "tos" : 0,
                                                      "state" : {
                                                         "metric" : 10
                                                      }
                                                   }
                                                ]
                                             },
                                             "state" : {
                                                "flags-description" : " : ABR ASBR",
                                                "flags" : 3,
                                                "number-links" : 1
                                             }
                                          }
                                       }
                                    ]
                                 }
                              },
                              {
                                 "type" : "openconfig-ospf-types:SUMMARY_ASBR_LSA",
                                 "state" : {
                                    "type" : "openconfig-ospf-types:SUMMARY_ASBR_LSA"
                                 },
                                 "lsas" : {
                                    "openconfig-ospfv2-ext:lsa-ext" : [
                                       {
                                          "advertising-router" : "5.5.5.4",
                                          "link-state-id" : "6.6.6.5",
                                          "summary-lsa" : {
                                             "types-of-service" : {
                                                "type-of-service" : [
                                                   {
                                                      "tos" : 0,
                                                      "state" : {
                                                         "metric" : 4
                                                      }
                                                   }
                                                ]
                                             },
                                             "state" : {
                                                "network-mask" : 0
                                             }
                                          },
                                          "state" : {
                                             "checksum" : 59716,
                                             "length" : 28,
                                             "flags" : 11,
                                             "display-sequence-number" : "80000001",
                                             "option" : 2,
                                             "option-expanded" : "*|-|-|-|-|-|E|-",
                                             "age" : 1187
                                          }
                                       }
                                    ]
                                 }
                              },
                              {
                                 "type" : "openconfig-ospf-types:SUMMARY_IP_NETWORK_LSA",
                                 "state" : {
                                    "type" : "openconfig-ospf-types:SUMMARY_IP_NETWORK_LSA"
                                 },
                                 "lsas" : {
                                    "openconfig-ospfv2-ext:lsa-ext" : [
                                       {
                                          "advertising-router" : "5.5.5.4",
                                          "link-state-id" : "21.1.0.0",
                                          "summary-lsa" : {
                                             "types-of-service" : {
                                                "type-of-service" : [
                                                   {
                                                      "tos" : 0,
                                                      "state" : {
                                                         "metric" : 14
                                                      }
                                                   }
                                                ]
                                             },
                                             "state" : {
                                                "network-mask" : 24
                                             }
                                          },
                                          "state" : {
                                             "checksum" : 18908,
                                             "length" : 28,
                                             "flags" : 11,
                                             "display-sequence-number" : "80000001",
                                             "option" : 2,
                                             "option-expanded" : "*|-|-|-|-|-|E|-",
                                             "age" : 1187
                                          }
                                       },
                                       {
                                          "advertising-router" : "5.5.5.4",
                                          "link-state-id" : "32.1.0.0",
                                          "summary-lsa" : {
                                             "types-of-service" : {
                                                "type-of-service" : [
                                                   {
                                                      "tos" : 0,
                                                      "state" : {
                                                         "metric" : 4
                                                      }
                                                   }
                                                ]
                                             },
                                             "state" : {
                                                "network-mask" : 24
                                             }
                                          },
                                          "state" : {
                                             "checksum" : 21967,
                                             "length" : 28,
                                             "flags" : 11,
                                             "display-sequence-number" : "80000001",
                                             "option" : 2,
                                             "option-expanded" : "*|-|-|-|-|-|E|-",
                                             "age" : 1229
                                          }
                                       }
                                    ]
                                 }
                              }
                           ]
                        },
                        "state" : {
                           "identifier" : "0.0.0.2"
                        }
                     },
                     "identifier" : "0.0.0.2",
                     "interfaces" : {
                        "interface" : [
                           {
                              "openconfig-ospfv2-ext:message-statistics" : {
                                 "state" : {
                                    "ls-update-transmit" : 3,
                                    "ls-request-receive" : 0,
                                    "ls-acknowledge-receive" : 3,
                                    "ls-acknowledge-transmit" : 2,
                                    "db-description-transmit" : 2,
                                    "db-description-receive" : 6,
                                    "hello-transmit" : 124,
                                    "hello-receive" : 125,
                                    "ls-request-transmit" : 1,
                                    "ls-update-receive" : 2
                                 }
                              },
                              "openconfig-ospfv2-ext:neighbours" : {
                                 "neighbour" : [
                                    {
                                       "neighbor-id" : "3.3.3.2",
                                       "neighbor-address" : "41.1.0.1",
                                       "state" : {
                                          "priority" : 0,
                                          "option-value" : 66,
                                          "gr-helper-status" : "None",
                                          "state-changes" : 6,
                                          "backup-designated-router" : "0.0.0.0",
                                          "adjacency-state" : "openconfig-ospf-types:FULL",
                                          "neighbor-id" : "6.6.6.5",
                                          "optional-capabilities" : "*|O|-|-|-|-|E|-",
                                          "thread-inactivity-timer" : true,
                                          "thread-ls-request-retransmission" : true,
                                          "thread-ls-update-retransmission" : true,
                                          "retransmit-summary-queue-length" : 0,
                                          "area-id" : "0.0.0.2",
                                          "gr-active-helper" : false,
                                          "interface-address" : "41.1.0.2",
                                          "neighbor-address" : "41.1.0.1",
                                          "last-established-time" : "1198302",
                                          "designated-router" : "41.1.0.2",
                                          "database-summary-queue-length" : 0,
                                          "interface-name" : "Ethernet120",
                                          "link-state-request-queue-length" : 0,
                                          "dead-time" : "34872"
                                       }
                                    }
                                 ]
                              },
                              "id" : "Ethernet120",
                              "timers" : {
                                 "state" : {
                                    "retransmission-interval" : 5,
                                    "dead-interval" : 40,
                                    "openconfig-ospfv2-ext:wait-time" : 40,
                                    "hello-interval" : 10000,
                                    "openconfig-ospfv2-ext:hello-due" : 1711
                                 }
                              },
                              "state" : {
                                 "priority" : 1,
                                 "openconfig-ospfv2-ext:member-of-ospf-all-routers" : true,
                                 "openconfig-ospfv2-ext:index" : 299,
                                 "openconfig-ospfv2-ext:mtu" : 9100,
                                 "openconfig-ospfv2-ext:if-flags" : "<UP,BROADCAST,RUNNING,MULTICAST>",
                                 "openconfig-ospfv2-ext:adjacency-count" : 1,
                                 "openconfig-ospfv2-ext:address-len" : 24,
                                 "openconfig-ospfv2-ext:ospf-enable" : true,
                                 "openconfig-ospfv2-ext:cost" : 10,
                                 "openconfig-ospfv2-ext:address" : "41.1.0.2",
                                 "id" : "Ethernet120",
                                 "openconfig-ospfv2-ext:adjacency-status" : "DR",
                                 "openconfig-ospfv2-ext:member-of-ospf-designated-routers" : true,
                                 "openconfig-ospfv2-ext:neighbor-count" : 1,
                                 "openconfig-ospfv2-ext:router-id" : "5.5.5.4",
                                 "openconfig-ospfv2-ext:broadcast-address" : "41.1.0.255",
                                 "openconfig-ospfv2-ext:bandwidth" : 10000,
                                 "openconfig-ospfv2-ext:area-id" : "0.0.0.2",
                                 "openconfig-ospfv2-ext:ospf-interface-type" : "Broadcast",
                                 "network-type" : "openconfig-ospf-types:BROADCAST_NETWORK",
                                 "openconfig-ospfv2-ext:transmit-delay" : 1,
                                 "openconfig-ospfv2-ext:operational-state" : "Up"
                              }
                           }
                        ]
                     },
                     "config" : {
                        "identifier" : "0.0.0.2"
                     },
                     "openconfig-ospfv2-ext:networks" : {
                        "network" : [
                           {
                              "address-prefix" : "41.1.0.0/24",
                              "config" : {
                                 "address-prefix" : "41.1.0.0/24",
                                 "description" : "Network prefix"
                              },
                              "state" : {
                                 "address-prefix" : "41.1.0.0/24",
                                 "description" : "Network prefix"
                              }
                           }
                        ]
                     },
                     "state" : {
                        "openconfig-ospfv2-ext:summary-lsa-count" : 2,
                        "openconfig-ospfv2-ext:nssa-lsa-checksum" : "0x00000000",
                        "openconfig-ospfv2-ext:authentication-type" : "no",
                        "openconfig-ospfv2-ext:asbr-summary-lsa-count" : 1,
                        "openconfig-ospfv2-ext:adjacency-count" : 1,
                        "openconfig-ospfv2-ext:opaque-area-lsa-count" : 0,
                        "openconfig-ospfv2-ext:opaque-area-lsa-checksum" : "0x00000000",
                        "openconfig-ospfv2-ext:lsa-count" : 6,
                        "openconfig-ospfv2-ext:opaque-link-lsa-checksum" : "0x00000000",
                        "openconfig-ospfv2-ext:virtual-link-adjacency-count" : 0,
                        "openconfig-ospfv2-ext:spf-execution-count" : 6,
                        "openconfig-ospfv2-ext:interface-count" : 1,
                        "openconfig-ospfv2-ext:network-lsa-count" : 1,
                        "openconfig-ospfv2-ext:asbr-summary-lsa-checksum" : "0x0000e944",
                        "openconfig-ospfv2-ext:router-lsa-checksum" : "0x000103a0",
                        "openconfig-ospfv2-ext:router-lsa-count" : 2,
                        "openconfig-ospfv2-ext:nssa-lsa-count" : 0,
                        "openconfig-ospfv2-ext:active-interface-count" : 1,
                        "openconfig-ospfv2-ext:shortcut" : "openconfig-ospfv2-ext:DEFAULT",
                        "openconfig-ospfv2-ext:summary-lsa-checksum" : "0x00009fab",
                        "openconfig-ospfv2-ext:network-lsa-checksum" : "0x000003f9",
                        "openconfig-ospfv2-ext:opaque-link-lsa-count" : 0
                     }
                  }
               ]
            }
         },
         "name" : "ospfv2",
         "config" : {
            "identifier" : "openconfig-policy-types:OSPF",
            "name" : "ospfv2"
         },
         "state" : {
            "identifier" : "openconfig-policy-types:OSPF",
            "name" : "ospfv2"
         }
      }
   ]
}
            `),
	},
}
