/*
Copyright 2020 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package plugin

import (
	"errors"
	"fmt"
	"path"
	"regexp"
	"strconv"
	"strings"

	"sigs.k8s.io/kubebuilder/pkg/internal/validation"
)

// DefaultNameQualifier is the suffix appended to all kubebuilder plugin names.
const DefaultNameQualifier = ".kubebuilder.io"

// Valid stage values.
const (
	// AlphaStage should be used for plugins that are frequently changed and may break between uses.
	AlphaStage = "alpha"
	// BetaStage should be used for plugins that are only changed in minor ways, ex. bug fixes.
	BetaStage = "beta"
)

// verRe defines the string format of a version.
var verRe = regexp.MustCompile("^(v)?([1-9][0-9]*)(-alpha|-beta)?$")

// Version is a plugin version containing a non-zero integer and an optional stage value
// that if present identifies a version as not stable to some degree.
type Version struct {
	// Number denotes the current version of a plugin. Two different numbers between versions
	// indicate that they are incompatible.
	Number int64
	// Stage indicates stability, and must be "alpha" or "beta".
	Stage string
}

func (v Version) String() string {
	if v.Stage != "" {
		return fmt.Sprintf("v%v-%s", v.Number, v.Stage)
	}
	return fmt.Sprintf("v%v", v.Number)
}

// Validate ensures v contains a number and if it has a stage suffix that it is either "alpha" or "beta".
func (v Version) Validate() error {
	if v.Number < 1 {
		return errors.New("integer value cannot be or be less than 0")
	}
	if v.Stage != "" && v.Stage != AlphaStage && v.Stage != BetaStage {
		return errors.New(`suffix must be "alpha" or "beta"`)
	}
	return nil
}

// ParseVersion parses version into a Version, assuming it adheres to format: (v)?[1-9][0-9]*(-(alpha|beta))?
func ParseVersion(version string) (v Version, err error) {
	if version == "" {
		return v, errors.New("plugin version is empty")
	}

	// A valid version string will have 4 submatches, each of which may be empty: the full string, "v",
	// the integer, and the stage suffix. Invalid version strings do not have 4 submatches.
	submatches := verRe.FindStringSubmatch(version)
	if len(submatches) != 4 {
		return v, fmt.Errorf("version format must match %s", verRe.String())
	}

	// Parse version number.
	versionNumStr := submatches[2]
	if versionNumStr == "" {
		return v, errors.New("version must contain an integer")
	}
	if v.Number, err = strconv.ParseInt(versionNumStr, 10, 64); err != nil {
		return v, err
	}

	// Parse stage suffix, if any.
	v.Stage = strings.TrimPrefix(submatches[3], "-")

	return v, v.Validate()
}

// Compare returns -1 if v < vp, 0 if v == vp, and 1 if v > vp.
func (v Version) Compare(vp Version) int {
	if v.Number == vp.Number {
		s, sp := v.Stage, vp.Stage
		if s == sp {
			return 0
		}
		// Since stages are not equal, check: stable > beta > alpha.
		if s == "" || (s == BetaStage && sp == AlphaStage) {
			return 1
		}
	} else if v.Number > vp.Number {
		return 1
	}
	return -1
}

// Key returns a unique identifying string for a plugin's name and version.
func Key(name, version string) string {
	if version == "" {
		return name
	}
	return path.Join(name, "v"+strings.TrimLeft(version, "v"))
}

// KeyFor returns a Base plugin's unique identifying string.
func KeyFor(p Base) string {
	return Key(p.Name(), p.Version().String())
}

// SplitKey returns a name and version for a plugin key.
func SplitKey(key string) (string, string) {
	if !strings.Contains(key, "/") {
		return key, ""
	}
	keyParts := strings.SplitN(key, "/", 2)
	return keyParts[0], keyParts[1]
}

// GetShortName returns plugin's short name (name before domain) if name
// is fully qualified (has a domain suffix), otherwise GetShortName returns name.
func GetShortName(name string) string {
	return strings.SplitN(name, ".", 2)[0]
}

// ValidateName ensures name is a valid DNS 1123 subdomain.
func ValidateName(name string) error {
	if errs := validation.IsDNS1123Subdomain(name); len(errs) != 0 {
		return fmt.Errorf("invalid plugin name %q: %v", name, errs)
	}
	return nil
}
