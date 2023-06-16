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

package translib

import (
	"fmt"
	"strings"
	"sync"

	"github.com/Azure/sonic-mgmt-common/translib/db"
	"github.com/Azure/sonic-mgmt-common/translib/internal/apis"
	"github.com/Azure/sonic-mgmt-common/translib/path"
	"github.com/Azure/sonic-mgmt-common/translib/tlerr"
	"github.com/Workiva/go-datastructures/queue"
	log "github.com/golang/glog"
	"github.com/openconfig/gnmi/proto/gnmi"
	"github.com/openconfig/ygot/ygot"
)

// SubscribeRequest holds the request data for Subscribe and Stream APIs.
type SubscribeRequest struct {
	Paths         []string
	Q             *queue.PriorityQueue
	Stop          chan struct{}
	User          UserRoles
	AuthEnabled   bool
	ClientVersion Version
	Session       *SubscribeSession
}

type SubscribeResponse struct {
	Path         string
	Update       ygot.ValidatedGoStruct // updated values
	Delete       []string               // deleted paths - relative to Path
	Timestamp    int64
	SyncComplete bool
	IsTerminated bool
}

type IsSubscribeRequest struct {
	Paths         []IsSubscribePath
	User          UserRoles
	AuthEnabled   bool
	ClientVersion Version
	Session       *SubscribeSession
}

type IsSubscribePath struct {
	ID   uint32           // Path ID for correlating with IsSubscribeResponse
	Path string           // Subscribe path
	Mode NotificationType // Requested subscribe mode
}

type IsSubscribeResponse struct {
	ID                  uint32 // Path ID
	Path                string
	IsSubPath           bool // Subpath of the requested path
	IsOnChangeSupported bool
	IsWildcardSupported bool // true if wildcard keys are supported in the path
	MinInterval         int
	Err                 error
	PreferredType       NotificationType
}

type NotificationType int

const (
	TargetDefined NotificationType = iota
	Sample
	OnChange
)

func (nt NotificationType) String() string {
	switch nt {
	case TargetDefined:
		return "TargetDefined"
	case Sample:
		return "Sample"
	case OnChange:
		return "OnChange"
	default:
		return fmt.Sprintf("NotificationType(%d)", nt)
	}
}

const (
	// MinSubscribeInterval is the lowest sample subscription interval supported by the system.
	// Value is in seconds. This is also used as the default value if apps do not provide one
	// or returns a lower value.
	MinSubscribeInterval = apis.SAMPLE_NOTIFICATION_MIN_INTERVAL
)

// Subscribe mutex for all the subscribe operations on the maps to be thread safe
var sMutex = &sync.Mutex{}

var defunct = tlerr.TranslibDBNotInit{}

// notificationInfo flags
const (
	niLeafPath        Bits = 1 << iota // Path represents a leaf node
	niWildcardPath                     // Path has wildcard keys
	niWildcardSubpath                  // Some dbFldYgPathInfo.rltvPath has wildcard keys
	niDeleteAsUpdate                   // Dont blindly treat db key delete as path delete
	niPartial                          // Db key represents only a partial subtree of the path
	niKeyFields                        // some db fields mapped to yang keys
	niDynamic                          // Path mapped to dynamic table (both db and non-db data)
)

type notificationInfo struct {
	flags       Bits
	table       *db.TableSpec
	key         *db.Key
	dbno        db.DBNum
	fields      []*dbFldYgPathInfo // map of db field to yang fields map
	path        *gnmi.Path         // Path to which the db key maps to
	handler     apis.ProcessOnChange
	appInfo     *appInfo
	sInfo       *subscribeInfo
	opaque      interface{} // App specific opaque data
	fldScanPatt string      // scan pattern to match the field names
	keyGroup    []int       // key component indices for the key group (for leaf-list)
}

// subscribeInfo holds the client data of Subscribe or Stream request.
// Should not be reused across multiple API calls.
type subscribeInfo struct {
	id       string // Subscribe request id
	syncDone bool   // Sync message has been sent
	termDone bool   // Terminate message has been sent
	q        *queue.PriorityQueue
	stop     chan struct{}
	sDBs     []*db.DB         //Subscription DB should be used only for keyspace notification unsubscription
	dbs      [db.MaxDB]*db.DB //used to perform get operations
}

