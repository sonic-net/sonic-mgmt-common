////////////////////////////////////////////////////////////////////////////////
//                                                                            //
//  Copyright 2023 Broadcom. The term Broadcom refers to Broadcom Inc. and/or //
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
	"time"

	"github.com/Azure/sonic-mgmt-common/translib/db"
	"github.com/Azure/sonic-mgmt-common/translib/internal/apis"
	"github.com/Azure/sonic-mgmt-common/translib/ocbinds"
	"github.com/Azure/sonic-mgmt-common/translib/path"
	"github.com/Azure/sonic-mgmt-common/translib/tlerr"
	log "github.com/golang/glog"
	"github.com/openconfig/gnmi/proto/gnmi"
	"github.com/openconfig/ygot/ygot"
)

// dbNotificationCounter counts number of db notification processed.
// Used to derive notificationID
var dbNotificationCounter Counter

// notificationEvent holds data about translib notification.
type notificationEvent struct {
	id    string             // Unique id for logging
	event db.SEvent          // DB notification type, if any
	key   *db.Key            // DB key, if any
	entry *db.Value          // DB entry
	nGrup *notificationGroup // Target notificationGroup for the event
	sInfo *subscribeInfo

	// Meta info for processSubscribe calls
	forceProcessSub bool
	appCache        map[*appInfo]appInterface
}

func notificationHandler(d *db.DB, sKey *db.SKey, key *db.Key, event db.SEvent) error {
	nid := fmt.Sprintf("n%d", dbNotificationCounter.Next())
	log.Infof("[%v] notificationHandler: d=%v, table=%v, kayPattern=%v, key=%v, event=%v",
		nid, dbInfo(d), tableInfo(sKey.Ts), keyInfo(sKey.Key), keyInfo(key), event)

	sMutex.Lock()
	defer sMutex.Unlock()

	switch event {
	case db.SEventHSet, db.SEventHDel, db.SEventDel:
		if nGrup, ok := sKey.Opaque.(*notificationGroup); ok {
			n := notificationEvent{
				id:    nid,
				event: event,
				key:   key,
				nGrup: nGrup,
			}
			n.process()
		} else {
			log.Warningf("[%v] notificationHandler: SKey corrupted; nil opaque. %v", nid, *sKey)
		}

	case db.SEventClose:
		// Close event would have been triggered due to unsubscribe on stop request
		delete(cleanupMap, d)

	case db.SEventErr:
		// Unexpected error in db layer.. Terminate the subscribe request.
		if sInfo, ok := cleanupMap[d]; ok && sInfo != nil && !sInfo.termDone {
			sendSyncNotification(sInfo, true)
			sInfo.termDone = true
		}
		delete(cleanupMap, d)
	}

	return nil
}

