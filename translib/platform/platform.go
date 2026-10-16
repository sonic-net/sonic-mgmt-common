package platform

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/Azure/sonic-mgmt-common/translib/tlerr"
	log "github.com/golang/glog"
)

/* Glossary - Terms like "port" and "interface" are often used interchangeably but
 * sometimes to refer to different things.  A few terms are defined here for their use
 * in this package.
 * Lane - single serdes lane with absolute numbering, e.g. 563
 * Lane set - Group of lanes which can be grouped in various ways to create an interface
 * Channel - relative numbered serdes lane in a lane set, e.g. 1..8
 * Interface - A logical grouping of lanes with a speed
 * Port - A.k.a Physical Port, a physical connector on the switch, e.g. OSFP cage w/ connector
 */
const (
	PLATFORM_JSON          = "/usr/share/sonic/hwsku/"
	PLATFORM_PLATFORM_JSON = "/usr/share/sonic/platform/"
)

type platformIntf struct {
	index         int
	isPrimary     bool
	name          string
	alias         string
	lanes         []int // For a given interface group, each interface has a slice of the same array
	primary       *platformIntf
	channelOffset uint16 // (First Lane) % (lane set size); used to derive channel index from lane
}

type platformConfig struct {
	intfs              map[string]*platformIntf
	intfNameToPortName map[string]string
	portNameToIntfName map[string]string
}

var (
	PLATFORM_JSON_PATH string
	platformCfg        platformConfig
	initErr            error
	once               sync.Once
)

func init() {
	searchBases := []string{PLATFORM_JSON, PLATFORM_PLATFORM_JSON}

	for _, baseDir := range searchBases {
		asicFile := filepath.Join(baseDir, "platform_asic")
		content, err := os.ReadFile(asicFile)
		if err != nil {
			log.Infof("Platform init: platform_asic not found in %s. Trying next..", baseDir)
			continue
		}

		var candidatePath string
		platform := strings.TrimSpace(string(content))

		if platform == "alpine_vs" || platform == "vs" {
			candidatePath = filepath.Join(baseDir, platform, "platform.json")

		} else {
			candidatePath = filepath.Join(baseDir, "platform.json")
		}

		if _, err := os.Stat(candidatePath); err == nil {
			PLATFORM_JSON_PATH = candidatePath
			log.Infof("Platform init: Found platform.json at %s\n", PLATFORM_JSON_PATH)
			return
		}
	}

	if PLATFORM_JSON_PATH == "" {
		log.Infof("Platform init: platform.json not found.")
	}
}

/* Lazy Initialization */
func ensureLoaded() error {
	once.Do(func() {
		if PLATFORM_JSON_PATH == "" {
			initErr = fmt.Errorf("platform.json path not found during init")
			return
		}
		initErr = doParsePlatformJson(PLATFORM_JSON_PATH)
	})
	return initErr
}

