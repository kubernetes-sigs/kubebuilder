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

	"sigs.k8s.io/kubebuilder/pkg/model"
	"sigs.k8s.io/kubebuilder/pkg/model/config"
	"sigs.k8s.io/kubebuilder/pkg/model/resource"
	"sigs.k8s.io/kubebuilder/pkg/plugin/internal/machinery"
	"sigs.k8s.io/kubebuilder/pkg/plugin/scaffold"
	"sigs.k8s.io/kubebuilder/pkg/plugin/v3/scaffolds/internal/templates"
	"sigs.k8s.io/kubebuilder/pkg/plugin/v3/scaffolds/internal/templates/config/api"
	"sigs.k8s.io/kubebuilder/pkg/plugin/v3/scaffolds/internal/templates/config/kdefault"
)

var _ scaffold.Scaffolder = &webhookScaffolder{}

type webhookScaffolder struct {
	config      *config.Config
	boilerplate string
	resource    *resource.Resource

	// Webhook type options.
	defaulting, validation, conversion bool
	webhookSpokeVersion                string
	webhookHubVersion                  string
}

// NewWebhookScaffolder returns a new Scaffolder for v2 webhook creation operations
func NewWebhookScaffolder(
	config *config.Config,
	boilerplate string,
	resource *resource.Resource,
	defaulting bool,
	validation bool,
	conversion bool,
	webhookSpokeVersion string,
	webhookHubVersion string,
) scaffold.Scaffolder {
	return &webhookScaffolder{
		config:              config,
		boilerplate:         boilerplate,
		resource:            resource,
		defaulting:          defaulting,
		validation:          validation,
		conversion:          conversion,
		webhookSpokeVersion: webhookSpokeVersion,
		webhookHubVersion:   webhookHubVersion,
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
	if s.webhookSpokeVersion != "" && s.webhookHubVersion != "" {
		if err := machinery.NewScaffold().Execute(
			s.newUniverse(),
			&api.Webhook{Defaulting: s.defaulting, Validating: s.validation},
			&api.HubConversion{HubVersion: s.webhookHubVersion},
			&api.SpokeConversion{SpokeVersion: s.webhookSpokeVersion},
			&templates.MainUpdater{WireWebhook: true},
			&kdefault.InjectCAPatch{},
		); err != nil {
			return err
		}
	} else {
		if s.conversion {
			fmt.Println(`Webhook server has been set up for you.		
 You need to implement the conversion.Hub and conversion.Convertible interfaces for your CRD types.`)
		}
		if err := machinery.NewScaffold().Execute(
			s.newUniverse(),
			&api.Webhook{Defaulting: s.defaulting, Validating: s.validation, Conversion: s.conversion},
			&templates.MainUpdater{WireWebhook: true},
			&kdefault.InjectCAPatch{},
		); err != nil {
			return err
		}
	}

	// TODO: Add test suite for conversion webhook after #1664 has been merged & conversion tests supported in envtest.
	if s.defaulting || s.validation {
		if err := machinery.NewScaffold().Execute(
			s.newUniverse(),
			&api.WebhookSuite{},
		); err != nil {
			return err
		}
	}

	return nil
}
