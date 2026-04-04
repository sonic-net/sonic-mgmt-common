////////////////////////////////////////////////////////////////////////////////
//                                                                            //
//  Copyright 2019 Broadcom. The term Broadcom refers to Broadcom Inc. and/or //
//  its subsidiaries.                                                         //
//                                                                            //
//  Licensed under the Apache License, Version 2.0 (the "License");           //
//  you may not use this file except in compliance with the License.          //
//  You may obtain a copy of the License at                                   //
//                                                                            //
//     http://www.apache.org/licenses/LICENSE-2.0                             //
//                                                                            //
//  Unless required by applicable law or agreed to in writing, software       //
//  distributed under the License is distributed on an "AS IS" BASIS,         //
//  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.  //
//  See the License for the specific language governing permissions and       //
//  limitations under the License.                                            //
//                                                                            //
////////////////////////////////////////////////////////////////////////////////

/*
Package db implements a wrapper over the go-redis/redis.

There may be an attempt to mimic sonic-py-swsssdk to ease porting of
code written in python using that SDK to Go Language.

Example:

  - Initialization:

    d, _ := db.NewDB(db.Options {
    DBNo              : db.ConfigDB,
    InitIndicator     : "CONFIG_DB_INITIALIZED",
    TableNameSeparator: "|",
    KeySeparator      : "|",
    })

  - Close:

    d.DeleteDB()

  - No-Transaction SetEntry

    tsa := db.TableSpec { Name: "ACL_TABLE" }
    tsr := db.TableSpec { Name: "ACL_RULE" }

    ca := make([]string, 1, 1)

    ca[0] = "MyACL1_ACL_IPV4"
    akey := db.Key { Comp: ca}
    avalue := db.Value {map[string]string {"ports":"eth0","type":"mirror" }}

    d.SetEntry(&tsa, akey, avalue)

  - GetEntry

    avalue, _ := d.GetEntry(&tsa, akey)

  - GetKeys

    keys, _ := d.GetKeys(&tsa);

  - GetKeysPattern

    keys, _ := d.GetKeys(&tsa, akeyPattern);

  - No-Transaction DeleteEntry

    d.DeleteEntry(&tsa, akey)

  - GetTable

    ta, _ := d.GetTable(&tsa)

  - No-Transaction DeleteTable

    d.DeleteTable(&ts)

  - Transaction

    rkey := db.Key { Comp: []string { "MyACL2_ACL_IPV4", "RULE_1" }}
    rvalue := db.Value { Field: map[string]string {
    "priority" : "0",
    "packet_action" : "eth1",
    },
    }

    d.StartTx([]db.WatchKeys { {Ts: &tsr, Key: &rkey} },
    []*db.TableSpec { &tsa, &tsr })

    d.SetEntry( &tsa, akey, avalue)
    d.SetEntry( &tsr, rkey, rvalue)

    e := d.CommitTx()

  - Transaction Abort

    d.StartTx([]db.WatchKeys {},
    []*db.TableSpec { &tsa, &tsr })
    d.DeleteEntry( &tsa, rkey)
    d.AbortTx()
*/
package db

import (
	"fmt"
	"strconv"

	//	"reflect"
	"errors"
	"strings"
	"time"

	"github.com/Azure/sonic-mgmt-common/cvl"
	cmn "github.com/Azure/sonic-mgmt-common/cvl/common"
	"github.com/Azure/sonic-mgmt-common/translib/tlerr"
	"github.com/go-redis/redis/v7"
	"github.com/golang/glog"
)

const (
	DefaultRedisUNIXSocket  string = "/var/run/redis/redis.sock"
	DefaultRedisLocalTCPEP  string = "localhost:6379"
	DefaultRedisRemoteTCPEP string = "127.0.0.1:6379"
	DefaultRedisUNIXNetwork string = "unix"
	DefaultRedisTCPNetwork  string = "tcp"
)

func init() {
	dbConfigInit()
}

// DBNum type indicates the type of DB (Eg: ConfigDB, ApplDB, ...).
type DBNum int

const (
	ApplDB        DBNum = iota // 0
	AsicDB                     // 1
	CountersDB                 // 2
	LogLevelDB                 // 3
	ConfigDB                   // 4
	FlexCounterDB              // 5
	StateDB                    // 6
	SnmpDB                     // 7
	ErrorDB                    // 8
	EventDB                    // 9
	// All DBs added above this line, please ----
	MaxDB //  The Number of DBs
)

func (dbNo DBNum) String() string {
	return getDBInstName(dbNo)
}

// ID returns the redis db id for this DBNum
func (dbNo DBNum) ID() int {
	name := getDBInstName(dbNo)
	if len(name) == 0 {
		panic("Invalid DBNum " + fmt.Sprintf("%d", dbNo))
	}
	return getDbId(name)
}

// Options gives parameters for opening the redis client.
type Options struct {
	DBNo               DBNum
	InitIndicator      string
	TableNameSeparator string //Overriden by the DB config file's separator.
	KeySeparator       string //Overriden by the DB config file's separator.
	IsWriteDisabled    bool   //Is write/set mode disabled ?
	IsCacheEnabled     bool   //Is cache (Per Connection) allowed?

	// OnChange caching for the DBs passed from Translib's Subscribe Infra
	// to the Apps. SDB is the SubscribeDB() returned handle on which
	// notifications of change are received.
	IsOnChangeEnabled bool // whether OnChange cache enabled

	SDB *DB //The subscribeDB handle (Future Use)

	IsSubscribeDB bool // Opened by SubscribeDB(Sess)?

	IsSession        bool // Is this a Candidate Config DB ?
	ConfigDBLazyLock bool // For Non-CCDB Action()/RPC (may write to ConfigDB)
	TxCmdsLim        int  // Tx Limit for Candidate Config DB

	IsReplaced  bool // Is candidate Config DB updated by config-replace operation.
	IsCommitted bool // Is candidate Config DB committed.

	// Alternate Datastore: By default, we query redis CONFIG_DB.
	// Front-end an alternate source of data. (Eg: config_db.cp.json
	// from a saved commit-id, or snapshot)
	Datastore DBDatastore

	DisableCVLCheck bool
}

func (o Options) String() string {
	return fmt.Sprintf(
		"{ DBNo: %v, InitIndicator: %v, TableNameSeparator: %v, KeySeparator: %v, IsWriteDisabled: %v, IsCacheEnabled: %v, IsOnChangeEnabled: %v, SDB: %v, DisableCVLCheck: %v, IsSession: %v, ConfigDBLazyLock: %v, TxCmdsLim: %v }",
		o.DBNo, o.InitIndicator, o.TableNameSeparator, o.KeySeparator,
		o.IsWriteDisabled, o.IsCacheEnabled, o.IsOnChangeEnabled, o.SDB,
		o.DisableCVLCheck, o.IsSession, o.ConfigDBLazyLock, o.TxCmdsLim)
}

type _txState int

const (
	txStateNone      _txState = iota // Idle (No transaction)
	txStateWatch                     // WATCH issued
	txStateSet                       // At least one Set|Mod|Delete done.
	txStateMultiExec                 // Between MULTI & EXEC
)

func (s _txState) String() string {
	var state string
	switch s {
	case txStateNone:
		state = "txStateNone"
	case txStateWatch:
		state = "txStateWatch"
	case txStateSet:
		state = "txStateSet"
	case txStateMultiExec:
		state = "txStateMultiExec"
	default:
		state = "Unknown _txState"
	}
	return state
}

const (
	InitialTxPipelineSize    int = 100
	InitialTablesCount       int = 20
	InitialTableEntryCount   int = 50
	InitialTablePatternCount int = 5
	InitialMapsCount         int = 10
	InitialMapKeyCount       int = 50
)

// TableSpec gives the name of the table, and other per-table customizations.
// (Eg: { Name: ACL_TABLE" }).
type TableSpec struct {
	Name string
	// https://github.com/project-arlo/sonic-mgmt-framework/issues/29
	// CompCt tells how many components in the key. Only the last component
	// can have TableSeparator as part of the key. Otherwise, we cannot
	// tell where the key component begins.
	CompCt int
	// NoDelete flag (if it is set to true) is to skip the row entry deletion from
	// the table when the "SetEntry" or "ModEntry" method is called with empty Value Field map.
	NoDelete bool
}

const (
	ConnectionClosed = tlerr.TranslibDBInvalidState("connection closed")
	OnChangeDisabled = tlerr.TranslibDBInvalidState("OnChange disabled")
	SupportsReadOnly = tlerr.TranslibDBInvalidState("Supported on read only")

	OnChangeNoSupport = tlerr.TranslibDBInvalidState("OnChange not supported")
	SupportsCfgDBOnly = tlerr.TranslibDBInvalidState("Supported on CfgDB only")
	UseGetEntry       = tlerr.TranslibDBInvalidState("Use GetEntry()")
)

