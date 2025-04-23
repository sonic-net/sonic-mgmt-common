package transformer

import (
	"errors"
	"strings"

	"github.com/godbus/dbus/v5"
	log "github.com/golang/glog"
)

// HostResult contains the body of the response and the error if any, when the
// endpoint finishes servicing the D-Bus request.
type HostResult struct {
	Body []interface{}
	Err  error
}

// checkQueryOutput checks if the host query output has errors.
func checkQueryOutput(r HostResult) (string, error) {
	if r.Err != nil {
		return "", errors.New("Internal SONiC HostService failure: " + r.Err.Error())
	}
	if len(r.Body) < 2 {
		return "", errors.New("Internal SONiC HostService failure: the response is too short.")
	}
	if _, ok := r.Body[0].(int32); !ok {
		return "", errors.New("Internal SONiC HostService failure: the first element is not int32.")
	}
	if _, ok := r.Body[1].(string); !ok {
		return "", errors.New("Internal SONiC HostService failure: second element is not string.")
	}
	if r.Body[0].(int32) != 0 {
		return "", errors.New("Internal SONiC HostService failure: " + r.Body[1].(string))
	}
	return r.Body[1].(string), nil
}

// HostQuery calls the corresponding D-Bus endpoint on the host and returns
// any error and response body
func HostQuery(endpoint string, args ...interface{}) (result HostResult) {
	log.Infof("HostQuery called")
	result_ch, err := hostQueryAsync(endpoint, args...)

	if err != nil {
		result.Err = err
		return
	}

	result = <-result_ch
	return
}

// hostQueryAsync calls the corresponding D-Bus endpoint on the host and returns
// a channel for the result, and any error
func hostQueryAsync(endpoint string, args ...interface{}) (chan HostResult, error) {
	log.Infof("HostQueryAsync called")
	var result_ch = make(chan HostResult, 1)
	conn, err := dbus.SystemBus()
	if err != nil {
		return result_ch, err
	}
	log.Infof("HostQueryAsync conn established")

	service := strings.SplitN(endpoint, ".", 2)
	const bus_name_base = "org.SONiC.HostService."
	bus_name := bus_name_base + service[0]
	bus_path := dbus.ObjectPath("/org/SONiC/HostService/" + service[0])

	obj := conn.Object(bus_name, bus_path)
	dest := bus_name_base + endpoint
	dbus_ch := make(chan *dbus.Call, 1)
	//log.Infof("HostQueryAsync dbus called %s "% string(bus_path))
	//log.Infof("HostQueryAsync dbus called %s  "% string(bus_name))

	go func() {
		var result HostResult

		// Wait for a read on the channel
		call := <-dbus_ch

		if call.Err != nil {
			log.Infof("HostQueryAsync Err is not nill while reading")
			result.Err = call.Err
		} else {
			log.Infof("HostQueryAsync Body is taken")
			result.Body = call.Body
		}

		// Write the result to the channel
		result_ch <- result
	}()

	log.Infof("HostQueryAsync Before objgo")
	call := obj.Go(dest, 0, dbus_ch, args...)

	if call.Err != nil {
		log.Infof("HostQueryAsync Err is not after obj.Go")
		return result_ch, call.Err
	}

	return result_ch, nil
}
