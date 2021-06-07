//////////////////////////////////////////////////////////////////////////
//
// Copyright 2021 Dell, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
//////////////////////////////////////////////////////////////////////////

package transformer

import (
	"encoding/json"
	"errors"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/Azure/sonic-mgmt-common/translib/db"
	"github.com/Azure/sonic-mgmt-common/translib/utils"
	log "github.com/golang/glog"
)

func init() {
	XlateFuncBind("DbToYang_event_severity_xfmr", DbToYang_event_severity_xfmr)
	XlateFuncBind("DbToYang_alarm_severity_xfmr", DbToYang_alarm_severity_xfmr)
	XlateFuncBind("rpc_unacknowledge_alarms", rpc_unacknowledge_alarms)
	XlateFuncBind("rpc_acknowledge_alarms", rpc_acknowledge_alarms)
	XlateFuncBind("rpc_get_events", rpc_get_events)
	XlateFuncBind("rpc_get_alarms", rpc_get_alarms)
	XlateFuncBind("YangToDb_event_stats_key_xfmr", YangToDb_event_stats_key_xfmr)
	XlateFuncBind("YangToDb_alarm_stats_key_xfmr", YangToDb_alarm_stats_key_xfmr)
}

var YangToDb_event_stats_key_xfmr KeyXfmrYangToDb = func(inParams XfmrParams) (string, error) {
	return "state", nil
}

var YangToDb_alarm_stats_key_xfmr KeyXfmrYangToDb = func(inParams XfmrParams) (string, error) {
	return "state", nil
}

func severity_xfmr(inParams XfmrParams, tbl string) (map[string]interface{}, error) {
	var err error
	result := make(map[string]interface{})
	data := (*inParams.dbDataMap)[inParams.curDb]
	pTbl := data[tbl]
	if _, ok := pTbl[inParams.key]; !ok {
		return result, err
	}

	pRec := pTbl[inParams.key]
	sev, ok := pRec.Field["severity"]
	if ok {
		result["severity"] = "openconfig-alarm-types:" + sev
	}
	return result, err
}

var DbToYang_event_severity_xfmr FieldXfmrDbtoYang = func(inParams XfmrParams) (map[string]interface{}, error) {
	return severity_xfmr(inParams, "EVENT")
}

var DbToYang_alarm_severity_xfmr FieldXfmrDbtoYang = func(inParams XfmrParams) (map[string]interface{}, error) {
	return severity_xfmr(inParams, "ALARM")
}

func get_events_with_filter_time(d *db.DB, tableName string, begin time.Time, end time.Time) ([]interface{}, error) {
	var events []interface{}

	log.Info(" Begin time: ", begin.UnixNano(), ", End time:", end.UnixNano())
	log.Infof("Begin time: %s, End time: %s", begin, end)

	table, err := d.GetTable(&db.TableSpec{Name: tableName})

	if err != nil {
		log.Error("Reading event Table failed.", err)
		return events, err
	}
	var keys []db.Key
	keys, err = table.GetKeys()

	if err != nil {
		log.Error("Get event table keys failed.", err)
		return events, err
	}

	for _, key := range keys {
		if entry, err := table.GetEntry(key); err == nil {

			var time_created string
			var ok bool
			if time_created, ok = entry.Field["time-created"]; !ok {
				continue
			}

			t_nano, _ := strconv.ParseInt(time_created, 10, 64)
			if t_nano <= end.UnixNano() && t_nano >= begin.UnixNano() {
				events = append(events, entry.Field)
			}
		}
	}

	return events, err
}