// notificationGroup is the grouping of notificationInfo by the key pattern.
type notificationGroup struct {
	nInfos map[string][]*notificationInfo
	//TODO move dbno, TS, key from notificationInfo to here
}

// subscribeCounter counts number of Subscribe calls.
var subscribeCounter Counter

var stopMap map[chan struct{}]*subscribeInfo
var cleanupMap map[*db.DB]*subscribeInfo

func init() {
	stopMap = make(map[chan struct{}]*subscribeInfo)
	cleanupMap = make(map[*db.DB]*subscribeInfo)
}

// Subscribe - Subscribes to the paths requested and sends notifications when the data changes in DB
func Subscribe(req SubscribeRequest) error {
	sid := subscribeContextId(req.Session)
	paths := req.Paths
	log.Infof("[%v] Subscribe: paths = %v", sid, paths)

	dbs, err := getAllDbs(withWriteDisable, withOnChange)
	if err != nil {
		return err
	}

	sInfo := &subscribeInfo{
		id:   sid,
		q:    req.Q,
		stop: req.Stop,
		dbs:  dbs,
	}

	sCtx := subscribeContext{
		id:      sid,
		sInfo:   sInfo,
		dbs:     dbs,
		version: req.ClientVersion,
		session: req.Session,
		recurse: true,
	}

	for _, path := range paths {
		err = sCtx.translateAndAddPath(path, OnChange)
		if err != nil {
			closeAllDbs(dbs[:])
			return err
		}
	}

	// Start db subscription and exit. DB objects will be
	// closed automatically when the subscription ends.
	err = sCtx.startSubscribe()

	return err
}

// Stream function streams the value for requested paths through a queue.
// Unlike Get, this function can return smaller chunks of response separately.
// Individual chunks are packed in a SubscribeResponse object and pushed to the req.Q.
// Pushes a SubscribeResponse with SyncComplete=true after data are pushed.
// Function will block until all values are returned. This can be used for
// handling "Sample" subscriptions (NotificationType.Sample).
// Client should be authorized to perform "subscribe" operation.
func Stream(req SubscribeRequest) error {
	sid := subscribeContextId(req.Session)
	log.Infof("[%v] Stream: paths = %v", sid, req.Paths)

	dbs, err := getAllDbs(withWriteDisable)
	if err != nil {
		return err
	}

	defer closeAllDbs(dbs[:])

	sc := subscribeContext{
		id:      sid,
		dbs:     dbs,
		version: req.ClientVersion,
		session: req.Session,
	}

	for _, path := range req.Paths {
		err := sc.translateAndAddPath(path, Sample)
		if err != nil {
			return err
		}
	}

	sInfo := &subscribeInfo{
		id:  sid,
		q:   req.Q,
		dbs: dbs,
	}

	for _, nInfo := range sc.tgtInfos {
		err = sendInitialUpdate(sInfo, nInfo)
		if err != nil {
			return err
		}
	}

	// Push a SyncComplete message at the end
	sInfo.syncDone = true
	sendSyncNotification(sInfo, false)
	return nil
}

// IsSubscribeSupported - Check if subscribe is supported on the given paths
func IsSubscribeSupported(req IsSubscribeRequest) ([]*IsSubscribeResponse, error) {
	reqID := subscribeContextId(req.Session)
	paths := req.Paths
	resp := make([]*IsSubscribeResponse, len(paths))

	for i := range resp {
		resp[i] = newIsSubscribeResponse(paths[i].ID, paths[i].Path)
	}

	log.Infof("[%v] IsSubscribeSupported: paths = %v", reqID, paths)

	dbs, err := getAllDbs(withWriteDisable)
	if err != nil {
		return resp, err
	}

	defer closeAllDbs(dbs[:])

	sc := subscribeContext{
		id:      reqID,
		dbs:     dbs,
		version: req.ClientVersion,
		session: req.Session,
		recurse: true,
	}

	for i, p := range paths {
		trInfo, errApp := sc.translateSubscribe(p.Path, p.Mode)
		if errApp != nil {
			resp[i].Err = errApp
			err = errApp
			continue
		}

		// Split target_defined request into separate on_change and sample
		// sub-requests if required.
		if p.Mode == TargetDefined {
			for _, xInfo := range trInfo.segregateSampleSubpaths() {
				xr := newIsSubscribeResponse(p.ID, xInfo.path)
				xr.IsSubPath = true
				resp = append(resp, xr)
				collectNotificationPreferences(xInfo.response.ntfAppInfoTrgt, xr)
				collectNotificationPreferences(xInfo.response.ntfAppInfoTrgtChlds, xr)
				xInfo.saveToSession()
			}
		}

		r := resp[i]
		collectNotificationPreferences(trInfo.response.ntfAppInfoTrgt, r)
		collectNotificationPreferences(trInfo.response.ntfAppInfoTrgtChlds, r)
		trInfo.saveToSession()
	}

	log.Infof("[%v] IsSubscribeSupported: returning %d IsSubscribeResponse; err=%v", reqID, len(resp), err)
	if log.V(5) {
		for i, r := range resp {
			log.Infof("[%v] IsSubscribeResponse[%d]: path=%s, onChg=%v, pref=%v, minInt=%d, err=%v",
				reqID, i, r.Path, r.IsOnChangeSupported, r.PreferredType, r.MinInterval, r.Err)
		}
	}

	return resp, err
}

