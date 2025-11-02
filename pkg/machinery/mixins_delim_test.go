/*
Copyright 2025 The Kubernetes Authors.

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

	"sigs.k8s.io/kubebuilder/v4/pkg/model/resource"
)

var _ = Describe("TemplateMixin Delimiters", func() {
	var tmp TemplateMixin

	BeforeEach(func() {
		tmp = TemplateMixin{}
	})

	Context("SetDelim and GetDelim", func() {
		It("should set and get custom delimiters", func() {
			tmp.SetDelim("[[", "]]")
			left, right := tmp.GetDelim()
			Expect(left).To(Equal("[["))
			Expect(right).To(Equal("]]"))
		})

		It("should return empty strings when delimiters are not set", func() {
			left, right := tmp.GetDelim()
			Expect(left).To(Equal(""))
			Expect(right).To(Equal(""))
		})

		It("should allow setting delimiters multiple times", func() {
			tmp.SetDelim("[[", "]]")
			left, right := tmp.GetDelim()
			Expect(left).To(Equal("[["))
			Expect(right).To(Equal("]]"))

			tmp.SetDelim("<%", "%>")
			left, right = tmp.GetDelim()
			Expect(left).To(Equal("<%"))
			Expect(right).To(Equal("%>"))
		})
	})
})

var _ = Describe("Mixins injection behaviors", func() {
	Context("DomainMixin", func() {
		It("should not overwrite existing domain", func() {
			tmp := DomainMixin{Domain: "existing.domain"}
			tmp.InjectDomain("new.domain")
			Expect(tmp.Domain).To(Equal("existing.domain"))
		})

		It("should inject domain when empty", func() {
			tmp := DomainMixin{}
			tmp.InjectDomain("new.domain")
			Expect(tmp.Domain).To(Equal("new.domain"))
		})
	})

	Context("RepositoryMixin", func() {
		It("should not overwrite existing repository", func() {
			tmp := RepositoryMixin{Repo: "existing.repo"}
			tmp.InjectRepository("new.repo")
			Expect(tmp.Repo).To(Equal("existing.repo"))
		})

		It("should inject repository when empty", func() {
			tmp := RepositoryMixin{}
			tmp.InjectRepository("new.repo")
			Expect(tmp.Repo).To(Equal("new.repo"))
		})
	})

	Context("ProjectNameMixin", func() {
		It("should not overwrite existing project name", func() {
			tmp := ProjectNameMixin{ProjectName: "existing"}
			tmp.InjectProjectName("new")
			Expect(tmp.ProjectName).To(Equal("existing"))
		})

		It("should inject project name when empty", func() {
			tmp := ProjectNameMixin{}
			tmp.InjectProjectName("new")
			Expect(tmp.ProjectName).To(Equal("new"))
		})
	})

	Context("BoilerplateMixin", func() {
		It("should not overwrite existing boilerplate", func() {
			tmp := BoilerplateMixin{Boilerplate: "existing"}
			tmp.InjectBoilerplate("new")
			Expect(tmp.Boilerplate).To(Equal("existing"))
		})

		It("should inject boilerplate when empty", func() {
			tmp := BoilerplateMixin{}
			tmp.InjectBoilerplate("new")
			Expect(tmp.Boilerplate).To(Equal("new"))
		})
	})

	Context("ResourceMixin", func() {
		It("should not overwrite existing resource", func() {
			existing := &resource.Resource{GVK: resource.GVK{Group: "existing"}}
			tmp := ResourceMixin{Resource: existing}
			tmp.InjectResource(&resource.Resource{GVK: resource.GVK{Group: "new"}})
			Expect(tmp.Resource.Group).To(Equal("existing"))
		})

		It("should inject resource when nil", func() {
			tmp := ResourceMixin{}
			res := &resource.Resource{GVK: resource.GVK{Group: "new"}}
			tmp.InjectResource(res)
			Expect(tmp.Resource.Group).To(Equal("new"))
		})
	})
})

var _ = Describe("IfNotExistsActionMixin", func() {
	Context("GetIfNotExistsAction", func() {
		It("should return the configured action", func() {
			tmp := IfNotExistsActionMixin{IfNotExistsAction: IgnoreFile}
			Expect(tmp.GetIfNotExistsAction()).To(Equal(IgnoreFile))
		})

		It("should return zero value when not set", func() {
			tmp := IfNotExistsActionMixin{}
			Expect(tmp.GetIfNotExistsAction()).To(Equal(IfNotExistsAction(0)))
		})
	})
})
