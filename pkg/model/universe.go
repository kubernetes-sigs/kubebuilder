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

package model

import (
	"sigs.k8s.io/kubebuilder/pkg/model/config"
	"sigs.k8s.io/kubebuilder/pkg/model/file"
	"sigs.k8s.io/kubebuilder/pkg/model/resource"
)

// Universe describes the entire state of file generation
type Universe struct {
	// Config stores the project configuration
	Config *config.Config `json:"config,omitempty"`

	// Boilerplate is the copyright comment added at the top of scaffolded files
	Boilerplate string `json:"boilerplate,omitempty"`

	// Resource contains the information of the API that is being scaffolded
	Resource *resource.Resource `json:"resource,omitempty"`

	// Files contains the model of the files that are being scaffolded
	Files []*file.File `json:"files,omitempty"`
}

// NewUniverse creates a new Universe
func NewUniverse(options ...UniverseOption) *Universe {
	universe := &Universe{}

	// Apply options
	for _, option := range options {
		option(universe)
	}

	return universe
}

// UniverseOption configure Universe
type UniverseOption func(*Universe)

// WithConfig stores the already loaded project configuration
func WithConfig(projectConfig *config.Config) UniverseOption {
	return func(universe *Universe) {
		universe.Config = projectConfig
	}
}

// WithBoilerplate stores the already loaded project configuration
func WithBoilerplate(boilerplate string) UniverseOption {
	return func(universe *Universe) {
		universe.Boilerplate = boilerplate
	}
}

// WithoutBoilerplate is used for files that do not require a boilerplate
func WithoutBoilerplate(universe *Universe) {
	universe.Boilerplate = ""
}

// WithResource stores the provided resource
func WithResource(resource *resource.Resource) UniverseOption {
	return func(universe *Universe) {
		universe.Resource = resource
	}
}

func (u Universe) InjectInto(t file.Template) {
	// Inject project configuration
	if u.Config != nil {
		if templateWithDomain, hasDomain := t.(file.HasDomain); hasDomain {
			templateWithDomain.InjectDomain(u.Config.Domain)
		}
		if templateWithRepository, hasRepository := t.(file.HasRepository); hasRepository {
			templateWithRepository.InjectRepository(u.Config.Repo)
		}
		if templateWithMultiGroup, hasMultiGroup := t.(file.HasMultiGroup); hasMultiGroup {
			templateWithMultiGroup.InjectMultiGroup(u.Config.MultiGroup)
		}
	}
	// Inject boilerplate
	if templateWithBoilerplate, hasBoilerplate := t.(file.HasBoilerplate); hasBoilerplate {
		templateWithBoilerplate.InjectBoilerplate(u.Boilerplate)
	}
	// Inject resource
	if u.Resource != nil {
		if templateWithResource, hasResource := t.(file.HasResource); hasResource {
			templateWithResource.InjectResource(u.Resource)
		}
	}
}
