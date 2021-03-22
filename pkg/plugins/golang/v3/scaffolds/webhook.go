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
	"sigs.k8s.io/kubebuilder/v3/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v3/pkg/model/resource"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugins"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugins/golang/v3/scaffolds/internal/templates"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugins/golang/v3/scaffolds/internal/templates/api"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugins/golang/v3/scaffolds/internal/templates/config/kdefault"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugins/golang/v3/scaffolds/internal/templates/config/webhook"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugins/golang/v3/scaffolds/internal/templates/hack"
)

var _ plugins.Scaffolder = &webhookScaffolder{}

type webhookScaffolder struct {
	config   config.Config
	resource resource.Resource

	// fs is the filesystem that will be used by the scaffolder
	fs machinery.Filesystem

	// force indicates whether to scaffold controller files even if it exists or not
	force bool
	// doScaffold indicates wheather the templates for the version need to be scaffolded
	doScaffold bool
	// spoke refers to the spoke version to be scaffolded
	spoke string
}

// NewWebhookScaffolder returns a new Scaffolder for v2 webhook creation operations
func NewWebhookScaffolder(config config.Config, resource resource.Resource, force bool,
	doScaffold bool, spoke string) plugins.Scaffolder {
	return &webhookScaffolder{
		config:     config,
		resource:   resource,
		force:      force,
		doScaffold: doScaffold,
		spoke:      spoke,
	}
}

// InjectFS implements cmdutil.Scaffolder
func (s *webhookScaffolder) InjectFS(fs machinery.Filesystem) {
	s.fs = fs
}

// Scaffold implements cmdutil.Scaffolder
func (s *webhookScaffolder) Scaffold() error {
	fmt.Println("Writing scaffold for you to edit...")

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
	hasSpoke := s.spoke != ""

	// add the spoke version to the resource, and update the
	// resource in the config.
	if s.resource.Webhooks == nil {
		s.resource.Webhooks = &resource.Webhooks{}
	}
	s.resource.Webhooks.Spokes = append(s.resource.Webhooks.Spokes, s.spoke)
	if err := s.config.UpdateResource(s.resource); err != nil {
		return fmt.Errorf("error updating resource: %w", err)
	}

	// if doScaffold is set to false, then scaffold only the spoke and return, since
	// rest of the templates are already scaffolded.
	if hasSpoke {
		if !s.doScaffold {
			if err := scaffold.Execute(
				&api.Conversion{Spoke: true, Version: s.spoke},
			); err != nil {
				return err
			}
			return nil
		}
	}

	if err := scaffold.Execute(
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

	// if spoke is specified then scaffold conversion webhook templates for the user.
	if hasSpoke && s.doScaffold {
		if err := scaffold.Execute(
			&api.Conversion{Hub: true, Version: s.resource.Version},
			&api.Conversion{Spoke: true, Version: s.spoke},
		); err != nil {
			return err
		}
	}

	if doConversion && !hasSpoke {
		fmt.Println(`Webhook server has been set up for you.
You need to implement the conversion.Hub and conversion.Convertible interfaces for your CRD types. 
You can also specify hub and spoke versions in the command to scaffold Convertible and Hub interfaces `)
	}

	// TODO: Add test suite for conversion webhook after #1664 has been merged & conversion tests supported in envtest.
	if doDefaulting || doValidation {
		if err := scaffold.Execute(
			&api.WebhookSuite{},
		); err != nil {
			return err
		}
	}

	return nil
}
