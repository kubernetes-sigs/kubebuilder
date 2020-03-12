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

package dependencies

import (
	"fmt"
	"strconv"
)

const semanticVersionPattern = `v?(?P<major>\d+)(?:\.(?P<minor>\d+)(?:\.(?P<patch>\d+))?)?(?P<extra>[\w\.-]+)?`

type semanticVersion struct {
	major int
	minor int
	patch int
	extra string
}

func (version semanticVersion) String() string {
	str := fmt.Sprintf("%d", version.major)
	if version.minor != 0 {
		str += fmt.Sprintf(".%d", version.minor)
		if version.patch != 0 {
			str += fmt.Sprintf(".%d", version.patch)
		}
	}
	str += version.extra
	return str
}

func newSemanticVersion(parts map[string]string) (version semanticVersion, err error) {
	if major, found := parts["major"]; found && major != "" {
		version.major, err = strconv.Atoi(major)
		if err != nil {
			return
		}
	}

	if minor, found := parts["minor"]; found && minor != "" {
		version.minor, err = strconv.Atoi(minor)
		if err != nil {
			return
		}
	}

	if patch, found := parts["patch"]; found && patch != "" {
		version.patch, err = strconv.Atoi(patch)
		if err != nil {
			return
		}
	}

	if extra, found := parts["extra"]; found && extra != "" {
		version.extra = extra
	}

	return
}

func (version semanticVersion) isEqualOrGreaterThan(other semanticVersion) bool {
	// Check major version
	if version.major < other.major {
		return false
	}
	if version.major > other.major {
		return true
	}

	// Mayor is equal, check minor version
	if version.minor < other.minor {
		return false
	}
	if version.minor > other.minor {
		return true
	}

	// Mayor and minor are equal, check patch
	if version.patch < other.patch {
		return false
	}
	if version.patch > other.patch {
		return true
	}

	// Mayor, minor and patch are equal, check extra
	if version.extra == other.extra {
		return true
	}
	// Extra is used for pre-releases, so a version with extra goes before than
	// the same version without extra
	if len(version.extra) == 0 && len(other.extra) != 0 {
		return true
	}
	if len(version.extra) != 0 && len(other.extra) == 0 {
		return false
	}
	// TODO: compare extra's if both are set and different
	panic(fmt.Errorf("comparison between %s and %s not implemented", version.extra, other.extra))
}
