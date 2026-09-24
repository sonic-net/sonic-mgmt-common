package translib

import (
	"errors"
	"fmt"
	"github.com/Azure/sonic-mgmt-common/translib/db"
	"testing"
)

// TLV EEPROM Data Types
const (
	_TLV_CODE_PRODUCT_NAME   = "0x21"
	_TLV_CODE_PART_NUMBER    = "0x22"
	_TLV_CODE_SERIAL_NUMBER  = "0x23"
	_TLV_CODE_MAC_BASE       = "0x24"
	_TLV_CODE_MANUF_DATE     = "0x25"
	_TLV_CODE_DEVICE_VERSION = "0x26"
	_TLV_CODE_LABEL_REVISION = "0x27"
	_TLV_CODE_PLATFORM_NAME  = "0x28"
	_TLV_CODE_ONIE_VERSION   = "0x29"
	_TLV_CODE_MAC_SIZE       = "0x2A"
	_TLV_CODE_MANUF_NAME     = "0x2B"
	_TLV_CODE_MANUF_COUNTRY  = "0x2C"
	_TLV_CODE_VENDOR_NAME    = "0x2D"
	_TLV_CODE_DIAG_VERSION   = "0x2E"
	_TLV_CODE_SERVICE_TAG    = "0x2F"
	_TLV_CODE_VENDOR_EXT     = "0xFD"
	_TLV_CODE_CRC_32         = "0xFE"
)

const (
	TEST_PRODUCT_NAME  = "6776-64X-O-AC-F"
	TEST_PART_NUMBER   = "FP123454321PF"
	TEST_PLATFORM_NAME = "x86_64-pfm_test-platform"
	TEST_SERVICE_TAG   = "6776X6776"
	TEST_MANUF_NAME    = "TestManufacture"
)

// Test transceiver fixture values for TRANSCEIVER_INFO|Ethernet0,
// TRANSCEIVER_DOM_SENSOR|Ethernet0 and TRANSCEIVER_DOM_THRESHOLD|Ethernet0
// in STATE_DB. The interface name is the same one used by the existing
// PORT_TABLE entries to make scoped GETs land on a real component path.
const (
	TEST_XCVR_IFNAME       = "Ethernet0"
	TEST_XCVR_COMPONENT    = "transceiver_Ethernet0"
	TEST_XCVR_SERIAL       = "TESTSERIAL0001"
	TEST_XCVR_MODEL        = "TESTMODEL-100G-LR"
	TEST_XCVR_CONNECTOR    = "LC"
	TEST_XCVR_MANUFACTURER = "TestVendor"
	TEST_XCVR_VENDOR_OUI   = "00-11-22"
	TEST_XCVR_VENDOR_REV   = "A1"
	TEST_XCVR_VENDOR_DATE  = "2024-01-01 00:00:00"
)

type EepromEntry struct {
	TlvType string
	Name    string
	Value   string
}

var testStateDbList = [...]EepromEntry{
	{_TLV_CODE_PRODUCT_NAME, "Product Name", TEST_PRODUCT_NAME},
	{_TLV_CODE_PLATFORM_NAME, "Platform Name", TEST_PLATFORM_NAME},
	{_TLV_CODE_SERVICE_TAG, "Service Tag", TEST_SERVICE_TAG},
	{_TLV_CODE_MANUF_NAME, "Manufacturer", TEST_MANUF_NAME},
	{_TLV_CODE_PART_NUMBER, "Part Number", TEST_PART_NUMBER},
}

func init() {
	fmt.Println("+++++  Init pfm_app_test  +++++")

	if err := clearPfmDataFromDb(); err == nil {
		fmt.Println("+++++  Removed All Platform Data from Db  +++++")
	} else {
		fmt.Printf("Failed to remove All Platform Data from Db: %v", err)
	}

	if err := clearTransceiverDataFromDb(); err != nil {
		fmt.Printf("Failed to remove Transceiver Data from Db: %v\n", err)
	}
}

// This will test GET on /openconfig-platform:components
func Test_PfmApp_TopLevelPath(t *testing.T) {
	url := "/openconfig-platform:components"

	t.Run("Default_Response_Top_Level", processGetRequest(url, bulkPfmShowDefaultResponse, false))

	//Set the factory DB with pre-defined EEPROM_INFO entry
	if err := createPfmFactoryDb(); err != nil {
		fmt.Printf("Failed to add Platform Data to Db: %v", err)
	}

	t.Run("Get_Full_Pfm_Tree_Top_Level", processGetRequest(url, bulkPfmShowAllJsonResponse, false))
}

