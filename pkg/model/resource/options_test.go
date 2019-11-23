package resource_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "sigs.k8s.io/kubebuilder/pkg/model/resource"
)

var _ = Describe("Resource Options", func() {
	Describe("scaffolding an API", func() {
		It("should succeed if the Options is valid", func() {
			options := &Options{Group: "crew", Version: "v1", Kind: "FirstMate"}
			Expect(options.Validate()).To(Succeed())
		})

		It("should fail if the Group is not specified", func() {
			options := &Options{Version: "v1", Kind: "FirstMate"}
			Expect(options.Validate()).NotTo(Succeed())
			Expect(options.Validate().Error()).To(ContainSubstring("group cannot be empty"))
		})

		It("should fail if the Group is not all lowercase", func() {
			options := &Options{Group: "Crew", Version: "v1", Kind: "FirstMate"}
			Expect(options.Validate()).NotTo(Succeed())
			Expect(options.Validate().Error()).To(ContainSubstring("group name is invalid: " +
				"([a DNS-1123 subdomain must consist of lower case alphanumeric characters"))
		})

		It("should fail if the Group contains non-alpha characters", func() {
			options := &Options{Group: "crew1*?", Version: "v1", Kind: "FirstMate"}
			Expect(options.Validate()).NotTo(Succeed())
			Expect(options.Validate().Error()).To(ContainSubstring("group name is invalid: " +
				"([a DNS-1123 subdomain must consist of lower case alphanumeric characters"))
		})

		It("should fail if the Version is not specified", func() {
			options := &Options{Group: "crew", Kind: "FirstMate"}
			Expect(options.Validate()).NotTo(Succeed())
			Expect(options.Validate().Error()).To(ContainSubstring("version cannot be empty"))
		})

		It("should fail if the Version does not match the version format", func() {
			options := &Options{Group: "crew", Version: "1", Kind: "FirstMate"}
			Expect(options.Validate()).NotTo(Succeed())
			Expect(options.Validate().Error()).To(ContainSubstring(
				`version must match ^v\d+(alpha\d+|beta\d+)?$ (was 1)`))

			options = &Options{Group: "crew", Version: "1beta1", Kind: "FirstMate"}
			Expect(options.Validate()).NotTo(Succeed())
			Expect(options.Validate().Error()).To(ContainSubstring(
				`version must match ^v\d+(alpha\d+|beta\d+)?$ (was 1beta1)`))

			options = &Options{Group: "crew", Version: "a1beta1", Kind: "FirstMate"}
			Expect(options.Validate()).NotTo(Succeed())
			Expect(options.Validate().Error()).To(ContainSubstring(
				`version must match ^v\d+(alpha\d+|beta\d+)?$ (was a1beta1)`))

			options = &Options{Group: "crew", Version: "v1beta", Kind: "FirstMate"}
			Expect(options.Validate()).NotTo(Succeed())
			Expect(options.Validate().Error()).To(ContainSubstring(
				`version must match ^v\d+(alpha\d+|beta\d+)?$ (was v1beta)`))

			options = &Options{Group: "crew", Version: "v1beta1alpha1", Kind: "FirstMate"}
			Expect(options.Validate()).NotTo(Succeed())
			Expect(options.Validate().Error()).To(ContainSubstring(
				`version must match ^v\d+(alpha\d+|beta\d+)?$ (was v1beta1alpha1)`))
		})

		It("should fail if the Kind is not specified", func() {
			options := &Options{Group: "crew", Version: "v1"}
			Expect(options.Validate()).NotTo(Succeed())
			Expect(options.Validate().Error()).To(ContainSubstring("kind cannot be empty"))
		})

		It("should fail if the Kind is not pascal cased", func() {
			// Base case
			options := &Options{Group: "crew", Version: "v1", Kind: "FirstMate"}
			Expect(options.Validate()).To(Succeed())

			// Can't detect this case :(
			options = &Options{Group: "crew", Version: "v1", Kind: "Firstmate"}
			Expect(options.Validate()).To(Succeed())

			options = &Options{Group: "crew", Version: "v1", Kind: "firstMate"}
			Expect(options.Validate()).NotTo(Succeed())
			Expect(options.Validate().Error()).To(ContainSubstring(
				`kind must be PascalCase (expected FirstMate was firstMate)`))

			options = &Options{Group: "crew", Version: "v1", Kind: "firstmate"}
			Expect(options.Validate()).NotTo(Succeed())
			Expect(options.Validate().Error()).To(ContainSubstring(
				`kind must be PascalCase (expected Firstmate was firstmate)`))
		})
	})
})