func newIsSubscribeResponse(id uint32, path string) *IsSubscribeResponse {
	return &IsSubscribeResponse{
		ID:                  id,
		Path:                path,
		IsOnChangeSupported: true,
		IsWildcardSupported: true,
		MinInterval:         MinSubscribeInterval,
		PreferredType:       OnChange,
	}
}

// collectNotificationPreferences computes overall notification preferences (is on-change
// supported, min sample interval, preferred mode etc) by combining individual table preferences
// from the notificationAppInfo array. Writes them to the IsSubscribeResponse object 'resp'.
func collectNotificationPreferences(nAppInfos []*notificationAppInfo, resp *IsSubscribeResponse) {
	if len(nAppInfos) == 0 {
		return
	}

	for _, nInfo := range nAppInfos {
		if !nInfo.isOnChangeSupported {
			resp.IsOnChangeSupported = false
		}
		if nInfo.isNonDB() {
			resp.IsWildcardSupported = false
			resp.IsOnChangeSupported = false
			resp.PreferredType = Sample
		}
		if nInfo.pType == Sample {
			resp.PreferredType = Sample
		}
		if nInfo.mInterval > resp.MinInterval {
			resp.MinInterval = nInfo.mInterval
		}
	}
}

func startDBSubscribe(opt db.Options, nGroups map[db.TableSpec]*notificationGroup, sInfo *subscribeInfo) error {
	var sKeyList []*db.SKey
	d := sInfo.dbs[int(opt.DBNo)]

	for tSpec, nGroup := range nGroups {
		skeys := nGroup.toSKeys()
		if len(skeys) == 0 {
			continue // should not happen
		}

		if log.V(2) {
			log.Infof("[%v] nGroup=%p:%v", sInfo.id, nGroup, nGroup.toString())
		}

		sKeyList = append(sKeyList, skeys...)

		d.RegisterTableForOnChangeCaching(&tSpec)
	}

	sDB, err := db.SubscribeDB(opt, sKeyList, notificationHandler)

	if err == nil {
		sInfo.sDBs = append(sInfo.sDBs, sDB)
		cleanupMap[sDB] = sInfo
	}

	return err
}

type subscribeContext struct {
	id      string // context id
	dbs     [db.MaxDB]*db.DB
	version Version
	session *SubscribeSession
	sInfo   *subscribeInfo
	recurse bool // collect notificationInfo for child paths too

	dbNInfos map[db.DBNum]map[db.TableSpec]*notificationGroup
	tgtInfos []*notificationInfo
}

func (sc *subscribeContext) addToNGroup(nInfo *notificationInfo) {
	if nInfo.table == nil || nInfo.dbno >= db.MaxDB {
		return // should not happen
	}

	d := nInfo.dbno
	tKey := *nInfo.table
	nGrp := sc.dbNInfos[d][tKey]
	if nGrp == nil {
		nGrp = new(notificationGroup)
		if tMap := sc.dbNInfos[d]; tMap != nil {
			tMap[tKey] = nGrp
		} else {
			sc.dbNInfos[d] = map[db.TableSpec]*notificationGroup{tKey: nGrp}
		}
	}

	nGrp.add(nInfo)
	nInfo.sInfo = sc.sInfo
}

