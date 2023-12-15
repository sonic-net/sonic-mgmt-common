////////////////////////////////////////////////////////////////////////////////
//                                                                            //
//  Copyright 2020 Broadcom. The term Broadcom refers to Broadcom Inc. and/or //
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

package util

import (
	"github.com/go-redis/redis/v7"
)

var FILTER_ENTRIES_LUASCRIPT *redis.Script = redis.NewScript(`
    --ARGV[1] => Key patterns
    --ARGV[2] => Key names separated by '|'
    --ARGV[3] => predicate patterns
    --ARGV[4] => Fields to return
    --ARGV[5] => Count of entries to return

    local tableData = {} ; local tbl = {}

    local keys = redis.call('KEYS', ARGV[1])
    if #keys == 0 then return end

    local count = -1
    if (ARGV[5] ~= nil and ARGV[5] ~= "") then count=tonumber(ARGV[5]) end

    local sepStart = string.find(keys[1], "|")
    if sepStart == nil then return ; end

    -- Function to load lua predicate code
    local function loadPredicateScript(str)
        if (str == nil or str == "") then return nil; end

        local f, err = loadstring("return function (k,h) " .. str .. " end")
        if f then return f(); else return nil;end
    end

    local keySetNames = {}
    ARGV[2]:gsub("([^|]+)",function(c) table.insert(keySetNames, c) end)

    local predicate = loadPredicateScript(ARGV[3])

    for _, key in ipairs(keys) do
        local hash = redis.call('HGETALL', key)
        local row = {}; local keySet = {}; local keyVal = {}
        local keyOnly = string.sub(key, sepStart+1)

        for index = 1, #hash, 2 do
            row[hash[index]] = hash[index + 1]
        end

        --Split key values
        keyOnly:gsub("([^|]+)", function(c)  table.insert(keyVal, c) end)

        for idx = 1, #keySetNames, 1 do
            keySet[keySetNames[idx]] = keyVal[idx]
        end

        if (predicate == nil) or (predicate(keySet, row) == true) then
            tbl[keyOnly] = row
        end

        if (count ~= -1 and #tbl == count) then break end

    end

    tableData[string.sub(keys[1], 0, sepStart-1)] = tbl

    return cjson.encode(tableData)
`)
