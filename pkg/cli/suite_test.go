/*
Copyright 2020 The Kubernetes Authors.

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

package cli

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugin"
)

func TestCLI(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "CLI Suite")
}

// Test plugin types and constructors.
var (
	_ plugin.Plugin = mockPlugin{}
	_ plugin.Plugin = mockDeprecatedPlugin{}
)

type mockPlugin struct { //nolint:maligned
	name            string
	version         plugin.Version
	projectVersions []config.Version
}

func newMockPlugin(name, version string, projVers ...config.Version) plugin.Plugin {
	var v plugin.Version
	if err := v.Parse(version); err != nil {
		panic(err)
	}
	return mockPlugin{name, v, projVers}
}

func (p mockPlugin) Name() string                               { return p.name }
func (p mockPlugin) Version() plugin.Version                    { return p.version }
func (p mockPlugin) SupportedProjectVersions() []config.Version { return p.projectVersions }

type mockDeprecatedPlugin struct { //nolint:maligned
	mockPlugin
	deprecation string
}

func newMockDeprecatedPlugin(name, version, deprecation string, projVers ...config.Version) plugin.Plugin {
	return mockDeprecatedPlugin{
		mockPlugin:  newMockPlugin(name, version, projVers...).(mockPlugin),
		deprecation: deprecation,
	}
}

func (p mockDeprecatedPlugin) DeprecationWarning() string { return p.deprecation }