func (sc *subscribeContext) translateAndAddPath(path string, mode NotificationType) error {
	trData := sc.getFromSession(path, mode)
	if trData == nil {
		trInfo, err := sc.translateSubscribe(path, mode)
		if err != nil {
			return err
		}
		trData = trInfo.getNInfos()
	}

	sc.tgtInfos = append(sc.tgtInfos, trData.targetInfos...)

	// Group nInfo by table and key pattern for OnChange.
	// Required for registering db subscriptions.
	if mode == OnChange {
		if sc.dbNInfos == nil {
			sc.dbNInfos = make(map[db.DBNum]map[db.TableSpec]*notificationGroup)
		}
		for _, nInfo := range trData.targetInfos {
			sc.addToNGroup(nInfo)
		}
		for _, nInfo := range trData.childInfos {
			sc.addToNGroup(nInfo)
		}
	}

	return nil
}

func (sc *subscribeContext) startSubscribe() error {
	var err error

	sMutex.Lock()
	defer sMutex.Unlock()

	sInfo := sc.sInfo

	stopMap[sInfo.stop] = sInfo

	for dbno, nGroups := range sc.dbNInfos {
		opt := getDBOptions(dbno, withWriteDisable)
		err = startDBSubscribe(opt, nGroups, sInfo)

		if err != nil {
			log.Warningf("[%v] db subscribe failed -- %v", sInfo.id, err)
			cleanup(sInfo.stop)
			return err
		}
	}

	for _, nInfo := range sc.tgtInfos {
		err := sendInitialUpdate(sInfo, nInfo)
		if err != nil {
			log.Warningf("[%v] init sync failed -- %v", sInfo.id, err)
			cleanup(sInfo.stop)
			return err
		}
	}

	sInfo.syncDone = true
	sendSyncNotification(sInfo, false)

	go stophandler(sInfo.stop)

	return err
}

// translateSubscribe resolves app module for a given path and calls translateSubscribe.
// Returns a translatedPathInfo containing app's response and other context data.
func (sc *subscribeContext) translateSubscribe(reqPath string, mode NotificationType) (*translatedPathInfo, error) {
	sid := sc.id
	app, appInfo, err := getAppModule(reqPath, sc.version)
	if err != nil {
		return nil, err
	}

	recurseSubpaths := sc.recurse || mode != Sample // always recurse for on_change and target_defined
	log.V(1).Infof("[%v] Calling translateSubscribe for path=\"%s\", mode=%v, recurse=%v",
		sid, reqPath, mode, recurseSubpaths)

	resp, err := (*app).translateSubscribe(
		translateSubRequest{
			ctxID:   sid,
			path:    reqPath,
			mode:    mode,
			recurse: recurseSubpaths,
			dbs:     sc.dbs,
		})

	if err != nil {
		log.Warningf("[%v] translateSubscribe failed for \"%s\"; err=%v", sid, reqPath, err)
		return nil, err
	}

	if len(resp.ntfAppInfoTrgt) == 0 {
		log.Warningf("[%v] translateSubscribe returned nil/empty response for path: %s", sid, reqPath)
		var pType string
		if path.StrHasWildcardKey(reqPath) {
			pType = "wildcard "
		}
		return nil, tlerr.NotSupported("%spath not supported: %s", pType, reqPath)
	}

	log.V(2).Infof("[%v] Path \"%s\" mapped to %d target and %d child notificationAppInfos",
		sid, reqPath, len(resp.ntfAppInfoTrgt), len(resp.ntfAppInfoTrgtChlds))
	for i, nAppInfo := range resp.ntfAppInfoTrgt {
		log.V(2).Infof("[%v] targetInfo[%d] = %v", sid, i, nAppInfo)
		if err = sc.validateAppInfo(nAppInfo, mode); err != nil {
			return nil, err
		}
	}
	for i, nAppInfo := range resp.ntfAppInfoTrgtChlds {
		log.V(2).Infof("[%v] childInfo[%d] = %v", sid, i, nAppInfo)
		if err = sc.validateAppInfo(nAppInfo, mode); err != nil {
			return nil, err
		}
	}

	return &translatedPathInfo{
		path:     reqPath,
		appInfo:  appInfo,
		sContext: sc,
		response: &resp,
	}, nil
}