type _txOp int

const (
	txOpNone  _txOp = iota // No Op
	txOpHMSet              // key, value gives the field:value to be set in key
	txOpHDel               // key, value gives the fields to be deleted in key
	txOpDel                // key
)

type _txCmd struct {
	ts    *TableSpec
	op    _txOp
	key   *Key
	value *Value
}

// DB is the main type.
type DB struct {
	client *redis.Client
	Opts   *Options

	txState      _txState
	txCmds       []_txCmd
	txTsEntryMap map[string]map[string]Value //map[TableSpec.Name]map[Entry]Value

	// For Config Session only, cache the HGetAll for restoring the
	// txTsEntryMap on error recovery/rollback. This avoids the duplicate
	// read for recovery/rollback. The Config DB is locked, therefore
	// it need not be read again.
	txTsEntryHGetAll map[string]map[string]Value //map[TableSpec.Name]map[Entry]Value

	cv                *cvl.CVL
	cvlHintsB4Open    map[string]interface{} // Hints set before CVLSess Opened
	cvlEditConfigData []cmn.CVLEditConfigData

	// If there is an error while Rollback (or similar), set this flag.
	// In this state, all writes are disabled, and this error is returned.
	err error // CVL Rollback/CVL/Commit error

	sPubSub     *redis.PubSub // PubSub. non-Nil implies SubscribeDB
	sCIP        bool          // Close in Progress
	sOnCCacheDB *DB           // Update this DB for PubSub notifications

	dbStatsConfig DBStatsConfig
	dbCacheConfig DBCacheConfig

	// DBStats is used by both PerConnection cache, and OnChange cache
	// On a DB handle, the two are mutually exclusive.
	stats DBStats
	cache dbCache

	onCReg dbOnChangeReg // holds OnChange enabled table names

	// PubSubRpc
	rPubSub *redis.PubSub

	// Non-Session Config DB Lock acquired
	configDBLocked bool
}

func (d DB) String() string {
	return fmt.Sprintf("{ client: %v, Opts: %v, txState: %v, tsCmds: %v }",
		d.client, d.Opts, d.txState, d.txCmds)
}

func (dbNo DBNum) Name() string {
	return (getDBInstName(dbNo))
}

func (d *DB) IsDirtified() bool {
	return (len(d.txCmds) > 0)
}

func GetDBInstName(dbNo DBNum) string {
	return getDBInstName(dbNo)
}

func getDBInstName(dbNo DBNum) string {
	switch dbNo {
	case ApplDB:
		return "APPL_DB"
	case AsicDB:
		return "ASIC_DB"
	case CountersDB:
		return "COUNTERS_DB"
	case LogLevelDB:
		return "LOGLEVEL_DB"
	case ConfigDB:
		return "CONFIG_DB"
	case FlexCounterDB:
		return "FLEX_COUNTER_DB"
	case StateDB:
		return "STATE_DB"
	case SnmpDB:
		return "SNMP_OVERLAY_DB"
	case ErrorDB:
		return "ERROR_DB"
	case EventDB:
		return "EVENT_DB"
	}
	return ""
}

func GetdbNameToIndex(dbName string) DBNum {
	dbIndex := ConfigDB
	switch dbName {
	case "APPL_DB":
		dbIndex = ApplDB
	case "ASIC_DB":
		dbIndex = AsicDB
	case "COUNTERS_DB":
		dbIndex = CountersDB
	case "LOGLEVEL_DB":
		dbIndex = LogLevelDB
	case "CONFIG_DB":
		dbIndex = ConfigDB
	case "FLEX_COUNTER_DB":
		dbIndex = FlexCounterDB
	case "STATE_DB":
		dbIndex = StateDB
	case "ERROR_DB":
		dbIndex = ErrorDB
	case "EVENT_DB":
		dbIndex = EventDB
	}
	return dbIndex
}

// NewDB is the factory method to create new DB's.
func NewDB(opt Options) (*DB, error) {

	var e error

	if glog.V(3) {
		glog.Info("NewDB: Begin: opt: ", opt)
	}

	// Time Start
	var now time.Time
	var dur time.Duration
	now = time.Now()

	d := DB{client: redis.NewClient(adjustRedisOpts(&opt)),
		Opts:              &opt,
		txState:           txStateNone,
		txCmds:            make([]_txCmd, 0, InitialTxPipelineSize),
		cvlEditConfigData: make([]cmn.CVLEditConfigData, 0, InitialTxPipelineSize),
		dbStatsConfig:     getDBStatsConfig(),
		stats:             DBStats{Tables: make(map[string]Stats, InitialTablesCount), Maps: make(map[string]Stats, InitialMapsCount)},
		dbCacheConfig:     getDBCacheConfig(),
		cache:             dbCache{Tables: make(map[string]Table, InitialTablesCount), Maps: make(map[string]MAP, InitialMapsCount)},
	}

	if d.client == nil {
		glog.Error("NewDB: Could not create redis client: ", d.Name())
		e = tlerr.TranslibDBCannotOpen{}
		goto NewDBExit
	}

	if opt.IsCacheEnabled && opt.IsOnChangeEnabled {
		glog.Error("Per Connection cache cannot be enabled with OnChange cache")
		glog.Error("Disabling Per Connection caching")
		opt.IsCacheEnabled = false
	}

	if opt.IsOnChangeEnabled && !opt.IsWriteDisabled {
		glog.Errorf("NewDB: IsEnableOnChange cannot be set on write enabled DB")
		e = tlerr.TranslibDBCannotOpen{}
		goto NewDBExit
	}

	if !d.Opts.IsWriteDisabled {
		if d.dbCacheConfig.PerConnection {
			glog.Info("NewDB: IsWriteDisabled false. Disable Cache")
		}
		d.dbCacheConfig.PerConnection = false
	}

	if !d.Opts.IsCacheEnabled {
		if d.dbCacheConfig.PerConnection {
			glog.Info("NewDB: IsCacheEnabled false. Disable Cache")
		}
		d.dbCacheConfig.PerConnection = false
	}

	if opt.IsSession && d.dbCacheConfig.PerConnection {
		if d.dbCacheConfig.PerConnection {
			glog.Info("NewDB: IsSession true. Disable Cache")
		}
		d.dbCacheConfig.PerConnection = false
	}

	if opt.IsSession && opt.IsOnChangeEnabled {
		glog.Error("NewDB: Subscription on Config Session not supported : ",
			d.Name())
		d.client.Close()
		e = tlerr.TranslibDBNotSupported{
			Description: "Subscription on Config Session not supported"}
		goto NewDBExit
	}

	if opt.IsSession && opt.DBNo != ConfigDB {
		glog.Error("NewDB: Non-Config DB on Config Session not supported : ",
			d.Name())
		d.client.Close()
		e = tlerr.TranslibDBNotSupported{
			Description: "Non-Config DB on Config Session not supported"}
		goto NewDBExit
	}

	if opt.IsOnChangeEnabled {
		d.onCReg = dbOnChangeReg{CacheTables: make(map[string]bool, InitialTablesCount)}
	}

	if opt.DBNo != ConfigDB {
		if glog.V(3) {
			glog.Info("NewDB: ! ConfigDB. Skip init. check.")
		}
		goto NewDBSkipInitIndicatorCheck
	}

	if len(d.Opts.InitIndicator) == 0 {

		if glog.V(3) {
			glog.Info("NewDB: Init indication not requested")
		}

	} else {
		glog.V(3).Info("NewDB: RedisCmd: ", d.Name(), ": ", "GET ",
			d.Opts.InitIndicator)
		if init, err := d.client.Get(d.Opts.InitIndicator).Int(); init != 1 {

			glog.Error("NewDB: Database not inited: ", d.Name(), ": GET ",
				d.Opts.InitIndicator)
			if err != nil {
				glog.Error("NewDB: Database not inited: ", d.Name(), ": GET ",
					d.Opts.InitIndicator, " returns err: ", err)
			}
			d.client.Close()
			e = tlerr.TranslibDBNotInit{}
			goto NewDBExit
		}
	}

	// Lazy ConfigDBLock, because Action()/RPC ConfigDB Modifiers do not
	// tell in advance whether they are going to perform a Write Operation,
	// and we do not want to block a Read Operation on Action()/RPC
	if opt.DBNo == ConfigDB && !opt.IsSession &&
		!opt.IsWriteDisabled && !opt.ConfigDBLazyLock {

		if e = ConfigDBTryLock(noSessionToken); e != nil {
			glog.Errorf("NewDB: ConfigDB possibly locked: %s", e)
			d.client.Close()
			goto NewDBExit
		}
		d.configDBLocked = true
	}

	// Register Candidate Config (Session) DBs
	if opt.IsSession {
		d.registerSessionDB()
	}

NewDBSkipInitIndicatorCheck:

NewDBExit:

	if d.dbStatsConfig.TimeStats {
		dur = time.Since(now)
	}

	dbGlobalStats.updateStats(d.Opts.DBNo, true, dur, &(d.stats))

	if glog.V(3) {
		glog.Info("NewDB: End: d: ", d, " e: ", e)
	}

	return &d, e
}

