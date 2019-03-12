package resource

import (
	"path/filepath"

	"fmt"

	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/kubebuilder/pkg/scaffold/input"
	"sigs.k8s.io/kubebuilder/pkg/scaffold/scaffoldtest"
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
			Expect(instance.Validate().Error()).To(ContainSubstring("group must match ^[a-z]+$ (was Crew)"))
		})

		It("should fail if the Group contains non-alpha characters", func() {
			instance := &Resource{Group: "crew1", Version: "v1", Kind: "FirstMate"}
			Expect(instance.Validate()).NotTo(Succeed())
			Expect(instance.Validate().Error()).To(ContainSubstring("group must match ^[a-z]+$ (was crew1)"))
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

		It("should fail if the Kind is not camel cased", func() {
			// Base case
			instance := &Resource{Group: "crew", Kind: "FirstMate", Version: "v1"}
			Expect(instance.Validate()).To(Succeed())

			// Can't detect this case :(
			instance = &Resource{Group: "crew", Kind: "Firstmate", Version: "v1"}
			Expect(instance.Validate()).To(Succeed())

			instance = &Resource{Group: "crew", Kind: "firstMate", Version: "v1"}
			Expect(instance.Validate()).NotTo(Succeed())
			Expect(instance.Validate().Error()).To(ContainSubstring(
				`Kind must be camelcase (expected FirstMate was firstMate)`))

			instance = &Resource{Group: "crew", Kind: "firstmate", Version: "v1"}
			Expect(instance.Validate()).NotTo(Succeed())
			Expect(instance.Validate().Error()).To(ContainSubstring(
				`Kind must be camelcase (expected Firstmate was firstmate)`))
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

		It("should keep the Resource if specified", func() {
			instance := &Resource{Group: "crew", Kind: "FirstMate", Version: "v1", Resource: "myresource"}
			Expect(instance.Validate()).To(Succeed())
			Expect(instance.Resource).To(Equal("myresource"))
		})
	})

	resources := []*Resource{
		{Group: "crew", Version: "v1", Kind: "FirstMate", Namespaced: true, CreateExampleReconcileBody: true},
		{Group: "ship", Version: "v1beta1", Kind: "Frigate", Namespaced: true, CreateExampleReconcileBody: false},
		{Group: "creatures", Version: "v2alpha1", Kind: "Kraken", Namespaced: false, CreateExampleReconcileBody: false},
	}

	for i := range resources {
		r := resources[i]
		Describe(fmt.Sprintf("scaffolding API %s", r.Kind), func() {
			files := []struct {
				instance input.File
				file     string
			}{
				{
					file: filepath.Join("pkg", "apis",
						fmt.Sprintf("addtoscheme_%s_%s.go", r.Group, r.Version)),
					instance: &AddToScheme{Resource: r},
				},
				{
					file:     filepath.Join("pkg", "apis", r.Group, r.Version, "doc.go"),
					instance: &Doc{Resource: r},
				},
				{
					file:     filepath.Join("pkg", "apis", r.Group, "group.go"),
					instance: &Group{Resource: r},
				},
				{
					file:     filepath.Join("pkg", "apis", r.Group, r.Version, "register.go"),
					instance: &Register{Resource: r},
				},
				{
					file: filepath.Join("pkg", "apis", r.Group, r.Version,
						strings.ToLower(r.Kind)+"_types.go"),
					instance: &Types{Resource: r},
				},
				{
					file: filepath.Join("pkg", "apis", r.Group, r.Version,
						strings.ToLower(r.Kind)+"_types_test.go"),
					instance: &TypesTest{Resource: r},
				},
				{
					file:     filepath.Join("pkg", "apis", r.Group, r.Version, r.Version+"_suite_test.go"),
					instance: &VersionSuiteTest{Resource: r},
				},
				{
					file: filepath.Join("config", "samples",
						fmt.Sprintf("%s_%s_%s.yaml", r.Group, r.Version, strings.ToLower(r.Kind))),
					instance: &CRDSample{Resource: r},
				},
			}

			for j := range files {
				f := files[j]
				Context(f.file, func() {
					It("should write a file matching the golden file", func() {
						s, result := scaffoldtest.NewTestScaffold(f.file, f.file)
						Expect(s.Execute(scaffoldtest.Options(), f.instance)).To(Succeed())
						Expect(result.Actual.String()).To(Equal(result.Golden), result.Actual.String())
					})
				})
			}
		})
	}
})
