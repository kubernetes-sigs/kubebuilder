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

var _ = Describe("Bundle", func() {
	const (
		name = "bundle.kubebuilder.io"
	)

	var (
		version = Version{Number: 1}

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

	Context("NewBundle", func() {
		It("should succeed for plugins with common supported project versions", func() {
			for _, plugins := range [][]Plugin{
				{p1, p2},
				{p1, p3},
				{p1, p4},
				{p2, p3},
				{p3, p4},

				{p1, p2, p3},
				{p1, p3, p4},
			} {

				b, err := NewBundleWithOptions(WithName(name),
					WithVersion(version),
					WithPlugins(plugins...))
				Expect(err).NotTo(HaveOccurred())
				Expect(b.Name()).To(Equal(name))
				Expect(b.Version().Compare(version)).To(Equal(0))
				versions := b.SupportedProjectVersions()
				sort.Slice(versions, func(i int, j int) bool {
					return versions[i].Compare(versions[j]) == -1
				})
				expectedVersions := CommonSupportedProjectVersions(plugins...)
				sort.Slice(expectedVersions, func(i int, j int) bool {
					return expectedVersions[i].Compare(expectedVersions[j]) == -1
				})
				Expect(versions).To(Equal(expectedVersions))
				Expect(b.Plugins()).To(Equal(plugins))
			}
		})

		It("should accept bundles as input", func() {
			var a, b Bundle
			var err error
			plugins := []Plugin{p1, p2, p3}
			a, err = NewBundleWithOptions(WithName("a"),
				WithVersion(version),
				WithPlugins(p1, p2))
			Expect(err).NotTo(HaveOccurred())
			b, err = NewBundleWithOptions(WithName("b"),
				WithVersion(version),
				WithPlugins(a, p3))
			Expect(err).NotTo(HaveOccurred())
			versions := b.SupportedProjectVersions()
			sort.Slice(versions, func(i int, j int) bool {
				return versions[i].Compare(versions[j]) == -1
			})
			expectedVersions := CommonSupportedProjectVersions(plugins...)
			sort.Slice(expectedVersions, func(i int, j int) bool {
				return expectedVersions[i].Compare(expectedVersions[j]) == -1
			})
			Expect(versions).To(Equal(expectedVersions))
			Expect(b.Plugins()).To(Equal(plugins))
		})

		It("should fail for plugins with no common supported project version", func() {
			for _, plugins := range [][]Plugin{
				{p2, p4},

				{p1, p2, p4},
				{p2, p3, p4},

				{p1, p2, p3, p4},
			} {
				_, err := NewBundleWithOptions(WithName(name),
					WithVersion(version),
					WithPlugins(plugins...))

				Expect(err).To(HaveOccurred())
			}
		})
	})

	Context("NewBundleWithOptions", func() {
		It("should succeed for plugins with common supported project versions", func() {
			for _, plugins := range [][]Plugin{
				{p1, p2},
				{p1, p3},
				{p1, p4},
				{p2, p3},
				{p3, p4},

				{p1, p2, p3},
				{p1, p3, p4},
			} {
				b, err := NewBundleWithOptions(WithName(name),
					WithVersion(version),
					WithDeprecationMessage(""),
					WithPlugins(plugins...),
				)
				Expect(err).NotTo(HaveOccurred())
				Expect(b.Name()).To(Equal(name))
				Expect(b.Version().Compare(version)).To(Equal(0))
				versions := b.SupportedProjectVersions()
				sort.Slice(versions, func(i int, j int) bool {
					return versions[i].Compare(versions[j]) == -1
				})
				expectedVersions := CommonSupportedProjectVersions(plugins...)
				sort.Slice(expectedVersions, func(i int, j int) bool {
					return expectedVersions[i].Compare(expectedVersions[j]) == -1
				})
				Expect(versions).To(Equal(expectedVersions))
				Expect(b.Plugins()).To(Equal(plugins))
			}
		})

		It("should accept bundles as input", func() {
			var a, b Bundle
			var err error
			plugins := []Plugin{p1, p2, p3}
			a, err = NewBundleWithOptions(WithName("a"),
				WithVersion(version),
				WithDeprecationMessage(""),
				WithPlugins(p1, p2),
			)
			Expect(err).NotTo(HaveOccurred())
			b, err = NewBundleWithOptions(WithName("b"),
				WithVersion(version),
				WithDeprecationMessage(""),
				WithPlugins(a, p3),
			)
			Expect(err).NotTo(HaveOccurred())
			versions := b.SupportedProjectVersions()
			sort.Slice(versions, func(i int, j int) bool {
				return versions[i].Compare(versions[j]) == -1
			})
			expectedVersions := CommonSupportedProjectVersions(plugins...)
			sort.Slice(expectedVersions, func(i int, j int) bool {
				return expectedVersions[i].Compare(expectedVersions[j]) == -1
			})
			Expect(versions).To(Equal(expectedVersions))
			Expect(b.Plugins()).To(Equal(plugins))
		})

		It("should fail for plugins with no common supported project version", func() {
			for _, plugins := range [][]Plugin{
				{p2, p4},

				{p1, p2, p4},
				{p2, p3, p4},

				{p1, p2, p3, p4},
			} {
				_, err := NewBundleWithOptions(WithName(name),
					WithVersion(version),
					WithDeprecationMessage(""),
					WithPlugins(plugins...),
				)
				Expect(err).To(HaveOccurred())
			}
		})
	})
})
