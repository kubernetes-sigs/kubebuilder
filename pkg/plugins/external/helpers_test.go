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

const (
	argGroup       = "--group"
	argVersion     = "--version"
	argKind        = "--kind"
	argHelp        = "--help"
	argFlag        = "--flag"
	argCustom      = "--custom"
	flagNameCustom = "custom"
	argFlag1       = "--flag1"
	argFlag2       = "--flag2"
	argValue       = "value"
)

var _ = Describe("helpers", func() {
	Context("isBooleanFlag", func() {
		It("should return true when next arg starts with --", func() {
			args := []string{argFlag1, argFlag2, argValue}
			Expect(isBooleanFlag(0, args)).To(BeTrue())
		})

		It("should return true when at end of args", func() {
			args := []string{argFlag1, argFlag2}
			Expect(isBooleanFlag(1, args)).To(BeTrue())
		})

		It("should return false when next arg is a value", func() {
			args := []string{argFlag1, argValue, argFlag2}
			Expect(isBooleanFlag(0, args)).To(BeFalse())
		})

		It("should return true for last argument", func() {
			args := []string{argFlag}
			Expect(isBooleanFlag(0, args)).To(BeTrue())
		})
	})

	Context("filterFlags", func() {
		var testFlags []external.Flag

		BeforeEach(func() {
			testFlags = []external.Flag{
				{Name: flagNameGroup, Type: flagTypeString},
				{Name: flagNameVersion, Type: flagTypeString},
				{Name: flagNameKind, Type: flagTypeString},
				{Name: flagNameCustom, Type: flagTypeString},
				{Name: flagNameHelp, Type: flagTypeBool},
			}
		})

		It("should filter flags based on single filter", func() {
			filter := func(flag external.Flag) bool {
				return flag.Name != flagNameGroup
			}
			result := filterFlags(testFlags, []externalFlagFilterFunc{filter})

			Expect(result).To(HaveLen(4))
			for _, flag := range result {
				Expect(flag.Name).NotTo(Equal(flagNameGroup))
			}
		})

		It("should filter flags based on multiple filters", func() {
			filter1 := func(flag external.Flag) bool {
				return flag.Name != flagNameGroup
			}
			filter2 := func(flag external.Flag) bool {
				return flag.Name != flagNameHelp
			}
			result := filterFlags(testFlags, []externalFlagFilterFunc{filter1, filter2})

			Expect(result).To(HaveLen(3))
			Expect(result).To(ContainElement(external.Flag{Name: flagNameVersion, Type: flagTypeString}))
			Expect(result).To(ContainElement(external.Flag{Name: flagNameKind, Type: flagTypeString}))
			Expect(result).To(ContainElement(external.Flag{Name: flagNameCustom, Type: flagTypeString}))
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
			args := []string{argGroup, argVersion, argCustom, argHelp}
			filter := func(arg string) bool {
				return arg != argGroup
			}
			result := filterArgs(args, []argFilterFunc{filter})

			Expect(result).To(Equal([]string{argVersion, argCustom, argHelp}))
		})

		It("should filter args based on multiple filters", func() {
			args := []string{argGroup, argVersion, argCustom}
			filter1 := func(arg string) bool {
				return arg != argGroup
			}
			filter2 := func(arg string) bool {
				return arg != argVersion
			}
			result := filterArgs(args, []argFilterFunc{filter1, filter2})

			Expect(result).To(Equal([]string{argCustom}))
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
			Expect(gvkArgFilter(flagNameGroup)).To(BeFalse())
			Expect(gvkArgFilter(argGroup)).To(BeFalse())
		})

		It("should filter out version flag", func() {
			Expect(gvkArgFilter(flagNameVersion)).To(BeFalse())
			Expect(gvkArgFilter(argVersion)).To(BeFalse())
		})

		It("should filter out kind flag", func() {
			Expect(gvkArgFilter(flagNameKind)).To(BeFalse())
			Expect(gvkArgFilter(argKind)).To(BeFalse())
		})

		It("should allow other flags", func() {
			Expect(gvkArgFilter(flagNameCustom)).To(BeTrue())
			Expect(gvkArgFilter(argCustom)).To(BeTrue())
			Expect(gvkArgFilter("domain")).To(BeTrue())
		})
	})

	Context("gvkFlagFilter", func() {
		It("should filter out group, version, kind flags", func() {
			Expect(gvkFlagFilter(external.Flag{Name: flagNameGroup})).To(BeFalse())
			Expect(gvkFlagFilter(external.Flag{Name: flagNameVersion})).To(BeFalse())
			Expect(gvkFlagFilter(external.Flag{Name: flagNameKind})).To(BeFalse())
		})

		It("should allow other flags", func() {
			Expect(gvkFlagFilter(external.Flag{Name: flagNameCustom})).To(BeTrue())
			Expect(gvkFlagFilter(external.Flag{Name: "domain"})).To(BeTrue())
		})
	})

	Context("helpArgFilter", func() {
		It("should filter out help flag", func() {
			Expect(helpArgFilter(flagNameHelp)).To(BeFalse())
			Expect(helpArgFilter(argHelp)).To(BeFalse())
		})

		It("should allow other flags", func() {
			Expect(helpArgFilter(flagNameCustom)).To(BeTrue())
			Expect(helpArgFilter(argCustom)).To(BeTrue())
			Expect(helpArgFilter("helpful")).To(BeTrue())
		})
	})

	Context("helpFlagFilter", func() {
		It("should filter out help flag", func() {
			Expect(helpFlagFilter(external.Flag{Name: flagNameHelp})).To(BeFalse())
		})

		It("should allow other flags", func() {
			Expect(helpFlagFilter(external.Flag{Name: flagNameCustom})).To(BeTrue())
			Expect(helpFlagFilter(external.Flag{Name: "helpful"})).To(BeTrue())
		})
	})
})
