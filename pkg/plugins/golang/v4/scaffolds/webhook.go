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
	"errors"
	"fmt"
	log "log/slog"
	"strings"

	"github.com/spf13/afero"

	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v4/pkg/model/resource"
	pluginutil "sigs.k8s.io/kubebuilder/v4/pkg/plugin/util"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/golang/v4/scaffolds/internal/templates/api"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/golang/v4/scaffolds/internal/templates/cmd"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/golang/v4/scaffolds/internal/templates/hack"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/golang/v4/scaffolds/internal/templates/test/e2e"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/golang/v4/scaffolds/internal/templates/webhooks"
)

var _ plugins.Scaffolder = &webhookScaffolder{}

type webhookScaffolder struct {
	config   config.Config
	resource resource.Resource

	// fs is the filesystem that will be used by the scaffolder
	fs machinery.Filesystem

	// force indicates whether to scaffold controller files even if it exists or not
	force bool

	// Deprecated - TODO: remove it for go/v5
	// isLegacy indicates that the resource should be created in the legacy path under the api
	isLegacy bool
}

// NewWebhookScaffolder returns a new Scaffolder for v2 webhook creation operations
func NewWebhookScaffolder(cfg config.Config, res resource.Resource, force bool, isLegacy bool) plugins.Scaffolder {
	return &webhookScaffolder{
		config:   cfg,
		resource: res,
		force:    force,
		isLegacy: isLegacy,
	}
}

// InjectFS implements cmdutil.Scaffolder
func (s *webhookScaffolder) InjectFS(fs machinery.Filesystem) {
	s.fs = fs
}