func (sc *subscribeContext) validateAppInfo(nAppInfo *notificationAppInfo, mode NotificationType) error {
	if nAppInfo == nil {
		log.Warningf("[%v] app returned nil notificationAppInfo", sc.id)
		return tlerr.New("internal error")
	}
	if nAppInfo.path == nil {
		log.Warningf("[%v] app returned nil path", sc.id)
		return tlerr.New("internal error")
	}
	if nAppInfo.isNonDB() {
		if err := sc.validateNonDbPath(nAppInfo, mode); err != nil {
			return err
		}
	}
	if nAppInfo.isDataSrcDynamic {
		if err := sc.validateDynamicSrcPath(nAppInfo); err != nil {
			return err
		}
	}
	if len(nAppInfo.keyGroupComps) != 0 {
		if err := sc.validateKeyGroup(nAppInfo); err != nil {
			return err
		}
	}
	return nil
}

func (sc *subscribeContext) validateNonDbPath(nAppInfo *notificationAppInfo, mode NotificationType) error {
	if nAppInfo.isOnChangeSupported { // Force disable on_change for non-db paths
		nAppInfo.isOnChangeSupported = false
		nAppInfo.pType = Sample
	}
	if path.HasWildcardKey(nAppInfo.path) {
		p := path.String(nAppInfo.path)
		log.V(1).Infof("[%v] Wildcard keys not supported for non-DB path: %s", sc.id, p)
		return tlerr.NotSupported("Wildcard keys not supported: %s", p)
	}
	return nil
}

func (sc *subscribeContext) validateDynamicSrcPath(nAppInfo *notificationAppInfo) error {
	if e0 := path.GetElemAt(nAppInfo.path, 0); !strings.HasPrefix(e0, "openconfig-") {
		p := path.String(nAppInfo.path)
		log.Warningf("[%v] Dynamic table handling supported only for OC yangs; found %s", sc.id, p)
		return tlerr.NotSupported("Subscribe not supported: %s", p)
	}
	if n := path.Len(nAppInfo.path); path.HasWildcardKey(path.SubPath(nAppInfo.path, 0, n-1)) {
		p := path.String(nAppInfo.path)
		log.V(1).Infof("[%v] Wildcard supported only at the last element for dynamic table path; found: %s", sc.id, p)
		return tlerr.NotSupported("Wildcard keys not supported: %s", p)
	}
	return nil
}

func (sc *subscribeContext) validateKeyGroup(nAppInfo *notificationAppInfo) error {
	if nAppInfo.isNonDB() {
		log.Warningf("[%v] keyGroups  not supported for non-DB path %s", sc.id, path.String(nAppInfo.path))
		return tlerr.New("internal error")
	}
	for _, k := range nAppInfo.keyGroupComps {
		if k < 0 || k >= nAppInfo.key.Len() {
			log.Warningf("[%v] keyGroups %v contains invalid values (>%d), path=%s",
				sc.id, nAppInfo.keyGroupComps, nAppInfo.key.Len(), path.String(nAppInfo.path))
			return tlerr.New("internal error")
		}
	}
	return nil
}

// getFromSession returns the translatedSubData for the path from the SubscribeSession.
// Returns nil if session does not exist or entry not found for the path.
func (sc *subscribeContext) getFromSession(path string, mode NotificationType) *translatedSubData {
	var trData *translatedSubData
	if sc.session != nil {
		trData = sc.session.get(path)
		log.V(2).Infof("[%v] found trData %p from session for \"%s\"", sc.id, trData, path)
	}
	return trData
}

// getNInfos returns the translated data as a translatedSubData.
func (tpInfo *translatedPathInfo) getNInfos() *translatedSubData {
	if tpInfo.trData != nil {
		return tpInfo.trData
	}

	trResp := tpInfo.response
	targetLen := len(trResp.ntfAppInfoTrgt)
	childLen := len(trResp.ntfAppInfoTrgtChlds)
	trData := &translatedSubData{
		targetInfos: make([]*notificationInfo, targetLen),
		childInfos:  make([]*notificationInfo, childLen),
	}

	for i := 0; i < targetLen; i++ {
		trData.targetInfos[i] = tpInfo.newNInfo(trResp.ntfAppInfoTrgt[i])
	}

	for i := 0; i < childLen; i++ {
		trData.childInfos[i] = tpInfo.newNInfo(trResp.ntfAppInfoTrgtChlds[i])
	}

	tpInfo.trData = trData
	return trData
}

