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

// Input is the input for scaffolding a file
type Input struct {
	// Path is the file to write
	Path string

	// IfExistsAction determines what to do if the file exists
	IfExistsAction IfExistsAction

	// TemplateBody is the template body to execute
	TemplateBody string

	// Domain is the domain for the APIs
	Domain string

	// Repo is the go project package
	Repo string
}

// Domain allows a domain to be set on an object
type Domain interface {
	// SetDomain sets the domain
	SetDomain(string)
}

// SetDomain sets the domain
func (i *Input) SetDomain(d string) {
	if i.Domain == "" {
		i.Domain = d
	}
}

// Repo allows a repo to be set on an object
type Repo interface {
	// SetRepo sets the repo
	SetRepo(string)
}

// SetRepo sets the repo
func (i *Input) SetRepo(r string) {
	if i.Repo == "" {
		i.Repo = r
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

// Template is a scaffoldable file template
type Template interface {
	// GetInput returns the Input for creating a scaffold file
	GetInput() (Input, error)
}

// RequiresValidation is a file that requires validation
type RequiresValidation interface {
	Template
	// Validate returns true if the template has valid values
	Validate() error
}
