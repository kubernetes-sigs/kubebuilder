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
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/spf13/afero"
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
	RegisterTestingT(t)

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
			scaffolder := NewAPIScaffolder(tt.config, tt.resource, false)
			apiScaffolder, ok := scaffolder.(*apiScaffolder)
			if !ok {
				t.Fatal("Expected apiScaffolder type")
			}

			// Initialize filesystem to prevent nil pointer dereference
			apiScaffolder.fs = machinery.Filesystem{FS: afero.NewMemMapFs()}

			gates := apiScaffolder.discoverFeatureGates()

			// In a real test environment, we'd create test files with feature gates
			// For now, we just verify the function doesn't panic and returns a slice
			Expect(gates).NotTo(BeNil())
			Expect(gates).To(BeAssignableToTypeOf([]string{}))
		})
	}
}

func TestAPIScaffolder_NewAPIScaffolder(t *testing.T) {
	RegisterTestingT(t)

	cfg := v3.New()
	res := resource.Resource{
		GVK: resource.GVK{
			Group:   "example.com",
			Version: "v1",
			Kind:    "MyResource",
		},
	}

	scaffolder := NewAPIScaffolder(cfg, res, false)
	Expect(scaffolder).NotTo(BeNil())

	apiScaffolder, ok := scaffolder.(*apiScaffolder)
	Expect(ok).To(BeTrue())

	Expect(apiScaffolder.config).To(Equal(cfg))
	Expect(apiScaffolder.resource).To(Equal(res))
	Expect(apiScaffolder.force).To(BeFalse())
}

func TestAPIScaffolder_discoverFeatureGates_Testdata(t *testing.T) {
	// Change to testdata directory
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(oldDir)

	// Change to testdata/project-v4 directory (from pkg/plugins/golang/v4/scaffolds/ to ../../../../../testdata/project-v4)
	testdataDir := filepath.Join("../../../../../testdata", "project-v4")
	if err := os.Chdir(testdataDir); err != nil {
		t.Fatalf("Failed to change to testdata directory: %v", err)
	}

	cfg := v3.New()
	res := resource.Resource{
		GVK: resource.GVK{
			Group:   "crew.example.com",
			Version: "v1",
			Kind:    "Captain",
		},
	}

	scaffolder := NewAPIScaffolder(cfg, res, false)
	apiScaffolder, ok := scaffolder.(*apiScaffolder)
	if !ok {
		t.Fatal("Expected apiScaffolder type")
	}

	// Initialize filesystem to prevent nil pointer dereference
	apiScaffolder.fs = machinery.Filesystem{FS: afero.NewOsFs()}

	featureGates := apiScaffolder.discoverFeatureGates()

	// The testdata contains a feature gate marker for "experimental-bar"
	expectedGates := []string{"experimental-bar"}
	if len(featureGates) != len(expectedGates) {
		t.Errorf("Expected %d feature gates from testdata, but found %d: %v", len(expectedGates), len(featureGates), featureGates)
	}

	for _, expectedGate := range expectedGates {
		found := false
		for _, gate := range featureGates {
			if gate == expectedGate {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected to find feature gate %s, but it was not discovered", expectedGate)
		}
	}
}
