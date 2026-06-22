package platform

import (
	"encoding/json"
	"os"
	"sort"
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
 * SubIndex - first channel of an interface
 * Interface - A logical grouping of lanes with a speed
 * Port - A.k.a Physical Port, a physical connector on the switch, e.g. OSFP cage w/ connector
 *
 * Interface naming scheme is <prefix>1/<index>/<subIndex>
 * Examples: Ethernet1/1/1, Ethernet1/32/8
 * The index corresponds to a port and may be zero-based or one-based depending
 * on the platform.  The subIndex is the first channel of the interface and may
 * be omitted for non-breakable interfaces such as SFP+, e.g. Ethernet1/1.
 */
const (
	PLATFORM_JSON          = "/usr/share/sonic/hwsku/platform.json"
	PLATFORM_PLATFORM_JSON = "/usr/share/sonic/platform/platform.json"
	HWSKU_JSON             = "/usr/share/sonic/hwsku/hwsku.json"
	GBPS_TO_MBPS           = 1000
)

type platformIntf struct {
	index         int
	subIndex      int
	isPrimary     bool
	name          string
	alias         string
	dfltMode      string   // Default breakout mode.
	modes         []string // List of supported breakout modes.
	lanes         []int    // For a given interface group, each interface has a slice of the same array
	speedsGbps    []int    // Possible speeds for the interface in Gbps
	primary       *platformIntf
	intfGroup     []*platformIntf // All interfaces in the group (including the primary)
	channelOffset uint16          // (First Lane) % (lane set size); used to derive channel index from lane
}

type InterfaceProperties struct {
	Index     int   // Index of the interface, Ethernet1/<index>/<subindex>
	SpeedMbps int   // Interface speed in Mbps
	Lanes     []int // Slice of lanes used by the interface from platformIntf
	Name      string
	Alias     string
}

type platformConfig struct {
	intfs              map[string]*platformIntf
	intfNameToPortName map[string]string
	portNameToIntfName map[string]string
}

type brkoutProperties struct {
	/* True when all interfaces in a mixed-mode breakout use the same number of
	 * channels.  1x400G(4)+1x200G(4) would be true since both the 400G and
	 * 200G interface use 4 channels even though the channels run at different
	 * serdes speeds.  1x400G(4)+2x200G(4) would be false since the 400G interface
	 * uses 4 channels and the 200G interfaces use 2 channels each. */
	isSymmetric bool
	/* The total number of interfaces in the breakout; sum of all the "<int>x"
	 * prefixes in the breakout mode string. 1x400G(4)+4x100G(4) would be 5. */
	numIntfs int
	/* An array of all speeds in the breakout, e.g. {"100G", "200G"}.  This is
	 * unsorted and may contain duplicates. */
	speedsGbps []string
}

var platformCfg platformConfig
var intfToDefaultMode = map[string]string{} // from hwsku.json

/* Functions */
func init() {
	parseHwskuJson()
	parsePlatformJson()
}

func parseThisThenThat(a, b string) error {
	if err := doParsePlatformJson(a); err != nil {
		return doParsePlatformJson(b)
	}
	return nil
}

var once = sync.OnceValue(func() error { return parseThisThenThat(PLATFORM_JSON, PLATFORM_PLATFORM_JSON) })
var onceagain = sync.OnceValue(func() error { return doParseHwskuJson(HWSKU_JSON) })

func parsePlatformJson() error {
	err := once()
	if err != nil {
		if log.V(3) {
			log.Info(err)
		}
	}
	return err
}

func parseHwskuJson() error {
	err := onceagain()
	if err != nil {
		if log.V(3) {
			log.Info(err)
		}
	}
	return err
}

// Returns true if the legacy format was detected.
// The distinction is primarily derived from the `breakout_modes` type.
// In the standard sonic format,`breakout_modes` is a dictionary
// but a string in the legacy format.
//
// So "An interface entry where all fields are strings" is used to make this
// determination.
//
// The input is expected to be the platform.json `interfaces` dictionary
// unmarshalled into a map. Here's an example json before unmarshalling:
//
//	"interfaces": {
//			"Ethernet1/1/1": {
//			"index": "1,1,1,1,1,1,1,1",
//			"default_brkout_mode": "8x100G",
//			"lanes": "9,10,11,12,13,14,15,16",
//			"alias_at_lanes": "Eth1/1,Eth1/2,Eth1/3,Eth1/4,Eth1/5,Eth1/6,Eth1/7,Eth1/8",
//			"breakout_modes": "1x1600G[800G], 2x800G[400G,200G,100G], 4x400G[200G], 8x200G[100G],"
//			},
//			"Ethernet1/1/2": { "index": "1", "breakout_modes": "1x200G[100G]" },
//	      ...
//	}
func legacyInterfacesFormatFound(intfsJson map[string]any) (bool, error) {
	for _, v := range intfsJson {
		intfJson, ok := v.(map[string]any)
		if !ok {
			return false, tlerr.InvalidArgs("Failed type assertion, unexpected interfaces format")
		}
		// are all values strings?
		for _, vv := range intfJson {
			if _, ok := vv.(string); !ok {
				return false, nil
			}
		}
		break
	}
	return true, nil
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
	intfsJson := parsedJson["interfaces"].(map[string]any)
	log.Infof("Found %d interface entries in platform json", len(intfsJson))
	if legacyFound, err := legacyInterfacesFormatFound(intfsJson); err != nil {
		return err
	} else if !legacyFound {
		return doParsePlatformJsonSonic(intfsJson)
	}
	for intfName, v := range intfsJson {
		/* All values are strings */
		intfJson := v.(map[string]any)
		intfStrMap := make(map[string]string)
		for k, v := range intfJson {
			intfStrMap[k] = v.(string)
		}
		indexJson := intfStrMap["index"]
		modesJson := intfStrMap["breakout_modes"]
		dfltModeJson := intfStrMap["default_brkout_mode"]
		lanesJson := intfStrMap["lanes"]

		/* GPINS naming convention is <StrPrefix>1/<index>/<subindex> where index
		 * corresponds to the front panel labeling and subindex is a one-based
		 * offset into the group of lanes.  The subindex may be ommitted for single
		 * lane interface such as SFP+ interfaces (hence the length may be 2 or 3). */
		nameArr := strings.Split(intfName, "/")
		if len(nameArr) != 2 && len(nameArr) != 3 {
			if log.V(3) {
				log.Infof("Platform json parsing error %s:malformed name", intfName)
			}
			return tlerr.New("Platform json parsing error %s:malformed name", intfName)
		}
		nameIndex, err := strconv.Atoi(nameArr[1])
		if err != nil {
			if log.V(3) {
				log.Infof("Platform json parsing error %s:nameIndex, err %v", intfName, err)
			}
			return err
		}
		var subIndex int
		if len(nameArr) > 2 {
			subIndex, err = strconv.Atoi(nameArr[2])
			if err != nil {
				if log.V(3) {
					log.Infof("Platform json parsing error %s:subIndex, err %v", intfName, err)
				}
				return err
			}
		}

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
		if index != nameIndex {
			if log.V(3) {
				log.Infof("Platform json parsing error %s:parsed index %d doesn't match name", intfName, index)
			}
			return tlerr.New("Platform json parsing error %s:parsed index %d doesn't match name", intfName, index)
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
		/* Primary interface must have the lane set.  They should also be the first
		 * sub-index in the interface group, i.e. subindex one.  Note that unbreakable
		 * interfaces have a subindex of zero. */
		isPrimary := len(lanesJson) > 0 && subIndex <= 1

		/* The primary interface is required to have breakout modes.  It is not required for other interfaces. */
		modes := parseBreakoutModes(modesJson)
		if len(modes) == 0 && isPrimary {
			if log.V(3) {
				log.Infof("Platform json parsing error %s:breakout_modes=\"%s\"", intfName, modesJson)
			}
			return tlerr.New("Platform json parsing error %s:breakout_modes=\"%s\"", intfName, modesJson)
		}
		/* Sanitize and validate the breakout mode string and default breakout mode string. */
		if len(dfltModeJson) > 0 {
			dfltMode, ok := sanitizeBreakoutMode(dfltModeJson)
			if !ok {
				return tlerr.New("Platform json parsing error %s:bad default_brkout_mode=\"%v\"", intfName, dfltMode)
			}
			dfltSpeed, err := brkoutModesToSpeeds([]string{dfltMode})
			if err != nil || len(dfltSpeed) != 1 {
				return tlerr.New("Platform json parsing error %s:bad default_brkout_mode=\"%v\"", intfName, dfltMode)
			}
		}
		for i, m := range modes {
			mCleaned, ok := sanitizeBreakoutMode(m)
			if !ok {
				return tlerr.New("Platform json parsing error %s:bad mode=\"%v\"", intfName, modes)
			}
			modes[i] = mCleaned
		}
		speeds, err := brkoutModesToSpeeds(modes)
		if err != nil {
			if log.V(3) {
				log.Infof("Platform json parsing error %s:breakout_modes=\"%s\" (%v)", intfName, modesJson, err)
			}
			return tlerr.New("Platform json parsing error %s:breakout_modes=\"%s\" (%v)", intfName, modesJson, err)
		}

		platformCfg.intfs[intfName] = &platformIntf{
			name:       intfName,
			index:      index,
			subIndex:   subIndex,
			lanes:      lanesInt,
			speedsGbps: speeds,
			modes:      modes,
			dfltMode:   dfltModeJson,
			isPrimary:  isPrimary,
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
		mJson := mJsonAny.(map[string]any)
		mJsonStrMap := make(map[string]string)
		for k, v := range mJson {
			mJsonStrMap[k] = v.(string)
		}

		/* Non-breakable interfaces will have a subindex of zero while breakable interfaces
		 * have a one-based index.  Check what case we are in and unify to a zero-
		 * based index. */
		subIndex := 0
		if intf.subIndex > 0 {
			subIndex = intf.subIndex - 1
		}

		aliases := strings.Split(mJsonStrMap["alias_at_lanes"], ",")
		if len(aliases) == 1 && aliases[0] == "" {
			/* No name aliases are defined, this is okay, use the interface name as the
			 * alias. */
			intf.alias = intf.name
		} else if len(aliases) > subIndex {
			/* Aliases are defined and our index is legal, save the alias. */
			intf.alias = strings.TrimSpace(aliases[subIndex])
		} else {
			/* Aliases are defined, but this interface's index is out of range. */
			if log.V(3) {
				log.Infof("Platform json parsing error %s:subIndex=%d, no alias=\"%s\"", intf.name, subIndex, aliases)
			}
			return tlerr.New("Platform json parsing error %s:subIndex=%d, no alias=\"%s\"", intf.name, subIndex, aliases)
		}

		/* Tag this interface with the lane set composed of its own lane as well as all
		 * lanes of subindexes larger than itself. */
		intf.lanes = intf.primary.lanes[subIndex:]
	}
	log.Infof("Updated %d platform interfaces from platform json", len(platformCfg.intfs))

	/* Loop over all interfaces to build the intfRel map capturing the primary intf to
	 * all interfaces in the group mapping. */
	intfRel := make(map[string][]string)
	for _, pIntf := range platformCfg.intfs {
		primaryIntf := pIntf.primary.name
		relatedIntfs, ok := intfRel[primaryIntf]
		if !ok {
			relatedIntfs = []string{}
		}
		intfRel[primaryIntf] = append(relatedIntfs, pIntf.name)
	}
	for k, v := range intfRel {
		sort.Strings(v)
		intfRel[k] = v
	}
	for primaryIntfName, intfGroup := range intfRel {
		priIntf := platformCfg.intfs[primaryIntfName]
		priIntf.intfGroup = make([]*platformIntf, len(intfGroup))
		for i, intfName := range intfGroup {
			pIntf := platformCfg.intfs[intfName]
			priIntf.intfGroup[i] = pIntf
		}
		for _, pIntf := range priIntf.intfGroup {
			pIntf.intfGroup = priIntf.intfGroup
		}
		if len(priIntf.intfGroup) > len(priIntf.lanes) {
			if log.V(3) {
				log.Infof("Platform json parsing error %s: %d lanes but %d intfs in group", primaryIntfName, len(priIntf.lanes), len(priIntf.intfGroup))
			}
			return tlerr.New("Platform json parsing error %s: %d lanes but %d intfs in group", primaryIntfName, len(priIntf.lanes), len(priIntf.intfGroup))
		}
	}
	log.Infof("Updated %d platform intfs from platform json", len(platformCfg.intfs))

	/* Loop over all interfaces to build the interface name (Ethernet1/1/1) to port
	 * name (1/1). */
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

func doParseHwskuJson(filename string) error {
	file, err := os.ReadFile(filename)
	if err != nil {
		if log.V(3) {
			log.Infof("Error reading hwsku json, file %s, err %v", filename, err)
		}
		return err
	}

	var parsedJson map[string]any
	if err := json.Unmarshal(file, &parsedJson); err != nil {
		if log.V(3) {
			log.Infof("Error parsing hwsku json, file %s, err %v", filename, err)
		}
		return err
	}

	// Pass over the json, map interface to default breakout mode
	intfsJson, ok := parsedJson["interfaces"].(map[string]any)
	if !ok {
		return tlerr.InvalidArgs("Type assertion failed converting interfaces")
	}
	log.Infof("Found %d interface entries in hwsku json", len(intfsJson))
	for intfName, v := range intfsJson {
		intfJson, ok := v.(map[string]any)
		if !ok {
			return tlerr.InvalidArgs("Type assertion failed converting interface")
		}
		intfStrMap := make(map[string]string)
		for k, v := range intfJson {
			intfStrMap[k], ok = v.(string)
			if !ok {
				return tlerr.InvalidArgs("Type assertion failed converting interface fields")
			}
		}
		intfToDefaultMode[intfName] = intfStrMap["default_brkout_mode"]
	}
	log.Infof("interface to default map: %v", intfToDefaultMode)
	return nil
}

func doParsePlatformJsonSonic(intfsJson map[string]any) error {
	for _, v := range intfsJson {
		intfJson := v.(map[string]any)
		indexJson := ""
		modesJson := make(map[string]interface{})
		lanesJson := ""
		for k, v := range intfJson {
			switch k {
			case "index":
				indexJson = v.(string)
			case "lanes":
				lanesJson = v.(string)
			case "breakout_modes":
				modesJson = v.(map[string]interface{})
			}
		}

		intfToMode, err := parseBreakoutModesDict(modesJson, lanesJson)
		if err != nil {
			if log.V(3) {
				log.Infof("parseBreakoutModesDict returned=%v", err)
			}
		}

		// Iterate through the interfaces discovered by parsing intfName's breakout modes
		for intf, modes := range intfToMode {
			speeds, err := brkoutModesToSpeeds(modes)
			if err != nil {
				if log.V(3) {
					log.Infof("brkoutModesToSpeeds returned=%v", err)
				}
			}

			/* GPINS naming convention is <StrPrefix>1/<index>/<subindex> where index
			* corresponds to the front panel labeling and subindex is a one-based
			* offset into the group of lanes.  The subindex may be ommitted for single
			* lane interface such as SFP+ interfaces (hence the length may be 2 or 3). */
			nameArr := strings.Split(intf, "/")
			switch len(nameArr) {
			case 2, 3:
				break
			default:
				if log.V(3) {
					log.Infof("Platform json parsing error %s:malformed name", intf)
				}
				return tlerr.New("Platform json parsing error %s:malformed name", intf)
			}
			nameIndex, err := strconv.Atoi(nameArr[1])
			if err != nil {
				if log.V(3) {
					log.Infof("Platform json parsing error %s:nameIndex, err %v", intf, err)
				}
				return err
			}
			var subIndex int
			if len(nameArr) > 2 {
				subIndex, err = strconv.Atoi(nameArr[2])
				if err != nil {
					if log.V(3) {
						log.Infof("Platform json parsing error %s:subIndex, err %v", intf, err)
					}
					return err
				}
			}

			/* Primary interface has a comma seperated list of all indexes, otherwise a single index. */
			indexes := strings.Split(indexJson, ",")
			if len(indexes) == 1 && indexes[0] == "" {
				if log.V(3) {
					log.Infof("Platform json parsing error %s:index=\"%s\" (missing)", intf, indexJson)
				}
				return tlerr.New("Platform json parsing error %s:index=\"%s\" (missing)", intf, indexJson)
			}
			index, err := strconv.Atoi(indexes[0])
			if err != nil {
				if log.V(3) {
					log.Infof("Platform json parsing error %s:index=\"%s\", err %v", intf, indexJson, err)
				}
				return err
			}
			if index != nameIndex {
				if log.V(3) {
					log.Infof("Platform json parsing error %s:parsed index %d doesn't match name", intf, index)
				}
				return tlerr.New("Platform json parsing error %s:parsed index %d doesn't match name", intf, index)
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
							log.Infof("Platform json parsing error %s:lanes=\"%s\", err %v", intf, lanesJson, err)
						}
						return err
					}
					lanesInt[i] = lane
				}
			}
			/* Primary interface must have the lane set.  They should also be the first
			* sub-index in the interface group, i.e. subindex one.  Note that unbreakable
			* interfaces have a subindex of zero. */
			isPrimary := len(lanesJson) > 0 && subIndex <= 1

			dfltModeJson := ""
			if dbm, ok := intfToDefaultMode[intf]; ok && isPrimary {
				dfltModeJson = dbm
			}

			platformCfg.intfs[intf] = &platformIntf{
				name:       intf,
				alias:      intf,
				index:      index,
				subIndex:   subIndex,
				lanes:      lanesInt,
				speedsGbps: speeds,
				modes:      modes,
				dfltMode:   dfltModeJson,
				isPrimary:  isPrimary,
			}
		}
	}

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

		/* Non-breakable interfaces will have a subindex of zero while breakable interfaces
		 * have a one-based index.  Check what case we are in and unify to a zero-
		 * based index. */
		subIndex := 0
		if intf.subIndex > 0 {
			subIndex = intf.subIndex - 1
		}

		/* Tag this interface with the lane set composed of its own lane as well as all
		 * lanes of subindexes larger than itself. */
		intf.lanes = intf.primary.lanes[subIndex:]
	}
	log.Infof("Updated %d platform interfaces from platform json", len(platformCfg.intfs))

	/* Loop over all interfaces to build the intfRel map capturing the primary intf to
	 * all interfaces in the group mapping. */
	intfRel := make(map[string][]string)
	for _, pIntf := range platformCfg.intfs {
		primaryIntf := pIntf.primary.name
		relatedIntfs, ok := intfRel[primaryIntf]
		if !ok {
			relatedIntfs = []string{}
		}
		intfRel[primaryIntf] = append(relatedIntfs, pIntf.name)
	}
	for k, v := range intfRel {
		sort.Strings(v)
		intfRel[k] = v
	}
	for primaryIntfName, intfGroup := range intfRel {
		priIntf := platformCfg.intfs[primaryIntfName]
		priIntf.intfGroup = make([]*platformIntf, len(intfGroup))
		for i, intfName := range intfGroup {
			pIntf := platformCfg.intfs[intfName]
			priIntf.intfGroup[i] = pIntf
		}
		for _, pIntf := range priIntf.intfGroup {
			pIntf.intfGroup = priIntf.intfGroup
		}
		if len(priIntf.intfGroup) > len(priIntf.lanes) {
			if log.V(3) {
				log.Infof("Platform json parsing error %s: %d lanes but %d intfs in group", primaryIntfName, len(priIntf.lanes), len(priIntf.intfGroup))
			}
			return tlerr.New("Platform json parsing error %s: %d lanes but %d intfs in group", primaryIntfName, len(priIntf.lanes), len(priIntf.intfGroup))
		}
	}
	log.Infof("Updated %d platform intfs from platform json", len(platformCfg.intfs))

	/* Loop over all interfaces to build the interface name (Ethernet1/1/1) to port
	 * name (1/1). */
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

func parseBreakoutModes(modes string) []string {
	// Example breakout_modes entry: "1x400G, 2x200G[100G,40G], 1x200G(4)+2x100G(4)"
	// Returns array length 3: {"1x400G", "2x200G[100G,40G]", "1x200G(4)+2x100G(4)"}
	isSqParens := false
	return strings.FieldsFunc(modes, func(c rune) bool {
		if c == '[' {
			isSqParens = true
		} else if c == ']' {
			isSqParens = false
		}
		if !isSqParens {
			return c == ','
		}
		return false
	})
}

// Parses the breakout_modes and
// returns map of interfaces to modes (they are part of)
func parseBreakoutModesDict(modes map[string]interface{}, lanes string) (map[string][]string, error) {
	// Example input dictionary:
	// breakout_modes": {
	// 	"1x1600G": ["Ethernet1/35/1"],
	// 	"2x800G": ["Ethernet1/35/1","Ethernet1/35/5"]
	// }
	intfToMode := make(map[string][]string)
	for mode, data := range modes {
		intfs := data.([]interface{})
		m, ok := sanitizeBreakoutMode(mode)
		if !ok {
			return nil, tlerr.InvalidArgs("Unable to sanitize mode:" + mode)
		}
		for _, intf := range intfs {
			i := intf.(string)
			intfToMode[i] = append(intfToMode[i], m)
		}
	}

	return intfToMode, nil
}

func sanitizeBreakoutMode(unsanitizedMode string) (string, bool) {
	mode := strings.TrimSpace(unsanitizedMode)
	if len(mode) == 0 || strings.ContainsAny(mode, " \t\n\r") {
		if log.V(3) {
			log.Infof("Breakout mode \"%s\" contains whitespace", mode)
		}
		return mode, false
	}
	numModes := strings.Count(mode, "+") + 1
	isMixed := numModes > 1
	mixedModes := strings.Split(mode, "+")
	if len(mixedModes) != numModes {
		if log.V(3) {
			log.Infof("Breakout mixed-mode \"%s\" expected %d modes, got %d", mode, numModes, len(mixedModes))
		}
		return mode, false
	}
	for _, mixedMode := range mixedModes {
		/* Modes must be prefixed by the interface count, "<integer>x..." */
		prefixSep := strings.Index(mixedMode, "x")
		if prefixSep == -1 {
			if log.V(3) {
				log.Infof("Breakout mode \"%s\" not prefixed by intf count", mode)
			}
			return mode, false
		}
		if _, err := strconv.Atoi(mixedMode[:prefixSep]); err != nil {
			if log.V(3) {
				log.Infof("Breakout mode \"%s\" prefixed by non-int", mode)
			}
			return mode, false
		}
		/* If this is a mixed mode, each mode must have a lane count suffix "...(<integer)" */
		if isMixed {
			suffixStart := strings.Index(mixedMode, "(")
			suffixEnd := strings.Index(mixedMode, ")")
			if suffixStart == -1 || suffixEnd != len(mixedMode)-1 || (suffixStart+1) >= suffixEnd {
				if log.V(3) {
					log.Infof("Breakout mode \"%s\" missing lane count suffix for mixed mode", mode)
				}
				return mode, false
			}
			_, err := strconv.Atoi(mixedMode[suffixStart+1 : suffixEnd])
			if err != nil {
				if log.V(3) {
					log.Infof("Breakout mode \"%s\" has non-int lane count suffix for mixed mode", mode)
				}
				return mode, false
			}
			mixedMode = mixedMode[:suffixStart]
		}
		/* The speeds are listed after the prefix but before the optional lane count. */
		speeds := mixedMode[prefixSep+1:]
		/* Speeds may be a single speed, "100G", or a set of speeds, "400G[200G,100G,50G]" */
		numOpen := strings.Count(speeds, "[")
		numClose := strings.Count(speeds, "]")
		if numOpen != numClose || numOpen > 1 {
			if log.V(3) {
				log.Infof("Breakout mode \"%s\" (%s) has malformed alternate speed list", mode, speeds)
			}
			return mode, false
		}
		if numOpen > 0 {
			/* We have an alternate speed list, verify it. */
			start := strings.Index(speeds, "[")
			end := strings.Index(speeds, "]")
			if (start+1) >= end || end != len(speeds)-1 {
				if log.V(3) {
					log.Infof("Breakout mode \"%s\" (%s) has malformed alternate speed list \"%s\"; start %d end %d", mode, mixedMode, speeds, start, end)
				}
				return mode, false
			}
			for _, altSpeed := range strings.Split(speeds[start+1:end], ",") {
				s, ok := strings.CutSuffix(altSpeed, "G")
				if !ok {
					if log.V(3) {
						log.Infof("mode \"%s\", alt-speed \"%s\" is missing G suffix", mixedMode, altSpeed)
					}
					return mode, false
				}
				if _, err := strconv.Atoi(s); err != nil {
					if log.V(3) {
						log.Infof("mode \"%s\", alt-speed \"%s\" is not an int (\"%s\")", mixedMode, altSpeed, s)
					}
					return mode, false
				}
			}
			speeds = speeds[:start]
		}
		s, ok := strings.CutSuffix(speeds, "G")
		if !ok {
			if log.V(3) {
				log.Infof("mode \"%s\", speeds \"%s\" is missing G suffix", mixedMode, speeds)
			}
			return mode, false
		}
		if _, err := strconv.Atoi(s); err != nil {
			if log.V(3) {
				log.Infof("mode \"%s\", speeds \"%s\", is not an int (\"%s\")", mixedMode, speeds, s)
			}
			return mode, false
		}
	}
	return mode, true
}

func brkoutModesToSpeeds(brkoutModes []string) ([]int, error) {
	speeds := map[int]int{}
	for _, mode := range brkoutModes {
		modeSpeeds, err := brkoutModeToSpeeds(mode)
		if err != nil {
			return nil, err
		}
		for _, speed := range modeSpeeds {
			/* Use map to remove duplicates. */
			speeds[speed] = speed
		}
	}
	speedList := make([]int, len(speeds))
	i := 0
	for speed := range speeds {
		speedList[i] = speed
		i++
	}
	sort.Ints(speedList)
	return speedList, nil
}
func brkoutModeToSpeeds(brkoutMode string) ([]int, error) {
	speeds := map[int]int{}
	modes := strings.FieldsFunc(brkoutMode, func(c rune) bool {
		return c == ',' || c == '[' || c == ']' || c == '+'
	})
	for _, mode := range modes {
		// Mode may contain, or be, the mixed mode channel count, e.g "(4)", if so skip it
		chanCountIndex := strings.Index(mode, "(")
		if chanCountIndex != -1 {
			mode = mode[:chanCountIndex]
			mode = strings.TrimSpace(mode)
			if len(mode) == 0 {
				continue
			}
		}
		// Mode may have a "<count>x" prefix; ignore it.
		before, after, found := strings.Cut(mode, "x")
		if found {
			mode = after
		} else {
			mode = before
		}
		// Mode may have a suffix of "(<num lanes>)"; ignore it.
		suffixStart := strings.Index(mode, "(")
		if suffixStart != -1 {
			mode = mode[:suffixStart]
		}
		// Mode must always end in "G" at this point.
		if mode[len(mode)-1] != 'G' {
			return nil, tlerr.InvalidArgs("Invalid breakout mode \"%s\" (%s)", brkoutMode, mode)
		}
		mode = mode[:len(mode)-1]
		speed, err := strconv.Atoi(mode)
		if err != nil {
			return nil, tlerr.InvalidArgs("Invalid breakout mode \"%s\" (%s)", brkoutMode, mode)
		}
		speeds[speed] = speed
	}
	speedList := make([]int, len(speeds))
	i := 0
	for speed := range speeds {
		speedList[i] = speed
		i++
	}
	sort.Ints(speedList)
	return speedList, nil
}

func calcChannelOffset() {
	// Derive the channel offset for each primary interface
	for _, intf := range platformCfg.intfs {
		if !intf.isPrimary {
			continue
		}
		intf.channelOffset = uint16(intf.lanes[0] % len(intf.lanes))

		// Propogate the offset to the rest of the group
		for _, i := range intf.intfGroup {
			i.channelOffset = intf.channelOffset
		}
	}
}

func ChannelOffset(intfName string) (uint16, error) {
	pIntf, err := platIntfByName(intfName)
	if err != nil {
		return 0, err
	}
	return pIntf.channelOffset, nil
}