// DeleteDB is the gentle way to close the DB connection.
func (d *DB) DeleteDB() error {
	if d == nil {
		return nil
	}
	if glog.V(3) {
		glog.Info("DeleteDB: Begin: d: ", d)
	}

	// Release the ConfigDB Lock if we placed on in NewDB()
	if d.configDBLocked {
		ConfigDBUnlock(noSessionToken)
		d.configDBLocked = false
	}

	if !d.IsOpen() {
		return ConnectionClosed
	}

	dbGlobalStats.updateStats(d.Opts.DBNo, false, 0, &(d.stats))

	if d.txState != txStateNone {
		glog.Warning("DeleteDB: not txStateNone, txState: ", d.txState)
	}

	// Unregister Candidate Config (Session) DB
	if d.Opts.IsSession {
		d.unRegisterSessionDB()
	}

	err := d.client.Close()
	d.client = nil
	return err
}

func (d *DB) IsOpen() bool {
	return d != nil && d.client != nil
}

func (d *DB) key2redis(ts *TableSpec, key Key) string {

	if glog.V(5) {
		glog.Info("key2redis: Begin: ",
			ts.Name+
				d.Opts.TableNameSeparator+
				strings.Join(key.Comp, d.Opts.KeySeparator))
	}
	return ts.Name +
		d.Opts.TableNameSeparator +
		strings.Join(key.Comp, d.Opts.KeySeparator)
}

func (d *DB) redis2key(ts *TableSpec, redisKey string) Key {

	splitTable := strings.SplitN(redisKey, d.Opts.TableNameSeparator, 2)

	if ts.CompCt > 0 {
		return Key{strings.SplitN(splitTable[1], d.Opts.KeySeparator, ts.CompCt)}
	} else {
		return Key{strings.Split(splitTable[1], d.Opts.KeySeparator)}
	}

}

// redis2ts_key  works only if keys don't contain the (Table|Key)Separator char
// (The TableSpec does not have the CompCt)
func (d *DB) redis2ts_key(redisKey string) (TableSpec, Key) {

	splitTable := strings.SplitN(redisKey, d.Opts.TableNameSeparator, 2)

	return TableSpec{Name: splitTable[0]},
		Key{strings.Split(splitTable[1], d.Opts.KeySeparator)}
}

func (d *DB) ts2redisUpdated(ts *TableSpec) string {

	if glog.V(5) {
		glog.Info("ts2redisUpdated: Begin: ", ts.Name)
	}

	var updated string

	if strings.Contains(ts.Name, "*") {
		updated = string("CONFIG_DB_UPDATED")
	} else {
		updated = string("CONFIG_DB_UPDATED_") + ts.Name
	}

	return updated
}

// GetEntry retrieves an entry(row) from the table.
func (d *DB) GetEntry(ts *TableSpec, key Key) (Value, error) {
	if !d.IsOpen() {
		return Value{}, ConnectionClosed
	}

	return d.getEntry(ts, key, false)
}

func (d *DB) getEntry(ts *TableSpec, key Key, forceReadDB bool) (Value, error) {

	if glog.V(3) {
		glog.Info("GetEntry: Begin: ", d.Name(), ": ts: ", ts, " key: ", key)
	}

	// GetEntryHits
	// Time Start
	var cacheHit bool
	var txCacheHit bool
	var now time.Time
	var dur time.Duration
	var stats Stats
	if d.dbStatsConfig.TimeStats {
		now = time.Now()
	}

	var table Table
	var value Value
	var e error
	var v map[string]string

	var ok bool
	entry := d.key2redis(ts, key)
	useCache := ((d.Opts.IsOnChangeEnabled && d.onCReg.isCacheTable(ts.Name)) ||
		(d.dbCacheConfig.PerConnection &&
			d.dbCacheConfig.isCacheTable(ts.Name)))

	// check in Tx cache first
	if value, ok = d.txTsEntryMap[ts.Name][entry]; !ok {
		// If cache GetFromCache (CacheHit?)
		if useCache && !forceReadDB {
			if table, ok = d.cache.Tables[ts.Name]; ok {
				if value, ok = table.entry[entry]; ok {
					value = value.Copy()
					cacheHit = true
				}
			}
		}
	} else {
		value = value.Copy()
		txCacheHit = true
	}

	if !cacheHit && !txCacheHit {
		// Increase (i.e. more verbose) V() level if it gets too noisy.
		if glog.V(3) {
			glog.Info("getEntry: RedisCmd: ", d.Name(), ": ", "HGETALL ", entry)
		}
		v, e = d.client.HGetAll(entry).Result()
		value = Value{Field: v}
	}

	if e != nil {
		if glog.V(2) {
			glog.Errorf("GetEntry: %s: HGetAll(%q) error: %v", d.Name(),
				entry, e)
		}
		value = Value{}

	} else if !value.IsPopulated() {
		if glog.V(4) {
			glog.Info("GetEntry: HGetAll(): empty map")
		}
		e = tlerr.TranslibRedisClientEntryNotExist{Entry: d.key2redis(ts, key)}

	} else if !cacheHit && !txCacheHit && useCache {
		if _, ok := d.cache.Tables[ts.Name]; !ok {
			if d.cache.Tables == nil {
				d.cache.Tables = make(map[string]Table, d.onCReg.size())
			}
			d.cache.Tables[ts.Name] = Table{
				ts:       ts,
				entry:    make(map[string]Value, InitialTableEntryCount),
				complete: false,
				patterns: make(map[string][]Key, InitialTablePatternCount),
				db:       d,
			}
		}
		d.cache.Tables[ts.Name].entry[entry] = value.Copy()
	}

	// Time End, Time, Peak
	if d.dbStatsConfig.TableStats {
		stats = d.stats.Tables[ts.Name]
	} else {
		stats = d.stats.AllTables
	}

	stats.Hits++
	stats.GetEntryHits++
	if cacheHit {
		stats.GetEntryCacheHits++
	}

	if d.dbStatsConfig.TimeStats {
		dur = time.Since(now)

		if dur > stats.Peak {
			stats.Peak = dur
		}
		stats.Time += dur

		if dur > stats.GetEntryPeak {
			stats.GetEntryPeak = dur
		}
		stats.GetEntryTime += dur
	}

	if d.dbStatsConfig.TableStats {
		d.stats.Tables[ts.Name] = stats
	} else {
		d.stats.AllTables = stats
	}

	if glog.V(3) {
		glog.Info("GetEntry: End: ", "value: ", value, " e: ", e)
	}

	return value, e
}

// GetKeys retrieves all entry/row keys.
func (d *DB) GetKeys(ts *TableSpec) ([]Key, error) {

	// If ts contains (Key|TableName)Separator (Eg: "|"), translate this to
	// a GetKeysPattern, by extracting the initial Key Comps from TableName
	// Slice into the t(able) (a)N(d) k(ey)Pat(tern) if any
	if tNkPat := strings.SplitN(ts.Name, d.Opts.TableNameSeparator,
		2); len(tNkPat) == 2 {

		tsNk := &TableSpec{Name: tNkPat[0], CompCt: ts.CompCt}
		pat := Key{Comp: append(strings.Split(tNkPat[1], d.Opts.KeySeparator),
			"*")}
		glog.Warningf("GetKeys: Separator in TableSpec %v is Deprecated. "+
			"Using TableSpec.Name %s, Pattern %v", ts, tsNk.Name, pat)
		return d.GetKeysPattern(tsNk, pat)
	}

	return d.GetKeysPattern(ts, Key{Comp: []string{"*"}})
}