// sendInitialUpdate sends the initial sync updates to the caller.
// Performs following steps:
//  1. Scan all keys for the table
//  2. Map each key to yang path
//  3. Get value for each path and send the notification message
func sendInitialUpdate(sInfo *subscribeInfo, nInfo *notificationInfo) error {
	ne := notificationEvent{
		id:    sInfo.id,
		sInfo: sInfo,
	}

	pathStr := path.String(nInfo.path)
	topNode := []*yangNodeInfo{new(yangNodeInfo)}

	log.V(1).Infof("[%s] initial update for path %s", ne.id, pathStr)

	if nInfo.table == nil { // non-db case
		log.V(1).Infof("[%s] Non-DB case.. notifying using direct GET", ne.id)
		ne.sendNotification(nInfo, "", topNode)
		return nil
	}

	// Workaround for paths mapped to a dynamic table -- notify using direct get
	if nInfo.flags.Has(niDynamic) {
		log.V(1).Infof("[%s] Dynamic table case.. notifying using direct GET", ne.id)
		nInfo2 := ne.prepareDynamicTablePath(nInfo)
		ne.sendNotification(nInfo2, "", topNode)
		return nil
	}

	// DB path.. iterate over keys and generate notification for each.

	d := sInfo.dbs[int(nInfo.dbno)]
	scanType := db.KeyScanType
	if len(nInfo.fldScanPatt) > 0 {
		scanType = db.FieldScanType
	}
	opts := db.ScanCursorOpts{ScanType: scanType, FldScanPatt: nInfo.fldScanPatt}
	cursor, err := d.NewScanCursor(nInfo.table, *nInfo.key, &opts)
	if err != nil {
		log.Warningf("[%s] Failed to create db cursor for %d/%s/%v; err=%v",
			ne.id, nInfo.dbno, nInfo.table.Name, nInfo.key, err)
		return err
	}

	defer cursor.DeleteScanCursor()
	var ddup map[string]bool
	var keys []db.Key

	if nInfo.key.IsPattern() && !nInfo.flags.Has(niWildcardPath) {
		log.V(5).Infof("[%s] db key is a glob pattern. Forcing processSubscribe..", ne.id)
		ne.forceProcessSub = true
	}

	if len(nInfo.keyGroup) != 0 {
		ddup = make(map[string]bool)
	}

	for done := false; !done; {
		var dbVal db.Value
		if scanType == db.FieldScanType {
			opts.CountHint = 1000
			dbVal, done, err = cursor.GetNextFields(&opts)
			keys = make([]db.Key, 0, len(dbVal.Field))
			for fldName := range dbVal.Field {
				keys = append(keys, db.Key{Comp: []string{fldName}})
			}
		} else {
			keys, done, err = cursor.GetNextKeys(&opts)
		}

		if err != nil {
			log.V(2).Infof("[%s] Failed to read db cursor for %d/%s/%v/%v; err=%v",
				ne.id, nInfo.dbno, nInfo.table.Name, nInfo.key, opts.FldScanPatt, err)
			return err
		}

		for _, k := range keys {
			ne.key = &k
			if ddup != nil {
				ddk := ne.getDdupKey(nInfo.keyGroup)
				if len(ddk) != 0 && ddup[ddk] {
					log.V(5).Infof("[%s] skip init sync for key %v; another key with matching comps %v has been processed",
						ne.id, k.Comp, nInfo.keyGroup)
					continue
				}
				ddup[ddk] = true
			}

			if scanType == db.FieldScanType {
				val := db.Value{Field: make(map[string]string)}
				val.Field[k.Comp[0]] = dbVal.Field[k.Comp[0]]
				ne.entry = &val
			} else {
				if v, err := d.GetEntry(nInfo.table, k); err != nil {
					log.V(2).Infof("[%v] Table %s key %v not found; skip initial sync",
						ne.id, nInfo.table.Name, k.Comp)
					continue
				} else {
					ne.entry = &v
				}
			}

			// Use apps's SubscribeOnChange handler to generate notifications.
			// Key dedup not required (for now).
			if nInfo.handler != nil {
				v := apis.EntryDiff{NewValue: *ne.entry, EntryCreated: true}
				ne.invokeAppHandler(nInfo, &v)
				continue
			}

			ne.sendNotification(nInfo, "", topNode)
		}
	}

	return nil
}

func sendSyncNotification(sInfo *subscribeInfo, isTerminated bool) {
	ne := notificationEvent{id: sInfo.id, sInfo: sInfo}
	log.Infof("[%v] Sending syncDone=%v, isTerminated=%v", ne.id, sInfo.syncDone, isTerminated)
	ne.send(&SubscribeResponse{
		Timestamp:    time.Now().UnixNano(),
		SyncComplete: sInfo.syncDone,
		IsTerminated: isTerminated,
	})
}

