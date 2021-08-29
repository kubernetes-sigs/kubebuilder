/*
Copyright 2021 The Kubernetes Authors.

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

package v1

import (
	"fmt"

	"sigs.k8s.io/kubebuilder/v3/pkg/config"
	"sigs.k8s.io/kubebuilder/v3/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugin"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugins/golang/declarative/v1/internal/templates"
)

var _ plugin.InitSubcommand = &initSubcommand{}

type initSubcommand struct {
	config config.Config
}

func (p *initSubcommand) InjectConfig(c config.Config) error {
	p.config = c

	return nil
}

func (p *initSubcommand) Scaffold(fs machinery.Filesystem) error {
	// Initialize the machinery.Scaffold that will write the files to disk
	scaffold := machinery.NewScaffold(fs,
		machinery.WithConfig(p.config),
	)

	if err := scaffold.Execute(
		&templates.Dockerfile{},
	); err != nil {
		return fmt.Errorf("error updating scaffold: %w", err)
	}

	return nil
}