func (d *DB) GetKeysPattern(ts *TableSpec, pat Key) ([]Key, error) {

	// GetKeysHits
	// Time Start
	var cacheHit bool
	var now time.Time
	var dur time.Duration
	var stats Stats
	var table Table
	var keys []Key
	var e error

	if glog.V(3) {
		glog.Info("GetKeys: Begin: ", d.Name(), ": ts: ", ts, "pat: ", pat)
	}

	if !d.IsOpen() {
		return keys, ConnectionClosed
	}

	defer func() {
		if e != nil {
			glog.Error("GetKeys: ts: ", ts, " e: ", e)
		}
		if glog.V(3) {
			glog.Info("GetKeys: End: ", "keys: ", keys, " e: ", e)
		}
	}()

	if d.dbStatsConfig.TimeStats {
		now = time.Now()
	}

	// If cache GetFromCache (CacheHit?)
	if d.dbCacheConfig.PerConnection && d.dbCacheConfig.isCacheTable(ts.Name) {
		var ok bool
		if table, ok = d.cache.Tables[ts.Name]; ok {
			if keys, ok = table.patterns[d.key2redis(ts, pat)]; ok {
				cacheHit = true
			}
		}
	}

	if !cacheHit {
		// Increase (i.e. more verbose) V() level if it gets too noisy.
		if glog.V(3) {
			glog.Info("GetKeysPattern: RedisCmd: ", d.Name(), ": ", "KEYS ", d.key2redis(ts, pat))
		}
		var redisKeys []string
		redisKeys, e = d.client.Keys(d.key2redis(ts, pat)).Result()

		keys = make([]Key, 0, len(redisKeys))
		// On error, return promptly
		if e != nil {
			return keys, e
		}

		for i := 0; i < len(redisKeys); i++ {
			keys = append(keys, d.redis2key(ts, redisKeys[i]))
		}

		// If cache SetCache (i.e. a cache miss)
		if d.dbCacheConfig.PerConnection && d.dbCacheConfig.isCacheTable(ts.Name) {
			if _, ok := d.cache.Tables[ts.Name]; !ok {
				d.cache.Tables[ts.Name] = Table{
					ts:       ts,
					entry:    make(map[string]Value, InitialTableEntryCount),
					complete: false,
					patterns: make(map[string][]Key, InitialTablePatternCount),
					db:       d,
				}
			}
			// Make a copy for the Per Connection cache which is always
			// *before* adjusting with Redis CAS Tx Cache.
			keysCopy := make([]Key, len(keys))
			for i, key := range keys {
				keysCopy[i] = key.Copy()
			}
			d.cache.Tables[ts.Name].patterns[d.key2redis(ts, pat)] = keysCopy
		}
	}

	for k := range d.txTsEntryMap[ts.Name] {
		if patternMatch(k, 0, d.key2redis(ts, pat), 0) {
			var present bool
			var index int
			key := d.redis2key(ts, k)
			for i := 0; i < len(keys); i++ {
				index = i
				if key.Equals(keys[i]) {
					present = true
					break
				}
			}
			if !present {
				if len(d.txTsEntryMap[ts.Name][k].Field) > 0 {
					keys = append(keys, key)
				}
			} else {
				if len(d.txTsEntryMap[ts.Name][k].Field) == 0 {
					keys = append(keys[:index], keys[index+1:]...)
				}
			}
		}
	}

	// Time End, Time, Peak
	if d.dbStatsConfig.TableStats {
		stats = d.stats.Tables[ts.Name]
	} else {
		stats = d.stats.AllTables
	}

	stats.Hits++
	stats.GetKeysPatternHits++
	if cacheHit {
		stats.GetKeysPatternCacheHits++
	}
	if (len(pat.Comp) == 1) && (pat.Comp[0] == "*") {
		stats.GetKeysHits++
		if cacheHit {
			stats.GetKeysCacheHits++
		}
	}

	if d.dbStatsConfig.TimeStats {
		dur = time.Since(now)

		if dur > stats.Peak {
			stats.Peak = dur
		}
		stats.Time += dur

		if dur > stats.GetKeysPatternPeak {
			stats.GetKeysPatternPeak = dur
		}
		stats.GetKeysPatternTime += dur

		if (len(pat.Comp) == 1) && (pat.Comp[0] == "*") {

			if dur > stats.GetKeysPeak {
				stats.GetKeysPeak = dur
			}
			stats.GetKeysTime += dur
		}
	}

	if d.dbStatsConfig.TableStats {
		d.stats.Tables[ts.Name] = stats
	} else {
		d.stats.AllTables = stats
	}

	return keys, e
}

// GetKeysByPattern retrieves all entry/row keys matching with the given pattern
// Deprecated: use GetKeysPattern()
func (d *DB) GetKeysByPattern(ts *TableSpec, pattern string) ([]Key, error) {
	glog.Warning("GetKeysByPattern() is deprecated and it will be removed in the future, please use GetKeysPattern()")
	return d.GetKeysPattern(ts, Key{Comp: []string{pattern}})
}

// DeleteKeys deletes all entry/row keys matching a pattern.
func (d *DB) DeleteKeys(ts *TableSpec, key Key) error {
	if glog.V(3) {
		glog.Info("DeleteKeys: Begin: ", d.Name(), ": ts: ", ts, " key: ", key)
	}

	if !d.IsOpen() {
		return ConnectionClosed
	}

	// This can be done via a LUA script as well. For now do this. TBD
	redisKeys, e := d.GetKeysPattern(ts, key)
	if glog.V(4) {
		glog.Info("DeleteKeys: redisKeys: ", redisKeys, " e: ", e)
	}

	for i := 0; i < len(redisKeys); i++ {
		if glog.V(4) {
			glog.Info("DeleteKeys: Deleting redisKey: ", redisKeys[i])
		}
		e = d.DeleteEntry(ts, redisKeys[i])
		if e != nil {
			glog.Warning("DeleteKeys: Deleting: ts: ", ts, " key",
				redisKeys[i], " : ", e)
		}
	}

	if glog.V(3) {
		glog.Info("DeleteKeys: End: e: ", e)
	}
	return e
}

func (d *DB) doCVL(ts *TableSpec, cvlOps []cmn.CVLOperation, key Key, vals []Value) error {
	var e error = nil

	var cvlRetCode cvl.CVLRetCode
	var cei cvl.CVLErrorInfo

	if d.err != nil {
		e = d.err
		glog.Error("doCVL: DB in error: ", e)
		goto doCVLExit
	}

	if d.Opts.DisableCVLCheck {
		glog.Info("doCVL: CVL Disabled. Skipping CVL")
		goto doCVLExit
	}

	// No Transaction case. No CVL.
	if d.txState == txStateNone {
		glog.Info("doCVL: No Transactions. Skipping CVL")
		goto doCVLExit
	}

	if len(cvlOps) != len(vals) {
		glog.Error("doCVL: Incorrect arguments len(cvlOps) != len(vals)")
		e = errors.New("CVL Incorrect args")
		return e
	}
	for i := 0; i < len(cvlOps); i++ {

		cvlEditConfigData := cmn.CVLEditConfigData{
			VType: cmn.VALIDATE_ALL,
			VOp:   cvlOps[i],
			Key:   d.key2redis(ts, key),
			// Await CVL PR ReplaceOp: isReplaceOp,
		}

		switch cvlOps[i] {
		case cmn.OP_CREATE, cmn.OP_UPDATE:
			cvlEditConfigData.Data = vals[i].Copy().Field
			d.cvlEditConfigData = append(d.cvlEditConfigData, cvlEditConfigData)

		case cmn.OP_DELETE:
			if len(vals[i].Field) == 0 {
				cvlEditConfigData.Data = map[string]string{}
			} else {
				cvlEditConfigData.Data = vals[i].Copy().Field
			}
			d.cvlEditConfigData = append(d.cvlEditConfigData, cvlEditConfigData)

		default:
			glog.Error("doCVL: Unknown, op: ", cvlOps[i])
			e = errors.New("Unknown Op: " + string(rune(cvlOps[i])))
		}

	}

	if e != nil {
		goto doCVLExit
	}

	if glog.V(3) {
		glog.Info("doCVL: calling ValidateEditConfig: ", d.cvlEditConfigData)
	}

	cei, cvlRetCode = d.cv.ValidateEditConfig(d.cvlEditConfigData)

	if cvl.CVL_SUCCESS != cvlRetCode {
		glog.Warning("doCVL: CVL Failure: ", cvlRetCode)
		// e = errors.New("CVL Failure: " + string(cvlRetCode))
		e = tlerr.TranslibCVLFailure{Code: int(cvlRetCode),
			CVLErrorInfo: cei}
		glog.Info("doCVL: ", len(d.cvlEditConfigData), len(cvlOps))
		d.cvlEditConfigData = d.cvlEditConfigData[:len(d.cvlEditConfigData)-len(cvlOps)]
	} else {
		for i := 0; i < len(cvlOps); i++ {
			d.cvlEditConfigData[len(d.cvlEditConfigData)-1-i].VType = cmn.VALIDATE_NONE
		}
	}

doCVLExit:

	if glog.V(3) {
		glog.Info("doCVL: End: e: ", e)
	}

	return e
}

