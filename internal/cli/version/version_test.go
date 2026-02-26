/*
Copyright 2026 The Kubernetes Authors.

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
	"runtime/debug"
	"strings"
	"testing"
)

func TestNew(t *testing.T) {
	t.Run("Environment variable override", func(t *testing.T) {
		expectedVersion := "v9.9.9-test"
		t.Setenv("KUBEBUILDER_TEST_VERSION", expectedVersion)

		v := New()

		if v.KubeBuilderVersion != expectedVersion {
			t.Errorf("expected version %s, got %s", expectedVersion, v.KubeBuilderVersion)
		}
		if v.GitCommit != "test-commit" {
			t.Errorf("expected gitCommit 'test-commit', got %s", v.GitCommit)
		}
	})

	t.Run("Fallback behavior", func(t *testing.T) {
		v := New()

		if v.KubernetesVendor != kubernetesVendorVersion {
			t.Errorf("expected vendor %s, got %s", kubernetesVendorVersion, v.KubernetesVendor)
		}
		if v.GoOs == "" || v.GoArch == "" {
			t.Error("GoOs or GoArch was not populated from runtime")
		}
	})

	t.Run("VCS metadata resolution with tagged release", func(t *testing.T) {
		v := &Version{KubeBuilderVersion: "v1.0.0"}
		settings := []debug.BuildSetting{
			{Key: "vcs.revision", Value: "abcdef123"},
			{Key: "vcs.modified", Value: "true"},
		}

		v.applyVCSMetadata(settings)

		if !strings.HasSuffix(v.GitCommit, "-dirty") {
			t.Errorf("expected commit to be dirty, got %s", v.GitCommit)
		}
		// For tagged releases, we ignore dirty flag to support GoReleaser builds
		if v.KubeBuilderVersion != "v1.0.0" {
			t.Errorf("expected version to remain v1.0.0, got %s", v.KubeBuilderVersion)
		}
	})

	t.Run("Version string formatting", func(t *testing.T) {
		v := Version{KubeBuilderVersion: "v1.2.3"}

		cleanVersion := v.GetKubeBuilderVersion()
		if cleanVersion != "1.2.3" {
			t.Errorf("expected 1.2.3, got %s", cleanVersion)
		}
	})
}

func TestGetKubeBuilderVersion(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{"strips v prefix", "v1.35.0", "1.35.0"},
		{"handles no prefix", "1.35.0", "1.35.0"},
		{"preserves devel", "(devel)", "(devel)"},
		{"handles empty", "", ""},
		{"handles dirty suffix", "v1.35.0-dirty", "1.35.0-dirty"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			v := Version{KubeBuilderVersion: tc.input}
			if actual := v.GetKubeBuilderVersion(); actual != tc.expected {
				t.Errorf("GetKubeBuilderVersion(%s) = %s; want %s", tc.input, actual, tc.expected)
			}
		})
	}
}

func TestResolveMainVersion(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"Valid Tag", "v4.10.4", "v4.10.4"},
		{"Development Build", "", develVersion},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			main := debug.Module{Version: tt.input}
			if got := resolveMainVersion(main); got != tt.expected {
				t.Errorf("resolveMainVersion() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestApplyVCSMetadata(t *testing.T) {
	tests := []struct {
		name           string
		initialVersion string
		settings       []debug.BuildSetting
		expectCommit   string
		expectVersion  string
		expectDate     string
	}{
		{
			name:           "Clean release build",
			initialVersion: "v4.10.4",
			settings: []debug.BuildSetting{
				{Key: "vcs.revision", Value: "abcdef123"},
				{Key: "vcs.time", Value: "2025-12-27T18:00:00Z"},
				{Key: "vcs.modified", Value: "false"},
			},
			expectCommit:  "abcdef123",
			expectVersion: "v4.10.4",
			expectDate:    "2025-12-27T18:00:00Z",
		},
		{
			name:           "Dirty development build",
			initialVersion: develVersion,
			settings: []debug.BuildSetting{
				{Key: "vcs.revision", Value: "abcdef123"},
				{Key: "vcs.modified", Value: "true"},
				{Key: "vcs.time", Value: "2025-12-29T19:30:00Z"},
			},
			expectCommit:  "abcdef123-dirty",
			expectVersion: "(devel)",
			expectDate:    "2025-12-29T19:30:00Z",
		},
		{
			name:           "Dirty tagged release (GoReleaser scenario)",
			initialVersion: "v4.5.3-rc.1",
			settings: []debug.BuildSetting{
				{Key: "vcs.revision", Value: "abcdef123"},
				{Key: "vcs.modified", Value: "true"},
				{Key: "vcs.time", Value: "2025-12-30T10:00:00Z"},
			},
			expectCommit:  "abcdef123-dirty",
			expectVersion: "v4.5.3-rc.1", // Stays clean for tagged releases
			expectDate:    "2025-12-30T10:00:00Z",
		},
		{
			name:           "Dirty pseudo-version",
			initialVersion: "v1.2.4-0.20191109021931-daa7c04131f5",
			settings: []debug.BuildSetting{
				{Key: "vcs.revision", Value: "abcdef123"},
				{Key: "vcs.modified", Value: "true"},
			},
			expectCommit:  "abcdef123-dirty",
			expectVersion: "(devel)", // Pseudo-versions become (devel) when dirty
			expectDate:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := &Version{
				KubeBuilderVersion: tt.initialVersion,
			}
			v.applyVCSMetadata(tt.settings)

			if v.GitCommit != tt.expectCommit {
				t.Errorf("GitCommit = %v, want %v", v.GitCommit, tt.expectCommit)
			}
			if v.KubeBuilderVersion != tt.expectVersion {
				t.Errorf("KubeBuilderVersion = %v, want %v", v.KubeBuilderVersion, tt.expectVersion)
			}
			if v.BuildDate != tt.expectDate {
				t.Errorf("BuildDate = %v, want %v", v.BuildDate, tt.expectDate)
			}
		})
	}
}

func TestPrintVersion(t *testing.T) {
	v := Version{
		KubeBuilderVersion: "v9.99.9",
		KubernetesVendor:   "9.99.9",
		GitCommit:          "9990f08847dd1",
		BuildDate:          "1970-01-12T12:12:12Z",
		GoOs:               "linux",
		GoArch:             "amd64",
	}

	expectedOutput := `KubeBuilder:          v9.99.9
Kubernetes:           9.99.9
Git Commit:           9990f08847dd1
Build Date:           1970-01-12T12:12:12Z
Go OS/Arch:           linux/amd64`

	actualOutput := v.PrintVersion()

	if actualOutput != expectedOutput {
		t.Errorf("different output in version subcommand.\nexpected:\n%v\ngot:\n%v",
			expectedOutput, actualOutput)
	}
}
