package resource_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "sigs.k8s.io/kubebuilder/pkg/scaffold/resource"
)

var _ = Describe("Resource", func() {
	Describe("scaffolding an API", func() {
		It("should succeed if the Resource is valid", func() {
			instance := &Resource{Group: "crew", Version: "v1", Kind: "FirstMate"}
			Expect(instance.Validate()).To(Succeed())
		})

		It("should fail if the Group is not specified", func() {
			instance := &Resource{Version: "v1", Kind: "FirstMate"}
			Expect(instance.Validate()).NotTo(Succeed())
			Expect(instance.Validate().Error()).To(ContainSubstring("group cannot be empty"))
		})

		It("should fail if the Group is not all lowercase", func() {
			instance := &Resource{Group: "Crew", Version: "v1", Kind: "FirstMate"}
			Expect(instance.Validate()).NotTo(Succeed())
			Expect(instance.Validate().Error()).To(ContainSubstring("group name is invalid: ([a DNS-1123 subdomain must consist of lower case alphanumeric characters"))
		})

		It("should fail if the Group contains non-alpha characters", func() {
			instance := &Resource{Group: "crew1*?", Version: "v1", Kind: "FirstMate"}
			Expect(instance.Validate()).NotTo(Succeed())
			Expect(instance.Validate().Error()).To(ContainSubstring("group name is invalid: ([a DNS-1123 subdomain must consist of lower case alphanumeric characters"))
		})

		It("should fail if the Version is not specified", func() {
			instance := &Resource{Group: "crew", Kind: "FirstMate"}
			Expect(instance.Validate()).NotTo(Succeed())
			Expect(instance.Validate().Error()).To(ContainSubstring("version cannot be empty"))
		})

		It("should fail if the Version does not match the version format", func() {
			instance := &Resource{Group: "crew", Version: "1", Kind: "FirstMate"}
			Expect(instance.Validate()).NotTo(Succeed())
			Expect(instance.Validate().Error()).To(ContainSubstring(
				`version must match ^v\d+(alpha\d+|beta\d+)?$ (was 1)`))

			instance = &Resource{Group: "crew", Version: "1beta1", Kind: "FirstMate"}
			Expect(instance.Validate()).NotTo(Succeed())
			Expect(instance.Validate().Error()).To(ContainSubstring(
				`version must match ^v\d+(alpha\d+|beta\d+)?$ (was 1beta1)`))

			instance = &Resource{Group: "crew", Version: "a1beta1", Kind: "FirstMate"}
			Expect(instance.Validate()).NotTo(Succeed())
			Expect(instance.Validate().Error()).To(ContainSubstring(
				`version must match ^v\d+(alpha\d+|beta\d+)?$ (was a1beta1)`))

			instance = &Resource{Group: "crew", Version: "v1beta", Kind: "FirstMate"}
			Expect(instance.Validate()).NotTo(Succeed())
			Expect(instance.Validate().Error()).To(ContainSubstring(
				`version must match ^v\d+(alpha\d+|beta\d+)?$ (was v1beta)`))

			instance = &Resource{Group: "crew", Version: "v1beta1alpha1", Kind: "FirstMate"}
			Expect(instance.Validate()).NotTo(Succeed())
			Expect(instance.Validate().Error()).To(ContainSubstring(
				`version must match ^v\d+(alpha\d+|beta\d+)?$ (was v1beta1alpha1)`))
		})

		It("should fail if the Kind is not specified", func() {
			instance := &Resource{Group: "crew", Version: "v1"}
			Expect(instance.Validate()).NotTo(Succeed())
			Expect(instance.Validate().Error()).To(ContainSubstring("kind cannot be empty"))
		})

		It("should fail if the Kind is not pascal cased", func() {
			// Base case
			instance := &Resource{Group: "crew", Kind: "FirstMate", Version: "v1"}
			Expect(instance.Validate()).To(Succeed())

			// Can't detect this case :(
			instance = &Resource{Group: "crew", Kind: "Firstmate", Version: "v1"}
			Expect(instance.Validate()).To(Succeed())

			instance = &Resource{Group: "crew", Kind: "firstMate", Version: "v1"}
			Expect(instance.Validate()).NotTo(Succeed())
			Expect(instance.Validate().Error()).To(ContainSubstring(
				`kind must be PascalCase (expected FirstMate was firstMate)`))

			instance = &Resource{Group: "crew", Kind: "firstmate", Version: "v1"}
			Expect(instance.Validate()).NotTo(Succeed())
			Expect(instance.Validate().Error()).To(ContainSubstring(
				`kind must be PascalCase (expected Firstmate was firstmate)`))
		})

		It("should default the Resource by pluralizing the Kind", func() {
			instance := &Resource{Group: "crew", Kind: "FirstMate", Version: "v1"}
			Expect(instance.Validate()).To(Succeed())
			Expect(instance.Resource).To(Equal("firstmates"))

			instance = &Resource{Group: "crew", Kind: "Fish", Version: "v1"}
			Expect(instance.Validate()).To(Succeed())
			Expect(instance.Resource).To(Equal("fish"))

			instance = &Resource{Group: "crew", Kind: "Helmswoman", Version: "v1"}
			Expect(instance.Validate()).To(Succeed())
			Expect(instance.Resource).To(Equal("helmswomen"))
		})

		It("should allow Cat as a Kind", func() {
			instance := &Resource{Group: "crew", Kind: "Cat", Version: "v1"}
			Expect(instance.Validate()).To(Succeed())
			Expect(instance.Resource).To(Equal("cats"))
		})

		It("should allow hyphens in group names", func() {
			instance := &Resource{Group: "my-project", Kind: "Cat", Version: "v1"}
			Expect(instance.Validate()).To(Succeed())
			Expect(instance.GroupImportSafe).To(Equal("myproject"))
		})

		It("should keep the Resource if specified", func() {
			instance := &Resource{Group: "crew", Kind: "FirstMate", Version: "v1", Resource: "myresource"}
			Expect(instance.Validate()).To(Succeed())
			Expect(instance.Resource).To(Equal("myresource"))
		})
	})
})
