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

package plugin

import (
	"testing"

	// nolint:revive
	. "github.com/onsi/ginkgo/v2"
	// nolint:revive
	. "github.com/onsi/gomega"

	"sigs.k8s.io/kubebuilder/v3/pkg/config"
)

func TestPlugin(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Plugin Suite")
}

type mockPlugin struct {
	name                     string
	version                  Version
	supportedProjectVersions []config.Version
}

func (p mockPlugin) Name() string                               { return p.name }
func (p mockPlugin) Version() Version                           { return p.version }
func (p mockPlugin) SupportedProjectVersions() []config.Version { return p.supportedProjectVersions }
