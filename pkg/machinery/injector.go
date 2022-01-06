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
	"sigs.k8s.io/kubebuilder/v3/pkg/config"
	"sigs.k8s.io/kubebuilder/v3/pkg/model/resource"
)

// injector is used to inject certain fields to file templates.
type injector struct {
	// config stores the project configuration.
	config config.Config

	// boilerplate is the copyright comment added at the top of scaffolded files.
	boilerplate string

	// resource contains the information of the API that is being scaffolded.
	resource *resource.Resource
}

// injectInto injects fields from the universe into the builder
func (i injector) injectInto(builder Builder) {
	// Inject project configuration
	if i.config != nil {
		if builderWithDomain, hasDomain := builder.(HasDomain); hasDomain {
			builderWithDomain.InjectDomain(i.config.GetDomain())
		}
		if builderWithRepository, hasRepository := builder.(HasRepository); hasRepository {
			builderWithRepository.InjectRepository(i.config.GetRepository())
		}
		if builderWithProjectName, hasProjectName := builder.(HasProjectName); hasProjectName {
			builderWithProjectName.InjectProjectName(i.config.GetProjectName())
		}
		if builderWithMultiGroup, hasMultiGroup := builder.(HasMultiGroup); hasMultiGroup {
			builderWithMultiGroup.InjectMultiGroup(i.config.IsMultiGroup())
		}
		if builderWithComponentConfig, hasComponentConfig := builder.(HasComponentConfig); hasComponentConfig {
			builderWithComponentConfig.InjectComponentConfig(i.config.IsComponentConfig())
		}
	}
	// Inject boilerplate
	if builderWithBoilerplate, hasBoilerplate := builder.(HasBoilerplate); hasBoilerplate {
		builderWithBoilerplate.InjectBoilerplate(i.boilerplate)
	}
	// Inject resource
	if i.resource != nil {
		if builderWithResource, hasResource := builder.(HasResource); hasResource {
			builderWithResource.InjectResource(i.resource)
		}
	}
}
