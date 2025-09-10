package transformer

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/Azure/sonic-mgmt-common/translib/db"
)

var (
	// pcMembers is a map to hold port channel members.
	pcMembers = make(map[string]bool)
)

const (
	// SFLOW_STATE_INTF_TBL represents the SFlow state interface table.
	SFLOW_STATE_INTF_TBL = "SFLOW_SESSION_TABLE"
)

// updateSubOpDataMap updates the transaction's sub-operation data map.
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

// GetIdFromFPQueueName extracts the queue ID from a formatted queue name.
func GetIdFromFPQueueName(qn string) (string, error) {
	parts := strings.Split(qn, ":")
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid queue name format: %s", qn)
	}
	return parts[len(parts)-1], nil
}

// processFieldLeafPairs processes a map of fields to leaves, converting and assigning values.
type fieldU64LeafPair struct {
	field string
	leaf  **uint64
}

func processFieldLeafPairs(entry *db.Value, pairs []fieldU64LeafPair) error {
	for _, pair := range pairs {
		if val, ok := entry.Field[pair.field]; ok {
			u, err := strconv.ParseUint(val, 10, 64)
			if err != nil {
				return err
			}
			*pair.leaf = &u
		}
	}
	return nil
}
