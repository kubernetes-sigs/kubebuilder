/*
Copyright 2025 The Kubernetes Authors.

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
	"testing"
)

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

func TestResolveKubernetesVendor(t *testing.T) {
	tests := []struct {
		name     string
		deps     []*debug.Module
		expected string
	}{
		{
			name: "Standard apimachinery version",
			deps: []*debug.Module{
				{Path: "k8s.io/apimachinery", Version: "v0.31.1"},
			},
			expected: "1.31.1",
		},
		{
			name: "Missing dependency",
			deps: []*debug.Module{
				{Path: "other/package", Version: "v1.0.0"},
			},
			expected: unknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := resolveKubernetesVendor(tt.deps); got != tt.expected {
				t.Errorf("resolveKubernetesVendor() = %v, want %v", got, tt.expected)
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
		},
		{
			name:           "Dirty development build",
			initialVersion: develVersion,
			settings: []debug.BuildSetting{
				{Key: "vcs.revision", Value: "abcdef123"},
				{Key: "vcs.modified", Value: "true"},
			},
			expectCommit:  "abcdef123-dirty",
			expectVersion: "(devel)-dirty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := &Version{KubeBuilderVersion: tt.initialVersion}
			v.applyVCSMetadata(tt.settings)

			if v.GitCommit != tt.expectCommit {
				t.Errorf("GitCommit = %v, want %v", v.GitCommit, tt.expectCommit)
			}
			if v.KubeBuilderVersion != tt.expectVersion {
				t.Errorf("KubeBuilderVersion = %v, want %v", v.KubeBuilderVersion, tt.expectVersion)
			}
		})
	}
}
