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
	"strings"

	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/golang/v4/scaffolds/internal/templates/api"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/afero"

	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v4/pkg/model/resource"
	pluginutil "sigs.k8s.io/kubebuilder/v4/pkg/plugin/util"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins"
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
func NewWebhookScaffolder(config config.Config, resource resource.Resource,
	force bool, isLegacy bool,
) plugins.Scaffolder {
	return &webhookScaffolder{
		config:   config,
		resource: resource,
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
	log.Println("Writing scaffold for you to edit...")

	// Load the boilerplate
	boilerplate, err := afero.ReadFile(s.fs.FS, hack.DefaultBoilerplatePath)
	if err != nil {
		if errors.Is(err, afero.ErrFileNotFound) {
			log.Warnf("Unable to find %s : %s .\n"+
				"This file is used to generate the license header in the project.\n"+
				"Note that controller-gen will also use this. Therefore, ensure that you "+
				"add the license file or configure your project accordingly.",
				hack.DefaultBoilerplatePath, err)
			boilerplate = []byte("")
		} else {
			return fmt.Errorf("error scaffolding webhook: unable to load boilerplate: %w", err)
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

	if err := s.config.UpdateResource(s.resource); err != nil {
		return fmt.Errorf("error updating resource: %w", err)
	}

	if err := scaffold.Execute(
		&webhooks.Webhook{Force: s.force, IsLegacyPath: s.isLegacy},
		&e2e.WebhookTestUpdater{WireWebhook: true},
		&cmd.MainUpdater{WireWebhook: true, IsLegacyPath: s.isLegacy},
		&webhooks.WebhookTest{Force: s.force, IsLegacyPath: s.isLegacy},
	); err != nil {
		return err
	}

	if doConversion {
		resourceFilePath := fmt.Sprintf("api/%s/%s_types.go",
			s.resource.Version, strings.ToLower(s.resource.Kind))
		if s.config.IsMultiGroup() {
			resourceFilePath = fmt.Sprintf("api/%s/%s/%s_types.go",
				s.resource.Group, s.resource.Version,
				strings.ToLower(s.resource.Kind))
		}

		err = pluginutil.InsertCodeIfNotExist(resourceFilePath,
			"// +kubebuilder:object:root=true",
			"\n// +kubebuilder:storageversion\n// +kubebuilder:conversion:hub")
		if err != nil {
			log.Errorf("Unable to insert storage version marker "+
				"(// +kubebuilder:storageversion) and the hub conversion (// +kubebuilder:conversion:hub) "+
				"in file %s: %v", resourceFilePath, err)
		}

		if err := scaffold.Execute(
			&api.Hub{Force: s.force},
		); err != nil {
			return err
		}

		for _, spoke := range s.resource.Webhooks.Spoke {
			log.Printf("Scaffolding for spoke version: %s\n", spoke)
			if err := scaffold.Execute(
				&api.Spoke{Force: s.force, SpokeVersion: spoke},
			); err != nil {
				return fmt.Errorf("failed to scaffold spoke %s: %w", spoke, err)
			}
		}

		log.Println(`Webhook server has been set up for you.
You need to implement the conversion.Hub and conversion.Convertible interfaces for your CRD types.`)
	}

	// TODO: Add test suite for conversion webhook after #1664 has been merged & conversion tests supported in envtest.
	if doDefaulting || doValidation {
		if err := scaffold.Execute(
			&webhooks.WebhookSuite{IsLegacyPath: s.isLegacy},
		); err != nil {
			return err
		}
	}

	// TODO: remove for go/v5
	if !s.isLegacy {
		if hasInternalController, err := pluginutil.HasFileContentWith("Dockerfile", "internal/controller"); err != nil {
			log.Error("Unable to read Dockerfile to check if webhook(s) will be properly copied: ", err)
		} else if hasInternalController {
			log.Warning("Dockerfile is copying internal/controller. To allow copying webhooks, " +
				"it will be edited, and `internal/controller` will be replaced by `internal/`.")

			if err := pluginutil.ReplaceInFile("Dockerfile", "internal/controller", "internal/"); err != nil {
				log.Error("Unable to replace \"internal/controller\" with \"internal/\" in the Dockerfile: ", err)
			}
		}
	}
	return nil
}
