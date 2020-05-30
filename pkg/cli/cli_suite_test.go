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

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/spf13/pflag"

	internalconfig "sigs.k8s.io/kubebuilder/internal/config"
	"sigs.k8s.io/kubebuilder/pkg/model/config"
	"sigs.k8s.io/kubebuilder/pkg/plugin"
)

func TestCLI(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "CLI Suite")
}

// Test plugin types and constructors.
type mockPlugin struct {
	name, version   string
	projectVersions []string
}

func (p mockPlugin) Name() string                       { return p.name }
func (p mockPlugin) Version() string                    { return p.version }
func (p mockPlugin) SupportedProjectVersions() []string { return p.projectVersions }

func (mockPlugin) UpdateContext(*plugin.Context) {}
func (mockPlugin) BindFlags(*pflag.FlagSet)      {}
func (mockPlugin) InjectConfig(*config.Config)   {}
func (mockPlugin) Run() error                    { return nil }

func makeBasePlugin(name, version string, projVers ...string) plugin.Base {
	return mockPlugin{name, version, projVers}
}

func makePluginsForKeys(keys ...string) (plugins []plugin.Base) {
	for _, key := range keys {
		n, v := plugin.SplitKey(key)
		plugins = append(plugins, makeBasePlugin(n, v, internalconfig.DefaultVersion))
	}
	return
}

type mockAllPlugin struct {
	mockPlugin
	mockInitPlugin
	mockCreateAPIPlugin
	mockCreateWebhookPlugin
}

type mockInitPlugin struct{ mockPlugin }
type mockCreateAPIPlugin struct{ mockPlugin }
type mockCreateWebhookPlugin struct{ mockPlugin }

func (p mockInitPlugin) GetInitPlugin() plugin.Init                            { return p }
func (p mockCreateAPIPlugin) GetCreateAPIPlugin() plugin.CreateAPI             { return p }
func (p mockCreateWebhookPlugin) GetCreateWebhookPlugin() plugin.CreateWebhook { return p }

func makeAllPlugin(name, version string, projectVersions ...string) plugin.Base {
	p := makeBasePlugin(name, version, projectVersions...).(mockPlugin)
	return mockAllPlugin{
		p,
		mockInitPlugin{p},
		mockCreateAPIPlugin{p},
		mockCreateWebhookPlugin{p},
	}
}

func makeSetByProjVer(ps ...plugin.Base) map[string][]plugin.Base {
	set := make(map[string][]plugin.Base)
	for _, p := range ps {
		for _, version := range p.SupportedProjectVersions() {
			set[version] = append(set[version], p)
		}
	}
	return set
}
