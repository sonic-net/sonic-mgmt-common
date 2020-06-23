package translib

import (
    "fmt"
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

// Set parses a version string in X.Y.Z into a Version object v.
func (v *Version) Set(s string) error {
    _, err := fmt.Sscanf(s, "%d.%d.%d", &v.Major, &v.Minor, &v.Patch)
    if err == nil {
        err = fmt.Errorf("Invalid version \"%s\"", s)
    }   

    return err 
}
