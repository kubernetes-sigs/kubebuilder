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
	"fmt"

	"sigs.k8s.io/kubebuilder/pkg/model"
	"sigs.k8s.io/kubebuilder/pkg/model/config"
	"sigs.k8s.io/kubebuilder/pkg/model/resource"
	"sigs.k8s.io/kubebuilder/pkg/scaffold/internal/machinery"
	"sigs.k8s.io/kubebuilder/pkg/scaffold/internal/templates"
	"sigs.k8s.io/kubebuilder/pkg/scaffold/internal/templates/webhook"
)

var _ Scaffolder = &webhookScaffolder{}

type webhookScaffolder struct {
	config      *config.Config
	boilerplate string
	resource    *resource.Resource

	// v2
	defaulting, validation, conversion bool
}

// NewWebhookScaffolder returns a new Scaffolder for v2 webhook creation operations
func NewWebhookScaffolder(
	config *config.Config,
	boilerplate string,
	resource *resource.Resource,
	defaulting bool,
	validation bool,
	conversion bool,
) Scaffolder {
	return &webhookScaffolder{
		config:      config,
		boilerplate: boilerplate,
		resource:    resource,
		defaulting:  defaulting,
		validation:  validation,
		conversion:  conversion,
	}
}

// Scaffold implements Scaffolder
func (s *webhookScaffolder) Scaffold() error {
	fmt.Println("Writing scaffold for you to edit...")

	switch {
	case s.config.IsV2(), s.config.IsV3():
		return s.scaffold()
	default:
		return fmt.Errorf("unknown project version %v", s.config.Version)
	}
}

func (s *webhookScaffolder) newUniverse() *model.Universe {
	return model.NewUniverse(
		model.WithConfig(s.config),
		model.WithBoilerplate(s.boilerplate),
		model.WithResource(s.resource),
	)
}

func (s *webhookScaffolder) scaffold() error {
	if s.conversion {
		fmt.Println(`Webhook server has been set up for you.
You need to implement the conversion.Hub and conversion.Convertible interfaces for your CRD types.`)
	}

	if err := machinery.NewScaffold().Execute(
		s.newUniverse(),
		&webhook.Webhook{Defaulting: s.defaulting, Validating: s.validation},
		&templates.MainUpdater{WireWebhook: true},
	); err != nil {
		return err
	}

	return nil
}
