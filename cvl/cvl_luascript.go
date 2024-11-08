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

package cvl

import (
	"errors"

	"github.com/go-redis/redis/v7"
)

// RunLua is a temporary API to run a named lua script in ConfigDb. Script name
// can be one of "count_entries" or "filter_entries".
// TODO move the script definitions to DBAL and remove this API
func RunLua(name string, args ...interface{}) (interface{}, error) {
	script, ok := luaScripts[name]
	if !ok || script == nil {
		return nil, errors.New("unknown script: " + name)
	}
	return script.Run(redisClient, []string{}, args...).Result()
}

// Redis server side script
func loadLuaScript(luaScripts map[string]*redis.Script) {

	//Find current number of entries in a table
	luaScripts["count_entries"] = redis.NewScript(`
	--ARGV[1] => Key patterns
	--ARGV[2] => Key names separated by '|'
	--ARGV[3] => predicate patterns
	--ARGV[4] => Field
	--ARGV[5] => Tx db entries

	local txEntries = cjson.decode(ARGV[5])

	-- count with filter
	local keys = redis.call('KEYS', ARGV[1])

	local cnt = 0

	-- Function to load lua predicate code
	local function loadPredicateScript(str)
		if (str == nil or str == "") then return nil; end

		local f, err = loadstring("return function (k,h) " .. str .. " end")
		if f then return f(); else return nil;end
	end

	local keySetNames = {}
	ARGV[2]:gsub("([^|]+)",function(c) table.insert(keySetNames, c) end)

	local predicate = loadPredicateScript(ARGV[3])

	local field = ""
	if (ARGV[4] ~= nil) then field = ARGV[4]; end

	local isRow = false
	if predicate ~= nil or #field > 0  then
		isRow = true
	end

	for _, k in ipairs(keys) do
		if txEntries[k] == nil then
			txEntries[k] = {}
		end
	end

	local tblKey = next(txEntries)
	if tblKey == nil then return 0 end

	local sepStart = string.find(tblKey, "|")
	if sepStart == nil then return ; end

    for key, val in pairs(txEntries) do
		if type(val) == 'table' then
			local keyOnly = string.sub(key, sepStart+1)
			local row = {}; local keySet = {}; local keyVal = {}
			if isRow then
				if next(val) ~= nil then
					row = val
				else
					local hash = redis.call('HGETALL', key)
					for index = 1, #hash, 2 do
						row[hash[index]] = hash[index + 1]
					end
				end
			end

			local incFlag = false
			if (predicate == nil) then
				incFlag = true
			else
				--Split key values
				keyOnly:gsub("([^|]+)", function(c)  table.insert(keyVal, c) end)

				if (#keySetNames == 0) then
					keySet = keyVal
				else
					for idx = 1, #keySetNames, 1 do
						keySet[keySetNames[idx]] = keyVal[idx]
					end
				end

				if (predicate(keySet, row) == true) then
					incFlag = true
				end
			end

			if (incFlag == true) then
				if (field ~= "") then
					if (row[field] ~= nil) then
						cnt = cnt + 1
					elseif (row[field.."@"] ~= nil) then
						row[field.."@"]:gsub("([^,]+)", function(c) cnt = cnt + 1 end)
					elseif (string.match(ARGV[2], field.."[|]?") ~= nil) then
						cnt = cnt + 1
					end
				else
					cnt = cnt + 1
				end
			end
		end
	end

	return cnt
	`)

	//Get filtered entries as per given key filters and predicate
	luaScripts["filter_entries"] = redis.NewScript(`
    --ARGV[1] => Key patterns
    --ARGV[2] => Key names separated by '|'
    --ARGV[3] => predicate patterns
    --ARGV[4] => Fields to return
    --ARGV[5] => Count of entries to return
	--ARGV[6] => Tx db entries

	local txEntries = cjson.decode(ARGV[6])

    local tableData = {} ; local tbl = {}

    local keys = redis.call('KEYS', ARGV[1])

    local count = -1
    if (ARGV[5] ~= nil and ARGV[5] ~= "") then count=tonumber(ARGV[5]) end

    -- Function to load lua predicate code
    local function loadPredicateScript(str)
        if (str == nil or str == "") then return nil; end

        local f, err = loadstring("return function (k,h) " .. str .. " end")
        if f then return f(); else return nil;end
    end

    local keySetNames = {}
    ARGV[2]:gsub("([^|]+)",function(c) table.insert(keySetNames, c) end)

    local predicate = loadPredicateScript(ARGV[3])

	for _, k in ipairs(keys) do
		if txEntries[k] == nil then
			txEntries[k] = {}
		end
	end

	local tblKey = next(txEntries)
	if tblKey == nil then return end

	local sepStart = string.find(tblKey, "|")
	if sepStart == nil then return ; end

    local entryCount = 0
    for key, val in pairs(txEntries) do
		if type(val) == 'table' then
			local row = {}; local keySet = {}; local keyVal = {}
			local keyOnly = string.sub(key, sepStart+1)

			if next(val) ~= nil then
				row = val
			else
				local hash = redis.call('HGETALL', key)

				for index = 1, #hash, 2 do
					row[hash[index]] = hash[index + 1]
				end
			end

			--Split key values
			keyOnly:gsub("([^|]+)", function(c)  table.insert(keyVal, c) end)

			if (#keySetNames == 0) then
				keySet = keyVal
			else
				for idx = 1, #keySetNames, 1 do
					keySet[keySetNames[idx]] = keyVal[idx]
				end
			end

			if (predicate == nil) or (predicate(keySet, row) == true) then
				tbl[keyOnly] = row
				entryCount = entryCount + 1
			end

			if (count ~= -1 and entryCount >= count) then break end
		end
    end

	if entryCount == 0 then return end

    tableData[string.sub(tblKey, 0, sepStart-1)] = tbl

    return cjson.encode(tableData)
`)
}
