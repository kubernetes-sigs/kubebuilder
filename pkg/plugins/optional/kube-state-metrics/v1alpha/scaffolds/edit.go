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

package scaffolds

import (
	"sigs.k8s.io/kubebuilder/v3/pkg/config"
	"sigs.k8s.io/kubebuilder/v3/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v3/pkg/model/resource"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugins"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugins/optional/kube-state-metrics/v1alpha/scaffolds/internal/templates"
)

var _ plugins.Scaffolder = &editScaffolder{}

type editScaffolder struct {
	fs     machinery.Filesystem
	config config.Config
}

func NewEditScaffolder(config config.Config) plugins.Scaffolder {
	return &editScaffolder{
		config: config,
	}
}

func (s *editScaffolder) InjectFS(fs machinery.Filesystem) {
	s.fs = fs
}

func (s *editScaffolder) Scaffold() error {
	rs, err := s.config.GetResources()
	if err != nil {
		return err
	}

	gvks := []resource.GVK{}
	for _, r := range rs {
		gvks = append(gvks, r.Copy().GVK)
	}

	scaffold := machinery.NewScaffold(s.fs)

	return scaffold.Execute(
		&templates.ConfigManifest{GVKs: gvks},
	)
}
