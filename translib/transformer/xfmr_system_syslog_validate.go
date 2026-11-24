package transformer

import (
	"encoding/json"
	"fmt"
	"net"
	"os/exec"
	"strings"

	"github.com/Azure/sonic-mgmt-common/translib/db"
	"github.com/Azure/sonic-mgmt-common/translib/tlerr"
)

const (
	syslogMgmtVrfTableName  = "MGMT_VRF_CONFIG"
	syslogMgmtVrfGlobalKey  = "vrf_global"
	syslogMgmtVrfEnabledKey = "mgmtVrfEnabled"
	syslogVrfTableName      = "VRF"
)

type syslogLinuxIPView struct {
	vrfList    []string
	vrfMembers map[string][]string
	intfIPs    map[string][]string
}

var syslogLinuxIPViewProvider = getSyslogLinuxIPView

func setSyslogLinuxIPViewForTest(view *syslogLinuxIPView) func() {
	original := syslogLinuxIPViewProvider
	if view == nil {
		syslogLinuxIPViewProvider = getSyslogLinuxIPView
	} else {
		syslogLinuxIPViewProvider = func() (*syslogLinuxIPView, error) {
			return view, nil
		}
	}
	return func() {
		syslogLinuxIPViewProvider = original
	}
}

func normalizeSyslogVrf(vrf string) string {
	vrf = strings.TrimSpace(vrf)
	if vrf == "" || vrf == "default" {
		return ""
	}
	return vrf
}

func isMgmtVrfEnabled(cfgDb *db.DB) bool {
	if cfgDb == nil {
		return false
	}
	entry, err := dbGetEntry(cfgDb, &db.TableSpec{Name: syslogMgmtVrfTableName}, db.Key{Comp: []string{syslogMgmtVrfGlobalKey}})
	if err != nil {
		return false
	}
	value := strings.ToLower(entry.Field[syslogMgmtVrfEnabledKey])
	return value == "true"
}

func validateSyslogVrfChoice(cfgDb *db.DB, vrf string) error {
	if vrf == "" {
		return nil
	}
	if vrf == "mgmt" {
		if !isMgmtVrfEnabled(cfgDb) {
			return tlerr.InvalidArgsError{Format: "Invalid value for network-instance: mgmt"}
		}
		return nil
	}
	if cfgDb == nil {
		return tlerr.InvalidArgsError{Format: fmt.Sprintf("Invalid value for network-instance: %s", vrf)}
	}
	if _, err := dbGetEntry(cfgDb, &db.TableSpec{Name: syslogVrfTableName}, db.Key{Comp: []string{vrf}}); err == nil {
		return nil
	}
	return tlerr.InvalidArgsError{Format: fmt.Sprintf("Invalid value for network-instance: %s", vrf)}
}

func validateSyslogSourceAddress(host, source string) error {
	sourceIP := net.ParseIP(source)
	if sourceIP == nil {
		return tlerr.InvalidArgsError{Format: fmt.Sprintf("Invalid value for source-address: %s", source)}
	}
	if sourceIP.IsLoopback() || sourceIP.IsMulticast() || sourceIP.IsLinkLocalUnicast() {
		return tlerr.InvalidArgsError{
			Format: fmt.Sprintf("Invalid value for source-address: %s is a loopback/multicast/link-local IP address", source),
		}
	}
	serverIP := net.ParseIP(host)
	if serverIP == nil {
		return tlerr.InvalidArgsError{Format: fmt.Sprintf("Invalid value for host: %s", host)}
	}
	if (serverIP.To4() == nil) != (sourceIP.To4() == nil) {
		return tlerr.InvalidArgsError{
			Format: fmt.Sprintf("Invalid value for host / source-address: %s / %s IP address family mismatch", host, source),
		}
	}
	return nil
}

func flattenSyslogIPAddrs(intfIPs map[string][]string) []string {
	seen := make(map[string]struct{})
	for _, addrs := range intfIPs {
		for _, addr := range addrs {
			seen[addr] = struct{}{}
		}
	}
	result := make([]string, 0, len(seen))
	for addr := range seen {
		result = append(result, addr)
	}
	return result
}

