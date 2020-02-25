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

package scaffold

import (
	"sigs.k8s.io/kubebuilder/pkg/model"
	"sigs.k8s.io/kubebuilder/pkg/model/config"
	"sigs.k8s.io/kubebuilder/pkg/scaffold/internal/machinery"
	templatesv1 "sigs.k8s.io/kubebuilder/pkg/scaffold/internal/templates/v1"
)

type updateScaffolder struct {
	config *config.Config
}

func NewUpdateScaffolder(config *config.Config) Scaffolder {
	return &updateScaffolder{
		config: config,
	}
}

func (s *updateScaffolder) Scaffold() error {
	return machinery.NewScaffold().Execute(
		model.NewUniverse(
			model.WithConfig(s.config),
			model.WithoutBoilerplate,
		),
		&templatesv1.GopkgToml{},
	)
}
