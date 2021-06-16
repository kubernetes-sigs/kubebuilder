/*
Copyright 2018 The Kubernetes Authors.

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

package golang

import (
	"sort"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("goVersion", func() {
	Context("parse", func() {
		var v goVersion

		BeforeEach(func() {
			v = goVersion{}
		})

		DescribeTable("should succeed for valid versions",
			func(version string, expected goVersion) {
				Expect(v.parse(version)).NotTo(HaveOccurred())
				Expect(v.major).To(Equal(expected.major))
				Expect(v.minor).To(Equal(expected.minor))
				Expect(v.patch).To(Equal(expected.patch))
				Expect(v.prerelease).To(Equal(expected.prerelease))
			},
			Entry("for minor release", "go1.15", goVersion{
				major: 1,
				minor: 15,
			}),
			Entry("for patch release", "go1.15.1", goVersion{
				major: 1,
				minor: 15,
				patch: 1,
			}),
			Entry("for alpha release", "go1.15alpha1", goVersion{
				major:      1,
				minor:      15,
				prerelease: "alpha1",
			}),
			Entry("for beta release", "go1.15beta1", goVersion{
				major:      1,
				minor:      15,
				prerelease: "beta1",
			}),
			Entry("for release candidate", "go1.15rc1", goVersion{
				major:      1,
				minor:      15,
				prerelease: "rc1",
			}),
		)

		DescribeTable("should fail for invalid versions",
			func(version string) { Expect(v.parse(version)).To(HaveOccurred()) },
			Entry("for invalid prefix", "g1.15"),
			Entry("for missing major version", "go.15"),
			Entry("for missing minor version", "go1."),
			Entry("for patch and prerelease version", "go1.15.1rc1"),
			Entry("for invalid major version", "goa.15"),
			Entry("for invalid minor version", "go1.a"),
			Entry("for invalid patch version", "go1.15.a"),
		)
	})

	Context("compare", func() {
		// Test compare() by sorting a list.
		var (
			versions = []goVersion{
				{major: 1, minor: 15, prerelease: "rc2"},
				{major: 1, minor: 15, patch: 1},
				{major: 1, minor: 16},
				{major: 1, minor: 15, prerelease: "beta1"},
				{major: 1, minor: 15, prerelease: "alpha2"},
				{major: 2, minor: 0},
				{major: 1, minor: 15, prerelease: "alpha1"},
				{major: 1, minor: 13},
				{major: 1, minor: 15, prerelease: "rc1"},
				{major: 1, minor: 15},
				{major: 1, minor: 15, patch: 2},
				{major: 1, minor: 14},
				{major: 1, minor: 15, prerelease: "beta2"},
				{major: 0, minor: 123},
			}

			sortedVersions = []goVersion{
				{major: 0, minor: 123},
				{major: 1, minor: 13},
				{major: 1, minor: 14},
				{major: 1, minor: 15, prerelease: "alpha1"},
				{major: 1, minor: 15, prerelease: "alpha2"},
				{major: 1, minor: 15, prerelease: "beta1"},
				{major: 1, minor: 15, prerelease: "beta2"},
				{major: 1, minor: 15, prerelease: "rc1"},
				{major: 1, minor: 15, prerelease: "rc2"},
				{major: 1, minor: 15},
				{major: 1, minor: 15, patch: 1},
				{major: 1, minor: 15, patch: 2},
				{major: 1, minor: 16},
				{major: 2, minor: 0},
			}
		)

		It("sorts a valid list of versions correctly", func() {
			sort.Slice(versions, func(i int, j int) bool {
				return versions[i].compare(versions[j]) == -1
			})
			Expect(versions).To(Equal(sortedVersions))
		})
	})
})

var _ = Describe("checkGoVersion", func() {
	DescribeTable("should return true for supported go versions",
		func(version string) { Expect(checkGoVersion(version)).NotTo(HaveOccurred()) },
		Entry("for go 1.13", "go1.13"),
		Entry("for go 1.13.1", "go1.13.1"),
		Entry("for go 1.13.2", "go1.13.2"),
		Entry("for go 1.13.3", "go1.13.3"),
		Entry("for go 1.13.4", "go1.13.4"),
		Entry("for go 1.13.5", "go1.13.5"),
		Entry("for go 1.13.6", "go1.13.6"),
		Entry("for go 1.13.7", "go1.13.7"),
		Entry("for go 1.13.8", "go1.13.8"),
		Entry("for go 1.13.9", "go1.13.9"),
		Entry("for go 1.13.10", "go1.13.10"),
		Entry("for go 1.13.11", "go1.13.11"),
		Entry("for go 1.13.12", "go1.13.12"),
		Entry("for go 1.13.13", "go1.13.13"),
		Entry("for go 1.13.14", "go1.13.14"),
		Entry("for go 1.13.15", "go1.13.15"),
		Entry("for go 1.14beta1", "go1.14beta1"),
		Entry("for go 1.14rc1", "go1.14rc1"),
		Entry("for go 1.14", "go1.14"),
		Entry("for go 1.14.1", "go1.14.1"),
		Entry("for go 1.14.2", "go1.14.2"),
		Entry("for go 1.14.3", "go1.14.3"),
		Entry("for go 1.14.4", "go1.14.4"),
		Entry("for go 1.14.5", "go1.14.5"),
		Entry("for go 1.14.6", "go1.14.6"),
		Entry("for go 1.14.7", "go1.14.7"),
		Entry("for go 1.14.8", "go1.14.8"),
		Entry("for go 1.14.9", "go1.14.9"),
		Entry("for go 1.14.10", "go1.14.10"),
		Entry("for go 1.14.11", "go1.14.11"),
		Entry("for go 1.14.12", "go1.14.12"),
		Entry("for go 1.14.13", "go1.14.13"),
		Entry("for go 1.14.14", "go1.14.14"),
		Entry("for go 1.14.15", "go1.14.15"),
		Entry("for go 1.15beta1", "go1.15beta1"),
		Entry("for go 1.15rc1", "go1.15rc1"),
		Entry("for go 1.15rc2", "go1.15rc2"),
		Entry("for go 1.15", "go1.15"),
		Entry("for go 1.15.1", "go1.15.1"),
		Entry("for go 1.15.2", "go1.15.2"),
		Entry("for go 1.15.3", "go1.15.3"),
		Entry("for go 1.15.4", "go1.15.4"),
		Entry("for go 1.15.5", "go1.15.5"),
		Entry("for go 1.15.6", "go1.15.6"),
		Entry("for go 1.15.7", "go1.15.7"),
		Entry("for go 1.15.8", "go1.15.8"),
		Entry("for go 1.16", "go1.16"),
		Entry("for go 1.16.1", "go1.16.1"),
		Entry("for go 1.16.2", "go1.16.2"),
		Entry("for go 1.16.3", "go1.16.3"),
		Entry("for go 1.16.4", "go1.16.4"),
	)

	DescribeTable("should return false for non-supported go versions",
		func(version string) { Expect(checkGoVersion(version)).To(HaveOccurred()) },
		Entry("for invalid go versions", "go"),
		Entry("for go 1.13beta1", "go1.13beta1"),
		Entry("for go 1.13rc1", "go1.13rc1"),
		Entry("for go 1.13rc2", "go1.13rc2"),
	)
})
