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
//go:build prune_xfmrtest
// +build prune_xfmrtest

package transformer

import (
	"reflect"

	"github.com/Azure/sonic-mgmt-common/translib/ocbinds"
)

func init() {
	xp_tests = append(xp_tests, acl_xp_tests...)
}

var acl_xp_tests = []xpTests{
	{ // ACL Identity Test Case (i.e. no QP)
		tid:        "ACL Identity",
		uri:        "/openconfig-acl:acl/acl-sets",
		requestUri: "/openconfig-acl:acl/acl-sets",
		payload: []byte(`
            {
               "acl-sets" : {
               "acl-set" : [
                  {
                     "name" : "MyACL3",
                     "config" : {
                        "name" : "MyACL3",
                        "type" : "ACL_IPV4",
                        "description" : "Description for MyACL3"
                     },
                     "state" : {
                        "name" : "MyACL3",
                        "type" : "ACL_IPV4",
                        "description" : "Description for MyACL3"
                     },
                     "type" : "ACL_IPV4"
                  }
               ]
               }
            }`),
		appRootType: reflect.TypeOf(ocbinds.OpenconfigAcl_Acl{}),
		queryParams: QueryParams{
			depthEnabled:  false,
			curDepth:      1,
			content:       QUERY_CONTENT_ALL,
			fields:        []string{},
			fieldsFillAll: false,
		},
		prunedPayload: []byte(`
            {
               "acl-sets" : {
               "acl-set" : [
                  {
                     "name" : "MyACL3",
                     "config" : {
                        "name" : "MyACL3",
                        "type" : "ACL_IPV4",
                        "description" : "Description for MyACL3"
                     },
                     "state" : {
                        "name" : "MyACL3",
                        "type" : "ACL_IPV4",
                        "description" : "Description for MyACL3"
                     },
                     "type" : "ACL_IPV4"
                  }
               ]
               }
            }`),
	},
	{ // ACL depth = 3
		tid:        "ACL Depth",
		uri:        "/openconfig-acl:acl/acl-sets",
		requestUri: "/openconfig-acl:acl/acl-sets",
		payload: []byte(`
            {
               "acl-sets" : {
               "acl-set" : [
                  {
                     "name" : "MyACL3",
                     "config" : {
                        "name" : "MyACL3",
                        "type" : "ACL_IPV4",
                        "description" : "Description for MyACL3"
                     },
                     "state" : {
                        "name" : "MyACL3",
                        "type" : "ACL_IPV4",
                        "description" : "Description for MyACL3"
                     },
                     "type" : "ACL_IPV4"
                  }
               ]
               }
            }`),
		appRootType: reflect.TypeOf(ocbinds.OpenconfigAcl_Acl{}),
		queryParams: QueryParams{
			depthEnabled:  true,
			curDepth:      3,
			content:       QUERY_CONTENT_ALL,
			fields:        []string{},
			fieldsFillAll: false,
		},
		prunedPayload: []byte(`
            {
               "acl-sets" : {
               "acl-set" : [
                  {
                     "name" : "MyACL3",
                     "config" : {
                     },
                     "state" : {
                     },
                     "type" : "ACL_IPV4"
                  }
               ]
               }
            }`),
	},
	{ // ACL content = QUERY_CONTENT_CONFIG
		tid:        "ACL Content Config",
		uri:        "/openconfig-acl:acl/acl-sets",
		requestUri: "/openconfig-acl:acl/acl-sets",
		payload: []byte(`
            {
               "acl-sets" : {
               "acl-set" : [
                  {
                     "name" : "MyACL3",
                     "config" : {
                        "name" : "MyACL3",
                        "type" : "ACL_IPV4",
                        "description" : "Description for MyACL3"
                     },
                     "state" : {
                        "name" : "MyACL3",
                        "type" : "ACL_IPV4",
                        "description" : "Description for MyACL3"
                     },
                     "type" : "ACL_IPV4"
                  }
               ]
               }
            }`),
		appRootType: reflect.TypeOf(ocbinds.OpenconfigAcl_Acl{}),
		queryParams: QueryParams{
			depthEnabled:  false,
			curDepth:      1,
			content:       QUERY_CONTENT_CONFIG,
			fields:        []string{},
			fieldsFillAll: false,
		},
		prunedPayload: []byte(`
            {
               "acl-sets" : {
               "acl-set" : [
                  {
                     "name" : "MyACL3",
                     "config" : {
                        "name" : "MyACL3",
                        "type" : "ACL_IPV4",
                        "description" : "Description for MyACL3"
                     },
                     "type" : "ACL_IPV4"
                  }
               ]
               }
            }`),
	},
	{ // ACL content = QUERY_CONTENT_NONCONFIG
		tid:        "ACL Content Nonconfig",
		uri:        "/openconfig-acl:acl/acl-sets",
		requestUri: "/openconfig-acl:acl/acl-sets",
		payload: []byte(`
            {
               "acl-sets" : {
               "acl-set" : [
                  {
                     "name" : "MyACL3",
                     "config" : {
                        "name" : "MyACL3",
                        "type" : "ACL_IPV4",
                        "description" : "Description for MyACL3"
                     },
                     "state" : {
                        "name" : "MyACL3",
                        "type" : "ACL_IPV4",
                        "description" : "Description for MyACL3"
                     },
                     "type" : "ACL_IPV4"
                  }
               ]
               }
            }`),
		appRootType: reflect.TypeOf(ocbinds.OpenconfigAcl_Acl{}),
		queryParams: QueryParams{
			depthEnabled:  false,
			curDepth:      1,
			content:       QUERY_CONTENT_NONCONFIG,
			fields:        []string{},
			fieldsFillAll: false,
		},
		prunedPayload: []byte(`
            {
               "acl-sets" : {
               "acl-set" : [
                  {
                     "name" : "MyACL3",
                     "state" : {
                        "name" : "MyACL3",
                        "type" : "ACL_IPV4",
                        "description" : "Description for MyACL3"
                     },
                     "type" : "ACL_IPV4"
                  }
               ]
               }
            }`),
	},
	{ // ACL fields
		tid:        "ACL Fields",
		uri:        "/openconfig-acl:acl/acl-sets/acl-set[name=MyACL3][type=ACL_IPV4]",
		requestUri: "/openconfig-acl:acl/acl-sets/acl-set[name=MyACL3][type=ACL_IPV4]",
		payload: []byte(`
            {
               "openconfig-acl:acl-set" : [
                  {
                     "name" : "MyACL3",
                     "config" : {
                        "name" : "MyACL3",
                        "type" : "ACL_IPV4",
                        "description" : "Description for MyACL3"
                     },
                     "state" : {
                        "name" : "MyACL3",
                        "type" : "ACL_IPV4",
                        "description" : "Description for MyACL3"
                     },
                     "type" : "ACL_IPV4"
                  }
               ]
            }`),
		appRootType: reflect.TypeOf(ocbinds.OpenconfigAcl_Acl{}),
		queryParams: QueryParams{
			depthEnabled:  false,
			curDepth:      1,
			content:       QUERY_CONTENT_ALL,
			fields:        []string{"config/description", "name", "type"},
			fieldsFillAll: false,
		},
		prunedPayload: []byte(`
            {
               "openconfig-acl:acl-set" : [
                  {
                     "name" : "MyACL3",
                     "config" : {
                        "description" : "Description for MyACL3"
                     },
                     "type" : "ACL_IPV4"
                  }
               ]
            }`),
	},
	{ // ACL depth = 3, content = QUERY_CONTENT_CONFIG
		tid:        "ACL Depth and content",
		uri:        "/openconfig-acl:acl/acl-sets",
		requestUri: "/openconfig-acl:acl/acl-sets",
		payload: []byte(`
            {
               "acl-sets" : {
               "acl-set" : [
                  {
                     "name" : "MyACL3",
                     "config" : {
                        "name" : "MyACL3",
                        "type" : "ACL_IPV4",
                        "description" : "Description for MyACL3"
                     },
                     "state" : {
                        "name" : "MyACL3",
                        "type" : "ACL_IPV4",
                        "description" : "Description for MyACL3"
                     },
                     "type" : "ACL_IPV4"
                  }
               ]
               }
            }`),
		appRootType: reflect.TypeOf(ocbinds.OpenconfigAcl_Acl{}),
		queryParams: QueryParams{
			depthEnabled:  true,
			curDepth:      3,
			content:       QUERY_CONTENT_CONFIG,
			fields:        []string{},
			fieldsFillAll: false,
		},
		prunedPayload: []byte(`
            {
               "acl-sets" : {
               "acl-set" : [
                  {
                     "name" : "MyACL3",
                     "config" : {
                     },
                     "type" : "ACL_IPV4"
                  }
               ]
               }
            }`),
	},
}
