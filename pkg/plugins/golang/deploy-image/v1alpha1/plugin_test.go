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

package v1alpha1

import (
	"testing"

	"sigs.k8s.io/kubebuilder/v4/pkg/plugin"
)

// TestGetPluginKeyForConfigIntegration tests that the plugin correctly resolves
// its key based on the plugin chain, supporting custom bundle names.
func TestGetPluginKeyForConfigIntegration(t *testing.T) {
	p := Plugin{}

	tests := []struct {
		name        string
		pluginChain []string
		expected    string
		description string
	}{
		{
			name:        "exact match",
			pluginChain: []string{"go.kubebuilder.io/v4", "deploy-image.go.kubebuilder.io/v1-alpha"},
			expected:    "deploy-image.go.kubebuilder.io/v1-alpha",
			description: "When plugin is used directly, it should use its own key",
		},
		{
			name:        "bundle match with custom domain",
			pluginChain: []string{"go.kubebuilder.io/v4", "deploy-image.custom-domain/v1-alpha"},
			expected:    "deploy-image.custom-domain/v1-alpha",
			description: "When plugin is wrapped in bundle with custom domain, it should use bundle's key",
		},
		{
			name:        "bundle match with operator-sdk domain",
			pluginChain: []string{"go.kubebuilder.io/v4", "deploy-image.operator-sdk.io/v1-alpha"},
			expected:    "deploy-image.operator-sdk.io/v1-alpha",
			description: "When plugin is wrapped in operator-sdk bundle, it should use bundle's key",
		},
		{
			name:        "no match - fallback to plugin key",
			pluginChain: []string{"go.kubebuilder.io/v4"},
			expected:    "deploy-image.go.kubebuilder.io/v1-alpha",
			description: "When no matching key in chain, fallback to plugin's own key",
		},
		{
			name:        "version mismatch - fallback",
			pluginChain: []string{"go.kubebuilder.io/v4", "deploy-image.custom-domain/v2-alpha"},
			expected:    "deploy-image.go.kubebuilder.io/v1-alpha",
			description: "When version doesn't match, fallback to plugin's own key",
		},
		{
			name:        "base name mismatch - fallback",
			pluginChain: []string{"go.kubebuilder.io/v4", "other-plugin.custom-domain/v1-alpha"},
			expected:    "deploy-image.go.kubebuilder.io/v1-alpha",
			description: "When base name doesn't match, fallback to plugin's own key",
		},
	}

	for _, tt := range tests {
		// capture range variable
		t.Run(tt.name, func(t *testing.T) {
			result := plugin.GetPluginKeyForConfig(tt.pluginChain, p)
			if result != tt.expected {
				t.Errorf("%s: Expected key %q, got %q", tt.description, tt.expected, result)
			}
		})
	}
}
