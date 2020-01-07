/*
Copyright 2020 The Kubernetes Authors.

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

package crd_test

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"sigs.k8s.io/kubebuilder/pkg/model"
	"sigs.k8s.io/kubebuilder/pkg/scaffold/input"
	"sigs.k8s.io/kubebuilder/pkg/scaffold/resource"
	"sigs.k8s.io/kubebuilder/pkg/scaffold/scaffoldtest"
	. "sigs.k8s.io/kubebuilder/pkg/scaffold/v1/crd"
)

func TestResource(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Resource Suite")
}

var _ = Describe("Resource", func() {
	resources := []*resource.Resource{
		{Group: "crew", Version: "v1", Kind: "FirstMate", Namespaced: true, CreateExampleReconcileBody: true},
		{Group: "ship", Version: "v1beta1", Kind: "Frigate", Namespaced: true, CreateExampleReconcileBody: false},
		{Group: "creatures", Version: "v2alpha1", Kind: "Kraken", Namespaced: false, CreateExampleReconcileBody: false},
	}

	for i := range resources {
		r := resources[i]
		_ = r.Validate()

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
