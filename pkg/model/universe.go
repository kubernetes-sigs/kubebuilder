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
	"io/ioutil"
	"strings"

	"github.com/gobuffalo/flect"

	"sigs.k8s.io/kubebuilder/pkg/model/config"
	"sigs.k8s.io/kubebuilder/pkg/scaffold/resource"
	"sigs.k8s.io/kubebuilder/pkg/scaffold/util"
)

// Universe describes the entire state of file generation
type Universe struct {
	// Config stores the project configuration
	Config *config.Config `json:"config,omitempty"`

	// Boilerplate is the copyright comment added at the top of scaffolded files
	Boilerplate string `json:"boilerplate,omitempty"`

	// Resource contains the information of the API that is being scaffolded
	Resource *Resource `json:"resource,omitempty"`

	// Files contains the model of the files that are being scaffolded
	Files []*File `json:"files,omitempty"`
}

// NewUniverse creates a new Universe
func NewUniverse(options ...UniverseOption) (*Universe, error) {
	universe := &Universe{}

	// Apply options
	for _, option := range options {
		if err := option(universe); err != nil {
			return nil, err
		}
	}

	return universe, nil
}

// UniverseOption configure Universe
type UniverseOption func(*Universe) error

// WithConfig stores the already loaded project configuration
func WithConfig(projectConfig *config.Config) UniverseOption {
	return func(universe *Universe) error {
		universe.Config = projectConfig
		return nil
	}
}

// WithBoilerplateFrom loads the boilerplate from the provided path
func WithBoilerplateFrom(path string) UniverseOption {
	return func(universe *Universe) error {
		boilerplate, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}

		universe.Boilerplate = string(boilerplate)
		return nil
	}
}

// WithBoilerplate stores the already loaded project configuration
func WithBoilerplate(boilerplate string) UniverseOption {
	return func(universe *Universe) error {
		universe.Boilerplate = string(boilerplate)
		return nil
	}
}

// WithoutBoilerplate is used for files that do not require a boilerplate
func WithoutBoilerplate(universe *Universe) error {
	universe.Boilerplate = "-"
	return nil
}

// WithResource stores the provided resource
func WithResource(resource *resource.Resource, project *config.Config) UniverseOption {
	return func(universe *Universe) error {
		resourceModel := &Resource{
			Namespaced: resource.Namespaced,
			Group:      resource.Group,
			Version:    resource.Version,
			Kind:       resource.Kind,
			Resource:   resource.Resource,
			Plural:     flect.Pluralize(strings.ToLower(resource.Kind)),
		}

		resourceModel.GoPackage, resourceModel.GroupDomain = util.GetResourceInfo(
			resource,
			project.Repo,
			project.Domain,
			project.MultiGroup,
		)

		universe.Resource = resourceModel
		return nil
	}
}