// Test_PfmApp_TransceiverState exercises the OpenConfig transceiver state
// subtree against STATE_DB fixtures for TRANSCEIVER_INFO, TRANSCEIVER_DOM_SENSOR
// and TRANSCEIVER_DOM_THRESHOLD. The assertions are intentionally lenient: we
// verify that each GET returns without error or panic, which is the regression
// guard for the nil-pointer concerns raised on PR #201 (map lookups against
// transceiverInfoTable / transceiverDomSensorTable / transceiverDomThresholdTable).
//
// Fixtures are loaded inside the test and torn down on cleanup so the
// existing Test_PfmApp_TopLevelPath bulk EEPROM test is unaffected by run
// ordering.
func Test_PfmApp_TransceiverState(t *testing.T) {
	if err := createTransceiverFactoryDb(); err != nil {
		t.Fatalf("Failed to add Transceiver data to Db: %v", err)
	}
	t.Cleanup(func() {
		if err := clearTransceiverDataFromDb(); err != nil {
			t.Logf("Cleanup: failed to remove Transceiver Data from Db: %v", err)
		}
	})

	cases := []struct {
		name string
		url  string
	}{
		{"State_Subtree", "/openconfig-platform:components/component[name=" + TEST_XCVR_COMPONENT + "]/openconfig-platform-transceiver:transceiver/state"},
		{"State_SerialNo", "/openconfig-platform:components/component[name=" + TEST_XCVR_COMPONENT + "]/openconfig-platform-transceiver:transceiver/state/serial-no"},
		{"State_Vendor", "/openconfig-platform:components/component[name=" + TEST_XCVR_COMPONENT + "]/openconfig-platform-transceiver:transceiver/state/vendor"},
		{"State_VendorPart", "/openconfig-platform:components/component[name=" + TEST_XCVR_COMPONENT + "]/openconfig-platform-transceiver:transceiver/state/vendor-part"},
		{"State_VendorRev", "/openconfig-platform:components/component[name=" + TEST_XCVR_COMPONENT + "]/openconfig-platform-transceiver:transceiver/state/vendor-rev"},
		{"State_DateCode", "/openconfig-platform:components/component[name=" + TEST_XCVR_COMPONENT + "]/openconfig-platform-transceiver:transceiver/state/date-code"},
		{"State_ConnectorType", "/openconfig-platform:components/component[name=" + TEST_XCVR_COMPONENT + "]/openconfig-platform-transceiver:transceiver/state/connector-type"},
		{"State_SupplyVoltage", "/openconfig-platform:components/component[name=" + TEST_XCVR_COMPONENT + "]/openconfig-platform-transceiver:transceiver/state/supply-voltage"},
		{"Thresholds_Critical", "/openconfig-platform:components/component[name=" + TEST_XCVR_COMPONENT + "]/openconfig-platform-transceiver:transceiver/thresholds/threshold[severity=CRITICAL]/state"},
		{"Thresholds_Warning", "/openconfig-platform:components/component[name=" + TEST_XCVR_COMPONENT + "]/openconfig-platform-transceiver:transceiver/thresholds/threshold[severity=WARNING]/state"},
	}

	for _, tc := range cases {
		t.Run(tc.name, verifyTransceiverGetNoError(tc.url))
	}
}

// Test_PfmApp_TransceiverState_Missing covers the case where no transceiver
// fixture is present in STATE_DB. The current pre-#201 code may return an
// empty response or surface a "data missing" tlerr depending on the path; the
// important regression property is that the handler does NOT panic from a
// nil map lookup on transceiverInfoTable / transceiverDomSensorTable /
// transceiverDomThresholdTable. Both nil-error and non-nil-error outcomes are
// accepted here; only a panic constitutes failure (recovered via t.Failed).
func Test_PfmApp_TransceiverState_Missing(t *testing.T) {
	// Ensure no transceiver entries exist.
	if err := clearTransceiverDataFromDb(); err != nil {
		t.Logf("setup: cleanup failed (continuing): %v", err)
	}

	cases := []struct {
		name string
		url  string
	}{
		{"Missing_State_SerialNo", "/openconfig-platform:components/component[name=transceiver_EthernetMissing]/openconfig-platform-transceiver:transceiver/state/serial-no"},
		{"Missing_State_SupplyVoltage", "/openconfig-platform:components/component[name=transceiver_EthernetMissing]/openconfig-platform-transceiver:transceiver/state/supply-voltage"},
		{"Missing_Thresholds_Critical", "/openconfig-platform:components/component[name=transceiver_EthernetMissing]/openconfig-platform-transceiver:transceiver/thresholds/threshold[severity=CRITICAL]/state"},
	}

	for _, tc := range cases {
		t.Run(tc.name, verifyTransceiverGetDoesNotPanic(tc.url))
	}
}

