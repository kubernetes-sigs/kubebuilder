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

package scaffolds

import (
	"fmt"

	"github.com/spf13/afero"

	"sigs.k8s.io/kubebuilder/v3/pkg/config"
	"sigs.k8s.io/kubebuilder/v3/pkg/model"
	"sigs.k8s.io/kubebuilder/v3/pkg/model/resource"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugins/golang/v3/scaffolds/internal/templates"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugins/golang/v3/scaffolds/internal/templates/api"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugins/golang/v3/scaffolds/internal/templates/config/kdefault"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugins/golang/v3/scaffolds/internal/templates/config/webhook"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugins/golang/v3/scaffolds/internal/templates/hack"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugins/internal/cmdutil"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugins/internal/machinery"
)

var _ cmdutil.Scaffolder = &webhookScaffolder{}

type webhookScaffolder struct {
	config      config.Config
	boilerplate string
	resource    resource.Resource

	// fs is the filesystem that will be used by the scaffolder
	fs afero.Fs

	// force indicates whether to scaffold controller files even if it exists or not
	force bool
}

// NewWebhookScaffolder returns a new Scaffolder for v2 webhook creation operations
func NewWebhookScaffolder(config config.Config, resource resource.Resource, force bool) cmdutil.Scaffolder {
	return &webhookScaffolder{
		config:   config,
		resource: resource,
		force:    force,
	}
}

// InjectFS implements cmdutil.Scaffolder
func (s *webhookScaffolder) InjectFS(fs afero.Fs) {
	s.fs = fs
}

func (s *webhookScaffolder) newUniverse() *model.Universe {
	return model.NewUniverse(
		model.WithConfig(s.config),
		model.WithBoilerplate(s.boilerplate),
		model.WithResource(&s.resource),
	)
}

// Scaffold implements cmdutil.Scaffolder
func (s *webhookScaffolder) Scaffold() error {
	fmt.Println("Writing scaffold for you to edit...")

	// Load the boilerplate
	bp, err := afero.ReadFile(s.fs, hack.DefaultBoilerplatePath)
	if err != nil {
		return fmt.Errorf("error scaffolding webhook: unable to load boilerplate: %w", err)
	}
	s.boilerplate = string(bp)

	// Keep track of these values before the update
	doDefaulting := s.resource.HasDefaultingWebhook()
	doValidation := s.resource.HasValidationWebhook()
	doConversion := s.resource.HasConversionWebhook()

	if err := s.config.UpdateResource(s.resource); err != nil {
		return fmt.Errorf("error updating resource: %w", err)
	}

	if err := machinery.NewScaffold(s.fs).Execute(
		s.newUniverse(),
		&api.Webhook{Force: s.force},
		&templates.MainUpdater{WireWebhook: true},
		&kdefault.WebhookCAInjectionPatch{},
		&kdefault.ManagerWebhookPatch{},
		&webhook.Kustomization{Force: s.force},
		&webhook.KustomizeConfig{},
		&webhook.Service{},
	); err != nil {
		return err
	}

	if doConversion {
		fmt.Println(`Webhook server has been set up for you.
You need to implement the conversion.Hub and conversion.Convertible interfaces for your CRD types.`)
	}

	// TODO: Add test suite for conversion webhook after #1664 has been merged & conversion tests supported in envtest.
	if doDefaulting || doValidation {
		if err := machinery.NewScaffold(s.fs).Execute(
			s.newUniverse(),
			&api.WebhookSuite{},
		); err != nil {
			return err
		}
	}

	return nil
}
