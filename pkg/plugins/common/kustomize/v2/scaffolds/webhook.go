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
	pluginutil "sigs.k8s.io/kubebuilder/v3/pkg/plugin/util"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugins/common/kustomize/v2/scaffolds/internal/templates/config/crd"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugins/common/kustomize/v2/scaffolds/internal/templates/config/crd/patches"

	"sigs.k8s.io/kubebuilder/v3/pkg/config"
	"sigs.k8s.io/kubebuilder/v3/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v3/pkg/model/resource"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugins"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugins/common/kustomize/v2/scaffolds/internal/templates/config/certmanager"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugins/common/kustomize/v2/scaffolds/internal/templates/config/kdefault"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugins/common/kustomize/v2/scaffolds/internal/templates/config/webhook"
)

var _ plugins.Scaffolder = &webhookScaffolder{}

type webhookScaffolder struct {
	config   config.Config
	resource resource.Resource

	// fs is the filesystem that will be used by the scaffolder
	fs machinery.Filesystem

	// force indicates whether to scaffold files even if they exist.
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
func (s *webhookScaffolder) InjectFS(fs machinery.Filesystem) { s.fs = fs }

// Scaffold implements cmdutil.Scaffolder
func (s *webhookScaffolder) Scaffold() error {
	log.Println("Writing kustomize manifests for you to edit...")

	// Initialize the machinery.Scaffold that will write the files to disk
	scaffold := machinery.NewScaffold(s.fs,
		machinery.WithConfig(s.config),
		machinery.WithResource(&s.resource),
	)

	if err := s.config.UpdateResource(s.resource); err != nil {
		return fmt.Errorf("error updating resource: %w", err)
	}

	if err := scaffold.Execute(
		&kdefault.WebhookCAInjectionPatch{},
		&kdefault.ManagerWebhookPatch{},
		&webhook.Kustomization{Force: s.force},
		&webhook.KustomizeConfig{},
		&webhook.Service{},
		&certmanager.Certificate{},
		&certmanager.Kustomization{},
		&certmanager.KustomizeConfig{},
		&patches.EnableWebhookPatch{},
		&patches.EnableCAInjectionPatch{},
		&crd.Kustomization{},
	); err != nil {
		return fmt.Errorf("error scaffolding kustomize webhook manifests: %v", err)
	}

	kustomizeFilePath := "config/default/kustomization.yaml"
	err := pluginutil.UncommentCode(kustomizeFilePath, "#- ../webhook", `#`)
	if err != nil {
		hasWebHookUncommented, err := pluginutil.HasFragment(kustomizeFilePath, "- ../webhook")
		if !hasWebHookUncommented || err != nil {
			log.Errorf("Unable to find the target #- ../webhook to uncomment in the file "+
				"%s.", kustomizeFilePath)
		}
	}

	err = pluginutil.UncommentCode(kustomizeFilePath, "#- path: manager_webhook_patch.yaml", `#`)
	if err != nil {
		hasWebHookUncommented, err := pluginutil.HasFragment(kustomizeFilePath, "- path: manager_webhook_patch.yaml")
		if !hasWebHookUncommented || err != nil {
			log.Errorf("Unable to find the target #- path: manager_webhook_patch.yaml to uncomment in the file "+
				"%s.", kustomizeFilePath)
		}
	}

	crdKustomizationsFilePath := "config/crd/kustomization.yaml"
	err = pluginutil.UncommentCode(crdKustomizationsFilePath, "#- path: patches/webhook", `#`)
	if err != nil {
		hasWebHookUncommented, err := pluginutil.HasFragment(crdKustomizationsFilePath, "- path: patches/webhook")
		if !hasWebHookUncommented || err != nil {
			log.Errorf("Unable to find the target(s) #- path: patches/webhook/* to uncomment in the file "+
				"%s.", crdKustomizationsFilePath)
		}
	}

	err = pluginutil.UncommentCode(crdKustomizationsFilePath, "#configurations:\n#- kustomizeconfig.yaml", `#`)
	if err != nil {
		hasWebHookUncommented, err := pluginutil.HasFragment(crdKustomizationsFilePath, "- kustomizeconfig.yaml")
		if !hasWebHookUncommented || err != nil {
			log.Errorf("Unable to find the target(s) #configurations:\n#- kustomizeconfig.yaml to uncomment in the file "+
				"%s.", crdKustomizationsFilePath)
		}
	}

	return nil
}
