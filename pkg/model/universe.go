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
	"sigs.k8s.io/kubebuilder/v3/pkg/config"
	"sigs.k8s.io/kubebuilder/v3/pkg/model/file"
	"sigs.k8s.io/kubebuilder/v3/pkg/model/resource"
)

// Universe describes the entire state of file generation
type Universe struct {
	// Config stores the project configuration
	Config config.Config `json:"config,omitempty"`

	// Boilerplate is the copyright comment added at the top of scaffolded files
	Boilerplate string `json:"boilerplate,omitempty"`

	// Resource contains the information of the API that is being scaffolded
	Resource *resource.Resource `json:"resource,omitempty"`

	// Files contains the model of the files that are being scaffolded
	Files map[string]*file.File `json:"files,omitempty"`
}

// InjectInto injects fields from the universe into the builder
func (u Universe) InjectInto(builder file.Builder) {
	// Inject project configuration
	if u.Config != nil {
		if builderWithDomain, hasDomain := builder.(file.HasDomain); hasDomain {
			builderWithDomain.InjectDomain(u.Config.GetDomain())
		}
		if builderWithRepository, hasRepository := builder.(file.HasRepository); hasRepository {
			builderWithRepository.InjectRepository(u.Config.GetRepository())
		}
		if builderWithProjectName, hasProjectName := builder.(file.HasProjectName); hasProjectName {
			builderWithProjectName.InjectProjectName(u.Config.GetProjectName())
		}
		if builderWithMultiGroup, hasMultiGroup := builder.(file.HasMultiGroup); hasMultiGroup {
			builderWithMultiGroup.InjectMultiGroup(u.Config.IsMultiGroup())
		}
		if builderWithComponentConfig, hasComponentConfig := builder.(file.HasComponentConfig); hasComponentConfig {
			builderWithComponentConfig.InjectComponentConfig(u.Config.IsComponentConfig())
		}
	}
	// Inject boilerplate
	if builderWithBoilerplate, hasBoilerplate := builder.(file.HasBoilerplate); hasBoilerplate {
		builderWithBoilerplate.InjectBoilerplate(u.Boilerplate)
	}
	// Inject resource
	if u.Resource != nil {
		if builderWithResource, hasResource := builder.(file.HasResource); hasResource {
			builderWithResource.InjectResource(u.Resource)
		}
	}
}