// add a notificationInfo to the notificationGroup
func (ng *notificationGroup) add(nInfo *notificationInfo) {
	keyStr := strings.Join(nInfo.key.Comp, "/")
	if ng.nInfos == nil {
		ng.nInfos = map[string][]*notificationInfo{keyStr: {nInfo}}
	} else {
		ng.nInfos[keyStr] = append(ng.nInfos[keyStr], nInfo)
	}
}

// toSKeys prepares DB subscribe keys for the notificationGroup
func (ng *notificationGroup) toSKeys() []*db.SKey {
	skeys := make([]*db.SKey, 0, len(ng.nInfos))
	for _, nInfoList := range ng.nInfos {
		// notificationInfo are already segregated by key patterns. So we can
		// just use 1st entry from this sub-group for getting table and key patterns.
		// TODO avoid redundant registrations of matching patterns (like "PORT|Eth1" and "PORT|*")
		nInfo := nInfoList[0]
		skeys = append(skeys, &db.SKey{
			Ts:     nInfo.table,
			Key:    nInfo.key,
			Opaque: ng,
		})
	}
	return skeys
}

func (ng *notificationGroup) toString() string {
	if ng == nil || len(ng.nInfos) == 0 {
		return "{}"
	}
	var nInfo *notificationInfo
	comps := make([][]string, 0, len(ng.nInfos))
	for _, nInfoList := range ng.nInfos {
		nInfo = nInfoList[0]
		comps = append(comps, nInfo.key.Comp)
	}
	return fmt.Sprintf("{dbno=%d, table=%s, patterns=%v}", nInfo.dbno, nInfo.table.Name, comps)
}

// translatedPathInfo holds the response of translateSubscribe for a path
// along with additional context info. Provides methods for convert this into
// translatedSubData (i.e, notificationInfo) and to save to SubscribeSession.
type translatedPathInfo struct {
	path     string
	appInfo  *appInfo
	sContext *subscribeContext
	response *translateSubResponse // returned by transalateSubscribe
	trData   *translatedSubData    // notificationInfo's derived from response
}

// translatedSubData holds translated subscription data for a path.
type translatedSubData struct {
	targetInfos []*notificationInfo
	childInfos  []*notificationInfo
}

// newNInfo creates a new *notificationInfo from given *notificationAppInfo.
// Uses the context information from this tpInfo; but does not update it.
func (tpInfo *translatedPathInfo) newNInfo(nAppInfo *notificationAppInfo) *notificationInfo {
	nInfo := &notificationInfo{
		dbno:        nAppInfo.dbno,
		table:       nAppInfo.table,
		key:         nAppInfo.key,
		fields:      nAppInfo.dbFldYgPathInfoList,
		path:        nAppInfo.path,
		handler:     nAppInfo.handlerFunc,
		appInfo:     tpInfo.appInfo,
		sInfo:       tpInfo.sContext.sInfo,
		opaque:      nAppInfo.opaque,
		fldScanPatt: nAppInfo.fieldScanPattern,
		keyGroup:    nAppInfo.keyGroupComps,
	}

	// Make sure field prefix path has a leading and trailing "/".
	// Helps preparing full path later by joining parts
	for _, pi := range nInfo.fields {
		if path.StrHasWildcardKey(pi.rltvPath) {
			nInfo.flags.Set(niWildcardSubpath)
		}
		if len(pi.rltvPath) != 0 && pi.rltvPath[0] != '/' {
			pi.rltvPath = "/" + pi.rltvPath
		}
		// Look for fields mapped to yang key - formatted as "{xyz}"
		for _, leaf := range pi.dbFldYgPathMap {
			if len(leaf) != 0 && leaf[0] == '{' {
				nInfo.flags.Set(niKeyFields)
			}
		}
	}

	if nAppInfo.isLeafPath() {
		nInfo.flags.Set(niLeafPath)
	}
	switch nAppInfo.deleteAction {
	case apis.InspectPathOnDelete:
		nInfo.flags.Set(niDeleteAsUpdate)
	case apis.InspectLeafOnDelete:
		nInfo.flags.Set(niDeleteAsUpdate | niPartial)
	}
	if path.HasWildcardKey(nAppInfo.path) {
		nInfo.flags.Set(niWildcardPath)
	}
	if nAppInfo.isDataSrcDynamic {
		nInfo.flags.Set(niDynamic)
	}

	return nInfo
}