func (ne *notificationEvent) prepareDynamicTablePath(nInfo *notificationInfo) *notificationInfo {
	if !nInfo.flags.Has(niWildcardPath) {
		return nInfo
	}

	// Remove the last list node from path if it has windcard keys - so that direct GET works.
	// Will not fetch extra data since list in always surrounded by a container with no other nodes.
	p := nInfo.path
	if k := path.Len(p) - 1; k > 0 && path.HasWildcardAtKey(p, k) {
		p = path.SubPath(p, 0, k)
		if log.V(2) {
			log.Infof("[%s] Modified path: %s", ne.id, path.String(p))
		}
	}

	// Return a copy of nInfo contraining the modified path
	nInfoCopy := new(notificationInfo)
	*nInfoCopy = *nInfo
	nInfoCopy.path = p
	nInfoCopy.flags.Unset(niWildcardPath)
	return nInfoCopy
}

func (ne *notificationEvent) getDdupKey(keyGroupComps []int) string {
	if len(keyGroupComps) == 0 {
		return ""
	}

	kLen := ne.key.Len()
	uniq := make([]string, len(keyGroupComps))
	for i, v := range keyGroupComps {
		if v < 0 || v >= kLen {
			log.Warningf("[%s] app returned invalid component index %d; key=%v",
				ne.id, i, ne.key.Comp)
			return ""
		}
		uniq[i] = ne.key.Get(v)
	}

	return strings.Join(uniq, "|")
}

// process translates db notification into SubscribeResponse and
// pushes to the caller.
func (ne *notificationEvent) process() {
	dbDiff, err := ne.DiffAndMergeOnChangeCache()
	if err == defunct {
		log.V(2).Infof("[%s] defunct subscription", ne.id)
		return
	}
	if err != nil {
		log.Warningf("[%s] error finding modified db fields: %v", ne.id, err)
		return
	}
	if dbDiff == nil || dbDiff.IsEmpty() {
		log.V(2).Infof("[%s] empty diff", ne.id)
		return
	}

	origId := ne.id

	// Find all key patterns that match current key
	for _, nInfos := range ne.nGrup.nInfos {
		keyPattern := *nInfos[0].key
		if !ne.key.Matches(keyPattern) {
			continue
		}

		log.V(2).Infof("[%s] Key %v matches registered pattern %v; has %d nInfos",
			ne.id, ne.key.Comp, keyPattern.Comp, len(nInfos))

		for _, nInfo := range nInfos {
			ne.sInfo = nInfo.sInfo
			ne.id = origId + "-" + nInfo.sInfo.id
			if log.V(2) {
				log.Infof("[%s] processing path: %s", ne.id, path.String(nInfo.path))
			}

			if nInfo.handler != nil {
				ne.invokeAppHandler(nInfo, dbDiff)
				continue
			}

			yInfos := ne.findModifiedFields(nInfo, dbDiff)
			changed := false
			if len(yInfos.old) != 0 {
				changed = true
				ne.entry = &dbDiff.OldValue
				ne.sendNotifications(nInfo, yInfos.old)
			}
			if len(yInfos.new) != 0 {
				changed = true
				ne.entry = &dbDiff.NewValue
				ne.sendNotifications(nInfo, yInfos.new)
			}
			if !changed {
				log.V(2).Infof("[%s] no fields updated", ne.id)
			}
		}

		ne.id = origId
	}
}

// DiffAndMergeOnChangeCache Compare modified entry with cached entry and
// return modified fields. Also update the cache with changes.
func (ne *notificationEvent) DiffAndMergeOnChangeCache() (*apis.EntryDiff, error) {
	var nInfo *notificationInfo
	for _, n := range ne.nGrup.nInfos {
		nInfo = n[0] // Pick any one nInfo from notificationGroup to read dbno and table spec
		break
	}
	if nInfo == nil {
		return nil, nil
	}

	ts := nInfo.table
	d := nInfo.sInfo.dbs[nInfo.dbno] // FIXME do not assume all nInfos belong to same sInfo
	var oldValue, newValue db.Value
	var err error

	if d == nil {
		return nil, defunct
	}
	if ne.event == db.SEventDel {
		oldValue, err = d.OnChangeCacheDelete(ts, *ne.key)
	} else {
		oldValue, newValue, err = d.OnChangeCacheUpdate(ts, *ne.key)
	}
	if err != nil {
		return nil, err // not found or redis error
	}

	cacheEntryDiff := apis.EntryCompare(oldValue, newValue)

	log.V(2).Infof("[%s] DiffAndMergeOnChangeCache: %v", ne.id, cacheEntryDiff)

	return cacheEntryDiff, nil
}

