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

package version

import (
	"fmt"
	"os"
	"runtime"
	"runtime/debug"
	"strings"
)

const (
	unknown                 = "unknown"
	develVersion            = "(devel)"
	kubernetesVendorVersion = "1.35.0"
)

type Version struct {
	KubeBuilderVersion string `json:"kubeBuilderVersion"`
	KubernetesVendor   string `json:"kubernetesVendor"`
	GitCommit          string `json:"gitCommit"`
	BuildDate          string `json:"buildDate"`
	GoOs               string `json:"goOs"`
	GoArch             string `json:"goArch"`
}

func New() Version {
	v := Version{
		KubeBuilderVersion: develVersion,
		KubernetesVendor:   kubernetesVendorVersion,
		GitCommit:          unknown,
		BuildDate:          unknown,
		GoOs:               runtime.GOOS,
		GoArch:             runtime.GOARCH,
	}

	if info, ok := debug.ReadBuildInfo(); ok {
		v.KubeBuilderVersion = resolveMainVersion(info.Main)
		v.applyVCSMetadata(info.Settings)
	}

	if testVersion := os.Getenv("KUBEBUILDER_TEST_VERSION"); testVersion != "" {
		v.KubeBuilderVersion = testVersion
		v.GitCommit = "test-commit"
		v.BuildDate = "1970-01-01T00:00:00Z"
	}

	return v
}

// GetKubeBuilderVersion returns only the CLI version string.
// Used for the cliVersion field in scaffolded PROJECT files.
func (v Version) GetKubeBuilderVersion() string {
	return strings.TrimPrefix(v.KubeBuilderVersion, "v")
}

func resolveMainVersion(main debug.Module) string {
	if main.Version != "" {
		return main.Version
	}
	return develVersion
}

func (v *Version) applyVCSMetadata(settings []debug.BuildSetting) {
	var isDirty bool

	for _, s := range settings {
		switch s.Key {
		case "vcs.revision":
			v.GitCommit = s.Value
		case "vcs.time":
			v.BuildDate = s.Value
		case "vcs.modified":
			isDirty = (s.Value == "true")
		}
	}

	if isDirty {
		if !strings.Contains(v.KubeBuilderVersion, "dirty") {
			v.KubeBuilderVersion += "-dirty"
		}

		if !strings.Contains(v.GitCommit, "dirty") {
			v.GitCommit += "-dirty"
		}
	}
}

func (v Version) PrintVersion() string {
	return fmt.Sprintf(`KubeBuilder:          %s
Kubernetes:           %s
Git Commit:           %s
Build Date:           %s
Go OS/Arch:           %s/%s`,
		v.KubeBuilderVersion,
		v.KubernetesVendor,
		v.GitCommit,
		v.BuildDate,
		v.GoOs,
		v.GoArch,
	)
}