// verifyTransceiverGetNoError returns a t.Run-compatible function that issues
// a GET and fails the test if it surfaces an error. Body content is not
// inspected; this is regression scaffolding around the handler reaching
// completion against valid fixtures.
func verifyTransceiverGetNoError(url string) func(*testing.T) {
	return func(t *testing.T) {
		if _, err := Get(GetRequest{Path: url}); err != nil {
			t.Fatalf("GET %s returned error: %v", url, err)
		}
	}
}

// verifyTransceiverGetDoesNotPanic asserts the handler returns (either a
// response or a recoverable error) instead of panicking. Both error and
// success are acceptable; only an unrecovered panic fails the test.
func verifyTransceiverGetDoesNotPanic(url string) func(*testing.T) {
	return func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("GET %s panicked: %v", url, r)
			}
		}()
		_, _ = Get(GetRequest{Path: url})
	}
}

// THis will delete Platform Table from DB
func clearPfmDataFromDb() error {
	var err error
	eepromTable := db.TableSpec{Name: "EEPROM_INFO"}

	d := getStateDB()
	if d == nil {
		err = errors.New("Failed to connect to state Db")
		return err
	}
	if err = d.DeleteTable(&eepromTable); err != nil {
		err = errors.New("Failed to delete Eeprom Table")
		return err
	}
	return err
}

func createPfmFactoryDb() error {
	var err error
	eepromTable := db.TableSpec{Name: "EEPROM_INFO"}

	d := getStateDB()
	if d == nil {
		err = errors.New("Failed to connect to state Db")
		return err
	}

	for _, dbItem := range testStateDbList {
		ca := make([]string, 1, 1)
		ca[0] = dbItem.TlvType

		akey := db.Key{Comp: ca}
		avalue := db.Value{Field: map[string]string{
			"Name":  dbItem.Name,
			"Value": dbItem.Value,
		},
		}
		d.SetEntry(&eepromTable, akey, avalue)
	}
	return err
}

// clearTransceiverDataFromDb deletes the TRANSCEIVER_INFO, TRANSCEIVER_DOM_SENSOR
// and TRANSCEIVER_DOM_THRESHOLD tables from STATE_DB. Errors from individual
// DeleteTable calls are collected but a missing table is not treated as fatal:
// the helper is also used in init() where the tables may not exist yet.
func clearTransceiverDataFromDb() error {
	d := getStateDB()
	if d == nil {
		return errors.New("Failed to connect to state Db")
	}
	for _, name := range []string{"TRANSCEIVER_INFO", "TRANSCEIVER_DOM_SENSOR", "TRANSCEIVER_DOM_THRESHOLD"} {
		ts := db.TableSpec{Name: name}
		_ = d.DeleteTable(&ts)
	}
	return nil
}