func (d *DB) doWrite(ts *TableSpec, op _txOp, k Key, val interface{}) error {
	var e error = nil
	var value Value

	key := k.Copy()

	if d.Opts.IsWriteDisabled {
		glog.Error("doWrite: Write to DB disabled")
		e = errors.New("Write to DB disabled during this operation")
		goto doWriteExit
	}

	if d.err != nil {
		e = d.err
		glog.Error("doWrite: DB in error: ", e)
		goto doWriteExit
	}

	if d.Opts.DBNo == ConfigDB && !d.Opts.IsSession && !d.configDBLocked {
		if e = ConfigDBTryLock(noSessionToken); e != nil {
			glog.Errorf("doWrite: ConfigDB possibly locked: %s", e)
			goto doWriteExit
		}
		d.configDBLocked = true
	}

	if d.Opts.IsSession && (d.Opts.TxCmdsLim != 0) &&
		(len(d.txCmds) >= d.Opts.TxCmdsLim) {

		glog.Infof("doWrite: TxCmdsLim exceeded %d >= %d", len(d.txCmds),
			d.Opts.TxCmdsLim)
		e = tlerr.TranslibDBTxCmdsLim{}
		goto doWriteExit
	}

	switch d.txState {
	case txStateNone:
		if glog.V(3) {
			glog.Info("doWrite: No Transaction.")
		}
	case txStateWatch:
		if glog.V(2) {
			glog.Info("doWrite: Change to txStateSet, txState: ", d.txState)
		}
		d.txState = txStateSet
	case txStateSet:
		if glog.V(5) {
			glog.Info("doWrite: Remain in txStateSet, txState: ", d.txState)
		}
	case txStateMultiExec:
		glog.Error("doWrite: Incorrect State, txState: ", d.txState)
		e = errors.New("Cannot issue {Set|Mod|Delete}Entry in txStateMultiExec")
	default:
		glog.Error("doWrite: Unknown, txState: ", d.txState)
		e = errors.New("Unknown State: " + string(rune(d.txState)))
	}

	if e != nil {
		goto doWriteExit
	}

	// No Transaction case. No CVL.
	if d.txState == txStateNone {

		glog.Info("doWrite: RedisCmd: ", d.Name(), ": ", getOperationName(op), " ", d.key2redis(ts, key), " ", getTableValuesInString(op, val))

		switch op {

		case txOpHMSet:
			value = Value{Field: make(map[string]string,
				len(val.(Value).Field))}
			vintf := make(map[string]interface{})
			for k, v := range val.(Value).Field {
				vintf[k] = v
			}
			e = d.client.HMSet(d.key2redis(ts, key), vintf).Err()

			if e != nil {
				glog.Error("doWrite: ", d.Name(), ": HMSet: ", key, " : ",
					value, " e: ", e)
			}

		case txOpHDel:
			fields := make([]string, 0, len(val.(Value).Field))
			for k := range val.(Value).Field {
				fields = append(fields, k)
			}

			e = d.client.HDel(d.key2redis(ts, key), fields...).Err()
			if e != nil {
				glog.Error("doWrite: ", d.Name(), ": HDel: ", key, " : ",
					fields, " e: ", e)
			}

		case txOpDel:
			e = d.client.Del(d.key2redis(ts, key)).Err()
			if e != nil {
				glog.Error("doWrite: ", d.Name(), ": Del: ", key, " : ", e)
			}

		default:
			glog.Error("doWrite: Unknown, op: ", op)
			e = errors.New("Unknown Op: " + string(rune(op)))
		}

		// No Transaction. Only update the config-timestamp, and
		// ignore the error, if any, since the actual operation succeeded.
		if d.Opts.DBNo == ConfigDB && e == nil {
			d.markConfigDBUpdated()
		}

		goto doWriteExit
	}

	// Transaction case.

	if _, ok := d.txTsEntryMap[ts.Name]; !ok {
		d.txTsEntryMap[ts.Name] = make(map[string]Value)
	}

	d.doTxSPsave(ts, key)

	switch op {
	case txOpHMSet, txOpHDel:
		value = val.(Value).Copy()
		entry := d.key2redis(ts, key)
		if _, ok := d.txTsEntryMap[ts.Name][entry]; !ok {
			var v map[string]string
			glog.Info("doWrite: RedisCmd: ", d.Name(), ": ", "HGETALL ", d.key2redis(ts, key))
			v, e = d.client.HGetAll(d.key2redis(ts, key)).Result()
			if len(v) != 0 {
				d.txTsEntryMap[ts.Name][entry] = Value{Field: v}
			} else {
				d.txTsEntryMap[ts.Name][entry] = Value{Field: make(map[string]string)}
			}
			d.doTxSPsaveHGetAll(ts, key, d.txTsEntryMap[ts.Name][entry])
		}
		if op == txOpHMSet {
			for k := range value.Field {
				d.txTsEntryMap[ts.Name][entry].Field[k] = value.Field[k]
			}
		} else {
			if _, ok := d.txTsEntryMap[ts.Name][entry]; ok {
				for k := range value.Field {
					delete(d.txTsEntryMap[ts.Name][entry].Field, k)
				}
			}
		}

	case txOpDel:
		entry := d.key2redis(ts, key)
		delete(d.txTsEntryMap[ts.Name], entry)
		d.txTsEntryMap[ts.Name][entry] = Value{Field: make(map[string]string)}

	default:
		glog.Error("doWrite: Unknown, op: ", op)
		e = errors.New("Unknown Op: " + string(rune(op)))
	}

	if e != nil {
		goto doWriteExit
	}

	d.txCmds = append(d.txCmds, _txCmd{
		ts:    ts,
		op:    op,
		key:   &key,
		value: &value,
	})
	d.stats.AllTables.TxCmdsLen = uint(len(d.txCmds))

	// Send Notification (if Candidate Config modified)
	if d.Opts.IsSession {
		op1 := op
		op2 := txOpNone
		if op1 == txOpHDel {
			// There could be a Delete Key as well if hash becomes empty
			entry := d.key2redis(ts, key)
			if v, ok := d.txTsEntryMap[ts.Name][entry]; !ok ||
				len(v.Field) == 0 {

				op2 = txOpDel
			}
		}
		d.sendSessionNotification(ts, &key, op1, op2)
	}

doWriteExit:

	if glog.V(3) {
		glog.Info("doWrite: End: e: ", e)
	}

	return e
}

// setEntry either Creates, or Sets an entry(row) in the table.
func (d *DB) setEntry(ts *TableSpec, key Key, value Value, isCreate bool) error {

	var e error = nil
	var valueComplement Value = Value{Field: make(map[string]string, len(value.Field))}
	var valueCurrent Value

	if glog.V(3) {
		glog.Info("setEntry: Begin: ", d.Name(), ": ts: ", ts, " key: ", key,
			" value: ", value, " isCreate: ", isCreate)
	}

	if len(value.Field) == 0 {
		if ts.NoDelete {
			glog.Info("setEntry: NoDelete flag is true, skipping deletion of the entry.")
		} else {
			glog.Info("setEntry: Mapping to DeleteEntry()")
			e = d.DeleteEntry(ts, key)
		}
		goto setEntryExit
	}

	if !isCreate {
		// Prepare the HDel list
		// Note: This is for compatibililty with PySWSSDK semantics.
		//       The CVL library will likely fail the SetEntry when
		//       the item exists.
		valueCurrent, e = d.GetEntry(ts, key)
		if e == nil {
			for k := range valueCurrent.Field {
				_, present := value.Field[k]
				if !present {
					valueComplement.Field[k] = string("")
				}
			}
		}
	}

	if !isCreate && e == nil {
		if glog.V(3) {
			glog.Info("setEntry: DoCVL for UPDATE")
		}
		if len(valueComplement.Field) == 0 {
			e = d.doCVL(ts, []cmn.CVLOperation{cmn.OP_UPDATE},
				key, []Value{value})
		} else {
			e = d.doCVL(ts, []cmn.CVLOperation{cmn.OP_UPDATE, cmn.OP_DELETE},
				key, []Value{value, valueComplement})
		}
	} else {
		if glog.V(3) {
			glog.Info("setEntry: DoCVL for CREATE")
		}
		e = d.doCVL(ts, []cmn.CVLOperation{cmn.OP_CREATE}, key, []Value{value})
	}

	if e != nil {
		goto setEntryExit
	}

	e = d.doWrite(ts, txOpHMSet, key, value)

	if (e == nil) && (len(valueComplement.Field) != 0) {
		if glog.V(3) {
			glog.Info("setEntry: DoCVL for HDEL (post-POC)")
		}
		e = d.doWrite(ts, txOpHDel, key, valueComplement)
	}

setEntryExit:
	return e
}

