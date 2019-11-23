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

package resource_test

import (
	"path"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"sigs.k8s.io/kubebuilder/pkg/model/config"
	. "sigs.k8s.io/kubebuilder/pkg/model/resource"
)

var _ = Describe("Resource", func() {
	Describe("scaffolding an API", func() {
		It("should succeed if the Resource is valid", func() {
			options := &Options{Group: "crew", Version: "v1", Kind: "FirstMate"}
			Expect(options.Validate()).To(Succeed())

			resource := options.NewResource(
				&config.Config{
					Version: config.Version2,
					Domain:  "test.io",
					Repo:    "test",
				},
				true,
			)
			Expect(resource.Namespaced).To(Equal(options.Namespaced))
			Expect(resource.Group).To(Equal(options.Group))
			Expect(resource.GroupPackageName).To(Equal("crew"))
			Expect(resource.Version).To(Equal(options.Version))
			Expect(resource.Kind).To(Equal(options.Kind))
			Expect(resource.Plural).To(Equal("firstmates"))
			Expect(resource.ImportAlias).To(Equal("crewv1"))
			Expect(resource.Package).To(Equal(path.Join("test", "api", "v1")))
			Expect(resource.Domain).To(Equal("crew.test.io"))
			Expect(resource.CreateExampleReconcileBody).To(Equal(false))

			resource = options.NewResource(
				&config.Config{
					Version:    config.Version2,
					Domain:     "test.io",
					Repo:       "test",
					MultiGroup: true,
				},
				true,
			)
			Expect(resource.Namespaced).To(Equal(options.Namespaced))
			Expect(resource.Group).To(Equal(options.Group))
			Expect(resource.GroupPackageName).To(Equal("crew"))
			Expect(resource.Version).To(Equal(options.Version))
			Expect(resource.Kind).To(Equal(options.Kind))
			Expect(resource.Plural).To(Equal("firstmates"))
			Expect(resource.ImportAlias).To(Equal("crewv1"))
			Expect(resource.Package).To(Equal(path.Join("test", "apis", "crew", "v1")))
			Expect(resource.Domain).To(Equal("crew.test.io"))
			Expect(resource.CreateExampleReconcileBody).To(Equal(false))

			resource = options.NewV1Resource(
				&config.Config{
					Version: config.Version1,
					Domain:  "test.io",
					Repo:    "test",
				},
				true,
			)
			Expect(resource.Namespaced).To(Equal(options.Namespaced))
			Expect(resource.Group).To(Equal(options.Group))
			Expect(resource.GroupPackageName).To(Equal("crew"))
			Expect(resource.Version).To(Equal(options.Version))
			Expect(resource.Kind).To(Equal(options.Kind))
			Expect(resource.Plural).To(Equal("firstmates"))
			Expect(resource.ImportAlias).To(Equal("crewv1"))
			Expect(resource.Package).To(Equal(path.Join("test", "pkg", "apis", "crew", "v1")))
			Expect(resource.Domain).To(Equal("crew.test.io"))
			Expect(resource.CreateExampleReconcileBody).To(Equal(options.CreateExampleReconcileBody))
		})

		It("should default the Plural by pluralizing the Kind", func() {
			singleGroupConfig := &config.Config{
				Version: config.Version2,
			}
			multiGroupConfig := &config.Config{
				Version:    config.Version2,
				MultiGroup: true,
			}
			v1Config := &config.Config{
				Version: config.Version1,
			}

			options := &Options{Group: "crew", Version: "v1", Kind: "FirstMate"}
			Expect(options.Validate()).To(Succeed())

			resource := options.NewResource(singleGroupConfig, true)
			Expect(resource.Plural).To(Equal("firstmates"))

			resource = options.NewResource(multiGroupConfig, true)
			Expect(resource.Plural).To(Equal("firstmates"))

			resource = options.NewV1Resource(v1Config, true)
			Expect(resource.Plural).To(Equal("firstmates"))

			options = &Options{Group: "crew", Version: "v1", Kind: "Fish"}
			Expect(options.Validate()).To(Succeed())

			resource = options.NewResource(singleGroupConfig, true)
			Expect(resource.Plural).To(Equal("fish"))

			resource = options.NewResource(multiGroupConfig, true)
			Expect(resource.Plural).To(Equal("fish"))

			resource = options.NewV1Resource(v1Config, true)
			Expect(resource.Plural).To(Equal("fish"))

			options = &Options{Group: "crew", Version: "v1", Kind: "Helmswoman"}
			Expect(options.Validate()).To(Succeed())

			resource = options.NewResource(singleGroupConfig, true)
			Expect(resource.Plural).To(Equal("helmswomen"))

			resource = options.NewResource(multiGroupConfig, true)
			Expect(resource.Plural).To(Equal("helmswomen"))

			resource = options.NewV1Resource(v1Config, true)
			Expect(resource.Plural).To(Equal("helmswomen"))
		})

		It("should keep the Plural if specified", func() {
			options := &Options{Group: "crew", Version: "v1", Kind: "FirstMate", Plural: "mates"}
			Expect(options.Validate()).To(Succeed())

			resource := options.NewResource(
				&config.Config{
					Version: config.Version2,
				},
				true,
			)
			Expect(resource.Plural).To(Equal("mates"))

			resource = options.NewResource(
				&config.Config{
					Version:    config.Version2,
					MultiGroup: true,
				},
				true,
			)
			Expect(resource.Plural).To(Equal("mates"))

			resource = options.NewV1Resource(
				&config.Config{
					Version: config.Version1,
				},
				true,
			)
			Expect(resource.Plural).To(Equal("mates"))
		})

		It("should allow hyphens and dots in group names", func() {
			singleGroupConfig := &config.Config{
				Version: config.Version2,
				Domain:  "test.io",
				Repo:    "test",
			}
			multiGroupConfig := &config.Config{
				Version:    config.Version2,
				Domain:     "test.io",
				Repo:       "test",
				MultiGroup: true,
			}
			v1Config := &config.Config{
				Version: config.Version1,
				Domain:  "test.io",
				Repo:    "test",
			}

			options := &Options{Group: "my-project", Version: "v1", Kind: "FirstMate"}
			Expect(options.Validate()).To(Succeed())

			resource := options.NewResource(singleGroupConfig, true)
			Expect(resource.Group).To(Equal(options.Group))
			Expect(resource.GroupPackageName).To(Equal("myproject"))
			Expect(resource.ImportAlias).To(Equal("myprojectv1"))
			Expect(resource.Package).To(Equal(path.Join("test", "api", "v1")))
			Expect(resource.Domain).To(Equal("my-project.test.io"))

			resource = options.NewResource(multiGroupConfig, true)
			Expect(resource.Group).To(Equal(options.Group))
			Expect(resource.GroupPackageName).To(Equal("myproject"))
			Expect(resource.ImportAlias).To(Equal("myprojectv1"))
			Expect(resource.Package).To(Equal(path.Join("test", "apis", "my-project", "v1")))
			Expect(resource.Domain).To(Equal("my-project.test.io"))

			resource = options.NewV1Resource(v1Config, true)
			Expect(resource.Group).To(Equal(options.Group))
			Expect(resource.GroupPackageName).To(Equal("myproject"))
			Expect(resource.ImportAlias).To(Equal("myprojectv1"))
			Expect(resource.Package).To(Equal(path.Join("test", "pkg", "apis", "myproject", "v1")))
			Expect(resource.Domain).To(Equal("my-project.test.io"))

			options = &Options{Group: "my.project", Version: "v1", Kind: "FirstMate"}
			Expect(options.Validate()).To(Succeed())

			resource = options.NewResource(singleGroupConfig, true)
			Expect(resource.Group).To(Equal(options.Group))
			Expect(resource.GroupPackageName).To(Equal("myproject"))
			Expect(resource.ImportAlias).To(Equal("myprojectv1"))
			Expect(resource.Package).To(Equal(path.Join("test", "api", "v1")))
			Expect(resource.Domain).To(Equal("my.project.test.io"))

			resource = options.NewResource(multiGroupConfig, true)
			Expect(resource.Group).To(Equal(options.Group))
			Expect(resource.GroupPackageName).To(Equal("myproject"))
			Expect(resource.ImportAlias).To(Equal("myprojectv1"))
			Expect(resource.Package).To(Equal(path.Join("test", "apis", "my.project", "v1")))
			Expect(resource.Domain).To(Equal("my.project.test.io"))

			resource = options.NewV1Resource(v1Config, true)
			Expect(resource.Group).To(Equal(options.Group))
			Expect(resource.GroupPackageName).To(Equal("myproject"))
			Expect(resource.ImportAlias).To(Equal("myprojectv1"))
			Expect(resource.Package).To(Equal(path.Join("test", "pkg", "apis", "myproject", "v1")))
			Expect(resource.Domain).To(Equal("my.project.test.io"))
		})

		It("should not append '.' if provided an empty domain", func() {
			options := &Options{Group: "crew", Version: "v1", Kind: "FirstMate"}
			Expect(options.Validate()).To(Succeed())

			resource := options.NewResource(
				&config.Config{
					Version: config.Version2,
				},
				true,
			)
			Expect(resource.Domain).To(Equal("crew"))

			resource = options.NewResource(
				&config.Config{
					Version:    config.Version2,
					MultiGroup: true,
				},
				true,
			)
			Expect(resource.Domain).To(Equal("crew"))

			resource = options.NewV1Resource(
				&config.Config{
					Version: config.Version1,
				},
				true,
			)
			Expect(resource.Domain).To(Equal("crew"))
		})

		It("should use core apis", func() {
			singleGroupConfig := &config.Config{
				Version: config.Version2,
				Domain:  "test.io",
				Repo:    "test",
			}
			multiGroupConfig := &config.Config{
				Version:    config.Version2,
				Domain:     "test.io",
				Repo:       "test",
				MultiGroup: true,
			}
			v1Config := &config.Config{
				Version: config.Version1,
				Domain:  "test.io",
				Repo:    "test",
			}

			options := &Options{Group: "apps", Version: "v1", Kind: "FirstMate"}
			Expect(options.Validate()).To(Succeed())

			resource := options.NewResource(singleGroupConfig, false)
			Expect(resource.Package).To(Equal(path.Join("k8s.io", "api", options.Group, options.Version)))
			Expect(resource.Domain).To(Equal("apps"))

			resource = options.NewResource(multiGroupConfig, false)
			Expect(resource.Package).To(Equal(path.Join("k8s.io", "api", options.Group, options.Version)))
			Expect(resource.Domain).To(Equal("apps"))

			resource = options.NewV1Resource(v1Config, false)
			Expect(resource.Package).To(Equal(path.Join("k8s.io", "api", options.Group, options.Version)))
			Expect(resource.Domain).To(Equal("apps"))

			options = &Options{Group: "authentication", Version: "v1", Kind: "FirstMate"}
			Expect(options.Validate()).To(Succeed())

			resource = options.NewResource(singleGroupConfig, false)
			Expect(resource.Package).To(Equal(path.Join("k8s.io", "api", options.Group, options.Version)))
			Expect(resource.Domain).To(Equal("authentication.k8s.io"))

			resource = options.NewResource(multiGroupConfig, false)
			Expect(resource.Package).To(Equal(path.Join("k8s.io", "api", options.Group, options.Version)))
			Expect(resource.Domain).To(Equal("authentication.k8s.io"))

			resource = options.NewV1Resource(v1Config, false)
			Expect(resource.Package).To(Equal(path.Join("k8s.io", "api", options.Group, options.Version)))
			Expect(resource.Domain).To(Equal("authentication.k8s.io"))
		})

		It("should create an example reconcile body if requested (v1 only)", func() {
			options := &Options{Group: "crew", Version: "v1", Kind: "FirstMate", CreateExampleReconcileBody: true}
			Expect(options.Validate()).To(Succeed())

			resource := options.NewResource(
				&config.Config{
					Version: config.Version2,
				},
				true,
			)
			Expect(resource.CreateExampleReconcileBody).To(Equal(false))

			resource = options.NewResource(
				&config.Config{
					Version:    config.Version2,
					MultiGroup: true,
				},
				true,
			)
			Expect(resource.CreateExampleReconcileBody).To(Equal(false))

			resource = options.NewV1Resource(
				&config.Config{
					Version: config.Version1,
				},
				true,
			)
			Expect(resource.CreateExampleReconcileBody).To(Equal(options.CreateExampleReconcileBody))
		})
	})
})
