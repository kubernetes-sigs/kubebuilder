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

package plugin

import (
	"sort"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	"sigs.k8s.io/kubebuilder/v4/pkg/model/stage"
)

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

var _ = Describe("CommonSupportedProjectVersions", func() {
	It("should return the common version", func() {
		var (
			p1 = mockPlugin{supportedProjectVersions: []config.Version{
				{Number: 1},
				{Number: 2},
				{Number: 3},
			}}
			p2 = mockPlugin{supportedProjectVersions: []config.Version{
				{Number: 1},
				{Number: 2, Stage: stage.Beta},
				{Number: 3, Stage: stage.Alpha},
			}}
			p3 = mockPlugin{supportedProjectVersions: []config.Version{
				{Number: 1},
				{Number: 2},
				{Number: 3, Stage: stage.Beta},
			}}
			p4 = mockPlugin{supportedProjectVersions: []config.Version{
				{Number: 2},
				{Number: 3},
			}}
		)

		for _, tc := range []struct {
			plugins  []Plugin
			versions []config.Version
		}{
			{plugins: []Plugin{p1, p2}, versions: []config.Version{{Number: 1}}},
			{plugins: []Plugin{p1, p3}, versions: []config.Version{{Number: 1}, {Number: 2}}},
			{plugins: []Plugin{p1, p4}, versions: []config.Version{{Number: 2}, {Number: 3}}},
			{plugins: []Plugin{p2, p3}, versions: []config.Version{{Number: 1}}},
			{plugins: []Plugin{p2, p4}, versions: []config.Version{}},
			{plugins: []Plugin{p3, p4}, versions: []config.Version{{Number: 2}}},

			{plugins: []Plugin{p1, p2, p3}, versions: []config.Version{{Number: 1}}},
			{plugins: []Plugin{p1, p2, p4}, versions: []config.Version{}},
			{plugins: []Plugin{p1, p3, p4}, versions: []config.Version{{Number: 2}}},
			{plugins: []Plugin{p2, p3, p4}, versions: []config.Version{}},

			{plugins: []Plugin{p1, p2, p3, p4}, versions: []config.Version{}},
		} {
			versions := CommonSupportedProjectVersions(tc.plugins...)
			sort.Slice(versions, func(i int, j int) bool {
				return versions[i].Compare(versions[j]) == -1
			})
			Expect(versions).To(Equal(tc.versions))
		}
	})
})
