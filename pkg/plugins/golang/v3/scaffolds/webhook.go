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

	"sigs.k8s.io/kubebuilder/v2/pkg/model"
	"sigs.k8s.io/kubebuilder/v2/pkg/model/config"
	"sigs.k8s.io/kubebuilder/v2/pkg/model/resource"
	"sigs.k8s.io/kubebuilder/v2/pkg/plugins/golang/v3/scaffolds/internal/templates"
	"sigs.k8s.io/kubebuilder/v2/pkg/plugins/golang/v3/scaffolds/internal/templates/api"
	"sigs.k8s.io/kubebuilder/v2/pkg/plugins/golang/v3/scaffolds/internal/templates/config/kdefault"
	"sigs.k8s.io/kubebuilder/v2/pkg/plugins/golang/v3/scaffolds/internal/templates/config/webhook"
	"sigs.k8s.io/kubebuilder/v2/pkg/plugins/internal/cmdutil"
	"sigs.k8s.io/kubebuilder/v2/pkg/plugins/internal/machinery"
)

var _ cmdutil.Scaffolder = &webhookScaffolder{}

type webhookScaffolder struct {
	config      *config.Config
	boilerplate string
	resource    *resource.Resource
}

// NewWebhookScaffolder returns a new Scaffolder for v2 webhook creation operations
func NewWebhookScaffolder(
	config *config.Config,
	boilerplate string,
	resource *resource.Resource,
) cmdutil.Scaffolder {
	return &webhookScaffolder{
		config:      config,
		boilerplate: boilerplate,
		resource:    resource,
	}
}

// Scaffold implements Scaffolder
func (s *webhookScaffolder) Scaffold() error {
	fmt.Println("Writing scaffold for you to edit...")
	return s.scaffold()
}

func (s *webhookScaffolder) newUniverse() *model.Universe {
	return model.NewUniverse(
		model.WithConfig(s.config),
		model.WithBoilerplate(s.boilerplate),
		model.WithResource(s.resource),
	)
}

func (s *webhookScaffolder) scaffold() error {
	// Check if we need to do the defaulting or validation before updating with previously existing info
	doDefaulting := s.resource.Webhooks.Defaulting
	doValidation := s.resource.Webhooks.Validation
	doConversion := s.resource.Webhooks.Conversion

	// Update the known data about resource
	var err error
	s.resource, err = s.config.UpdateResources(s.resource)
	if err != nil {
		return fmt.Errorf("error updating resources in config: %w", err)
	}

	if err := machinery.NewScaffold().Execute(
		s.newUniverse(),
		&api.Webhook{},
		&templates.MainUpdater{WireWebhook: true},
		&kdefault.WebhookCAInjectionPatch{},
		&kdefault.ManagerWebhookPatch{},
		&webhook.Kustomization{},
		&webhook.KustomizeConfig{},
		&webhook.Service{},
	); err != nil {
		return err
	}

	// TODO: Add test suite for conversion webhook after #1664 has been merged & conversion tests supported in envtest.
	if doDefaulting || doValidation {
		if err := machinery.NewScaffold().Execute(
			s.newUniverse(),
			&api.WebhookSuite{},
		); err != nil {
			return err
		}
	}

	if doConversion {
		fmt.Println(`Webhook server has been set up for you.
You need to implement the conversion.Hub and conversion.Convertible interfaces for your CRD types.`)
	}

	return nil
}
