package transformer

import (
	"github.com/Azure/sonic-mgmt-common/translib/tlerr"
)

// SaveStartupConfig initiates the operation of saving the current content of
// the ConfigDB to a file that is then used to populate the database during the startup.
func SaveStartupConfig() error {
	r := HostQuery("cfg_mgmt.save", []string{})
	if r.Err != nil {
		return tlerr.New(Format: "internal SONiC Hostservice communication failure: %w", r.Err.Error())
	}
	if len(r.Body) < 2 {
		return tlerr.New("internal SONiC Hostservice communication failure: the response is too short.")
	}
	if _, ok := r.Body[0].(int32); !ok {
		return tlerr.New("internal SONiC Hostservice communication failure: first element is not int32.")
	}
	if _, ok := r.Body[1].(string); !ok {
		return tlerr.New("internal SONiC Hostservice communication failure: second element is not string.")
	}
	if r.Body[0].(int32) != 0 {
		return tlerr.New(r.Body[1].(string))
	}
	return nil
}
