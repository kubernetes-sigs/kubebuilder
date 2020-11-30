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

	"sigs.k8s.io/kubebuilder/v2/pkg/model"
	"sigs.k8s.io/kubebuilder/v2/pkg/model/config"
	"sigs.k8s.io/kubebuilder/v2/pkg/model/resource"
	"sigs.k8s.io/kubebuilder/v2/pkg/scaffold/internal/machinery"
	managerv1 "sigs.k8s.io/kubebuilder/v2/pkg/scaffold/internal/templates/v1/manager"
	webhookv1 "sigs.k8s.io/kubebuilder/v2/pkg/scaffold/internal/templates/v1/webhook"
	templatesv2 "sigs.k8s.io/kubebuilder/v2/pkg/scaffold/internal/templates/v2"
	webhookv2 "sigs.k8s.io/kubebuilder/v2/pkg/scaffold/internal/templates/v2/webhook"
)

var _ Scaffolder = &webhookScaffolder{}

type webhookScaffolder struct {
	config      *config.Config
	boilerplate string
	resource    *resource.Resource
	// v1
	server      string
	webhookType string
	operations  []string
	// v2
	defaulting, validation, conversion bool
}

// NewV1WebhookScaffolder returns a new Scaffolder for v1 webhook creation operations
func NewV1WebhookScaffolder(
	config *config.Config,
	boilerplate string,
	resource *resource.Resource,
	server string,
	webhookType string,
	operations []string,
) Scaffolder {
	return &webhookScaffolder{
		config:      config,
		boilerplate: boilerplate,
		resource:    resource,
		server:      server,
		webhookType: webhookType,
		operations:  operations,
	}
}

// NewV2WebhookScaffolder returns a new Scaffolder for v2 webhook creation operations
func NewV2WebhookScaffolder(
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
	case s.config.IsV1():
		return s.scaffoldV1()
	case s.config.IsV2():
		return s.scaffoldV2()
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

func (s *webhookScaffolder) scaffoldV1() error {
	webhookConfig := webhookv1.Config{Server: s.server, Type: s.webhookType, Operations: s.operations}

	return machinery.NewScaffold().Execute(
		s.newUniverse(),
		&managerv1.Webhook{},
		&webhookv1.AdmissionHandler{Config: webhookConfig},
		&webhookv1.AdmissionWebhookBuilder{Config: webhookConfig},
		&webhookv1.AdmissionWebhooks{Config: webhookConfig},
		&webhookv1.AddAdmissionWebhookBuilderHandler{Config: webhookConfig},
		&webhookv1.Server{Config: webhookConfig},
		&webhookv1.AddServer{Config: webhookConfig},
	)
}

func (s *webhookScaffolder) scaffoldV2() error {
	if s.conversion {
		fmt.Println(`Webhook server has been set up for you.
You need to implement the conversion.Hub and conversion.Convertible interfaces for your CRD types.`)
	}

	if err := machinery.NewScaffold().Execute(
		s.newUniverse(),
		&webhookv2.Webhook{Defaulting: s.defaulting, Validating: s.validation},
		&templatesv2.MainUpdater{WireWebhook: true},
	); err != nil {
		return err
	}

	return nil
}
