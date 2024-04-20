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

package resource

import (
	"path"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = DescribeTable("safeImport should remove unsupported characters",
	func(unsafe, safe string) { Expect(safeImport(unsafe)).To(Equal(safe)) },
	Entry("no dots nor dashes", "text", "text"),
	Entry("one dot", "my.domain", "mydomain"),
	Entry("several dots", "example.my.domain", "examplemydomain"),
	Entry("one dash", "example-text", "exampletext"),
	Entry("several dashes", "other-example-text", "otherexampletext"),
	Entry("both dots and dashes", "my-example.my.domain", "myexamplemydomain"),
)

var _ = Describe("APIPackagePath", func() {
	const (
		repo    = "github.com/kubernetes-sigs/kubebuilder"
		group   = "group"
		version = "v1"
	)

	DescribeTable("should work",
		func(repo, group, version string, multiGroup bool, p string) {
			Expect(APIPackagePath(repo, group, version, multiGroup)).To(Equal(p))
		},
		Entry("single group setup", repo, group, version, false, path.Join(repo, "api", version)),
		Entry("multiple group setup", repo, group, version, true, path.Join(repo, "api", group, version)),
		Entry("multiple group setup with empty group", repo, "", version, true, path.Join(repo, "api", version)),
	)
})

var _ = Describe("APIPackagePathLegacy", func() {
	const (
		repo    = "github.com/kubernetes-sigs/kubebuilder"
		group   = "group"
		version = "v1"
	)

	DescribeTable("should work",
		func(repo, group, version string, multiGroup bool, p string) {
			Expect(APIPackagePathLegacy(repo, group, version, multiGroup)).To(Equal(p))
		},
		Entry("single group setup", repo, group, version, false, path.Join(repo, "api", version)),
		Entry("multiple group setup", repo, group, version, true, path.Join(repo, "apis", group, version)),
		Entry("multiple group setup with empty group", repo, "", version, true, path.Join(repo, "apis", version)),
	)
})

var _ = DescribeTable("RegularPlural should return the regular plural form",
	func(singular, plural string) { Expect(RegularPlural(singular)).To(Equal(plural)) },
	Entry("basic singular", "firstmate", "firstmates"),
	Entry("capitalized singular", "Firstmate", "firstmates"),
	Entry("camel-cased singular", "FirstMate", "firstmates"),
	Entry("irregular well-known plurals", "fish", "fish"),
	Entry("irregular well-known plurals", "helmswoman", "helmswomen"),
)
