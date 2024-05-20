/*
Copyright 2022 The Kubernetes Authors.

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

package machinery

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	cfgv3 "sigs.k8s.io/kubebuilder/v4/pkg/config/v3"
	"sigs.k8s.io/kubebuilder/v4/pkg/model/resource"
)

type templateBase struct {
	path           string
	ifExistsAction IfExistsAction
}

func (t templateBase) GetPath() string {
	return t.path
}

func (t templateBase) GetIfExistsAction() IfExistsAction {
	return t.ifExistsAction
}

type templateWithDomain struct {
	templateBase
	domain string
}

func (t *templateWithDomain) InjectDomain(domain string) {
	t.domain = domain
}

type templateWithRepository struct {
	templateBase
	repository string
}

func (t *templateWithRepository) InjectRepository(repository string) {
	t.repository = repository
}

type templateWithProjectName struct {
	templateBase
	projectName string
}

func (t *templateWithProjectName) InjectProjectName(projectName string) {
	t.projectName = projectName
}

type templateWithMultiGroup struct {
	templateBase
	multiGroup bool
}

func (t *templateWithMultiGroup) InjectMultiGroup(multiGroup bool) {
	t.multiGroup = multiGroup
}

type templateWithBoilerplate struct {
	templateBase
	boilerplate string
}

func (t *templateWithBoilerplate) InjectBoilerplate(boilerplate string) {
	t.boilerplate = boilerplate
}

type templateWithResource struct {
	templateBase
	resource *resource.Resource
}

func (t *templateWithResource) InjectResource(res *resource.Resource) {
	t.resource = res
}

var _ = Describe("injector", func() {
	tmp := templateBase{
		path:           "my/path/to/file",
		ifExistsAction: Error,
	}

	Context("injectInto", func() {
		Context("Config", func() {
			var c config.Config

			BeforeEach(func() {
				c = cfgv3.New()
			})

			Context("Domain", func() {
				var template *templateWithDomain

				BeforeEach(func() {
					template = &templateWithDomain{templateBase: tmp}
				})

				It("should not inject anything if the config is nil", func() {
					injector{}.injectInto(template)
					Expect(template.domain).To(Equal(""))
				})

				It("should not inject anything if the config doesn't have a domain set", func() {
					injector{config: c}.injectInto(template)
					Expect(template.domain).To(Equal(""))
				})

				It("should inject if the config has a domain set", func() {
					const domain = "my.domain"
					Expect(c.SetDomain(domain)).To(Succeed())

					injector{config: c}.injectInto(template)
					Expect(template.domain).To(Equal(domain))
				})
			})

			Context("Repository", func() {
				var template *templateWithRepository

				BeforeEach(func() {
					template = &templateWithRepository{templateBase: tmp}
				})

				It("should not inject anything if the config is nil", func() {
					injector{}.injectInto(template)
					Expect(template.repository).To(Equal(""))
				})

				It("should not inject anything if the config doesn't have a repository set", func() {
					injector{config: c}.injectInto(template)
					Expect(template.repository).To(Equal(""))
				})

				It("should inject if the config has a repository set", func() {
					const repo = "test"
					Expect(c.SetRepository(repo)).To(Succeed())

					injector{config: c}.injectInto(template)
					Expect(template.repository).To(Equal(repo))
				})
			})

			Context("Project name", func() {
				var template *templateWithProjectName

				BeforeEach(func() {
					template = &templateWithProjectName{templateBase: tmp}
				})

				It("should not inject anything if the config is nil", func() {
					injector{}.injectInto(template)
					Expect(template.projectName).To(Equal(""))
				})

				It("should not inject anything if the config doesn't have a project name set", func() {
					injector{config: c}.injectInto(template)
					Expect(template.projectName).To(Equal(""))
				})

				It("should inject if the config has a project name set", func() {
					const projectName = "my project"
					Expect(c.SetProjectName(projectName)).To(Succeed())

					injector{config: c}.injectInto(template)
					Expect(template.projectName).To(Equal(projectName))
				})
			})

			Context("Multi-group", func() {
				var template *templateWithMultiGroup

				BeforeEach(func() {
					template = &templateWithMultiGroup{templateBase: tmp}
				})

				It("should not inject anything if the config is nil", func() {
					injector{}.injectInto(template)
					Expect(template.multiGroup).To(BeFalse())
				})

				It("should not set the flag if the config doesn't have the multi-group flag set", func() {
					injector{config: c}.injectInto(template)
					Expect(template.multiGroup).To(BeFalse())
				})

				It("should set the flag if the config has the multi-group flag set", func() {
					Expect(c.SetMultiGroup()).To(Succeed())

					injector{config: c}.injectInto(template)
					Expect(template.multiGroup).To(BeTrue())
				})
			})
		})

		Context("Boilerplate", func() {
			var template *templateWithBoilerplate

			BeforeEach(func() {
				template = &templateWithBoilerplate{templateBase: tmp}
			})

			It("should not inject anything if no boilerplate was set", func() {
				injector{}.injectInto(template)
				Expect(template.boilerplate).To(Equal(""))
			})

			It("should inject if the a boilerplate was set", func() {
				const boilerplate = `Copyright "The Kubernetes Authors"`

				injector{boilerplate: boilerplate}.injectInto(template)
				Expect(template.boilerplate).To(Equal(boilerplate))
			})
		})

		Context("Resource", func() {
			var template *templateWithResource

			BeforeEach(func() {
				template = &templateWithResource{templateBase: tmp}
			})

			It("should not inject anything if the resource is nil", func() {
				injector{}.injectInto(template)
				Expect(template.resource).To(BeNil())
			})

			It("should inject if the config has a domain set", func() {
				res := &resource.Resource{
					GVK: resource.GVK{
						Group:   "group",
						Domain:  "my.domain",
						Version: "v1",
						Kind:    "Kind",
					},
				}

				injector{resource: res}.injectInto(template)
				Expect(template.resource).To(Equal(res))
			})
		})
	})
})
