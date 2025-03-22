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
	"runtime/debug"
)

const unknown = "unknown"

// var needs to be used instead of const as ldflags is used to fill this
// information in the release process
var (
	kubeBuilderVersion      = unknown
	kubernetesVendorVersion = "1.32.1"
	goos                    = unknown
	goarch                  = unknown
	gitCommit               = "$Format:%H$" // sha1 from git, output of $(git rev-parse HEAD)

	buildDate = "1970-01-01T00:00:00Z" // build date in ISO8601 format, output of $(date -u +'%Y-%m-%dT%H:%M:%SZ')
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
	if kubeBuilderVersion == unknown {
		if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" {
			kubeBuilderVersion = info.Main.Version
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
