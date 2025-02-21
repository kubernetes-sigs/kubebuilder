/*
Copyright 2017 The Kubernetes Authors.

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

package cmd

import (
	"fmt"
	"runtime"
	"runtime/debug"
	"strings"
	"time"
)

const unknown = "unknown"

// var needs to be used instead of const as ldflags is used to fill this
// information in the release process
var (
	kubeBuilderVersion      = unknown
	kubernetesVendorVersion = "1.31.0"
	goos                    = unknown
	goarch                  = unknown
	gitCommit               = unknown // "$Format:%H$" sha1 from git, output of $(git rev-parse HEAD)
	buildDate               = unknown // "1970-01-01T00:00:00Z" build date in ISO8601 format, output of $(date -u +'%Y-%m-%dT%H:%M:%SZ')
)

// version contains all the information related to the CLI version
type version struct {
	KubeBuilderVersion string `json:"kubeBuilderVersion"`
	KubernetesVendor   string `json:"kubernetesVendor"`
	GitCommit          string `json:"gitCommit"`
	BuildDate          string `json:"buildDate"`
	GoOs               string `json:"goOs"`
	GoArch             string `json:"goArch"`
}

// versionString returns the CLI version
func versionString() string {
	info, ok := debug.ReadBuildInfo()

	if ok && info.Main.Version != "" {
		if kubeBuilderVersion == unknown {
			kubeBuilderVersion = info.Main.Version
		}

		if goos == unknown {
			goos = runtime.GOOS
		}

		if goarch == unknown {
			goarch = runtime.GOARCH
		}

		if gitCommit == unknown && info.Main.Version != "" {
			mainVersionSplit := strings.Split(info.Main.Version, "-")

			// For released semvers like "v4.5.0"
			// Result: info.Main.Version == "semver"
			if len(mainVersionSplit) == 1 {
				gitCommit = info.Main.Version
			}

			// For unreleased refs like "<commit-hash>"
			// Result (go install): info.Main.Version == "<semver>-<build-date>-<commit-hash>" E.g "v4.5.1-0.20250121092837-7ee23df2b97c"
			if len(mainVersionSplit) == 3 {
				gitCommit = mainVersionSplit[2]

				buildDateFromVersion := func() string {
					buildDatesplit := strings.Split(
						mainVersionSplit[1],
						".",
					)

					if len(buildDatesplit) == 2 {
						return buildDatesplit[1]
					}

					return buildDatesplit[0]
				}()

				// format build date
				if t, err := time.Parse("20060102150405", buildDateFromVersion); err == nil {
					buildDate = t.Format(time.RFC3339)
				}
			}
		}
	}

	return fmt.Sprintf("Version: %#v", version{
		kubeBuilderVersion,
		kubernetesVendorVersion,
		gitCommit,
		buildDate,
		goos,
		goarch,
	})
}
