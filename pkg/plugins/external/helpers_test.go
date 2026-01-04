/*
Copyright 2026 The Kubernetes Authors.

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

package external

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"sigs.k8s.io/kubebuilder/v4/pkg/plugin/external"
)

var _ = Describe("helpers", func() {
	Context("isBooleanFlag", func() {
		It("should return true when next arg starts with --", func() {
			args := []string{"--flag1", "--flag2", "value"}
			Expect(isBooleanFlag(0, args)).To(BeTrue())
		})

		It("should return true when at end of args", func() {
			args := []string{"--flag1", "--flag2"}
			Expect(isBooleanFlag(1, args)).To(BeTrue())
		})

		It("should return false when next arg is a value", func() {
			args := []string{"--flag1", "value", "--flag2"}
			Expect(isBooleanFlag(0, args)).To(BeFalse())
		})

		It("should return true for last argument", func() {
			args := []string{"--flag"}
			Expect(isBooleanFlag(0, args)).To(BeTrue())
		})
	})

	Context("filterFlags", func() {
		var testFlags []external.Flag

		BeforeEach(func() {
			testFlags = []external.Flag{
				{Name: "group", Type: "string"},
				{Name: "version", Type: "string"},
				{Name: "kind", Type: "string"},
				{Name: "custom", Type: "string"},
				{Name: "help", Type: "bool"},
			}
		})

		It("should filter flags based on single filter", func() {
			filter := func(flag external.Flag) bool {
				return flag.Name != "group"
			}
			result := filterFlags(testFlags, []externalFlagFilterFunc{filter})

			Expect(result).To(HaveLen(4))
			for _, flag := range result {
				Expect(flag.Name).NotTo(Equal("group"))
			}
		})

		It("should filter flags based on multiple filters", func() {
			filter1 := func(flag external.Flag) bool {
				return flag.Name != "group"
			}
			filter2 := func(flag external.Flag) bool {
				return flag.Name != "help"
			}
			result := filterFlags(testFlags, []externalFlagFilterFunc{filter1, filter2})

			Expect(result).To(HaveLen(3))
			Expect(result).To(ContainElement(external.Flag{Name: "version", Type: "string"}))
			Expect(result).To(ContainElement(external.Flag{Name: "kind", Type: "string"}))
			Expect(result).To(ContainElement(external.Flag{Name: "custom", Type: "string"}))
		})

		It("should return empty when all flags filtered out", func() {
			filter := func(_ external.Flag) bool {
				return false
			}
			result := filterFlags(testFlags, []externalFlagFilterFunc{filter})
			Expect(result).To(BeEmpty())
		})

		It("should return all flags when no filters reject", func() {
			filter := func(_ external.Flag) bool {
				return true
			}
			result := filterFlags(testFlags, []externalFlagFilterFunc{filter})
			Expect(result).To(Equal(testFlags))
		})
	})

	Context("filterArgs", func() {
		It("should filter args based on single filter", func() {
			args := []string{"--group", "--version", "--custom", "--help"}
			filter := func(arg string) bool {
				return arg != "--group"
			}
			result := filterArgs(args, []argFilterFunc{filter})

			Expect(result).To(Equal([]string{"--version", "--custom", "--help"}))
		})

		It("should filter args based on multiple filters", func() {
			args := []string{"--group", "--version", "--custom"}
			filter1 := func(arg string) bool {
				return arg != "--group"
			}
			filter2 := func(arg string) bool {
				return arg != "--version"
			}
			result := filterArgs(args, []argFilterFunc{filter1, filter2})

			Expect(result).To(Equal([]string{"--custom"}))
		})

		It("should return empty when all args filtered out", func() {
			args := []string{"--arg1", "--arg2"}
			filter := func(_ string) bool {
				return false
			}
			result := filterArgs(args, []argFilterFunc{filter})
			Expect(result).To(BeEmpty())
		})

		It("should return all args when no filters reject", func() {
			args := []string{"--arg1", "--arg2"}
			filter := func(_ string) bool {
				return true
			}
			result := filterArgs(args, []argFilterFunc{filter})
			Expect(result).To(Equal(args))
		})
	})

	Context("gvkArgFilter", func() {
		It("should filter out group flag", func() {
			Expect(gvkArgFilter("group")).To(BeFalse())
			Expect(gvkArgFilter("--group")).To(BeFalse())
		})

		It("should filter out version flag", func() {
			Expect(gvkArgFilter("version")).To(BeFalse())
			Expect(gvkArgFilter("--version")).To(BeFalse())
		})

		It("should filter out kind flag", func() {
			Expect(gvkArgFilter("kind")).To(BeFalse())
			Expect(gvkArgFilter("--kind")).To(BeFalse())
		})

		It("should allow other flags", func() {
			Expect(gvkArgFilter("custom")).To(BeTrue())
			Expect(gvkArgFilter("--custom")).To(BeTrue())
			Expect(gvkArgFilter("domain")).To(BeTrue())
		})
	})

	Context("gvkFlagFilter", func() {
		It("should filter out group, version, kind flags", func() {
			Expect(gvkFlagFilter(external.Flag{Name: "group"})).To(BeFalse())
			Expect(gvkFlagFilter(external.Flag{Name: "version"})).To(BeFalse())
			Expect(gvkFlagFilter(external.Flag{Name: "kind"})).To(BeFalse())
		})

		It("should allow other flags", func() {
			Expect(gvkFlagFilter(external.Flag{Name: "custom"})).To(BeTrue())
			Expect(gvkFlagFilter(external.Flag{Name: "domain"})).To(BeTrue())
		})
	})

	Context("helpArgFilter", func() {
		It("should filter out help flag", func() {
			Expect(helpArgFilter("help")).To(BeFalse())
			Expect(helpArgFilter("--help")).To(BeFalse())
		})

		It("should allow other flags", func() {
			Expect(helpArgFilter("custom")).To(BeTrue())
			Expect(helpArgFilter("--custom")).To(BeTrue())
			Expect(helpArgFilter("helpful")).To(BeTrue())
		})
	})

	Context("helpFlagFilter", func() {
		It("should filter out help flag", func() {
			Expect(helpFlagFilter(external.Flag{Name: "help"})).To(BeFalse())
		})

		It("should allow other flags", func() {
			Expect(helpFlagFilter(external.Flag{Name: "custom"})).To(BeTrue())
			Expect(helpFlagFilter(external.Flag{Name: "helpful"})).To(BeTrue())
		})
	})
})
