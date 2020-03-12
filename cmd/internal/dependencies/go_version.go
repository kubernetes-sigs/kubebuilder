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
	"os/exec"
	"regexp"
	"runtime"
	"strings"
)

// Minimum required go version
var (
	minGoVersion = semanticVersion{
		major: 1,
		minor: 11,
	}
)

const (
	goVersionStringPattern = `go` + semanticVersionPattern
	goVersionOsArchPattern = `(?P<os>\w+)/(?P<arch>\w+)`

	goVersionRawPattern = `^` + goVersionStringPattern + `$`
	goVersionCmdPattern = `^go version ` + goVersionStringPattern + ` ` + goVersionOsArchPattern + `$`
)

var (
	goVersionRawRegexp = regexp.MustCompile(goVersionRawPattern)
	goVersionCmdRegexp = regexp.MustCompile(goVersionCmdPattern)

	compilationGoInfo = goBuildInfo{
		os:   runtime.GOOS,
		arch: runtime.GOARCH,
	}
)

func init() {
	var err error
	compilationGoInfo.version, err = newSemanticVersion(FindStringSubmatch(goVersionRawRegexp, runtime.Version()))
	if err != nil {
		panic(err)
	}
}

type goBuildInfo struct {
	version semanticVersion
	os      string
	arch    string
}

func (info goBuildInfo) isSameAsCompiled() bool {
	return info == compilationGoInfo
}

func (info goBuildInfo) isValidVersion() bool {
	return info.version.isEqualOrGreaterThan(minGoVersion)
}

func newGoInfo(str string) (info goBuildInfo, err error) {
	parts := FindStringSubmatch(goVersionCmdRegexp, str)
	info.version, err = newSemanticVersion(parts)
	if err != nil {
		return
	}
	info.os = parts["os"]
	info.arch = parts["arch"]

	return
}

// CheckGo verifies that the available go command is available and up to date
func CheckGo() error {
	cmd := exec.Command("go", "version")
	out, err := cmd.Output()
	if err != nil {
		return err
	}

	return checkGo(strings.TrimSpace(string(out)), false)
}

// checkCustomize is used for tests
func checkGo(str string, skipCompileCheck bool) error {
	info, err := newGoInfo(str)
	if err != nil {
		return CmdParseError{"go version", err}
	}

	if !info.isValidVersion() {
		return RequiredVersionError{"go", info.version, minGoVersion}
	}

	// This errors has to be reported last as we may want to skip it for backwards compatibility
	if !skipCompileCheck && !info.isSameAsCompiled() {
		return CompilationVersionMatchError{info}
	}

	return nil
}