func get_events_with_filter_id(d *db.DB, tableName string, begin string, end string) ([]interface{}, error) {
	var begin_id, end_id uint64
	var events []interface{}
	var keys []db.Key

	table, err := d.GetTable(&db.TableSpec{Name: tableName})

	if err != nil {
		log.Error("Reading the table failed.", err)
		return events, err
	}

	keys, err = table.GetKeys()

	if err != nil {
		log.Error("Get event table keys failed.", err)
		return events, err
	}

	if end_id, err = strconv.ParseUint(end, 10, 64); err != nil {
		end_id = math.MaxUint64
	}

	if begin_id, err = strconv.ParseUint(begin, 10, 64); err != nil {
		begin_id = 0
	}

	for _, key := range keys {
		if entry, err := table.GetEntry(key); err == nil {
			var id_str string
			var ok bool
			if id_str, ok = entry.Field["id"]; !ok {
				continue
			}
			id, _ := strconv.ParseUint(id_str, 10, 64)

			if id <= end_id && id >= begin_id {
				events = append(events, entry.Field)
			}
		}
	}
	return events, err
}

func get_events_with_filter_severity(d *db.DB, tableName string, severity string) ([]interface{}, error) {
	var events []interface{}
	table, err := d.GetTable(&db.TableSpec{Name: tableName})

	if err != nil {
		log.Error("Reading the table failed.", err)
		return events, err
	}
	var keys []db.Key
	keys, err = table.GetKeys()

	if err != nil {
		log.Error("Get event table keys failed.", err)
		return events, err
	}

	for _, key := range keys {
		if entry, err := table.GetEntry(key); err == nil {
			var ev_sev string
			var ok bool
			if ev_sev, ok = entry.Field["severity"]; !ok {
				continue
			}
			entry.Field["id"] = key.Comp[0]
			if strings.EqualFold(ev_sev, severity) {
				events = append(events, entry.Field)
			}
		}
	}
	return events, err
}

func process_show_request(dbs [db.MaxDB]*db.DB, table string, input interface{}) ([]interface{}, error) {
	var err error
	var events []interface{}

	mapData := input.(map[string]interface{})

	if id, ok := mapData["id"].(interface{}); ok {
		idMap := id.(map[string]interface{})
		var begin, end string
		if begin, ok = idMap["begin"].(string); !ok {
			begin = ""
		}

		if end, ok = idMap["end"].(string); !ok {
			end = ""
		}
		log.Info("begin ", begin, " :end ", end)
		events, err = get_events_with_filter_id(dbs[db.EventDB], table, begin, end)

	} else if interval, ok := mapData["interval"].(string); ok {
		end_time := time.Now()
		begin_time := time.Now()
		is_interval_valid := true
		if interval == "5_MINUTES" {
			begin_time = begin_time.Add(-time.Minute * 5)
		} else if interval == "60_MINUTES" {
			begin_time = begin_time.Add(-time.Minute * 60)
		} else if interval == "24_HOURS" {
			begin_time = begin_time.Add(-time.Hour * 24)
		} else {
			is_interval_valid = false
			log.Info("Error interval time ", interval)
			err = errors.New("Invalid interval")
		}
		if is_interval_valid {
			events, err = get_events_with_filter_time(dbs[db.EventDB], table, begin_time, end_time)
		}

	} else if time_filter, ok := mapData["time"].(interface{}); ok {

		timeMap := time_filter.(map[string]interface{})

		begin := time.Time{}
		end := time.Now()

		if begin_time, ok := timeMap["begin"].(string); ok {
			begin, err = time.Parse(time.RFC3339, begin_time)
			if err != nil {
				log.Info("Error parsing begin time ", begin_time)
			}
		}

		if end_time, ok := timeMap["end"].(string); ok {
			end, err = time.Parse(time.RFC3339, end_time)
			if err != nil {
				log.Info("Error parsing end time ", end_time)
			}
		}
		events, err = get_events_with_filter_time(dbs[db.EventDB], table, begin, end)
	} else if severity, ok := mapData["severity"].(string); ok {

		events, err = get_events_with_filter_severity(dbs[db.EventDB], table, severity)
	}
	return events, err
}

