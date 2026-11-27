package transformer

import (
	"strings"

	"github.com/Azure/sonic-mgmt-common/translib/db"
)

// Temporary stand-ins for helpers that the syslog transformer
// (xfmr_system_syslog.go) uses and that nokia-cpn defines elsewhere.
//
// Remove each definition from this file when the original lands:
//   - nullKey, dbGetKeys, dbGetEntry: xfmr_system_ntp.go
//   - updateSubOpDataMap:             xfmr_port_breakout.go
//   - isMoreSpecificPath:             xfmr_vrrp.go
//
// Keeping them in a separate file avoids duplicate-symbol errors being buried
// in the syslog code, and makes the later removal a one-file delete.

const nullKey = "NULL"

// Function variables for mocking in unit tests.
var (
	dbGetKeys  = (*db.DB).GetKeys
	dbGetEntry = (*db.DB).GetEntry
)

// updateSubOpDataMap merges subOpMap into inParams.subOpDataMap[oper] so the
// infra applies those ConfigDB mutations as a sub-operation (e.g. DELETE of
// stale rows during a REPLACE).
func updateSubOpDataMap(subOpMap map[db.DBNum]map[string]map[string]db.Value, oper Operation, inParams XfmrParams) {
	if len(subOpMap[db.ConfigDB]) == 0 {
		return
	}
	if inParams.subOpDataMap[oper] == nil {
		inParams.subOpDataMap[oper] = &subOpMap
		return
	}
	if (*inParams.subOpDataMap[oper])[db.ConfigDB] == nil {
		(*inParams.subOpDataMap[oper])[db.ConfigDB] = make(map[string]map[string]db.Value)
	}
	mapCopy((*inParams.subOpDataMap[oper])[db.ConfigDB], subOpMap[db.ConfigDB])
}

// isMoreSpecificPath reports whether requestYangPath is a strict descendant of
// callbackYangPath.
func isMoreSpecificPath(requestYangPath string, callbackYangPath string) bool {
	if requestYangPath == "" || callbackYangPath == "" {
		return false
	}
	return strings.HasPrefix(requestYangPath, callbackYangPath) && len(requestYangPath) > len(callbackYangPath)
}