func doParsePlatformJson(filename string) error {
	file, err := os.ReadFile(filename)
	if err != nil {
		if log.V(3) {
			log.Infof("Error reading platform json, file %s, err %v", filename, err)
		}
		return err
	}

	var parsedJson map[string]any
	if err := json.Unmarshal(file, &parsedJson); err != nil {
		if log.V(3) {
			log.Infof("Error parsing platform json, file %s, err %v", filename, err)
		}
		return err
	}

	/* Perform an initial pass over the json to create the platformIntfs and
	 * populate most fields. */
	platformCfg = platformConfig{}
	platformCfg.intfs = map[string]*platformIntf{}
	intfsJson, ok := parsedJson["interfaces"].(map[string]any)
	if !ok {
		return tlerr.InvalidArgs("Failed type assertion, 'interfaces' is missing or not a JSON object")
	}

	log.Infof("Found %d interface entries in platform json", len(intfsJson))
	for intfName, v := range intfsJson {
		intfJson, ok := v.(map[string]any)
		if !ok {
			return tlerr.InvalidArgs("Failed type assertion, interface entry '%s' is not a JSON object", intfName)
		}

		indexRaw, exists := intfJson["index"]
		if !exists {
			return tlerr.InvalidArgs("Field 'index' missing in interface '%s'", intfName)
		}
		indexJson, ok := indexRaw.(string)
		if !ok {
			return tlerr.InvalidArgs("Field 'index' in interface '%s' must be a string", intfName)
		}

		lanesJson, _ := intfJson["lanes"].(string)

		/* Primary interface has a comma seperated list of all indexes, otherwise a single index. */
		indexes := strings.Split(indexJson, ",")
		if len(indexes) == 1 && indexes[0] == "" {
			if log.V(3) {
				log.Infof("Platform json parsing error %s:index=\"%s\" (missing)", intfName, indexJson)
			}
			return tlerr.New("Platform json parsing error %s:index=\"%s\" (missing)", intfName, indexJson)
		}
		index, err := strconv.Atoi(indexes[0])
		if err != nil {
			if log.V(3) {
				log.Infof("Platform json parsing error %s:index=\"%s\", err %v", intfName, indexJson, err)
			}
			return err
		}

		/* Only the primary interface will have a comma seperated list of lanes. */
		var lanesInt []int = nil
		if len(lanesJson) > 0 {
			lanesArr := strings.Split(lanesJson, ",")
			lanesInt = make([]int, len(lanesArr))
			for i, l := range lanesArr {
				lane, err := strconv.Atoi(l)
				if err != nil {
					if log.V(3) {
						log.Infof("Platform json parsing error %s:lanes=\"%s\", err %v", intfName, lanesJson, err)
					}
					return err
				}
				lanesInt[i] = lane
			}
		}
		/* Primary interface must have the lane set. */
		isPrimary := len(lanesJson) > 0

		platformCfg.intfs[intfName] = &platformIntf{
			name:      intfName,
			index:     index,
			lanes:     lanesInt,
			isPrimary: isPrimary,
		}
		log.Infof("Added %s platform interface", intfName)
	}
	log.Infof("Created %d platform interfaces from platform json", len(platformCfg.intfs))

	/* Perform a second pass linking interfaces in the same group and adding aliases. */
	for _, intf := range platformCfg.intfs {
		if intf.isPrimary {
			intf.primary = intf
		} else {
			for _, pIntf := range platformCfg.intfs {
				/* Skip non-primary interfaces.  Skip primary interfaces for other indexes. */
				if !pIntf.isPrimary || intf.index != pIntf.index {
					continue
				}
				intf.primary = pIntf
				break
			}
		}
		if intf.primary == nil {
			if log.V(3) {
				log.Infof("Platform json parsing error %s: no primary", intf.name)
			}
			return tlerr.New("Platform json parsing error %s: no primary", intf.name)
		}

		mJsonAny, ok := intfsJson[intf.primary.name]
		if !ok {
			if log.V(3) {
				log.Infof("Platform json parsing error %s:primary=%s, no json entry", intf.name, intf.primary.name)
			}
			return tlerr.New("Platform json parsing error %s:primary=%s, no json entry", intf.name, intf.primary.name)
		}
		mJson, ok := mJsonAny.(map[string]any)
		if !ok {
			return tlerr.InvalidArgs("Failed type assertion, data for primary interface '%s' is not a JSON object", intf.primary.name)
		}

		aliasAtLanes, _ := mJson["alias_at_lanes"].(string)
		aliases := strings.Split(aliasAtLanes, ",")
		if len(aliases) == 1 && aliases[0] == "" {
			/* No name aliases are defined, this is okay, use the interface name as the
			 * alias. */
			intf.alias = intf.name
		} else if len(aliases) > 0 {
			intf.alias = strings.TrimSpace(aliases[0])
		}
	}
	log.Infof("Updated %d platform interfaces from platform json", len(platformCfg.intfs))

	platformCfg.intfNameToPortName = make(map[string]string)
	platformCfg.portNameToIntfName = make(map[string]string)
	for _, pIntf := range platformCfg.intfs {
		if !pIntf.isPrimary {
			continue
		}
		intfName := pIntf.name
		portName := "1/" + strconv.Itoa(pIntf.index)
		platformCfg.intfNameToPortName[intfName] = portName
		platformCfg.portNameToIntfName[portName] = intfName
	}
	calcChannelOffset()
	log.Infof("Built port name maps for %d (%d) entries", len(platformCfg.intfNameToPortName), len(platformCfg.portNameToIntfName))

	return nil
}

func platIntfByName(intfName string) (platformIntf, error) {
	rv, ok := platformCfg.intfs[intfName]
	if !ok {
		return platformIntf{}, tlerr.InvalidArgs("platformIntf \"%s\" not found", intfName)
	}
	return *rv, nil
}

func calcChannelOffset() {
	// Derive the channel offset for each primary interface
	for _, intf := range platformCfg.intfs {
		if !intf.isPrimary {
			continue
		}
		if len(intf.lanes) == 0 {
			if log.V(3) {
				log.Infof("Primary interface %s has 0 physical lanes configured, skipping offset calculation", intf.name)
			}
			continue
		}
		intf.channelOffset = uint16(intf.lanes[0] % len(intf.lanes))
	}
}

func ChannelOffset(intfName string) (uint16, error) {
	if err := ensureLoaded(); err != nil {
		if log.V(3) {
			log.Infof("Error in loading platform file. err %v", err)
		}
		return 0, err
	}
	pIntf, err := platIntfByName(intfName)
	if err != nil {
		return 0, err
	}
	return pIntf.channelOffset, nil
}