// Scaffold implements cmdutil.Scaffolder
func (s *webhookScaffolder) Scaffold() error {
	log.Info("Writing scaffold for you to edit...")

	// Load the boilerplate
	boilerplate, err := afero.ReadFile(s.fs.FS, hack.DefaultBoilerplatePath)
	if err != nil {
		if errors.Is(err, afero.ErrFileNotFound) {
			log.Warn("unable to find boilerplate file. "+
				"This file is used to generate the license header in the project..\n"+
				"Note that controller-gen will also use this. Ensure that you "+
				"add the license file or configure your project accordingly",
				"file_path", hack.DefaultBoilerplatePath, "error", err)
			boilerplate = []byte("")
		} else {
			return fmt.Errorf("error scaffolding webhook: failed to load boilerplate: %w", err)
		}
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

	if err = s.config.UpdateResource(s.resource); err != nil {
		return fmt.Errorf("error updating resource: %w", err)
	}

	// Check if webhook files exist
	webhookFilePath := s.getWebhookFilePath()
	webhookFileExists := false
	if _, statErr := s.fs.FS.Stat(webhookFilePath); statErr == nil {
		webhookFileExists = true
	}

	webhookTestFilePath := s.getWebhookTestFilePath()
	webhookTestFileExists := false
	if _, statErr := s.fs.FS.Stat(webhookTestFilePath); statErr == nil {
		webhookTestFileExists = true
	}

	// Scaffold or update webhook file (for all webhook types)
	// Note: Conversion webhooks also need a webhook.go file with minimal setup (.For(&Type{}).Complete())
	// This is how controller-runtime discovers Hub/Convertible interfaces
	if doDefaulting || doValidation || doConversion {
		if err = s.scaffoldWebhookFile(scaffold, webhookFileExists); err != nil {
			return err
		}

		// Update main.go to wire webhook setup function (for all webhook types)
		if err = scaffold.Execute(
			&cmd.MainUpdater{WireWebhook: true, IsLegacyPath: s.isLegacy},
		); err != nil {
			return fmt.Errorf("error updating main.go: %w", err)
		}
	}

	// Scaffold or update webhook test file (for all webhook types)
	if err = s.scaffoldWebhookTestFile(scaffold, webhookTestFileExists); err != nil {
		return err
	}

	// Update e2e tests
	// WireWebhook controls webhook service readiness checks (for defaulting/validation)
	// But conversion webhooks still need CA injection tests (handled inside updater)
	if err = scaffold.Execute(
		&e2e.WebhookTestUpdater{WireWebhook: doDefaulting || doValidation},
	); err != nil {
		return fmt.Errorf("error updating e2e tests: %w", err)
	}

	if doConversion {
		// Update the types file to add storage version marker
		if err = scaffold.Execute(&api.TypesUpdater{}); err != nil {
			return fmt.Errorf("error updating types file with storage version marker: %w", err)
		}

		if err = scaffold.Execute(&api.Hub{Force: s.force}); err != nil {
			return fmt.Errorf("error scaffold resource with hub: %w", err)
		}

		for _, spoke := range s.resource.Webhooks.Spoke {
			log.Info("Scaffolding for spoke version", "version", spoke)
			if err = scaffold.Execute(&api.Spoke{Force: s.force, SpokeVersion: spoke}); err != nil {
				return fmt.Errorf("failed to scaffold spoke %s: %w", spoke, err)
			}
		}

		log.Info(`Webhook server has been set up for you.
You need to implement the conversion.Hub and conversion.Convertible interfaces for your CRD types.`)
	}

	// Scaffold webhook suite test for all webhook types
	// Note: Conversion webhooks also need the suite to register with envtest
	if doDefaulting || doValidation || doConversion {
		if err = scaffold.Execute(&webhooks.WebhookSuite{IsLegacyPath: s.isLegacy}); err != nil {
			return fmt.Errorf("error scaffold webhook suite: %w", err)
		}
	}

	// TODO: remove for go/v5
	if !s.isLegacy {
		if hasInternalController, err := pluginutil.HasFileContentWith("Dockerfile", "internal/controller"); err != nil {
			log.Error("failed to read Dockerfile to check if webhook(s) will be properly copied", "error", err)
		} else if hasInternalController {
			log.Warn("Dockerfile is copying internal/controller; to allow copying webhooks, " +
				"it will be edited, and `internal/controller` will be replaced by `internal/`")

			if err = pluginutil.ReplaceInFile("Dockerfile", "internal/controller", "internal/"); err != nil {
				log.Error("failed to replace \"internal/controller\" with \"internal/\" in the Dockerfile", "error", err)
			}
		}
	}
	return nil
}

// getWebhookFilePath returns the path to the webhook file
func (s *webhookScaffolder) getWebhookFilePath() string {
	baseDir := "api"
	if !s.isLegacy {
		baseDir = "internal/webhook"
	}

	var path string
	if s.config.IsMultiGroup() && s.resource.Group != "" {
		path = fmt.Sprintf("%s/%s/%s/%s_webhook.go",
			baseDir, s.resource.Group, s.resource.Version, strings.ToLower(s.resource.Kind))
	} else {
		path = fmt.Sprintf("%s/%s/%s_webhook.go",
			baseDir, s.resource.Version, strings.ToLower(s.resource.Kind))
	}

	return path
}

// getWebhookTestFilePath returns the path to the webhook test file
func (s *webhookScaffolder) getWebhookTestFilePath() string {
	baseDir := "api"
	if !s.isLegacy {
		baseDir = "internal/webhook"
	}

	var path string
	if s.config.IsMultiGroup() && s.resource.Group != "" {
		path = fmt.Sprintf("%s/%s/%s/%s_webhook_test.go",
			baseDir, s.resource.Group, s.resource.Version, strings.ToLower(s.resource.Kind))
	} else {
		path = fmt.Sprintf("%s/%s/%s_webhook_test.go",
			baseDir, s.resource.Version, strings.ToLower(s.resource.Kind))
	}

	return path
}

// scaffoldWebhookFile creates or updates the webhook implementation file
func (s *webhookScaffolder) scaffoldWebhookFile(scaffold *machinery.Scaffold, fileExists bool) error {
	if !fileExists || s.force {
		if err := scaffold.Execute(
			&webhooks.Webhook{Force: s.force, IsLegacyPath: s.isLegacy},
		); err != nil {
			return fmt.Errorf("error creating webhook: %w", err)
		}
	} else if fileExists && !s.force && !s.isLegacy {
		log.Info("Adding new webhook type to existing file")
		if err := scaffold.Execute(
			&webhooks.WebhookUpdater{},
		); err != nil {
			return fmt.Errorf("error updating webhook: %w", err)
		}
	}
	return nil
}

// scaffoldWebhookTestFile creates or updates the webhook test file
func (s *webhookScaffolder) scaffoldWebhookTestFile(scaffold *machinery.Scaffold, fileExists bool) error {
	if !fileExists || s.force {
		if err := scaffold.Execute(
			&webhooks.WebhookTest{Force: s.force, IsLegacyPath: s.isLegacy},
		); err != nil {
			return fmt.Errorf("error creating webhook test: %w", err)
		}
	} else if fileExists && !s.force && !s.isLegacy {
		if err := scaffold.Execute(
			&webhooks.WebhookTestUpdater{},
		); err != nil {
			return fmt.Errorf("error updating webhook test: %w", err)
		}
	}
	return nil
}