// findModifiedFields determines db fields changed since last notification
func (ne *notificationEvent) findModifiedFields(nInfo *notificationInfo, entryDiff *apis.EntryDiff) yangNodeInfoSet {
	var yInfos yangNodeInfoSet
	targetPathCreate := entryDiff.EntryCreated
	targetPathDelete := entryDiff.EntryDeleted

	if nInfo.flags.Has(niKeyFields) && !targetPathCreate && !targetPathDelete {
		targetPathCreate, targetPathDelete = ne.processKeyFields(nInfo, entryDiff)
	}

	// When a new db entry is created, the notification infra can fetch full
	// content of target path.
	if targetPathCreate {
		log.V(2).Infof("[%s] Entry created;", ne.id)
		yInfos.new = append(yInfos.new, &yangNodeInfo{})
	}

	// Treat entry delete as update when 'partial' flag is set
	if entryDiff.EntryDeleted && nInfo.flags.Has(niDeleteAsUpdate) {
		log.V(2).Infof("[%s] Entry deleted; but treating it as update", ne.id)
		if nInfo.flags.Has(niPartial) {
			delFields := apis.EntryFields(entryDiff.OldValue)
			yInfos.old = ne.createYangPathInfos(nInfo, delFields, "update")
		}
		if len(yInfos.old) == 0 {
			yInfos.old = append(yInfos.old, &yangNodeInfo{})
		}
		return yInfos
	}

	// When entry is deleted, mark the whole target path as deleted
	if targetPathDelete {
		log.V(2).Infof("[%s] Entry deleted;", ne.id)
		yInfos.old = append(yInfos.old, &yangNodeInfo{deleted: true})
	}

	if targetPathCreate || targetPathDelete {
		log.V(5).Infof("[%s] findModifiedFields returns %v", ne.id, yInfos)
		return yInfos
	}

	// Collect yang leaf info for updated fields
	if len(entryDiff.UpdatedFields) != 0 {
		yInfos.new = ne.createYangPathInfos(nInfo, entryDiff.UpdatedFields, "update")
	}

	// Collect yang leaf info for created fields
	if len(entryDiff.CreatedFields) != 0 {
		yy := ne.createYangPathInfos(nInfo, entryDiff.CreatedFields, "create")
		if len(yy) != 0 {
			yInfos.new = append(yInfos.new, yy...)
		}
	}

	// Collect yang leaf info for deleted fields
	if len(entryDiff.DeletedFields) != 0 {
		yy := ne.createYangPathInfos(nInfo, entryDiff.DeletedFields, "delete")
		if len(yy) != 0 {
			if nInfo.flags.Has(niLeafPath | niDeleteAsUpdate) {
				log.V(2).Infof("[%s] Treating field delete as target leaf path update", ne.id)
				for _, y := range yy {
					y.deleted = false
				}
			}
			yInfos.new = append(yInfos.new, yy...)
		}
	}

	log.V(5).Infof("[%s] findModifiedFields returns %v", ne.id, yInfos)
	return yInfos
}