// CreateEntry creates an entry(row) in the table.
func (d *DB) CreateEntry(ts *TableSpec, key Key, value Value) error {

	if !d.IsOpen() {
		return ConnectionClosed
	}

	return d.setEntry(ts, key, value, true)
}

// SetEntry sets an entry(row) in the table.
func (d *DB) SetEntry(ts *TableSpec, key Key, value Value) error {
	if !d.IsOpen() {
		return ConnectionClosed
	}

	return d.setEntry(ts, key, value, false)
}

func (d *DB) Publish(channel string, message interface{}) error {
	if !d.IsOpen() {
		return ConnectionClosed
	}

	e := d.client.Publish(channel, message).Err()
	return e
}

func (d *DB) RunScript(script *redis.Script, keys []string, args ...interface{}) *redis.Cmd {
	if !d.IsOpen() {
		return nil
	}

	if d.Opts.DBNo == ConfigDB {
		glog.Info("RunScript: Not supported for ConfigDB")
		return nil
	}

	return script.Run(d.client, keys, args...)
}

// DeleteEntry deletes an entry(row) in the table.
func (d *DB) DeleteEntry(ts *TableSpec, key Key) error {

	var e error = nil
	if glog.V(3) {
		glog.Info("DeleteEntry: Begin: ", d.Name(), ": ts: ", ts, " key: ", key)
	}

	if !d.IsOpen() {
		return ConnectionClosed
	}

	if glog.V(3) {
		glog.Info("DeleteEntry: DoCVL for DELETE")
	}
	e = d.doCVL(ts, []cmn.CVLOperation{cmn.OP_DELETE}, key, []Value{Value{}})

	if e == nil {
		e = d.doWrite(ts, txOpDel, key, nil)
	}

	return e
}

// ModEntry modifies an entry(row) in the table.
func (d *DB) ModEntry(ts *TableSpec, key Key, value Value) error {

	var e error = nil

	if glog.V(3) {
		glog.Info("ModEntry: Begin: ", d.Name(), ": ts: ", ts, " key: ", key,
			" value: ", value)
	}

	if !d.IsOpen() {
		return ConnectionClosed
	}

	if len(value.Field) == 0 {
		if ts.NoDelete {
			if glog.V(3) {
				glog.Info("ModEntry: NoDelete flag is true, skipping deletion of the entry.")
			}
		} else {
			if glog.V(3) {
				glog.Info("ModEntry: Mapping to DeleteEntry()")
			}
			e = d.DeleteEntry(ts, key)
		}
		goto ModEntryExit
	}

	if glog.V(3) {
		glog.Info("ModEntry: DoCVL for UPDATE")
	}
	e = d.doCVL(ts, []cmn.CVLOperation{cmn.OP_UPDATE}, key, []Value{value})

	if e == nil {
		e = d.doWrite(ts, txOpHMSet, key, value)
	}

ModEntryExit:

	return e
}

// DeleteEntryFields deletes some fields/columns in an entry(row) in the table.
func (d *DB) DeleteEntryFields(ts *TableSpec, key Key, value Value) error {

	if glog.V(3) {
		glog.Info("DeleteEntryFields: Begin: ", d.Name(), ": ts: ", ts,
			" key: ", key, " value: ", value)
	}

	if !d.IsOpen() {
		return ConnectionClosed
	}

	if glog.V(3) {
		glog.Info("DeleteEntryFields: DoCVL for HDEL (post-POC)")
	}

	if glog.V(3) {
		glog.Info("DeleteEntryFields: DoCVL for HDEL")
	}

	e := d.doCVL(ts, []cmn.CVLOperation{cmn.OP_DELETE}, key, []Value{value})

	if e == nil {
		e = d.doWrite(ts, txOpHDel, key, value)
	}

	return e
}

// DeleteTable deletes the entire table.
func (d *DB) DeleteTable(ts *TableSpec) error {
	if glog.V(3) {
		glog.Info("DeleteTable: Begin: ", d.Name(), ": ts: ", ts)
	}

	if !d.IsOpen() {
		return ConnectionClosed
	}

	// This can be done via a LUA script as well. For now do this. TBD
	// Read Keys
	keys, e := d.GetKeys(ts)
	if e != nil {
		glog.Error("DeleteTable: GetKeys: " + e.Error())
		goto DeleteTableExit
	}

	// For each key in Keys
	// 	Delete the entry
	for i := 0; i < len(keys); i++ {
		// Don't define/declare a nested scope ``e''
		e = d.DeleteEntry(ts, keys[i])
		if e != nil {
			glog.Warning("DeleteTable: DeleteEntry: " + e.Error())
			break
		}
	}
DeleteTableExit:
	if glog.V(3) {
		glog.Info("DeleteTable: End: ")
	}
	return e
}

//////////////////////////////////////////////////////////////////////////
// The Transaction API for translib infra
//////////////////////////////////////////////////////////////////////////

// WatchKeys is array of (TableSpec, Key) tuples to be watched in a Transaction.
type WatchKeys struct {
	Ts  *TableSpec
	Key *Key
}

func (w WatchKeys) String() string {
	return fmt.Sprintf("{ Ts: %v, Key: %v }", w.Ts, w.Key)
}

// Tables2TableSpecs - Convenience function to make TableSpecs from strings.
// This only works on Tables having key components without TableSeparator
// as part of the key.
func Tables2TableSpecs(tables []string) []*TableSpec {
	var tss []*TableSpec

	tss = make([]*TableSpec, 0, len(tables))

	for i := 0; i < len(tables); i++ {
		tss = append(tss, &(TableSpec{Name: tables[i]}))
	}

	return tss
}

// StartSessTx CommitSessTx AbortSessTx
// Originally (roughly 4.0.x and before) we have StartTx(), CommitTx(), and
// AbortTx() which represented the API for invoking redis CAS (Check-And-Set)
// Transactions. With the attempt at Config Sessions a.k.a. Two-Phase Commit,
// these APIs may have to be re-arranged, because a few of the Action Callback
// handlers in the transformer invoke these API directly. Ideally all
// db.[Start|Commit|Abort]Tx need to be rewritten to cs.[Start|Commit|Abort]Tx.
// But, that would mean changing App code.
func (d *DB) StartSessTx(w []WatchKeys, tss []*TableSpec) error {
	glog.V(3).Info("StartSessTx:")
	return d.startTx(w, tss)
}

func (d *DB) CommitSessTx() error {
	glog.V(3).Info("CommitSessTx:")
	return d.commitTx()
}

func (d *DB) AbortSessTx() error {
	glog.V(3).Info("AbortSessTx:")
	return d.abortTx()
}

func (d *DB) StartTx(w []WatchKeys, tss []*TableSpec) error {
	if d.Opts.IsSession {
		return d.DeclareSP()
	}
	return d.startTx(w, tss)
}

func (d *DB) CommitTx() error {
	defer d.clearCVLHint("")
	if d.Opts.IsSession {
		return d.ReleaseSP()
	}
	return d.commitTx()
}

func (d *DB) AbortTx() error {
	if d.Opts.IsSession {
		// Rollback creates the CVL Session again -- with only the
		// pre-DeclareSP() CVL Hints.
		return d.Rollback2SP()
	}
	d.clearCVLHint("")
	return d.abortTx()
}

// startTx method is used by infra to start a check-and-set Transaction.
func (d *DB) startTx(w []WatchKeys, tss []*TableSpec) error {

	if d.Opts.DBNo != ConfigDB {
		e := tlerr.TranslibDBNotSupported{
			Description: "Transactions are only supported on ConfigDB"}
		glog.Errorf("StartTx: %v", e)
		return e
	}

	if glog.V(3) {
		glog.Info("StartTx: Begin: w: ", w, " tss: ", tss)
	}

	d.txTsEntryMap = make(map[string]map[string]Value)
	d.txTsEntryHGetAll = make(map[string]map[string]Value)

	var e error = nil

	//Start CVL session
	if d.cv, e = d.NewValidationSession(); e != nil {
		goto StartTxExit
	}

	// Validate State
	if d.txState != txStateNone {
		glog.Error("StartTx: Incorrect State, txState: ", d.txState)
		e = errors.New("Transaction already in progress")
		goto StartTxExit
	}

	// Hints which were saved before the Session was opened.
	for k, v := range d.cvlHintsB4Open {
		if e = d.StoreCVLHint(k, v); e != nil {
			glog.Errorf("StartTx: k: %v, v: %v, error: %v", k, v, e)
			goto StartTxExit
		}
	}
	d.cvlHintsB4Open = nil

	e = d.performWatch(w, tss)

StartTxExit:

	if glog.V(3) {
		glog.Info("StartTx: End: e: ", e)
	}
	return e
}

