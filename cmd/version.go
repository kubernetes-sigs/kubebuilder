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
	"encoding/json"
	"fmt"
	"runtime"
	"runtime/debug"
)

const unknown = "unknown"

// These are filled via ldflags during build
var (
	kubeBuilderVersion      = unknown
	kubernetesVendorVersion = "1.33.0"
	goos                    = unknown
	goarch                  = unknown
	gitCommit               = "$Format:%H$"
	buildDate               = "1970-01-01T00:00:00Z"
)

// VersionInfo holds all CLI version-related information
type VersionInfo struct {
	KubeBuilderVersion string `json:"kubeBuilderVersion"`
	KubernetesVendor   string `json:"kubernetesVendor"`
	GitCommit          string `json:"gitCommit"`
	BuildDate          string `json:"buildDate"`
	GoOS               string `json:"goOs"`
	GoArch             string `json:"goArch"`
}

// resolveBuildInfo ensures dynamic fields are populated
func resolveBuildInfo() {
	if kubeBuilderVersion == unknown {
		if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" {
			kubeBuilderVersion = info.Main.Version
		}
	}
	if goos == unknown {
		goos = runtime.GOOS
	}
	if goarch == unknown {
		goarch = runtime.GOARCH
	}
	if gitCommit == "$Format:%H$" || gitCommit == "" {
		gitCommit = unknown
	}
}

// getVersionInfo returns populated VersionInfo
func getVersionInfo() VersionInfo {
	resolveBuildInfo()
	return VersionInfo{
		KubeBuilderVersion: kubeBuilderVersion,
		KubernetesVendor:   kubernetesVendorVersion,
		GitCommit:          gitCommit,
		BuildDate:          buildDate,
		GoOS:               goos,
		GoArch:             goarch,
	}
}

// versionString returns a human-friendly string version
func versionString() string {
	v := getVersionInfo()
	return fmt.Sprintf(`KubeBuilder Version: %s
Kubernetes Vendor:   %s
Git Commit:          %s
Build Date:          %s
Go OS/Arch:          %s/%s`,
		v.KubeBuilderVersion,
		v.KubernetesVendor,
		v.GitCommit,
		v.BuildDate,
		v.GoOS,
		v.GoArch,
	)
}

// getKubeBuilderVersion returns just the CLI version
func getKubeBuilderVersion() string {
	resolveBuildInfo()
	return kubeBuilderVersion
}

// versionJSON returns version as JSON string
func versionJSON() string {
	v := getVersionInfo()
	b, _ := json.MarshalIndent(v, "", "  ")
	return string(b)
}