func (ne *notificationEvent) processKeyFields(nInfo *notificationInfo, entryDiff *apis.EntryDiff) (keyCreate, keyDelete bool) {
	keyFields := map[string]bool{}
	for _, nDbFldInfo := range nInfo.fields {
		for field, leaf := range nDbFldInfo.dbFldYgPathMap {
			if len(leaf) != 0 && leaf[0] == '{' {
				keyFields[field] = true
			}
		}
	}
	for _, f := range entryDiff.DeletedFields {
		if keyFields[f] {
			log.V(2).Infof("[%s] deleted field %s is mapped to yang key; treat as path delete", ne.id, f)
			keyDelete = true
			break
		}
	}
	for _, f := range entryDiff.CreatedFields {
		if keyFields[f] {
			log.V(2).Infof("[%s] created field %s is mapped to yang key; treat as path create", ne.id, f)
			keyCreate = true
			break
		}
	}
	for _, f := range entryDiff.UpdatedFields {
		if keyFields[f] {
			log.V(2).Infof("[%s] updated field %s is mapped to yang key; treat as path delete+create", ne.id, f)
			keyDelete = true
			keyCreate = true
			break
		}
	}
	return
}

func (ne *notificationEvent) createYangPathInfos(nInfo *notificationInfo, fields []string, action string) []*yangNodeInfo {
	var yInfos []*yangNodeInfo
	isDelete := (action == "delete")

	for _, f := range fields {
		for _, nDbFldInfo := range nInfo.fields {
			leaf, ok := nDbFldInfo.dbFldYgPathMap[f]
			if !ok {
				continue
			}
			// DB field mapped to single leaf node
			if strings.IndexByte(leaf, ',') == -1 {
				log.V(2).Infof("[%s] %s field=%s, path=%s/%s", ne.id, action, f, nDbFldInfo.rltvPath, leaf)
				yInfos = append(yInfos, &yangNodeInfo{
					parentPrefix: nDbFldInfo.rltvPath,
					leafName:     leaf,
					deleted:      isDelete,
				})
				continue
			}
			// DB field mapped to multiple leaf nodes -- a comma separated value
			for _, s := range strings.Split(leaf, ",") {
				s = strings.TrimSpace(s)
				log.V(2).Infof("[%s] %s field=%s, path=%s/%s", ne.id, action, f, nDbFldInfo.rltvPath, s)
				yInfos = append(yInfos, &yangNodeInfo{
					parentPrefix: nDbFldInfo.rltvPath,
					leafName:     s,
					deleted:      isDelete,
				})
			}
		}
	}

	return yInfos
}

func (ne *notificationEvent) invokeAppHandler(nInfo *notificationInfo, entryDiff *apis.EntryDiff) {
	log.V(2).Infof("[%s] invoking custom handler: %v", ne.id, nInfo.handler)
	nc := apis.NotificationContext{
		Path:      path.Clone(nInfo.path),
		Db:        ne.sInfo.dbs[int(nInfo.dbno)],
		Table:     nInfo.table,
		Key:       ne.key,
		AllDb:     ne.sInfo.dbs,
		EntryDiff: *entryDiff,
		Opaque:    nInfo.opaque,
	}
	ctx := pocCallContext{ne: ne, nInfo: nInfo}
	nInfo.handler(&nc, &ctx)
}

func (ne *notificationEvent) getApp(nInfo *notificationInfo) appInterface {
	if app := ne.appCache[nInfo.appInfo]; app != nil {
		return app
	}

	app, _ := getAppInterface(nInfo.appInfo.appType)
	if ne.appCache == nil {
		ne.appCache = map[*appInfo]appInterface{nInfo.appInfo: app}
	} else {
		ne.appCache[nInfo.appInfo] = app
	}
	return app
}

func (ne *notificationEvent) getValue(nInfo *notificationInfo, path string) (ygot.ValidatedGoStruct, error) {
	var payload ygot.ValidatedGoStruct
	app := ne.getApp(nInfo)
	appInfo := nInfo.appInfo
	dbs := ne.sInfo.dbs

	err := appInitialize(&app, appInfo, path, nil, &appOptions{}, GET)

	if err != nil {
		return payload, err
	}

	err = app.translateGet(dbs)

	if err != nil {
		return payload, err
	}

	resp, err := app.processGet(dbs, TRANSLIB_FMT_YGOT)
	if err == nil {
		if resp.ValueTree == nil {
			err = tlerr.NotFound("app returned nil")
		} else if isEmptyYgotStruct(resp.ValueTree) {
			err = tlerr.NotFound("app returned empty %T", resp.ValueTree)
		} else {
			payload = resp.ValueTree
		}
	}

	return payload, err
}

