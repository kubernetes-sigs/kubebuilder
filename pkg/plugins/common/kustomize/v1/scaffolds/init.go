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
	log "github.com/sirupsen/logrus"

	"sigs.k8s.io/kubebuilder/v3/pkg/config"
	"sigs.k8s.io/kubebuilder/v3/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugins"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugins/common/kustomize/v1/scaffolds/internal/templates/config/kdefault"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugins/common/kustomize/v1/scaffolds/internal/templates/config/manager"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugins/common/kustomize/v1/scaffolds/internal/templates/config/prometheus"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugins/common/kustomize/v1/scaffolds/internal/templates/config/rbac"
)

const (
	imageName = "controller:latest"
)

var _ plugins.Scaffolder = &initScaffolder{}

type initScaffolder struct {
	config config.Config

	// fs is the filesystem that will be used by the scaffolder
	fs machinery.Filesystem
}

// NewInitScaffolder returns a new Scaffolder for project initialization operations
func NewInitScaffolder(config config.Config) plugins.Scaffolder {
	return &initScaffolder{
		config: config,
	}
}

// InjectFS implements cmdutil.Scaffolder
func (s *initScaffolder) InjectFS(fs machinery.Filesystem) {
	s.fs = fs
}

// Scaffold implements cmdutil.Scaffolder
func (s *initScaffolder) Scaffold() error {
	log.Println("Writing kustomize manifests for you to edit...")

	// Initialize the machinery.Scaffold that will write the files to disk
	scaffold := machinery.NewScaffold(s.fs,
		machinery.WithConfig(s.config),
	)

	templates := []machinery.Builder{
		&rbac.Kustomization{},
		&rbac.AuthProxyRole{},
		&rbac.AuthProxyRoleBinding{},
		&rbac.AuthProxyService{},
		&rbac.AuthProxyClientRole{},
		&rbac.RoleBinding{},
		&rbac.LeaderElectionRole{},
		&rbac.LeaderElectionRoleBinding{},
		&rbac.ServiceAccount{},
		&manager.Kustomization{},
		&manager.Config{Image: imageName},
		&kdefault.Kustomization{},
		&kdefault.ManagerAuthProxyPatch{},
		&kdefault.ManagerConfigPatch{},
		&prometheus.Kustomization{},
		&prometheus.Monitor{},
	}

	if s.config.IsComponentConfig() {
		templates = append(templates, &manager.ControllerManagerConfig{})
	}

	return scaffold.Execute(templates...)
}
