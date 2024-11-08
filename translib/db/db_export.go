////////////////////////////////////////////////////////////////////////////////
//                                                                            //
//  Copyright 2022 Broadcom. The term Broadcom refers to Broadcom Inc. and/or //
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

package db

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Export writes the full DB contents to a file in sonic db json format.
// Includes contents from transaction cache, if present. Uses the sonic-cfggen
// tool to remove system default entries/keys and apply standard format.
//
// If filePath is empty or has '*', it will be expanded to a random name similar to
// the os.CreateTemp() API. Actual file path will be returned to the caller (outFile).
// Contents will be written and formatted in multiple steps. Hence outFile may a valid
// path (but with garbage contents) even if there was an error. Caller must disregard
// its contents and cleanup the file when outFile != "" && err != nil.
func (d *DB) Export(filePath string) (outFile string, err error) {
	if d.Opts.DBNo != ConfigDB {
		return "", fmt.Errorf("Export not supported on %v", d.Opts.DBNo)
	}
	var rawDump string
	if len(d.txCmds) != 0 { // avoid temporary raw dump if there are no tx cache
		rawDump, err = d.ExportRaw("db_dump_*.json")
		if len(rawDump) != 0 {
			defer os.Remove(rawDump)
		}
	}
	if err == nil {
		outFile, err = generateDbDump(filePath, rawDump)
	}
	return
}

// ExportRaw is similar to Export(), but does not process the output using sonic-cfggen tool.
func (d *DB) ExportRaw(filePath string) (outFile string, err error) {
	// Load everything from DB+txCache
	var tables map[TableSpec]Table
	opts := GetConfigOptions{AllowWritable: true}
	tables, err = d.GetConfig([]*TableSpec{}, &opts)
	if err != nil {
		return
	}

	// Prepare to db json map -- {"TABLE":{"KEY":{"FIELD": "VALUE", ...}, ...}, ...}
	jData := make(map[string]map[string]map[string]interface{})
	for ts, table := range tables {
		entryMap := make(map[string]map[string]interface{})
		keys, _ := table.GetKeys()
		for _, key := range keys {
			entry, _ := table.GetEntry(key)
			entryKey := strings.Join(key.Comp, d.Opts.KeySeparator)
			values := make(map[string]interface{}, len(entry.Field))
			for k, v := range entry.Field {
				switch {
				case k == "NULL": // skip the dummy NULL field
				case k[len(k)-1] == '@': // split leaf-list
					values[k[:len(k)-1]] = strings.Split(v, ",")
				default:
					values[k] = v
				}
			}
			entryMap[entryKey] = values
		}
		jData[ts.Name] = entryMap
	}

	// Open file for writing the DB contents
	var f *os.File
	f, err = createFile(filePath)
	if err != nil {
		err = fmt.Errorf("Failed to create dump file: %w", err)
		return
	}

	defer f.Close()
	outFile = f.Name()
	f.Chmod(0664) // make it readable for everyone

	// Dump db json to f, no pretty print
	err = json.NewEncoder(f).Encode(jData)
	if err != nil {
		err = fmt.Errorf("Failed to write dump file: %w", err)
	}
	return
}

// generateDbDump runs sonic-cfggen tool to generate db dump file using data
// from a json file (rawDumpFile) or from config_db (if rawDumpFile=="").
func generateDbDump(filePath, rawDumpFile string) (string, error) {
	f, err := createFile(filePath)
	if err != nil {
		return "", fmt.Errorf("Failed to create dump file: %w", err)
	}

	defer f.Close()
	f.Chmod(0664)

	args := make([]string, 0, 3)
	args = append(args, "--print-data")
	if len(rawDumpFile) != 0 {
		args = append(args, "-j", rawDumpFile)
	} else {
		args = append(args, "-d")
	}

	// Process using sonic-cfggen -- removes system defaults (copp, breakout etc) and pretty print
	cmd := exec.Command("sonic-cfggen", args...)
	cmd.Stdout = f // redirect stdout to f
	err = cmd.Run()
	if err != nil {
		err = fmt.Errorf("sonic-cfggen failed: %w", err)
	}

	return f.Name(), err
}

func createFile(template string) (*os.File, error) {
	dirname, basename := filepath.Split(template)
	if len(basename) == 0 || strings.IndexByte(basename, '*') >= 0 {
		//TODO change to os.CreateTemp after upgrading to the latest golang
		return ioutil.TempFile(dirname, basename)
	}
	return os.Create(template)
}
