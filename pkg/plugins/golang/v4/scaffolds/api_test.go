/*
Copyright 2022 The Kubernetes Authors.

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

package scaffolds

import (
	"os"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	v3 "sigs.k8s.io/kubebuilder/v4/pkg/config/v3"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v4/pkg/model/resource"
)

// mockConfig implements config.Config for testing
type mockConfig struct {
	multiGroup bool
}

func (m *mockConfig) GetVersion() config.Version                { return config.Version{} }
func (m *mockConfig) GetCliVersion() string                     { return "" }
func (m *mockConfig) SetCliVersion(version string) error        { return nil }
func (m *mockConfig) GetDomain() string                         { return "example.com" }
func (m *mockConfig) SetDomain(domain string) error             { return nil }
func (m *mockConfig) GetRepository() string                     { return "" }
func (m *mockConfig) SetRepository(repository string) error     { return nil }
func (m *mockConfig) GetProjectName() string                    { return "" }
func (m *mockConfig) SetProjectName(name string) error          { return nil }
func (m *mockConfig) GetPluginChain() []string                  { return nil }
func (m *mockConfig) SetPluginChain(pluginChain []string) error { return nil }
func (m *mockConfig) IsMultiGroup() bool                        { return m.multiGroup }
func (m *mockConfig) SetMultiGroup() error                      { return nil }
func (m *mockConfig) ClearMultiGroup() error                    { return nil }
func (m *mockConfig) ResourcesLength() int                      { return 0 }
func (m *mockConfig) HasResource(gvk resource.GVK) bool         { return false }
func (m *mockConfig) GetResource(gvk resource.GVK) (resource.Resource, error) {
	return resource.Resource{}, nil
}
func (m *mockConfig) GetResources() ([]resource.Resource, error) { return nil, nil }
func (m *mockConfig) AddResource(res resource.Resource) error    { return nil }
func (m *mockConfig) UpdateResource(res resource.Resource) error { return nil }
func (m *mockConfig) HasGroup(group string) bool                 { return false }
func (m *mockConfig) ListCRDVersions() []string                  { return nil }
func (m *mockConfig) ListWebhookVersions() []string              { return nil }
func (m *mockConfig) DecodePluginConfig(key string, configObj interface{}) error {
	return nil
}
func (m *mockConfig) EncodePluginConfig(key string, configObj interface{}) error {
	return nil
}
func (m *mockConfig) MarshalYAML() ([]byte, error) { return nil, nil }
func (m *mockConfig) UnmarshalYAML([]byte) error   { return nil }

func TestAPIScaffolder_discoverFeatureGates(t *testing.T) {
	tests := []struct {
		name          string
		config        config.Config
		resource      resource.Resource
		expectedGates []string
	}{
		{
			name: "single group project",
			config: &mockConfig{
				multiGroup: false,
			},
			resource: resource.Resource{
				GVK: resource.GVK{
					Group:   "example.com",
					Version: "v1",
					Kind:    "MyResource",
				},
			},
			expectedGates: []string{},
		},
		{
			name: "multi group project",
			config: &mockConfig{
				multiGroup: true,
			},
			resource: resource.Resource{
				GVK: resource.GVK{
					Group:   "example.com",
					Version: "v1",
					Kind:    "MyResource",
				},
			},
			expectedGates: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock filesystem
			fs := afero.NewMemMapFs()

			scaffolder := &apiScaffolder{
				config:   tt.config,
				resource: tt.resource,
				fs:       machinery.Filesystem{FS: fs},
			}

			gates := scaffolder.discoverFeatureGates()

			// In a real test environment, we'd create test files with feature gates
			// For now, we just verify the function doesn't panic and returns a slice
			assert.NotNil(t, gates, "Expected gates slice to be returned")
			assert.IsType(t, []string{}, gates, "Expected gates to be a string slice")
		})
	}
}

func TestAPIScaffolder_NewAPIScaffolder(t *testing.T) {
	cfg := &mockConfig{}
	res := resource.Resource{
		GVK: resource.GVK{
			Group:   "example.com",
			Version: "v1",
			Kind:    "MyResource",
		},
	}

	scaffolder := NewAPIScaffolder(cfg, res, false)
	require.NotNil(t, scaffolder, "Expected scaffolder to be created")

	apiScaffolder, ok := scaffolder.(*apiScaffolder)
	require.True(t, ok, "Expected apiScaffolder type")

	assert.Equal(t, cfg, apiScaffolder.config)
	assert.Equal(t, res, apiScaffolder.resource)
	assert.False(t, apiScaffolder.force)
}

func TestAPIScaffolder_discoverFeatureGates_Testdata(t *testing.T) {
	// Test with actual testdata to ensure commented markers are ignored
	scaffolder := &apiScaffolder{
		config: v3.New(),
		resource: resource.Resource{
			GVK: resource.GVK{
				Group: "crew",
			},
		},
		fs: machinery.Filesystem{
			FS: afero.NewOsFs(),
		},
	}

	// Change to the testdata directory
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir("testdata/project-v4"); err != nil {
		t.Skipf("Skipping test - testdata directory not available: %v", err)
	}

	// Discover feature gates from the testdata
	featureGates := scaffolder.discoverFeatureGates()

	// Should not find any feature gates since they're all commented out
	if len(featureGates) > 0 {
		t.Errorf("Expected no feature gates from testdata, but found %d: %v", len(featureGates), featureGates)
	}
}
