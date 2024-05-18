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

	log "github.com/sirupsen/logrus"
	"github.com/spf13/afero"

	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v4/pkg/model/resource"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/golang/v4/scaffolds/internal/templates"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/golang/v4/scaffolds/internal/templates/api"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/golang/v4/scaffolds/internal/templates/hack"
)

var _ plugins.Scaffolder = &webhookScaffolder{}

type webhookScaffolder struct {
	config   config.Config
	resource resource.Resource

	// fs is the filesystem that will be used by the scaffolder
	fs machinery.Filesystem

	// force indicates whether to scaffold controller files even if it exists or not
	force bool
}

// NewWebhookScaffolder returns a new Scaffolder for v2 webhook creation operations
func NewWebhookScaffolder(config config.Config, resource resource.Resource, force bool) plugins.Scaffolder {
	return &webhookScaffolder{
		config:   config,
		resource: resource,
		force:    force,
	}
}

// InjectFS implements cmdutil.Scaffolder
func (s *webhookScaffolder) InjectFS(fs machinery.Filesystem) {
	s.fs = fs
}

// Scaffold implements cmdutil.Scaffolder
func (s *webhookScaffolder) Scaffold() error {
	log.Println("Writing scaffold for you to edit...")

	// Load the boilerplate
	boilerplate, err := afero.ReadFile(s.fs.FS, hack.DefaultBoilerplatePath)
	if err != nil {
		return fmt.Errorf("error scaffolding webhook: unable to load boilerplate: %w", err)
	}

	// Initialize the machinery.Scaffold that will write the files to disk
	scaffold := machinery.NewScaffold(s.fs,
		machinery.WithConfig(s.config),
		machinery.WithBoilerplate(string(boilerplate)),
		machinery.WithResource(&s.resource),
	)

	// Keep track of these values before the update
	doDefaulting := s.resource.HasDefaultingWebhook()
	doValidation := s.resource.HasValidationWebhook()
	doConversion := s.resource.HasConversionWebhook()

	if err := s.config.UpdateResource(s.resource); err != nil {
		return fmt.Errorf("error updating resource: %w", err)
	}

	if err := scaffold.Execute(
		&api.Webhook{Force: s.force},
		&templates.MainUpdater{WireWebhook: true},
		&api.WebhookTest{Force: s.force},
	); err != nil {
		return err
	}

	if doConversion {
		log.Println(`Webhook server has been set up for you.
You need to implement the conversion.Hub and conversion.Convertible interfaces for your CRD types.`)
	}

	// TODO: Add test suite for conversion webhook after #1664 has been merged & conversion tests supported in envtest.
	if doDefaulting || doValidation {
		if err := scaffold.Execute(
			&api.WebhookSuite{K8SVersion: EnvtestK8SVersion},
		); err != nil {
			return err
		}
	}

	return nil
}
