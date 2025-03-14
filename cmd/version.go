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
)

const unknown = "unknown"

// var needs to be used instead of const as ldflags is used to fill this
// information in the release process
var (
	kubeBuilderVersion      = unknown
	kubernetesVendorVersion = "1.32.1"
	goVersion               = unknown
	goOs                    = unknown
	goArch                  = unknown
	gitCommit               = unknown // "$Format:%H$" sha1 from git, output of $(git rev-parse HEAD)

	// "1970-01-01T00:00:00Z" build date in ISO8601 format
	// Output of $(date -u +'%Y-%m-%dT%H:%M:%SZ')
	buildDate = unknown
)

// version contains all the information related to the CLI version
type version struct {
	KubeBuilderVersion string `json:"kubeBuilderVersion"`
	KubernetesVendor   string `json:"kubernetesVendor"`
	GoVersion          string `json:"goVersion"`
	GoOs               string `json:"goOs"`
	GoArch             string `json:"goArch"`
	GitCommit          string `json:"gitCommit"`
	BuildDate          string `json:"buildDate"`
}

// versionString returns the CLI version
func versionString() string {
	if goVersion == unknown {
		goVersion = runtime.Version()
	}

	if goOs == unknown {
		goOs = runtime.GOOS
	}

	if goArch == unknown {
		goArch = runtime.GOARCH
	}

	info, ok := debug.ReadBuildInfo()

	if ok {
		if info.Main.Version != "" {
			if kubeBuilderVersion == unknown {
				kubeBuilderVersion = info.Main.Version
			}
		}

		for _, setting := range info.Settings {
			if buildDate == unknown && setting.Key == "vcs.revision" {
				buildDate = setting.Value
			}

			if gitCommit == unknown && setting.Key == "vcs.revision" {
				gitCommit = setting.Value
			}
		}
	}

	return fmt.Sprintf("Version: %#v", version{
		kubeBuilderVersion,
		kubernetesVendorVersion,
		goVersion,
		goOs,
		goArch,
		gitCommit,
		buildDate,
	})
}
