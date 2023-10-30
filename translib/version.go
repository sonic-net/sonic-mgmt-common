////////////////////////////////////////////////////////////////////////////////
//                                                                            //
//  Copyright 2020 Broadcom. The term Broadcom refers to Broadcom Inc. and/or //
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
	"encoding/xml"
	"fmt"
	"github.com/Azure/sonic-mgmt-common/translib/tlerr"
	"github.com/golang/glog"
	"os"
	"path/filepath"
)

// theYangBundleVersion indicates the current yang bundle version.
// It is read from the version.xml config file.
var theYangBundleVersion Version

// theYangBaseVersion indicates the minimum yang bundle version
// supported by translib APIs. This will be usually 1 major version
// below current ynag bundle version {theYangBundleVersion.Major-1, 0, 0}
var theYangBaseVersion Version

// Version represents the semantic version number in Major.Minor.Patch
// format.
type Version struct {
	Major uint32 // Major version number
	Minor uint32 // Minor version number
	Patch uint32 // Patch number
}

// NewVersion creates a Version object from given version string
func NewVersion(s string) (Version, error) {
	var v Version
	err := v.Set(s)
	return v, err
}

func (v Version) String() string {
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}

// Set parses a version string in X.Y.Z into a Version object v.
func (v *Version) Set(s string) error {
	_, err := fmt.Sscanf(s, "%d.%d.%d", &v.Major, &v.Minor, &v.Patch)
	if err == nil && v.IsNull() {
		err = fmt.Errorf("Invalid version \"%s\"", s)
	}

	return err
}

// IsNull checks if the version v is a null version (0.0.0)
func (v *Version) IsNull() bool {
	return v == nil || (v.Major == 0 && v.Minor == 0 && v.Patch == 0)
}

// GreaterThan checks if the Version v is more than another version
func (v *Version) GreaterThan(other Version) bool {
	return (v.Major > other.Major) ||
		(v.Major == other.Major && v.Minor > other.Minor) ||
		(v.Major == other.Major && v.Minor == other.Minor && v.Patch > other.Patch)
}

// GetCompatibleBaseVersion returns the compatible base version for
// current version.
func (v *Version) GetCompatibleBaseVersion() Version {
	switch v.Major {
	case 0:
		// 0.x.x versions are always considered as developement versions
		// and are not backward compatible. So, base = current
		return *v
	case 1:
		// Base is 1.0.0 if yang bundle version is 1.x.x
		return Version{Major: 1}
	default:
		// For 2.x.x or higher versions the base will be (N-1).0.0
		return Version{Major: v.Major - 1}
	}
}

// validateClientVersion verifies API client client is compatible with this server.
func validateClientVersion(clientVer Version, path string, appInfo *appInfo) error {
	if clientVer.IsNull() {
		// Client did not povide version info
		return nil
	}

	if appInfo.isNative {
		return validateClientVersionForNonYangAPI(clientVer, path, appInfo)
	}

	return validateClientVersionForYangAPI(clientVer, path, appInfo)
}

// validateClientVersionForYangAPI checks theYangBaseVersion <= clientVer <= theYangBundleVersion
func validateClientVersionForYangAPI(clientVer Version, path string, appInfo *appInfo) error {
	if theYangBaseVersion.GreaterThan(clientVer) {
		glog.Errorf("Client version %s is less than base version %s", clientVer, theYangBaseVersion)
	} else if clientVer.GreaterThan(theYangBundleVersion) {
		glog.Errorf("Client version %s is more than server vesion %s", clientVer, theYangBundleVersion)
	} else {
		return nil
	}

	return tlerr.TranslibUnsupportedClientVersion{
		ClientVersion:     clientVer.String(),
		ServerVersion:     theYangBundleVersion.String(),
		ServerBaseVersion: theYangBaseVersion.String(),
	}
}

// validateClientVersionForNonYangAPI checks client version for non-yang APIs
func validateClientVersionForNonYangAPI(clientVer Version, path string, appInfo *appInfo) error {
	// Version checks are not supported for non-yang APIs.
	// Client versions are ignored.
	return nil
}

//========= initialization =========

func init() {
	path := filepath.Join(GetYangPath(), "version.xml")
	if err := initYangVersionConfigFromFile(path); err != nil {
		glog.Errorf("Error loading version config file; err=%v", err)
		glog.Errorf("API VERSION CHECK IS DISABLED")
	} else {
		glog.Infof("*** Yang bundle version = %s", theYangBundleVersion)
		glog.Infof("*** Yang base version   = %s", theYangBaseVersion)
	}
}

// Load version config from file
func initYangVersionConfigFromFile(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}

	defer f.Close()

	var configData struct {
		YangBundleVersion Version `xml:"yang-bundle-version"`
	}

	err = xml.NewDecoder(f).Decode(&configData)
	if err != nil {
		return err
	}

	theYangBundleVersion = configData.YangBundleVersion
	theYangBaseVersion = theYangBundleVersion.GetCompatibleBaseVersion()

	return nil
}

// GetYangBundleVersion returns the API version for yang bundle hosted
// on this server.
func GetYangBundleVersion() Version {
	return theYangBundleVersion
}

// GetYangBaseVersion returns the base version or min version of yang APIs
// supported by this server.
func GetYangBaseVersion() Version {
	return theYangBaseVersion
}