func validateSyslogSourceInVrf(source, vrf string, view *syslogLinuxIPView) error {
	if source == "" {
		if vrf == "" {
			return nil
		}
		for _, linuxVrf := range view.vrfList {
			if linuxVrf == vrf {
				return nil
			}
		}
		return tlerr.InvalidArgsError{
			Format: fmt.Sprintf("Invalid value for network-instance: %s VRF doesn't exist in Linux", vrf),
		}
	}

	if vrf != "" {
		for _, linuxVrf := range view.vrfList {
			if linuxVrf == vrf {
				filtered := make(map[string][]string)
				for ifName, addrs := range view.intfIPs {
					for _, member := range view.vrfMembers[vrf] {
						if ifName == member {
							filtered[ifName] = addrs
						}
					}
				}
				if !containsString(flattenSyslogIPAddrs(filtered), source) {
					return tlerr.InvalidArgsError{
						Format: fmt.Sprintf("Invalid value for source-address: %s IP doesn't exist in Linux %s VRF", source, vrf),
					}
				}
				return nil
			}
		}
		return tlerr.InvalidArgsError{
			Format: fmt.Sprintf("Invalid value for network-instance: %s VRF doesn't exist in Linux", vrf),
		}
	}

	excludedIfs := make(map[string]struct{})
	for _, vrfName := range view.vrfList {
		excludedIfs[vrfName] = struct{}{}
	}
	for _, members := range view.vrfMembers {
		for _, member := range members {
			excludedIfs[member] = struct{}{}
		}
	}
	defaultVRFIPs := make(map[string][]string)
	for ifName, addrs := range view.intfIPs {
		if _, skip := excludedIfs[ifName]; skip {
			continue
		}
		defaultVRFIPs[ifName] = addrs
	}
	if !containsString(flattenSyslogIPAddrs(defaultVRFIPs), source) {
		return tlerr.InvalidArgsError{
			Format: fmt.Sprintf("Invalid value for source-address: %s IP doesn't exist in Linux default VRF", source),
		}
	}
	return nil
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func validateSyslogServerFields(cfgDb *db.DB, host string, fields map[string]string) error {
	source := fields[SYSLOG_FIELD_SOURCE]
	vrf := normalizeSyslogVrf(fields[SYSLOG_FIELD_VRF])

	if vrf != "" || source != "" {
		if err := validateSyslogVrfChoice(cfgDb, vrf); err != nil {
			return err
		}
	}
	if source == "" {
		return validateSyslogSourceInVrf("", vrf, mustGetSyslogLinuxIPView())
	}
	if err := validateSyslogSourceAddress(host, source); err != nil {
		return err
	}
	view, err := syslogLinuxIPViewProvider()
	if err != nil {
		return tlerr.InvalidArgsError{Format: err.Error()}
	}
	return validateSyslogSourceInVrf(source, vrf, view)
}

// syslogIntfAddrFamilies reports which address families ifName can supply as
// a syslog source. It mirrors rsyslog-config.sh, which takes any IPv4 address
// but only scope-global IPv6 addresses of SYSLOG_CONFIG|GLOBAL:src_intf.
func syslogIntfAddrFamilies(ifName string, view *syslogLinuxIPView) (hasV4, hasV6 bool) {
	if view == nil {
		return false, false
	}
	for _, addr := range view.intfIPs[ifName] {
		ip := net.ParseIP(addr)
		if ip == nil {
			continue
		}
		if ip.To4() != nil {
			hasV4 = true
		} else if !ip.IsLinkLocalUnicast() && !ip.IsLoopback() {
			hasV6 = true
		}
	}
	return hasV4, hasV6
}

// validateSyslogSrcIntfForServer rejects a global source-interface that has no
// address of the remote server's family. Servers with a per-server
// source-address and non-IP hosts do not use the source-interface address.
// The family test matches rsyslog.conf.j2 (':' in host means IPv6).
func validateSyslogSrcIntfForServer(host, source, srcIntf string, view *syslogLinuxIPView) error {
	if srcIntf == "" || source != "" || net.ParseIP(host) == nil {
		return nil
	}
	hasV4, hasV6 := syslogIntfAddrFamilies(srcIntf, view)
	if strings.Contains(host, ":") {
		if !hasV6 {
			return tlerr.InvalidArgsError{
				Format: fmt.Sprintf("Invalid value for source-interface: %s has no global IPv6 address for remote-server %s (IP address family mismatch)", srcIntf, host),
			}
		}
		return nil
	}
	if !hasV4 {
		return tlerr.InvalidArgsError{
			Format: fmt.Sprintf("Invalid value for source-interface: %s has no IPv4 address for remote-server %s (IP address family mismatch)", srcIntf, host),
		}
	}
	return nil
}

// validateSyslogSrcIntfForServers checks srcIntf against every remote server
// (host -> per-server source-address) that would fall back to it.
func validateSyslogSrcIntfForServers(srcIntf string, servers map[string]string) error {
	if srcIntf == "" || len(servers) == 0 {
		return nil
	}
	view, err := syslogLinuxIPViewProvider()
	if err != nil {
		return tlerr.InvalidArgsError{Format: err.Error()}
	}
	for host, source := range servers {
		if err := validateSyslogSrcIntfForServer(host, source, srcIntf, view); err != nil {
			return err
		}
	}
	return nil
}

func mustGetSyslogLinuxIPView() *syslogLinuxIPView {
	view, err := syslogLinuxIPViewProvider()
	if err != nil {
		return &syslogLinuxIPView{}
	}
	return view
}

func getSyslogLinuxIPView() (*syslogLinuxIPView, error) {
	vrfList, err := getLinuxVrfList()
	if err != nil {
		return nil, err
	}
	vrfMembers, err := getLinuxVrfMemberDict(vrfList)
	if err != nil {
		return nil, err
	}
	intfIPs, err := getLinuxIPAddrDict()
	if err != nil {
		return nil, err
	}
	return &syslogLinuxIPView{
		vrfList:    vrfList,
		vrfMembers: vrfMembers,
		intfIPs:    intfIPs,
	}, nil
}

func runIPJSONCommand(args ...string) ([]byte, error) {
	output, err := exec.Command("ip", args...).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to inspect Linux networking state: %s", strings.TrimSpace(string(output)))
	}
	return output, nil
}

