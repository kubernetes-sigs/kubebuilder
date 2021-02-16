package resource_test

import (
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	. "sigs.k8s.io/kubebuilder/v2/pkg/model/resource"
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

		DescribeTable("valid Kind values-according to core Kubernetes",
			func(kind string) {
				options := &Options{Group: "crew", Kind: kind, Version: "v1"}
				Expect(options.Validate()).To(Succeed())
			},
			Entry("should pass validation if Kind is camelcase", "FirstMate"),
			Entry("should pass validation if Kind has more than one caps at the start", "FIRSTMate"),
		)

		It("should fail if Kind is too long", func() {
			kind := strings.Repeat("a", 64)

			options := &Options{Group: "crew", Kind: kind, Version: "v1"}
			err := options.Validate()
			Expect(err).To(MatchError(ContainSubstring("must be no more than 63 characters")))
		})

		DescribeTable("invalid Kind values-according to core Kubernetes",
			func(kind string) {
				options := &Options{Group: "crew", Kind: kind, Version: "v1"}
				Expect(options.Validate()).To(MatchError(
					ContainSubstring("a DNS-1035 label must consist of lower case alphanumeric characters")))
			},
			Entry("should fail validation if Kind contains whitespaces", "Something withSpaces"),
			Entry("should fail validation if Kind ends in -", "KindEndingIn-"),
			Entry("should fail validation if Kind starts with number", "0ValidityKind"),
		)

		It("should fail if Kind starts with a lowercase character", func() {
			options := &Options{Group: "crew", Kind: "lOWERCASESTART", Version: "v1"}
			err := options.Validate()
			Expect(err).To(MatchError(ContainSubstring("kind must start with an uppercase character")))
		})
	})
})
