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
	"fmt"
	log "log/slog"

	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/optional/autoupdate/v1alpha/scaffolds/internal/github"
)

var _ plugins.Scaffolder = &editScaffolder{}

type editScaffolder struct {
	config config.Config

	// fs is the filesystem that will be used by the scaffolder
	fs machinery.Filesystem

	// useGHModels determines if GitHub Models AI summary should be enabled
	useGHModels bool

	// openGHIssue determines if the workflow should create GitHub Issues
	openGHIssue bool

	// openGHPR determines if the workflow should create GitHub Pull Requests
	openGHPR bool
}

// NewInitScaffolder returns a new Scaffolder for project initialization operations
func NewInitScaffolder(useGHModels, openGHIssue, openGHPR bool) plugins.Scaffolder {
	return &editScaffolder{
		useGHModels: useGHModels,
		openGHIssue: openGHIssue,
		openGHPR:    openGHPR,
	}
}

// InjectFS implements cmdutil.Scaffolder
func (s *editScaffolder) InjectFS(fs machinery.Filesystem) {
	s.fs = fs
}

// Scaffold implements cmdutil.Scaffolder
func (s *editScaffolder) Scaffold() error {
	log.Info("Writing scaffold for you to edit...")

	scaffold := machinery.NewScaffold(s.fs,
		machinery.WithConfig(s.config),
	)

	err := scaffold.Execute(
		&github.AutoUpdate{
			UseGHModels: s.useGHModels,
			OpenGHIssue: s.openGHIssue,
			OpenGHPR:    s.openGHPR,
		},
	)
	if err != nil {
		return fmt.Errorf("failed to execute init scaffold: %w", err)
	}

	return nil
}