func getLinuxVrfList() ([]string, error) {
	output, err := runIPJSONCommand("--json", "vrf", "show")
	if err != nil {
		return nil, err
	}
	var vrfData []struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(output, &vrfData); err != nil {
		return nil, err
	}
	vrfList := make([]string, 0, len(vrfData))
	for _, entry := range vrfData {
		if entry.Name != "" {
			vrfList = append(vrfList, entry.Name)
		}
	}
	return vrfList, nil
}

func getLinuxVrfMemberDict(vrfList []string) (map[string][]string, error) {
	vrfMembers := make(map[string][]string, len(vrfList))
	for _, vrfName := range vrfList {
		output, err := runIPJSONCommand("--json", "link", "show", "vrf", vrfName)
		if err != nil {
			return nil, err
		}
		var linkData []struct {
			IfName string `json:"ifname"`
		}
		if err := json.Unmarshal(output, &linkData); err != nil {
			return nil, err
		}
		members := make([]string, 0, len(linkData))
		for _, entry := range linkData {
			if entry.IfName != "" {
				members = append(members, entry.IfName)
			}
		}
		vrfMembers[vrfName] = members
	}
	return vrfMembers, nil
}

func getLinuxIPAddrDict() (map[string][]string, error) {
	output, err := runIPJSONCommand("--json", "address", "show")
	if err != nil {
		return nil, err
	}
	var addrData []struct {
		IfName   string `json:"ifname"`
		AddrInfo []struct {
			Local string `json:"local"`
		} `json:"addr_info"`
	}
	if err := json.Unmarshal(output, &addrData); err != nil {
		return nil, err
	}
	intfIPs := make(map[string][]string)
	for _, entry := range addrData {
		if entry.IfName == "" {
			continue
		}
		for _, addr := range entry.AddrInfo {
			if addr.Local != "" {
				intfIPs[entry.IfName] = append(intfIPs[entry.IfName], addr.Local)
			}
		}
	}
	return intfIPs, nil
}
