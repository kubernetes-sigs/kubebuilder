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
	"os/exec"
	"regexp"
	"strings"
)

// Minimum required kustomize version
var (
	minKustomizeVersion = semanticVersion{
		major: 3,
		minor: 1,
	}
)

const (
	versionFieldName = "Version"

	kustomizePairPattern       = `\w+:[\w-:\./]+`
	kustomizeVersionCmdPattern = `^(?:Version: )?\{(` + kustomizePairPattern + `(?: ` + kustomizePairPattern + `))\}$`
	kustomizeVersionRawPattern = `^kustomize/` + semanticVersionPattern + `$`
)

var (
	kustomizeVersionCmdRegexp = regexp.MustCompile(kustomizeVersionCmdPattern)
	kustomizeVersionRawRegexp = regexp.MustCompile(kustomizeVersionRawPattern)
)

type kustomizeInfo struct {
	version semanticVersion
}

func (info kustomizeInfo) isValidVersion() bool {
	return info.version.isEqualOrGreaterThan(minKustomizeVersion)
}

func newKustomizeInfo(str string) (info kustomizeInfo, err error) {
	// kustomizeVersionCmdRegexp has only one capture group
	match := kustomizeVersionCmdRegexp.FindStringSubmatch(str)[1]
	pairs := strings.Split(match, " ")
	var mapping = make(map[string]string, len(pairs))
	for _, pair := range pairs {
		splitPair := strings.SplitN(pair, " ", 2)
		if len(splitPair) >= 2 {
			mapping[splitPair[0]] = splitPair[1]
		}
	}

	version, found := mapping[versionFieldName]
	if !found {
		err = fmt.Errorf("unexpected format")
		return
	}

	info.version, err = newSemanticVersion(FindStringSubmatch(kustomizeVersionRawRegexp, version))

	return
}

// CheckKustomize verifies that the available kustomize command is available and up to date
func CheckKustomize() error {
	cmd := exec.Command("kustomize", "version")
	out, err := cmd.Output()
	if err != nil {
		return err
	}

	return checkKustomize(string(out))
}

// checkCustomize is used for tests
func checkKustomize(str string) error {
	info, err := newKustomizeInfo(str)
	if err != nil {
		return CmdParseError{"kustomize version", err}
	}

	if !info.isValidVersion() {
		return RequiredVersionError{"kustomize", info.version, minKustomizeVersion}
	}

	return nil
}
