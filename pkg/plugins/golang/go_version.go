/*
Copyright 2018 The Kubernetes Authors.

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

package golang

import (
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

const (
	goVerPattern = `^go(?P<major>[0-9]+)\.(?P<minor>[0-9]+)(?:\.(?P<patch>[0-9]+)|(?P<pre>(?:alpha|beta|rc)[0-9]+))?$`
)

var goVerRegexp = regexp.MustCompile(goVerPattern)

// GoVersion describes a Go version.
type GoVersion struct {
	major, minor, patch int
	prerelease          string
}

func (v GoVersion) String() string {
	switch {
	case v.patch != 0:
		return fmt.Sprintf("go%d.%d.%d", v.major, v.minor, v.patch)
	case v.prerelease != "":
		return fmt.Sprintf("go%d.%d%s", v.major, v.minor, v.prerelease)
	}
	return fmt.Sprintf("go%d.%d", v.major, v.minor)
}

// MustParse will panic if verStr does not match the expected Go version string spec.
func MustParse(verStr string) (v GoVersion) {
	if err := v.parse(verStr); err != nil {
		panic(err)
	}
	return v
}

func (v *GoVersion) parse(verStr string) error {
	m := goVerRegexp.FindStringSubmatch(verStr)
	if m == nil {
		return fmt.Errorf("invalid version string")
	}

	var err error

	v.major, err = strconv.Atoi(m[1])
	if err != nil {
		return fmt.Errorf("error parsing major version '%s': %s", m[1], err)
	}

	v.minor, err = strconv.Atoi(m[2])
	if err != nil {
		return fmt.Errorf("error parsing minor version '%s': %s", m[2], err)
	}

	if m[3] != "" {
		v.patch, err = strconv.Atoi(m[3])
		if err != nil {
			return fmt.Errorf("error parsing patch version '%s': %s", m[2], err)
		}
	}

	v.prerelease = m[4]

	return nil
}

// Compare returns -1, 0, or 1 if v < other, v == other, or v > other, respectively.
func (v GoVersion) Compare(other GoVersion) int {
	if v.major > other.major {
		return 1
	}
	if v.major < other.major {
		return -1
	}

	if v.minor > other.minor {
		return 1
	}
	if v.minor < other.minor {
		return -1
	}

	if v.patch > other.patch {
		return 1
	}
	if v.patch < other.patch {
		return -1
	}

	if v.prerelease == other.prerelease {
		return 0
	}
	if v.prerelease == "" {
		return 1
	}
	if other.prerelease == "" {
		return -1
	}
	if v.prerelease > other.prerelease {
		return 1
	}
	return -1
}

// ValidateGoVersion verifies that Go is installed and the current go version is supported by a plugin.
func ValidateGoVersion(minVersion, maxVersion GoVersion) error {
	err := fetchAndCheckGoVersion(minVersion, maxVersion)
	if err != nil {
		return fmt.Errorf("%s. You can skip this check using the --skip-go-version-check flag", err)
	}
	return nil
}

func fetchAndCheckGoVersion(minVersion, maxVersion GoVersion) error {
	cmd := exec.Command("go", "version")
	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to retrieve 'go version': %v", string(out))
	}

	split := strings.Split(string(out), " ")
	if len(split) < 3 {
		return fmt.Errorf("found invalid Go version: %q", string(out))
	}
	goVer := split[2]
	if err := checkGoVersion(goVer, minVersion, maxVersion); err != nil {
		return fmt.Errorf("go version '%s' is incompatible because '%s'", goVer, err)
	}
	return nil
}

func checkGoVersion(verStr string, minVersion, maxVersion GoVersion) error {
	var version GoVersion
	if err := version.parse(verStr); err != nil {
		return err
	}

	if version.Compare(minVersion) < 0 || version.Compare(maxVersion) >= 0 {
		return fmt.Errorf("plugin requires %s <= version < %s", minVersion, maxVersion)
	}

	return nil
}
