/*
Copyright 2018 The Kubernetes Authors.

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

package file

import (
	"sigs.k8s.io/kubebuilder/pkg/model/resource"
)

// PathMixin provides file builders with a path field
type PathMixin struct {
	// Path is the of the file
	Path string
}

// GetPath implements Builder
func (t *PathMixin) GetPath() string {
	return t.Path
}

// IfExistsActionMixin provides file builders with a if-exists-action field
type IfExistsActionMixin struct {
	// IfExistsAction determines what to do if the file exists
	IfExistsAction IfExistsAction
}

// GetIfExistsAction implements Builder
func (t *IfExistsActionMixin) GetIfExistsAction() IfExistsAction {
	return t.IfExistsAction
}

// TemplateMixin is the mixin that should be embedded in Template builders
type TemplateMixin struct {
	PathMixin
	IfExistsActionMixin

	// TemplateBody is the template body to execute
	TemplateBody string
}

// GetBody implements Template
func (t *TemplateMixin) GetBody() string {
	return t.TemplateBody
}

// InserterMixin is the mixin that should be embedded in Inserter builders
type InserterMixin struct {
	PathMixin
}

// GetIfExistsAction implements Builder
func (t *InserterMixin) GetIfExistsAction() IfExistsAction {
	// Inserter builders always need to overwrite previous files
	return Overwrite
}

// DomainMixin provides templates with a injectable domain field
type DomainMixin struct {
	// Domain is the domain for the APIs
	Domain string
}

// InjectDomain implements HasDomain
func (m *DomainMixin) InjectDomain(domain string) {
	if m.Domain == "" {
		m.Domain = domain
	}
}

// RepositoryMixin provides templates with a injectable repository field
type RepositoryMixin struct {
	// Repo is the go project package path
	Repo string
}

// InjectRepository implements HasRepository
func (m *RepositoryMixin) InjectRepository(repository string) {
	if m.Repo == "" {
		m.Repo = repository
	}
}

// MultiGroupMixin provides templates with a injectable multi-group flag field
type MultiGroupMixin struct {
	// MultiGroup is the multi-group flag
	MultiGroup bool
}

// InjectMultiGroup implements HasMultiGroup
func (m *MultiGroupMixin) InjectMultiGroup(flag bool) {
	m.MultiGroup = flag
}

// BoilerplateMixin provides templates with a injectable boilerplate field
type BoilerplateMixin struct {
	// Boilerplate is the contents of a Boilerplate go header file
	Boilerplate string
}

// InjectBoilerplate implements HasBoilerplate
func (m *BoilerplateMixin) InjectBoilerplate(boilerplate string) {
	if m.Boilerplate == "" {
		m.Boilerplate = boilerplate
	}
}

// ResourceMixin provides templates with a injectable resource field
type ResourceMixin struct {
	Resource *resource.Resource
}

// InjectResource implements HasResource
func (m *ResourceMixin) InjectResource(res *resource.Resource) {
	if m.Resource == nil {
		m.Resource = res
	}
}