// saveToSession saves the derived notificationInfo data into the
// SubscribeSession.
func (tpInfo *translatedPathInfo) saveToSession() {
	session := tpInfo.sContext.session
	if session == nil {
		return
	}

	trData := tpInfo.getNInfos()
	path := tpInfo.path
	if trData == nil {
		log.V(2).Infof("[%v] no trData for \"%s\"", tpInfo.sContext.id, path)
		return
	}
	log.V(2).Infof("[%v] set trData %p in session for \"%s\"", tpInfo.sContext.id, trData, path)
	session.put(path, trData)
}

// segregateSampleSubpaths inspects the translateSubscribe response and prepares
// extra SAMPLE subscribe sub-paths for the TARGET_DEFINED subscription if needed.
// Returns nil if current path & its subtree prefers only ON_CHANGE or only SAMPLE.
// If path prefers ON_CHANGE but some subtree nodes prefer SAMPLE, only the ON_CHANGE
// related notificationAppInfo will be retained in this translatedPathInfo. Returns new
// translatedPathInfo for each of the subtree segments that prefer SAMPLE.
func (tpInfo *translatedPathInfo) segregateSampleSubpaths() []*translatedPathInfo {
	sid := tpInfo.sContext.id
	var samplePInfos []*translatedPathInfo
	var samplePaths []string
	var onchgInfos, sampleInfos []*notificationAppInfo

	log.V(2).Infof("[%v] segregate on_change and sample segments for %s", sid, tpInfo.path)

	// Segregate ON_CHANGE and SAMPLE nAppInfos from targetInfos
	for _, nAppInfo := range tpInfo.response.ntfAppInfoTrgt {
		if !nAppInfo.isOnChangeSupported || nAppInfo.pType == Sample {
			sampleInfos = append(sampleInfos, nAppInfo)
		} else {
			onchgInfos = append(onchgInfos, nAppInfo)
		}
	}

	log.V(2).Infof("[%v] found %d on_change and %d sample targetInfos",
		sid, len(onchgInfos), len(sampleInfos))

	// Path does not support ON_CHANGE. No extra re-grouping required.
	if len(onchgInfos) == 0 {
		log.V(1).Infof("[%v] on_change={}; sample={%s}", sid, tpInfo.path)
		return nil
	}

	// Some targetInfos prefer SAMPLE mode.. Remove them from this translatedPathInfo
	// and create separate translatedPathInfo for them.
	if len(sampleInfos) != 0 {
		tpInfo.response.ntfAppInfoTrgt = onchgInfos
		for _, nAppInfo := range sampleInfos {
			matchingPInfo := findTranslatedParentInfo(samplePInfos, nAppInfo.path)
			if matchingPInfo != nil {
				// Matching entry exists.. append to its targetInfo.
				// Required for paths that map to multiple tables.
				matchingPInfo.response.ntfAppInfoTrgt = append(
					matchingPInfo.response.ntfAppInfoTrgt, nAppInfo)
			} else {
				newPInfo := tpInfo.cloneForSubPath(nAppInfo)
				samplePInfos = append(samplePInfos, newPInfo)
				samplePaths = append(samplePaths, newPInfo.path)
			}
		}
	}

	// Segregate ON_CHANGE and SAMPLE childInfos.
	onchgInfos = nil
	for _, nAppInfo := range tpInfo.response.ntfAppInfoTrgtChlds {
		if nAppInfo.isOnChangeSupported && nAppInfo.pType != Sample {
			onchgInfos = append(onchgInfos, nAppInfo)
			continue
		}

		parentPInfo := findTranslatedParentInfo(samplePInfos, nAppInfo.path)
		if parentPInfo != nil {
			// Parent path of nAppInfo is already recorded. Add as childInfo.
			parentPInfo.response.ntfAppInfoTrgtChlds = append(
				parentPInfo.response.ntfAppInfoTrgtChlds, nAppInfo)
		} else {
			// nAppInfo does not belong to any of the already recorded paths.
			// Treat it as a new targetInfo.
			newPInfo := tpInfo.cloneForSubPath(nAppInfo)
			samplePInfos = append(samplePInfos, newPInfo)
			samplePaths = append(samplePaths, newPInfo.path)
		}
	}

	if log.V(2) {
		nSample := len(tpInfo.response.ntfAppInfoTrgtChlds) - len(onchgInfos)
		log.Infof("[%v] found %d on_change and %d sample childInfos", sid, len(onchgInfos), nSample)
	}

	// Retain only ON_CHANGE childInfos in this translatedPathInfo.
	// SAMPLE mode childInfos would have been already moved to samplePInfos.
	if len(onchgInfos) != len(tpInfo.response.ntfAppInfoTrgtChlds) {
		tpInfo.response.ntfAppInfoTrgtChlds = onchgInfos
	}

	log.V(1).Infof("[%v] on_change=[%s]; sample=%v", sid, tpInfo.path, samplePaths)
	return samplePInfos
}

