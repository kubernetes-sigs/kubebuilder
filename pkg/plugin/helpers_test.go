/*
Copyright 2021 The Kubernetes Authors.

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
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	"sigs.k8s.io/kubebuilder/v3/pkg/config"
	"sigs.k8s.io/kubebuilder/v3/pkg/model/stage"
)

type mockPlugin struct {
	name                     string
	version                  Version
	supportedProjectVersions []config.Version
}

func (p mockPlugin) Name() string                               { return p.name }
func (p mockPlugin) Version() Version                           { return p.version }
func (p mockPlugin) SupportedProjectVersions() []config.Version { return p.supportedProjectVersions }

const (
	short = "go"
	name  = "go.kubebuilder.io"
	key   = "go.kubebuilder.io/v1"
)

var (
	version                  = Version{Number: 1}
	supportedProjectVersions = []config.Version{
		{Number: 2},
		{Number: 3},
	}
)

var _ = Describe("KeyFor", func() {
	It("should join plugins name and version", func() {
		plugin := mockPlugin{
			name:    name,
			version: version,
		}
		Expect(KeyFor(plugin)).To(Equal(key))
	})
})

var _ = Describe("SplitKey", func() {
	It("should split keys with versions", func() {
		n, v := SplitKey(key)
		Expect(n).To(Equal(name))
		Expect(v).To(Equal(version.String()))
	})

	It("should split keys without versions", func() {
		n, v := SplitKey(name)
		Expect(n).To(Equal(name))
		Expect(v).To(Equal(""))
	})
})

var _ = Describe("GetShortName", func() {
	It("should extract base names from domains", func() {
		Expect(GetShortName(name)).To(Equal(short))
	})
})

var _ = Describe("Validate", func() {
	It("should succeed for valid plugins", func() {
		plugin := mockPlugin{
			name:                     name,
			version:                  version,
			supportedProjectVersions: supportedProjectVersions,
		}
		Expect(Validate(plugin)).To(Succeed())
	})

	DescribeTable("should fail",
		func(plugin Plugin) {
			Expect(Validate(plugin)).NotTo(Succeed())
		},
		Entry("for invalid plugin names", mockPlugin{
			name:                     "go_kubebuilder.io",
			version:                  version,
			supportedProjectVersions: supportedProjectVersions,
		}),
		Entry("for invalid plugin versions", mockPlugin{
			name:                     name,
			version:                  Version{Number: -1},
			supportedProjectVersions: supportedProjectVersions,
		}),
		Entry("for no supported project version", mockPlugin{
			name:                     name,
			version:                  version,
			supportedProjectVersions: nil,
		}),
		Entry("for invalid supported project version", mockPlugin{
			name:                     name,
			version:                  version,
			supportedProjectVersions: []config.Version{{Number: -1}},
		}),
	)
})

var _ = Describe("ValidateKey", func() {
	It("should succeed for valid keys", func() {
		Expect(ValidateKey(key)).To(Succeed())
	})

	DescribeTable("should fail",
		func(key string) {
			Expect(ValidateKey(key)).NotTo(Succeed())
		},
		Entry("for invalid plugin names", "go_kubebuilder.io/v1"),
		Entry("for invalid versions", "go.kubebuilder.io/a"),
	)
})

var _ = Describe("SupportsVersion", func() {
	plugin := mockPlugin{
		supportedProjectVersions: supportedProjectVersions,
	}

	It("should return true for supported versions", func() {
		Expect(SupportsVersion(plugin, config.Version{Number: 2})).To(BeTrue())
		Expect(SupportsVersion(plugin, config.Version{Number: 3})).To(BeTrue())
	})

	It("should return false for non-supported versions", func() {
		Expect(SupportsVersion(plugin, config.Version{Number: 1})).To(BeFalse())
		Expect(SupportsVersion(plugin, config.Version{Number: 3, Stage: stage.Alpha})).To(BeFalse())
	})
})