func (ne *notificationEvent) processSubscribe(nInfo *notificationInfo, subpath string) *gnmi.Path {
	inPath := nInfo.path
	if len(subpath) != 0 { // Include path suffix, if provided
		inPath = path.Copy(nInfo.path)
		_, err := path.AppendPathStr(inPath, subpath)
		if err != nil {
			log.Warningf("[%s] Invalid suffix \"%s\"; err: %v", ne.id, subpath, err)
			return nil
		}
	}

	in := processSubRequest{
		ctxID:  ne.id,
		dbno:   nInfo.dbno,
		table:  nInfo.table,
		key:    ne.key,
		entry:  ne.entry,
		dbs:    ne.sInfo.dbs,
		opaque: nInfo.opaque,
		path:   path.Clone(inPath),
	}

	if log.V(1) {
		log.Infof("[%s] Call processSubscribe with dbno=%d, table=%s, key=%v; subpath=%s",
			ne.id, in.dbno, tableInfo(in.table), keyInfo(in.key), subpath)
	}

	app := ne.getApp(nInfo)
	out, err := app.processSubscribe(in)
	if err != nil {
		log.Warningf("[%s] processSubscribe returned err: %v", ne.id, err)
		return nil
	}

	if out.path == nil {
		log.Warningf("[%s] processSubscribe returned nil path", ne.id)
		return nil
	}

	if !path.Matches(out.path, inPath) {
		log.Warningf("[%s] processSubscribe returned: %s", ne.id, path.String(out.path))
		log.Warningf("[%s] Expected path template   : %s", ne.id, path.String(inPath))
		return nil
	}

	// Trim the output path if it is longer than nInfo.path
	if tLen := path.Len(inPath); path.Len(out.path) > tLen {
		out.path = path.SubPath(out.path, 0, tLen)
	}

	if path.HasWildcardKey(out.path) {
		log.Warningf("[%s] processSubscribe did not resolve all wildcards: \"%s\"",
			ne.id, path.String(out.path))
		return nil
	}

	if log.V(1) {
		log.Infof("[%s] processSubscribe returned: %v", ne.id, out)
	}

	return out.path
}

// sendNotifications prepares and sends one or more notifications for modified fields.
// If niWildcardSubpath flag is set, it groups the fields based on wildcard field prefixes
// and sends one notification for each group. Otherwise sends only one notification message.
func (ne *notificationEvent) sendNotifications(nInfo *notificationInfo, fields []*yangNodeInfo) {
	if !nInfo.flags.Has(niWildcardSubpath) {
		ne.sendNotification(nInfo, "", fields)
		return
	}
	// Optimization for single field case -- avoids grouping logic
	if len(fields) == 1 {
		prefix := fields[0].parentPrefix
		fields[0].parentPrefix = ""
		ne.sendNotification(nInfo, prefix, fields)
		return
	}
	// Group fields by their parent prefix and process each group separately.
	groups := make(map[string][]*yangNodeInfo)
	for _, f := range fields {
		prefix := f.parentPrefix
		f.parentPrefix = ""
		groups[prefix] = append(groups[prefix], f)
	}
	for prefix, groupFields := range groups {
		ne.sendNotification(nInfo, prefix, groupFields)
	}
}