func (d *DB) AppendWatchTx(w []WatchKeys, tss []*TableSpec) error {
	if glog.V(3) {
		glog.Info("AppendWatchTx: Begin: w: ", w, " tss: ", tss)
	}

	var e error = nil

	// Validate State
	if d.txState == txStateNone {
		glog.Error("AppendWatchTx: Incorrect State, txState: ", d.txState)
		e = errors.New("Transaction has not started")
		goto AppendWatchTxExit
	}

	e = d.performWatch(w, tss)

AppendWatchTxExit:

	if glog.V(3) {
		glog.Info("AppendWatchTx: End: e: ", e)
	}
	return e
}

func (d *DB) performWatch(w []WatchKeys, tss []*TableSpec) error {
	var e error
	var first_e error
	var args []interface{}

	// For each watchkey
	//   If a pattern, Get the keys, appending results to Cmd args.
	//   Else append keys to the Cmd args
	//   Note: (LUA scripts do not support WATCH)

	args = make([]interface{}, 0, len(w)+len(tss)+1)
	args = append(args, "WATCH")
	for i := 0; i < len(w); i++ {

		redisKey := d.key2redis(w[i].Ts, *(w[i].Key))

		if !strings.Contains(redisKey, "*") {
			args = append(args, redisKey)
			continue
		}

		glog.Info("performWatch: RedisCmd: ", d.Name(), ": ", "KEYS ", redisKey)
		redisKeys, e := d.client.Keys(redisKey).Result()
		if e != nil {
			glog.Warning("performWatch: Keys: " + e.Error())
			if first_e == nil {
				first_e = e
			}
			continue
		}
		for j := 0; j < len(redisKeys); j++ {
			args = append(args, d.redis2key(w[i].Ts, redisKeys[j]))
		}
	}

	// for each TS, append to args the CONFIG_DB_UPDATED_<TABLENAME> key

	for i := 0; i < len(tss); i++ {
		args = append(args, d.ts2redisUpdated(tss[i]))
	}

	if len(args) == 1 {
		glog.Warning("performWatch: Empty WatchKeys. Skipping WATCH")
		goto SkipWatch
	}

	// Issue the WATCH
	glog.Info("performWatch: Do: ", args)
	_, e = d.client.Do(args...).Result()

	if e != nil {
		glog.Warning("performWatch: Do: WATCH ", args, " e: ", e.Error())
		if first_e == nil {
			first_e = e
		}
	}

SkipWatch:

	// Switch State
	d.txState = txStateWatch

	return first_e
}

// CommitTx method is used by infra to commit a check-and-set Transaction.
func (d *DB) commitTx() error {
	if glog.V(3) {
		glog.Info("CommitTx: Begin:")
	}

	var e error = nil
	var tsmap map[TableSpec]bool = make(map[TableSpec]bool, len(d.txCmds)) // UpperBound

	// Validate State
	switch d.txState {
	case txStateNone:
		glog.Error("CommitTx: No WATCH done, txState: ", d.txState)
		e = errors.New("StartTx() not done. No Transaction active.")
	case txStateWatch:
		if glog.V(1) {
			glog.Info("CommitTx: No SET|DEL done, txState: ", d.txState)
		}
	case txStateSet:
		break
	case txStateMultiExec:
		glog.Error("CommitTx: Incorrect State, txState: ", d.txState)
		e = errors.New("Cannot issue MULTI in txStateMultiExec")
	default:
		glog.Error("CommitTx: Unknown, txState: ", d.txState)
		e = errors.New("Unknown State: " + string(rune(d.txState)))
	}

	if d.err != nil {
		e = d.err
		glog.Error("CommitTx: DB in error: ", e)
	}

	if e != nil {
		goto CommitTxExit
	}

	// Issue MULTI
	glog.Info("CommitTx: Do: MULTI")
	_, e = d.client.Do("MULTI").Result()

	if e != nil {
		glog.Warning("CommitTx: Do: MULTI e: ", e.Error())
		goto CommitTxExit
	}

	// For each cmd in txCmds
	//   Invoke it
	for i := 0; i < len(d.txCmds); i++ {

		var args []interface{}

		redisKey := d.key2redis(d.txCmds[i].ts, *(d.txCmds[i].key))

		// Add TS to the map of watchTables
		tsmap[*(d.txCmds[i].ts)] = true

		switch d.txCmds[i].op {

		case txOpHMSet:

			args = make([]interface{}, 0, len(d.txCmds[i].value.Field)*2+2)
			args = append(args, "HMSET", redisKey)

			for k, v := range d.txCmds[i].value.Field {
				args = append(args, k, v)
			}

			_, e = d.client.Do(args...).Result()

		case txOpHDel:

			args = make([]interface{}, 0, len(d.txCmds[i].value.Field)+2)
			args = append(args, "HDEL", redisKey)

			for k := range d.txCmds[i].value.Field {
				args = append(args, k)
			}

			_, e = d.client.Do(args...).Result()

		case txOpDel:

			args = make([]interface{}, 0, 2)
			args = append(args, "DEL", redisKey)

			_, e = d.client.Do(args...).Result()

		default:
			glog.Error("CommitTx: Unknown, op: ", d.txCmds[i].op)
			e = errors.New("Unknown Op: " + string(rune(d.txCmds[i].op)))
		}

		glog.Info("CommitTx: RedisCmd: ", d.Name(), ": ", args)

		if e != nil {
			glog.Warning("CommitTx: Do: ", args, " e: ", e.Error())
			break
		}
	}

	if e != nil {
		goto CommitTxExit
	}

	// Flag the Tables as updated.
	for ts := range tsmap {
		if glog.V(4) {
			glog.Info("CommitTx: Do: SET ", d.ts2redisUpdated(&ts), " 1")
		}
		_, e = d.client.Do("SET", d.ts2redisUpdated(&ts), "1").Result()
		if e != nil {
			glog.Warning("CommitTx: Do: SET ",
				d.ts2redisUpdated(&ts), " 1: e: ",
				e.Error())
			break
		}
	}

	if e != nil {
		goto CommitTxExit
	}

	if e = d.markConfigDBUpdated(); e != nil {
		goto CommitTxExit
	}

	// Issue EXEC
	glog.Info("CommitTx: Do: EXEC")
	_, e = d.client.Do("EXEC").Result()

	if e != nil {
		glog.Warning("CommitTx: Do: EXEC e: ", e.Error())
		e = tlerr.TranslibTransactionFail{}
	}

CommitTxExit:
	// Switch State, Clear Command list
	d.txState = txStateNone
	d.txCmds = d.txCmds[:0]
	d.cvlEditConfigData = d.cvlEditConfigData[:0]
	d.txTsEntryMap = make(map[string]map[string]Value)
	d.txTsEntryHGetAll = make(map[string]map[string]Value)

	//Close CVL session
	if d.cv != nil {
		if ret := cvl.ValidationSessClose(d.cv); ret != cvl.CVL_SUCCESS {
			glog.Error("CommitTx: End: Error in closing CVL session: ret: ",
				cvl.GetErrorString(ret))
		}
		d.cv = nil
	}

	if glog.V(3) {
		glog.Info("CommitTx: End: e: ", e)
	}
	return e
}

// AbortTx method is used by infra to abort a check-and-set Transaction.
func (d *DB) abortTx() error {
	if glog.V(3) {
		glog.Info("AbortTx: Begin:")
	}

	var e error = nil

	// Validate State
	switch d.txState {
	case txStateNone:
		glog.Error("AbortTx: No WATCH done, txState: ", d.txState)
		e = errors.New("StartTx() not done. No Transaction active.")
	case txStateWatch:
		if glog.V(1) {
			glog.Info("AbortTx: No SET|DEL done, txState: ", d.txState)
		}
	case txStateSet:
		break
	case txStateMultiExec:
		glog.Error("AbortTx: Incorrect State, txState: ", d.txState)
		e = errors.New("Cannot issue UNWATCH in txStateMultiExec")
	default:
		glog.Error("AbortTx: Unknown, txState: ", d.txState)
		e = errors.New("Unknown State: " + string(rune(d.txState)))
	}

	if e != nil {
		goto AbortTxExit
	}

	// Issue UNWATCH
	glog.Info("AbortTx: Do: UNWATCH")
	_, e = d.client.Do("UNWATCH").Result()

	if e != nil {
		glog.Warning("AbortTx: Do: UNWATCH e: ", e.Error())
	}

AbortTxExit:
	// Switch State, Clear Command list
	d.txState = txStateNone
	d.txCmds = d.txCmds[:0]
	d.cvlEditConfigData = d.cvlEditConfigData[:0]
	d.txTsEntryMap = make(map[string]map[string]Value)
	d.txTsEntryHGetAll = make(map[string]map[string]Value)

	//Close CVL session
	if d.cv != nil {
		if ret := cvl.ValidationSessClose(d.cv); ret != cvl.CVL_SUCCESS {
			glog.Error("AbortTx: End: Error in closing CVL session: ret: ",
				cvl.GetErrorString(ret))
		}
		d.cv = nil
	}

	if glog.V(3) {
		glog.Info("AbortTx: End: e: ", e)
	}
	return e
}

