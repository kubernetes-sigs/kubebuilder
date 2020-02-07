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

// Template is a scaffoldable file template
type Template interface {
	// GetTemplateMixin returns the TemplateMixin for creating a scaffold file
	GetTemplateMixin() (TemplateMixin, error)
	// GetPath returns the path to the file location
	GetPath() string
	// GetBody returns the template body
	GetBody() string
	// GetIfExistsAction returns the behavior when creating a file that already exists
	GetIfExistsAction() IfExistsAction
}

// TemplateMixin is the input for scaffolding a file
type TemplateMixin struct {
	// Path is the file to write
	Path string

	// TemplateBody is the template body to execute
	TemplateBody string

	// IfExistsAction determines what to do if the file exists
	IfExistsAction IfExistsAction
}

func(t *TemplateMixin) GetPath() string {
	return t.Path
}

func(t *TemplateMixin) GetBody() string {
	return t.TemplateBody
}

func(t *TemplateMixin) GetIfExistsAction() IfExistsAction {
	return t.IfExistsAction
}

// HasDomain allows the domain to be used on a template
type HasDomain interface {
	// InjectDomain sets the template domain
	InjectDomain(string)
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

// HasRepository allows the repository to be used on a template
type HasRepository interface {
	// InjectRepository sets the template repository
	InjectRepository(string)
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

// HasMultiGroup allows the multi-group flag to be used on a template
type HasMultiGroup interface {
	// InjectMultiGroup sets the template multi-group flag
	InjectMultiGroup(bool)
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

// HasBoilerplate allows a boilerplate to be used on a template
type HasBoilerplate interface {
	// InjectBoilerplate sets the template boilerplate
	InjectBoilerplate(string)
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

// HasResource allows a resource to be used on a template
type HasResource interface {
	// InjectResource sets the template resource
	InjectResource(*resource.Resource)
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

// RequiresValidation is a file that requires validation
type RequiresValidation interface {
	Template
	// Validate returns true if the template has valid values
	Validate() error
}