// sendNotification fetches data for a set of fields and sends SubscribeResponse to the
// translib client. relativePrefix is the common prefix for all the fields, relative to nInfo.path.
// It should be empty unless fields have wildcard prefixes. If presnet, processSubscribe is invoked
// to resolve wildcards in nInfo.path+relativePrefix. Fields must have empty parentPrefix.
func (ne *notificationEvent) sendNotification(nInfo *notificationInfo, relativePrefix string, fields []*yangNodeInfo) {
	var prefix *gnmi.Path
	switch {
	case len(relativePrefix) != 0:
		prefix = ne.processSubscribe(nInfo, relativePrefix)
	case nInfo.flags.Has(niWildcardPath) || ne.forceProcessSub:
		prefix = ne.processSubscribe(nInfo, "")
	default:
		prefix = path.Clone(nInfo.path)
	}
	if prefix == nil {
		log.Warningf("[%s] skip notification -- processSubscribe failed", ne.id)
		return
	}

	sInfo := ne.sInfo
	prefixStr, err := ygot.PathToString(prefix)
	if err != nil {
		log.Warningf("[%s] skip notification -- %v", ne.id, err)
		return
	}

	resp := &SubscribeResponse{
		Path:      prefixStr,
		Timestamp: time.Now().UnixNano(),
	}

	log.Infof("[%s] preparing SubscribeResponse for %s", ne.id, prefixStr)
	var updatePaths []string

	for _, lv := range fields {
		leafPath := lv.getPath()

		if lv.deleted {
			log.Infof("[%s] '%s' deleted", ne.id, leafPath)
			resp.Delete = append(resp.Delete, leafPath)
			continue
		}

		data, err := ne.getValue(nInfo, prefixStr+leafPath)

		if sInfo.syncDone && isNotFoundError(err) {
			log.Infof("[%s] '%s' not found", ne.id, leafPath)
			resp.Delete = append(resp.Delete, leafPath)
			continue
		}
		if err != nil {
			log.V(2).Infof("[%s] '%s' skipped due to error: %v", ne.id, leafPath, err)
			continue
		}

		log.Infof("[%s] '%s' updated", ne.id, leafPath)
		if log.V(5) {
			log.Infof("update value = %T %s", data, objPrinter.Sprint(data))
		}
		lv.valueTree = data
		updatePaths = append(updatePaths, leafPath)
	}

	numUpdate := len(updatePaths)
	numDelete := len(resp.Delete)
	if numUpdate == 0 && numDelete == 0 {
		log.Warningf("[%s] skip notification -- no data", ne.id)
		return
	}

	switch {
	case numUpdate == 0:
		// No updates; retain resp.Path=prefixStr and resp.Update=nil
	case numUpdate == 1 && numDelete == 0:
		// There is only one update and no deletes. Overwrite the resp.Path
		// to the parent node (because processGet returns GoStruct for the parent)
		lv, _ := nextYangNodeForUpdate(fields, 0)
		n := path.Len(prefix)
		if nInfo.flags.Has(niLeafPath) {
			resp.Path, _ = path.SplitLastElem(prefixStr)
		} else if !lv.isTargetNode(nInfo) {
			resp.Path = prefixStr + lv.parentPrefix
		} else {
			// Optimization for init sync/entry create of non-leaf target -- use the
			// GoStruct of the target node and retain full target path in resp.Path.
			// This longer prefix will produce more compact notification message.
			cp := path.SubPath(prefix, n-1, n)
			lv.valueTree, err = getYgotAtPath(lv.valueTree, cp)
		}

		resp.Update = lv.valueTree
		log.V(2).Infof("[%s] Single update case; %T", ne.id, resp.Update)

	default:
		// There are > 1 updates or 1 update with few delete paths. Hence retain resp.Path
		// as prefixStr itself. Coalesce the values by merging them into a new data tree.
		tmpRoot := new(ocbinds.Device)
		resp.Update, err = mergeYgotAtPath(tmpRoot, prefix, nil)
		if err != nil {
			break
		}

		log.V(2).Infof("[%s] Coalesce %d updates into %T", ne.id, numUpdate, resp.Update)
		lv, i := nextYangNodeForUpdate(fields, 0)
		for lv != nil && err == nil {
			_, err = mergeYgotAtPathStr(tmpRoot, prefixStr+lv.parentPrefix, lv.valueTree)
			lv, i = nextYangNodeForUpdate(fields, i+1)
		}

		// If resp.Update is a list, clear the key attributes set by ygot APIs.
		// Assumes none of the update paths are list keys -- key change should be treated as
		// instance del+create; and should not land here. If it does, its an app error!
		if err == nil {
			err = clearListKeys(resp.Update)
		}
	}

	if err != nil {
		log.Warningf("[%s] skip notification -- %v", ne.id, err)
		return
	}

	log.Infof("[%s] Sending %d updates and %d deletes", ne.id, numUpdate, numDelete)
	ne.send(resp)
}