func (d *DB) markConfigDBUpdated() error {
	if glog.V(4) {
		glog.Info("markConfigDBUpdated: Do: SET ",
			d.ts2redisUpdated(&TableSpec{Name: "*"}), " 1")
	}
	_, e := d.client.Do("SET", d.ts2redisUpdated(&TableSpec{Name: "*"}),
		strconv.FormatInt(time.Now().UnixNano(), 10)).Result()
	if e != nil {
		glog.Warning("markConfigDBUpdated: Do: SET ",
			"CONFIG_DB_UPDATED", " 1: e: ", e.Error())
	}
	return e
}

func getOperationName(op _txOp) string {
	switch op {
	case txOpNone:
		return "No Operation"
	case txOpHMSet:
		return "HMSET"
	case txOpHDel:
		return "HDEL"
	case txOpDel:
		return "DEL"
	}
	return ""
}

func getTableValuesInString(op _txOp, val interface{}) string {
	var values string

	switch op {

	case txOpHMSet:
		for k, v := range val.(Value).Field {
			values += k + " " + v + " "
		}
	case txOpHDel:
		for k := range val.(Value).Field {
			values += k + " "
		}
	}

	return values
}

func (d *DB) Name() string {
	return (getDBInstName(d.Opts.DBNo))
}

// GetEntries retrieves the entries from the table for the given keys
// using redis pipelining, if the key is not present in the cache.
// returns slice of value and error; Note: error slice will be nil,
// if there is no error occurred for any of the given keys.
func (d *DB) GetEntries(ts *TableSpec, keys []Key) ([]Value, []error) {
	if (d == nil) || (d.client == nil) {
		values := make([]Value, len(keys))
		errors := make([]error, len(keys))
		for i := range errors {
			errors[i] = tlerr.TranslibDBConnectionReset{}
		}

		return values, errors
	}

	return d.getEntries(ts, keys, false)
}

func (d *DB) getEntries(ts *TableSpec, keys []Key, forceReadDB bool) ([]Value, []error) {

	if glog.V(3) {
		glog.Info("GetEntries: Begin: ", d.Name(), ": ts: ", ts, " keys: ", keys)
	}

	var now time.Time
	if d.dbStatsConfig.TimeStats {
		now = time.Now()
	}

	var values = make([]Value, len(keys))
	var errors []error

	var dur time.Duration
	var stats Stats
	var cacheHit bool
	var txCacheHit bool
	var cacheChk bool
	var tblExist bool

	var tbl Table

	if (d.dbCacheConfig.PerConnection &&
		d.dbCacheConfig.isCacheTable(ts.Name)) ||
		(d.Opts.IsOnChangeEnabled && d.onCReg.isCacheTable(ts.Name)) {
		cacheChk = true
		tbl, tblExist = d.cache.Tables[ts.Name]
	}

	if d.dbStatsConfig.TableStats {
		stats = d.stats.Tables[ts.Name]
	} else {
		stats = d.stats.AllTables
	}

	// to keep the order of the input keys
	var keyIdxs []int
	var dbKeys []string

	for idx, key := range keys {
		cacheHit = false
		txCacheHit = false
		entry := d.key2redis(ts, key)

		if valueTx, exist := d.txTsEntryMap[ts.Name][entry]; !exist {
			if cacheChk && !forceReadDB {
				if value, ok := tbl.entry[entry]; ok {
					values[idx] = value.Copy()
					cacheHit = true
				}
			}
		} else {
			values[idx] = valueTx.Copy()
			txCacheHit = true
			if len(valueTx.Field) == 0 {
				keyErr := tlerr.TranslibRedisClientEntryNotExist{Entry: entry}
				setError(keyErr, idx, &errors, len(keys))
			}
		}

		if !cacheHit && !txCacheHit {
			keyIdxs = append(keyIdxs, idx)
			dbKeys = append(dbKeys, entry)
		}

		if cacheHit {
			stats.GetEntryCacheHits++
		}
	}

	if len(dbKeys) > 0 {
		// get the values for the keys using redis pipeline
		entryList, err := d.getMultiEntry(ts, dbKeys)
		if err != nil {
			glog.Error("GetEntries: ", d.Name(),
				": error in getMultiEntry(", ts.Name, "): ", err.Error())
			if errors == nil {
				errors = make([]error, len(keys))
			}
			for i, dbKey := range dbKeys {
				keyIdx := keyIdxs[i]
				values[keyIdx] = Value{}
				errors[keyIdx] = tlerr.TranslibRedisClientEntryNotExist{Entry: dbKey}
			}
		} else {
			// iterate the keys to fill the value and error slice
			for i, dbKey := range dbKeys {
				keyIdx := keyIdxs[i]
				v := entryList[i]

				if v == nil {
					values[keyIdx] = Value{}
					keyErr := tlerr.TranslibRedisClientEntryNotExist{Entry: dbKey}
					setError(keyErr, keyIdx, &errors, len(keys))
					continue
				}

				dbValue := Value{}
				res, e := v.Result()
				if e != nil {
					values[keyIdx] = dbValue
					setError(e, keyIdx, &errors, len(keys))
					glog.Warningf("GetEntries: %s: error %s; for the key %s",
						d.Name(), e.Error(), dbKey)
				} else {
					dbValue.Field = res
					values[keyIdx] = dbValue
				}

				if len(dbValue.Field) != 0 {
					if cacheChk {
						if !tblExist {
							d.cache.Tables[ts.Name] = Table{
								ts:       ts,
								entry:    make(map[string]Value, InitialTableEntryCount),
								complete: false,
								patterns: make(map[string][]Key, InitialTablePatternCount),
								db:       d,
							}
							tblExist = true
						}
						d.cache.Tables[ts.Name].entry[dbKey] = dbValue.Copy()
					}
				} else if e == nil {
					if glog.V(4) {
						glog.Info("GetEntries: pipe.HGetAll(): empty map for the key: ", dbKey)
					}
					keyErr := tlerr.TranslibRedisClientEntryNotExist{Entry: dbKey}
					setError(keyErr, keyIdx, &errors, len(keys))
				}
			}
		}
	}

	stats.GetEntryHits = stats.GetEntryHits + uint(len(keys))
	stats.Hits++
	stats.GetEntriesHits++

	if d.dbStatsConfig.TimeStats {
		dur = time.Since(now)

		if dur > stats.Peak {
			stats.Peak = dur
		}
		stats.Time += dur

		if dur > stats.GetEntriesPeak {
			stats.GetEntriesPeak = dur
		}
		stats.GetEntriesTime += dur
	}

	if d.dbStatsConfig.TableStats {
		d.stats.Tables[ts.Name] = stats
	} else {
		d.stats.AllTables = stats
	}

	if glog.V(3) {
		glog.Info("GetEntries: End: ", "ts: ", ts, "values: ", values, " errors: ", errors)
	}

	return values, errors
}

func setError(e error, idx int, errors *[]error, numKeys int) {
	if *errors == nil {
		*errors = make([]error, numKeys)
	}
	(*errors)[idx] = e
}

// getMultiEntry retrieves the entries of the given keys using "redis pipeline".
func (d *DB) getMultiEntry(ts *TableSpec, keys []string) ([]*redis.StringStringMapCmd, error) {

	if glog.V(3) {
		glog.Info("getMultiEntry: Begin: ts: ", ts)
	}

	var results = make([]*redis.StringStringMapCmd, len(keys))

	pipe := d.client.Pipeline()
	defer pipe.Close()

	if glog.V(3) {
		glog.Info("getMultiEntry: RedisCmd: ", d.Name(), ": ", "pipe.HGetAll for the ", keys)
	}

	for i, key := range keys {
		results[i] = pipe.HGetAll(key)
	}

	if glog.V(3) {
		glog.Info("getMultiEntry: RedisCmd: ", d.Name(), ": ", "pipe.Exec")
	}
	_, err := pipe.Exec()

	if glog.V(3) {
		glog.Info("getMultiEntry: End: ts: ", ts, "results: ", results, "err: ", err)
	}

	return results, err
}
