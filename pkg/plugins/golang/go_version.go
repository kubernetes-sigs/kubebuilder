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

var (
	go113 = goVersion{
		major: 1,
		minor: 13,
	}
	goVerMax = goVersion{
		major:      1,
		minor:      17,
		prerelease: "alpha1",
	}

	goVerRegexp = regexp.MustCompile(goVerPattern)
)

type goVersion struct {
	major, minor, patch int
	prerelease          string
}

func (v *goVersion) parse(verStr string) error {
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

func (v goVersion) compare(other goVersion) int {
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

// ValidateGoVersion verifies that Go is installed and the current go version is supported by kubebuilder
func ValidateGoVersion() error {
	err := fetchAndCheckGoVersion()
	if err != nil {
		return fmt.Errorf("%s. You can skip this check using the --skip-go-version-check flag", err)
	}
	return nil
}

func fetchAndCheckGoVersion() error {
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
	if err := checkGoVersion(goVer); err != nil {
		return fmt.Errorf("go version '%s' is incompatible because '%s'", goVer, err)
	}
	return nil
}

// checkGoVersion should only ever check if the Go version >= 1.13, since the kubebuilder binary only cares
// that the go binary supports go modules which were stabilized in that version (i.e. in go 1.13) by default
func checkGoVersion(verStr string) error {
	var version goVersion
	if err := version.parse(verStr); err != nil {
		return err
	}

	if version.compare(go113) < 0 || version.compare(goVerMax) >= 0 {
		return fmt.Errorf("requires 1.13 <= version < 1.17")
	}

	return nil
}