var rpc_get_events RpcCallpoint = func(body []byte, dbs [db.MaxDB]*db.DB) ([]byte, error) {

	var mapData map[string]interface{}
	var events []interface{}

	err := json.Unmarshal(body, &mapData)
	if err != nil {
		log.Info("Failed to unmarshall given input data")
		return nil, err
	}

	var result struct {
		Output struct {
			Status        int32  `json:"status"`
			Status_detail string `json:"status-detail"`
			EVENT         struct {
				EVENT_LIST []interface{} `json:"EVENT_LIST"`
			} `json:"EVENT"`
		} `json:"sonic-event:output"`
	}

	log.Info("rpc_get_events map_data: ", mapData)

	input := mapData["sonic-event:input"]
	events, err = process_show_request(dbs, "EVENT", input)
	if err == nil {
		result.Output.Status = 0
		result.Output.EVENT.EVENT_LIST = events
	} else {
		result.Output.Status = 1
		result.Output.Status_detail = string(err.Error())
	}

	return json.Marshal(&result)

}

var rpc_get_alarms RpcCallpoint = func(body []byte, dbs [db.MaxDB]*db.DB) ([]byte, error) {

	var mapData map[string]interface{}
	var alarms []interface{}

	err := json.Unmarshal(body, &mapData)
	if err != nil {
		log.Info("Failed to unmarshall given input data")
		return nil, err
	}

	var result struct {
		Output struct {
			Status        int32  `json:"status"`
			Status_detail string `json:"status-detail"`
			ALARM         struct {
				ALARM_LIST []interface{} `json:"ALARM_LIST"`
			} `json:"ALARM"`
		} `json:"sonic-alarm:output"`
	}

	log.Info("rpc_get_alarms map_data: ", mapData)

	input := mapData["sonic-alarm:input"]

	alarms, err = process_show_request(dbs, "ALARM", input)
	if err == nil {
		result.Output.Status = 0
		result.Output.ALARM.ALARM_LIST = alarms
	} else {
		result.Output.Status = 1
		result.Output.Status_detail = string(err.Error())
	}

	return json.Marshal(&result)
}

func ack_alarm(body []byte, dbs [db.MaxDB]*db.DB, op string) ([]byte, error) {

	var mapData map[string]interface{}

	err := json.Unmarshal(body, &mapData)
	if err != nil {
		log.Info("Failed to unmarshal given input data")
		return nil, err
	}

	var result struct {
		Output struct {
			Status        int32  `json:"status"`
			Status_detail string `json:"status-detail"`
		} `json:"sonic-alarm:output"`
	}

	log.Infof("In ack_alarm operation %s, input %s ", op, mapData)

	input := mapData["sonic-alarm:input"]

	mapData = input.(map[string]interface{})
	error_ids := ""

	action := utils.UNACKNOWLEDGE
	if op == "true" {
		action = utils.ACKNOWLEDGE
	}

	if ids, ok := mapData["id"].([]interface{}); ok {
		for _, id := range ids {

			if _, err = strconv.ParseInt(id.(string), 10, 64); err == nil {
				err = utils.EventNotify(dbs, "", id.(string), action, "Alarm id %s %s.", id.(string), utils.GetActionStr(action))
			}

			if err != nil {
				log.Info("Unable to write ack mode to alarm id ", id.(string))
				error_ids += id.(string)
				error_ids += ","
			}
		}
		error_ids = strings.TrimSuffix(error_ids, ",")
		if error_ids != "" {
			result.Output.Status = 1
			op_str := "ack "
			if op == "false" {
				op_str = "unack "
			}
			result.Output.Status_detail = "Failed to " + op_str + " " + error_ids
		} else {
			result.Output.Status = 0
		}
	}
	return json.Marshal(&result)
}

var rpc_acknowledge_alarms RpcCallpoint = func(body []byte, dbs [db.MaxDB]*db.DB) ([]byte, error) {

	return ack_alarm(body, dbs, "true")
}

var rpc_unacknowledge_alarms RpcCallpoint = func(body []byte, dbs [db.MaxDB]*db.DB) ([]byte, error) {

	return ack_alarm(body, dbs, "false")
}
