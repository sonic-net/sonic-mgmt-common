package transformer

import (
	"github.com/godbus/dbus/v5"

	"fmt"

	lvl "github.com/Azure/sonic-mgmt-common/translib/log"
	"github.com/golang/glog"
)

// roleToGroup maps the user role to a list of groups in the host
func roleToGroup(role string) []string {
	switch role {
	case "admin":
		return []string{"admin", "sudo", "docker"}
	case "operator":
		return []string{"operator", "docker"}
	default:
		return []string{}
	}
}

// hostAccountCallObject returns a dbus.BusObject which can be used to call
// the requested method
func hostAccountCallObject(method string) (dbus.BusObject, string, error) {
	conn, err := dbus.SystemBus()
	if err != nil {
		return nil, "", err
	}

	const bus_name_base = "org.SONiC.HostAccountManagement"
	bus_name := "ham.accounts." + method
	bus_path := dbus.ObjectPath("/org/SONiC/HostAccountManagement")

	obj := conn.Object(bus_name_base, bus_path)

	return obj, bus_name, nil
}

func hostAccountParseCallReturn(call *dbus.Call) (bool, string) {
	if call.Err != nil {
		glog.V(lvl.ERROR).Info(call.Err.Error())
		return false, call.Err.Error()
	}

	body := call.Body[0].([]interface{})
	success := body[0].(bool)
	errmsg := body[1].(string)

	return success, errmsg
}

// hostAccountUserMod calls the HAM usermod function over D-Bus
func hostAccountUserMod(login, role, hashed_pw string) (bool, string) {
	obj, dest, err := hostAccountCallObject("usermod")
	if err != nil {
		return false, err.Error()
	}

	roles := roleToGroup(role)
	if len(roles) == 0 {
		return false, fmt.Sprintf("Invalid role %s", role)
	}

	return hostAccountParseCallReturn(obj.Call(dest, 0, login, roles, hashed_pw))
}

// hostAccountUserDel calls the HAM userdel over D-Bus
func hostAccountUserDel(login string) (bool, string) {
	obj, dest, err := hostAccountCallObject("userdel")
	if err != nil {
		return false, err.Error()
	}

	return hostAccountParseCallReturn(obj.Call(dest, 0, login))
}