// createTransceiverFactoryDb loads a single Ethernet0 transceiver entry into
// each of TRANSCEIVER_INFO, TRANSCEIVER_DOM_SENSOR and TRANSCEIVER_DOM_THRESHOLD
// in STATE_DB. Values are static and mirror the schema fields consumed by
// translib/pfm_app.go's getCompTransceiver*FromDb functions.
func createTransceiverFactoryDb() error {
	d := getStateDB()
	if d == nil {
		return errors.New("Failed to connect to state Db")
	}

	infoTable := db.TableSpec{Name: "TRANSCEIVER_INFO"}
	infoKey := db.Key{Comp: []string{TEST_XCVR_IFNAME}}
	infoValue := db.Value{Field: map[string]string{
		"serial":       TEST_XCVR_SERIAL,
		"model":        TEST_XCVR_MODEL,
		"connector":    TEST_XCVR_CONNECTOR,
		"manufacturer": TEST_XCVR_MANUFACTURER,
		"vendor_oui":   TEST_XCVR_VENDOR_OUI,
		"vendor_rev":   TEST_XCVR_VENDOR_REV,
		"vendor_date":  TEST_XCVR_VENDOR_DATE,
	}}
	if err := d.SetEntry(&infoTable, infoKey, infoValue); err != nil {
		return fmt.Errorf("SetEntry TRANSCEIVER_INFO: %w", err)
	}

	sensorTable := db.TableSpec{Name: "TRANSCEIVER_DOM_SENSOR"}
	sensorValue := db.Value{Field: map[string]string{
		"voltage":     "3.30",
		"temperature": "42.5",
		"tx1power":    "1.05",
		"tx2power":    "1.05",
		"tx3power":    "1.05",
		"tx4power":    "1.05",
		"tx5power":    "1.05",
		"tx6power":    "1.05",
		"tx7power":    "1.05",
		"tx8power":    "1.05",
		"rx1power":    "-2.10",
		"rx2power":    "-2.10",
		"rx3power":    "-2.10",
		"rx4power":    "-2.10",
		"rx5power":    "-2.10",
		"rx6power":    "-2.10",
		"rx7power":    "-2.10",
		"rx8power":    "-2.10",
		"tx1bias":     "8.50",
		"tx2bias":     "8.50",
		"tx3bias":     "8.50",
		"tx4bias":     "8.50",
		"tx5bias":     "8.50",
		"tx6bias":     "8.50",
		"tx7bias":     "8.50",
		"tx8bias":     "8.50",
	}}
	if err := d.SetEntry(&sensorTable, infoKey, sensorValue); err != nil {
		return fmt.Errorf("SetEntry TRANSCEIVER_DOM_SENSOR: %w", err)
	}

	thresholdTable := db.TableSpec{Name: "TRANSCEIVER_DOM_THRESHOLD"}
	thresholdValue := db.Value{Field: map[string]string{
		"temphighalarm":      "85.0",
		"templowalarm":       "-10.0",
		"vcchighalarm":       "3.6",
		"vcclowalarm":        "3.0",
		"temphighwarning":    "75.0",
		"templowwarning":     "-5.0",
		"vcchighwarning":     "3.5",
		"vcclowwarning":      "3.1",
		"txpowerhighalarm":   "3.0",
		"txpowerlowalarm":    "-5.0",
		"rxpowerhighalarm":   "3.0",
		"rxpowerlowalarm":    "-15.0",
		"txbiashighalarm":    "12.0",
		"txbiaslowalarm":     "2.0",
		"txpowerhighwarning": "2.5",
		"txpowerlowwarning":  "-4.0",
		"rxpowerhighwarning": "2.5",
		"rxpowerlowwarning":  "-13.0",
		"txbiashighwarning":  "11.0",
		"txbiaslowwarning":   "3.0",
	}}
	if err := d.SetEntry(&thresholdTable, infoKey, thresholdValue); err != nil {
		return fmt.Errorf("SetEntry TRANSCEIVER_DOM_THRESHOLD: %w", err)
	}

	return nil
}

func getStateDB() *db.DB {
	stateDb, _ := db.NewDB(db.Options{
		DBNo:               db.StateDB,
		InitIndicator:      "STATE_DB_INITIALIZED",
		TableNameSeparator: "|",
		KeySeparator:       "|",
	})

	return stateDb
}

/***************************************************************************/
///////////                  JSON Data for Tests              ///////////////
/***************************************************************************/
var bulkPfmShowDefaultResponse string = "{\"openconfig-platform:components\":{\"component\":[{\"name\":\"System Eeprom\",\"state\":{\"empty\":false,\"location\":\"Slot 1\",\"name\":\"System Eeprom\",\"oper-status\":\"openconfig-platform-types:ACTIVE\",\"removable\":false}}]}}"

var bulkPfmShowAllJsonResponse string = "{\"openconfig-platform:components\":{\"component\":[{\"name\":\"System Eeprom\",\"state\":{\"description\":\"" + TEST_PLATFORM_NAME + "\",\"empty\":false,\"id\":\"" + TEST_PRODUCT_NAME + "\",\"location\":\"Slot 1\",\"mfg-name\":\"" + TEST_MANUF_NAME + "\",\"name\":\"System Eeprom\",\"oper-status\":\"openconfig-platform-types:ACTIVE\",\"part-no\":\"" + TEST_PART_NUMBER + "\",\"removable\":false,\"serial-no\":\"" + TEST_SERVICE_TAG + "\"}}]}}"
