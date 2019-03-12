package controller

import (
	"fmt"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/kubebuilder/pkg/scaffold/input"
	"sigs.k8s.io/kubebuilder/pkg/scaffold/resource"
	"sigs.k8s.io/kubebuilder/pkg/scaffold/scaffoldtest"
)

var _ = Describe("Controller", func() {
	resources := []*resource.Resource{
		{Group: "crew", Version: "v1", Kind: "FirstMate", Namespaced: true, CreateExampleReconcileBody: true},
		{Group: "ship", Version: "v1beta1", Kind: "Frigate", Namespaced: true, CreateExampleReconcileBody: false},
		{Group: "creatures", Version: "v2alpha1", Kind: "Kraken", Namespaced: false, CreateExampleReconcileBody: false},
		{Group: "core", Version: "v1", Kind: "Namespace", Namespaced: false, CreateExampleReconcileBody: false},
	}

	for i := range resources {
		r := resources[i]
		Describe(fmt.Sprintf("scaffolding Controller %s", r.Kind), func() {
			files := []struct {
				instance input.File
				file     string
			}{
				{
					file: filepath.Join("pkg", "controller",
						fmt.Sprintf("add_%s.go", strings.ToLower(r.Kind))),
					instance: &AddController{Resource: r},
				},
				{
					file: filepath.Join("pkg", "controller", strings.ToLower(r.Kind),
						strings.ToLower(r.Kind)+"_controller.go"),
					instance: &Controller{Resource: r},
				},
				{
					file: filepath.Join("pkg", "controller",
						strings.ToLower(r.Kind), strings.ToLower(r.Kind)+"_controller_suite_test.go"),
					instance: &SuiteTest{Resource: r},
				},
				{
					file: filepath.Join("pkg", "controller",
						strings.ToLower(r.Kind), strings.ToLower(r.Kind)+"_controller_test.go"),
					instance: &Test{Resource: r},
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
