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

	"sigs.k8s.io/kubebuilder/v4/pkg/model/resource"
)

type mockTemplate struct {
	TemplateMixin
	DomainMixin
	RepositoryMixin
	ProjectNameMixin
	MultiGroupMixin
	BoilerplateMixin
	ResourceMixin
}

type mockInserter struct {
	// InserterMixin requires a different type because it collides with TemplateMixin
	InserterMixin
}

var _ = Describe("TemplateMixin", func() {
	const (
		path           = "path/to/file.go"
		ifExistsAction = SkipFile
		body           = "content"
	)

	tmp := mockTemplate{
		TemplateMixin: TemplateMixin{
			PathMixin:           PathMixin{path},
			IfExistsActionMixin: IfExistsActionMixin{ifExistsAction},
			TemplateBody:        body,
		},
	}

	Context("GetPath", func() {
		It("should return the path", func() {
			Expect(tmp.GetPath()).To(Equal(path))
		})
	})

	Context("GetIfExistsAction", func() {
		It("should return the if-exists action", func() {
			Expect(tmp.GetIfExistsAction()).To(Equal(ifExistsAction))
		})
	})

	Context("GetBody", func() {
		It("should return the body", func() {
			Expect(tmp.GetBody()).To(Equal(body))
		})
	})
})

var _ = Describe("InserterMixin", func() {
	const path = "path/to/file.go"

	tmp := mockInserter{
		InserterMixin: InserterMixin{
			PathMixin: PathMixin{path},
		},
	}

	Context("GetPath", func() {
		It("should return the path", func() {
			Expect(tmp.GetPath()).To(Equal(path))
		})
	})

	Context("GetIfExistsAction", func() {
		It("should return overwrite file always", func() {
			Expect(tmp.GetIfExistsAction()).To(Equal(OverwriteFile))
		})
	})
})

var _ = Describe("DomainMixin", func() {
	const domain = "my.domain"

	tmp := mockTemplate{}

	Context("InjectDomain", func() {
		It("should inject the provided domain", func() {
			tmp.InjectDomain(domain)
			Expect(tmp.Domain).To(Equal(domain))
		})
	})
})

var _ = Describe("RepositoryMixin", func() {
	const repo = "test"

	tmp := mockTemplate{}

	Context("InjectRepository", func() {
		It("should inject the provided repository", func() {
			tmp.InjectRepository(repo)
			Expect(tmp.Repo).To(Equal(repo))
		})
	})
})

var _ = Describe("ProjectNameMixin", func() {
	const name = "my project"

	tmp := mockTemplate{}

	Context("InjectProjectName", func() {
		It("should inject the provided project name", func() {
			tmp.InjectProjectName(name)
			Expect(tmp.ProjectName).To(Equal(name))
		})
	})
})

var _ = Describe("MultiGroupMixin", func() {
	tmp := mockTemplate{}

	Context("InjectMultiGroup", func() {
		It("should inject the provided multi group flag", func() {
			tmp.InjectMultiGroup(true)
			Expect(tmp.MultiGroup).To(BeTrue())
		})
	})
})

var _ = Describe("BoilerplateMixin", func() {
	const boilerplate = "Copyright"

	tmp := mockTemplate{}

	Context("InjectBoilerplate", func() {
		It("should inject the provided boilerplate", func() {
			tmp.InjectBoilerplate(boilerplate)
			Expect(tmp.Boilerplate).To(Equal(boilerplate))
		})
	})
})

var _ = Describe("ResourceMixin", func() {
	res := &resource.Resource{GVK: resource.GVK{
		Group:   "group",
		Domain:  "my.domain",
		Version: "v1",
		Kind:    "Kind",
	}}

	tmp := mockTemplate{}

	Context("InjectResource", func() {
		It("should inject the provided resource", func() {
			tmp.InjectResource(res)
			Expect(tmp.Resource.GVK.IsEqualTo(res.GVK)).To(BeTrue())
		})
	})
})
