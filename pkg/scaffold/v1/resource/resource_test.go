package resource_test

import (
	"path/filepath"

	"fmt"

	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/kubebuilder/pkg/model"
	"sigs.k8s.io/kubebuilder/pkg/scaffold/input"
	"sigs.k8s.io/kubebuilder/pkg/scaffold/scaffoldtest"
	"sigs.k8s.io/kubebuilder/pkg/scaffold/v1/resource"
)

var _ = Describe("Resource", func() {
	Describe("scaffolding an API", func() {
		It("should succeed if the Resource is valid", func() {
			instance := &resource.Resource{Group: "crew", Version: "v1", Kind: "FirstMate"}
			Expect(instance.Validate()).To(Succeed())
		})

		It("should fail if the Group is not specified", func() {
			instance := &resource.Resource{Version: "v1", Kind: "FirstMate"}
			Expect(instance.Validate()).NotTo(Succeed())
			Expect(instance.Validate().Error()).To(ContainSubstring("group cannot be empty"))
		})

		It("should fail if the Group is not all lowercase", func() {
			instance := &resource.Resource{Group: "Crew", Version: "v1", Kind: "FirstMate"}
			Expect(instance.Validate()).NotTo(Succeed())
			Expect(instance.Validate().Error()).To(ContainSubstring("group name is invalid: ([a DNS-1123 subdomain must consist of lower case alphanumeric characters"))
		})

		It("should fail if the Group contains non-alpha characters", func() {
			instance := &resource.Resource{Group: "crew1*?", Version: "v1", Kind: "FirstMate"}
			Expect(instance.Validate()).NotTo(Succeed())
			Expect(instance.Validate().Error()).To(ContainSubstring("group name is invalid: ([a DNS-1123 subdomain must consist of lower case alphanumeric characters"))
		})

		It("should fail if the Version is not specified", func() {
			instance := &resource.Resource{Group: "crew", Kind: "FirstMate"}
			Expect(instance.Validate()).NotTo(Succeed())
			Expect(instance.Validate().Error()).To(ContainSubstring("version cannot be empty"))
		})

		It("should fail if the Version does not match the version format", func() {
			instance := &resource.Resource{Group: "crew", Version: "1", Kind: "FirstMate"}
			Expect(instance.Validate()).NotTo(Succeed())
			Expect(instance.Validate().Error()).To(ContainSubstring(
				`version must match ^v\d+(alpha\d+|beta\d+)?$ (was 1)`))

			instance = &resource.Resource{Group: "crew", Version: "1beta1", Kind: "FirstMate"}
			Expect(instance.Validate()).NotTo(Succeed())
			Expect(instance.Validate().Error()).To(ContainSubstring(
				`version must match ^v\d+(alpha\d+|beta\d+)?$ (was 1beta1)`))

			instance = &resource.Resource{Group: "crew", Version: "a1beta1", Kind: "FirstMate"}
			Expect(instance.Validate()).NotTo(Succeed())
			Expect(instance.Validate().Error()).To(ContainSubstring(
				`version must match ^v\d+(alpha\d+|beta\d+)?$ (was a1beta1)`))

			instance = &resource.Resource{Group: "crew", Version: "v1beta", Kind: "FirstMate"}
			Expect(instance.Validate()).NotTo(Succeed())
			Expect(instance.Validate().Error()).To(ContainSubstring(
				`version must match ^v\d+(alpha\d+|beta\d+)?$ (was v1beta)`))

			instance = &resource.Resource{Group: "crew", Version: "v1beta1alpha1", Kind: "FirstMate"}
			Expect(instance.Validate()).NotTo(Succeed())
			Expect(instance.Validate().Error()).To(ContainSubstring(
				`version must match ^v\d+(alpha\d+|beta\d+)?$ (was v1beta1alpha1)`))
		})

		It("should fail if the Kind is not specified", func() {
			instance := &resource.Resource{Group: "crew", Version: "v1"}
			Expect(instance.Validate()).NotTo(Succeed())
			Expect(instance.Validate().Error()).To(ContainSubstring("kind cannot be empty"))
		})

		It("should fail if the Kind is not pascal cased", func() {
			// Base case
			instance := &resource.Resource{Group: "crew", Kind: "FirstMate", Version: "v1"}
			Expect(instance.Validate()).To(Succeed())

			// Can't detect this case :(
			instance = &resource.Resource{Group: "crew", Kind: "Firstmate", Version: "v1"}
			Expect(instance.Validate()).To(Succeed())

			instance = &resource.Resource{Group: "crew", Kind: "firstMate", Version: "v1"}
			Expect(instance.Validate()).NotTo(Succeed())
			Expect(instance.Validate().Error()).To(ContainSubstring(
				`kind must be PascalCase (expected FirstMate was firstMate)`))

			instance = &resource.Resource{Group: "crew", Kind: "firstmate", Version: "v1"}
			Expect(instance.Validate()).NotTo(Succeed())
			Expect(instance.Validate().Error()).To(ContainSubstring(
				`kind must be PascalCase (expected FirstMate was firstmate)`))
		})

		It("should default the Resource by pluralizing the Kind", func() {
			instance := &resource.Resource{Group: "crew", Kind: "FirstMate", Version: "v1"}
			Expect(instance.Validate()).To(Succeed())
			Expect(instance.Resource).To(Equal("firstmates"))

			instance = &resource.Resource{Group: "crew", Kind: "Fish", Version: "v1"}
			Expect(instance.Validate()).To(Succeed())
			Expect(instance.Resource).To(Equal("fish"))

			instance = &resource.Resource{Group: "crew", Kind: "Helmswoman", Version: "v1"}
			Expect(instance.Validate()).To(Succeed())
			Expect(instance.Resource).To(Equal("helmswomen"))
		})

		It("should allow Cat as a Kind", func() {
			instance := &resource.Resource{Group: "crew", Kind: "Cat", Version: "v1"}
			Expect(instance.Validate()).To(Succeed())
			Expect(instance.Resource).To(Equal("cats"))
		})

		It("should allow hyphens in group names", func() {
			instance := &resource.Resource{Group: "my-project", Kind: "Cat", Version: "v1"}
			Expect(instance.Validate()).To(Succeed())
			Expect(instance.GroupImportSafe).To(Equal("myproject"))
		})

		It("should keep the Resource if specified", func() {
			instance := &resource.Resource{Group: "crew", Kind: "FirstMate", Version: "v1", Resource: "myresource"}
			Expect(instance.Validate()).To(Succeed())
			Expect(instance.Resource).To(Equal("myresource"))
		})
	})

	resources := []*resource.Resource{
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
					instance: &resource.AddToScheme{Resource: r},
				},
				{
					file:     filepath.Join("pkg", "apis", r.Group, r.Version, "doc.go"),
					instance: &resource.Doc{Resource: r},
				},
				{
					file:     filepath.Join("pkg", "apis", r.Group, "group.go"),
					instance: &resource.Group{Resource: r},
				},
				{
					file:     filepath.Join("pkg", "apis", r.Group, r.Version, "register.go"),
					instance: &resource.Register{Resource: r},
				},
				{
					file: filepath.Join("pkg", "apis", r.Group, r.Version,
						strings.ToLower(r.Kind)+"_types.go"),
					instance: &resource.Types{Resource: r},
				},
				{
					file: filepath.Join("pkg", "apis", r.Group, r.Version,
						strings.ToLower(r.Kind)+"_types_test.go"),
					instance: &resource.TypesTest{Resource: r},
				},
				{
					file:     filepath.Join("pkg", "apis", r.Group, r.Version, r.Version+"_suite_test.go"),
					instance: &resource.VersionSuiteTest{Resource: r},
				},
				{
					file: filepath.Join("config", "samples",
						fmt.Sprintf("%s_%s_%s.yaml", r.Group, r.Version, strings.ToLower(r.Kind))),
					instance: &resource.CRDSample{Resource: r},
				},
			}

			for j := range files {
				f := files[j]
				Context(f.file, func() {
					It(fmt.Sprintf("should write a file matching the golden file %s", f.file), func() {
						s, result := scaffoldtest.NewTestScaffold(f.file, f.file)
						Expect(s.Execute(&model.Universe{}, scaffoldtest.Options(), f.instance)).To(Succeed())
						Expect(result.Actual.String()).To(Equal(result.Golden), result.Actual.String())
					})
				})
			}
		})
	}
})