// cloneForSubPath creates a clone of this *translatedPathInfo with a
// fake response created from given *notificationAppInfo.
func (tpInfo *translatedPathInfo) cloneForSubPath(nAppInfo *notificationAppInfo) *translatedPathInfo {
	return &translatedPathInfo{
		path:     path.String(nAppInfo.path),
		appInfo:  tpInfo.appInfo,
		sContext: tpInfo.sContext,
		response: &translateSubResponse{
			ntfAppInfoTrgt: []*notificationAppInfo{nAppInfo},
		},
	}
}

func findTranslatedParentInfo(tpInfos []*translatedPathInfo, p *gnmi.Path) *translatedPathInfo {
	for _, tpInfo := range tpInfos {
		for _, targetInfo := range tpInfo.response.ntfAppInfoTrgt {
			if path.Matches(p, targetInfo.path) {
				return tpInfo
			}
		}
	}
	return nil
}

// translatedPathCache is the per-path cache in SubscribeSession.
type translatedPathCache struct {
	pathData map[string]*translatedSubData
}

func (tpCache *translatedPathCache) put(path string, trData *translatedSubData) {
	if tpCache.pathData == nil {
		tpCache.pathData = make(map[string]*translatedSubData)
	}
	tpCache.pathData[path] = trData
}

func (tpCache *translatedPathCache) get(path string) *translatedSubData {
	return tpCache.pathData[path]
}

func (tpCache *translatedPathCache) reset() {
	tpCache.pathData = nil
}

func stophandler(stop chan struct{}) {
	for {
		<-stop
		sMutex.Lock()
		defer sMutex.Unlock()
		cleanup(stop)
		return
	}
}

func cleanup(stop chan struct{}) {
	if sInfo, ok := stopMap[stop]; ok {
		log.Infof("[%v] stopping..", sInfo.id)

		for _, sDB := range sInfo.sDBs {
			sDB.UnsubscribeDB()
		}

		sInfo.sDBs = nil
		closeAllDbs(sInfo.dbs[:])

		delete(stopMap, stop)
	}
	//printAllMaps()
}

// SubscribeSession is used to share session data between subscription
// related APIs - IsSubscribeSupported, Subscribe and Stream.
type SubscribeSession struct {
	ID          string  // session id
	callCounter Counter // API call counter
	translatedPathCache
}

// NewSubscribeSession creates a new SubscribeSession. Caller
// MUST close the session object through CloseSubscribeSession
// call at the end.
func NewSubscribeSession() *SubscribeSession {
	return &SubscribeSession{
		ID: fmt.Sprintf("s%d", subscribeCounter.Next()),
	}
}

// Close a SubscribeSession and release all resources it held by it.
// API client MUST close the sessions it creates; and not reuse the session after closing.
func (ss *SubscribeSession) Close() {
	if ss != nil {
		ss.reset()
	}
}

func subscribeContextId(ss *SubscribeSession) string {
	if ss != nil {
		return fmt.Sprintf("%s.%d", ss.ID, ss.callCounter.Next()-1)
	}
	return fmt.Sprintf("s%d.0", subscribeCounter.Next())
}