func (ne *notificationEvent) send(resp *SubscribeResponse) {
	if log.V(5) {
		log.Infof("[%s] SubscribeResponse %s", ne.id, objPrinter.Sprint(resp))
	}
	if err := ne.sInfo.q.Put(resp); err != nil {
		log.Warningf("[%v] Response queue error: %v", ne.id, err)
	}
}

// pocCallContext holds context info about a ProcessOnChange handler call.
// Will be passed as the NotificationSender argument to the handler.
type pocCallContext struct {
	ne    *notificationEvent
	nInfo *notificationInfo
}

// Send implements apis.NotificationSender interface
func (poc *pocCallContext) Send(n *apis.Notification) {
	if n.Update != nil || len(n.Delete) != 0 {
		poc.send(n)
	}
	// Load each UpdatePaths value and send as separate notification response.
	// Merging them into one requires additional computation of common prefix
	// for UpdatePaths+Delete values (because they may belong to different keys).
	for _, p := range n.UpdatePaths {
		var nu apis.Notification
		if poc.fillUpdate(&nu, n.Path+p) == nil {
			poc.send(&nu)
		}
	}
}

func (poc *pocCallContext) fillUpdate(n *apis.Notification, p string) error {
	data, err := poc.ne.getValue(poc.nInfo, p)
	if err == nil {
		n.Path, _ = path.SplitLastElem(p)
		n.Update = data
		return nil
	}
	if poc.ne.sInfo.syncDone && isNotFoundError(err) {
		log.V(2).Infof("[%s] %s not found (%v)", poc.ne.id, p, err)
		n.Delete = append(n.Delete, p)
		return nil
	}
	log.Warningf("[%s] skip notification for POC UpdatePath '%s'; err=%v", poc.ne.id, p, err)
	return err
}

func (poc *pocCallContext) send(n *apis.Notification) {
	ne := poc.ne
	resp := SubscribeResponse{
		Path:      n.Path,
		Delete:    n.Delete,
		Update:    n.Update,
		Timestamp: time.Now().UnixNano(),
	}

	log.Infof("[%v] Sending %d updates and %d deletes", ne.id, btoi(n.Update != nil), len(n.Delete))
	ne.send(&resp)
}

func btoi(b bool) int {
	if b {
		return 1
	}
	return 0
}

func nextYangNodeForUpdate(nodes []*yangNodeInfo, indx int) (*yangNodeInfo, int) {
	for n := len(nodes); indx < n; indx++ {
		if nodes[indx].valueTree != nil {
			return nodes[indx], indx
		}
	}
	return nil, -1
}

// yangNodeInfoSet contains yangNodeInfo mappings for old and new db entries.
// Old mappings usually include db entry delete operations. New mappings
// include entry create or update operations (including field delete).
type yangNodeInfoSet struct {
	old []*yangNodeInfo
	new []*yangNodeInfo
}

// yangNodeInfo holds path and value for a yang leaf
type yangNodeInfo struct {
	parentPrefix string
	leafName     string
	deleted      bool
	valueTree    ygot.ValidatedGoStruct
}

func (lv *yangNodeInfo) getPath() string {
	if len(lv.leafName) == 0 {
		return lv.parentPrefix
	}
	return lv.parentPrefix + "/" + lv.leafName
}

// isTargetLeaf checks if this yang node is the target path of the notificationInfo.
func (lv *yangNodeInfo) isTargetNode(nInfo *notificationInfo) bool {
	return len(lv.parentPrefix) == 0 && len(lv.leafName) == 0
}
