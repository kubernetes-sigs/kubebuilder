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
	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v4/pkg/model/resource"
	pluginutil "sigs.k8s.io/kubebuilder/v4/pkg/plugin/util"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/common/kustomize/v2/scaffolds/internal/templates/config/certmanager"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/common/kustomize/v2/scaffolds/internal/templates/config/crd"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/common/kustomize/v2/scaffolds/internal/templates/config/crd/patches"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/common/kustomize/v2/scaffolds/internal/templates/config/kdefault"
	network_policy "sigs.k8s.io/kubebuilder/v4/pkg/plugins/common/kustomize/v2/scaffolds/internal/templates/config/network-policy"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/common/kustomize/v2/scaffolds/internal/templates/config/webhook"
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
func NewWebhookScaffolder(cfg config.Config, res resource.Resource, force bool) plugins.Scaffolder {
	return &webhookScaffolder{
		config:   cfg,
		resource: res,
		force:    force,
	}
}

// InjectFS implements cmdutil.Scaffolder
func (s *webhookScaffolder) InjectFS(fs machinery.Filesystem) { s.fs = fs }

// Scaffold implements cmdutil.Scaffolder
func (s *webhookScaffolder) Scaffold() error {
	log.Println("Writing kustomize manifests for you to edit...")

	// Will validate the scaffold
	// Users that scaffolded the project previously
	// with the bugs will receive a message to help
	// them out fix their scaffold.
	validateScaffoldedProject()

	// Initialize the machinery.Scaffold that will write the files to disk
	scaffold := machinery.NewScaffold(s.fs,
		machinery.WithConfig(s.config),
		machinery.WithResource(&s.resource),
	)

	if err := s.config.UpdateResource(s.resource); err != nil {
		return fmt.Errorf("error updating resource: %w", err)
	}

	buildScaffold := []machinery.Builder{
		&kdefault.ManagerWebhookPatch{},
		&webhook.Kustomization{Force: s.force},
		&webhook.KustomizeConfig{},
		&webhook.Service{},
		&certmanager.Certificate{},
		&certmanager.Issuer{},
		&certmanager.MetricsCertificate{},
		&certmanager.Kustomization{},
		&certmanager.KustomizeConfig{},
		&network_policy.PolicyAllowWebhooks{},
	}

	// Only scaffold the following patches if is a conversion webhook
	if s.resource.Webhooks.Conversion {
		buildScaffold = append(buildScaffold, &patches.EnableWebhookPatch{})
		buildScaffold = append(buildScaffold, &kdefault.KustomizationCAConversionUpdater{})
	}

	if !s.resource.External && !s.resource.Core {
		buildScaffold = append(buildScaffold, &crd.Kustomization{})
	}

	if err := scaffold.Execute(buildScaffold...); err != nil {
		return fmt.Errorf("error scaffolding kustomize webhook manifests: %v", err)
	}

	policyKustomizeFilePath := "config/network-policy/kustomization.yaml"
	err := pluginutil.InsertCodeIfNotExist(policyKustomizeFilePath,
		"resources:", allowWebhookTrafficFragment)
	if err != nil {
		log.Errorf("Unable to add the line '- allow-webhook-traffic.yaml' at the end of the file"+
			"%s to allow webhook traffic.", policyKustomizeFilePath)
	}

	kustomizeFilePath := "config/default/kustomization.yaml"
	err = pluginutil.UncommentCode(kustomizeFilePath, "#- ../webhook", `#`)
	if err != nil {
		hasWebHookUncommented, errCheck := pluginutil.HasFileContentWith(kustomizeFilePath, "- ../webhook")
		if !hasWebHookUncommented || errCheck != nil {
			log.Errorf("Unable to find the target #- ../webhook to uncomment in the file "+
				"%s.", kustomizeFilePath)
		}
	}

	err = pluginutil.UncommentCode(kustomizeFilePath, "#patches:", `#`)
	if err != nil {
		hasWebHookUncommented, errCheck := pluginutil.HasFileContentWith(kustomizeFilePath, "patches:")
		if !hasWebHookUncommented || errCheck != nil {
			log.Errorf("Unable to find the line '#patches:' to uncomment in the file "+
				"%s.", kustomizeFilePath)
		}
	}

	err = pluginutil.UncommentCode(kustomizeFilePath, `#- path: manager_webhook_patch.yaml
#  target:
#    kind: Deployment`, `#`)
	if err != nil {
		hasWebHookUncommented, errCheck := pluginutil.HasFileContentWith(kustomizeFilePath,
			"- path: manager_webhook_patch.yaml")
		if !hasWebHookUncommented || errCheck != nil {
			log.Errorf("Unable to find the target #- path: manager_webhook_patch.yaml to uncomment in the file "+
				"%s.", kustomizeFilePath)
		}
	}

	if s.resource.Webhooks.Conversion {
		crdKustomizationsFilePath := "config/crd/kustomization.yaml"
		err = pluginutil.UncommentCode(crdKustomizationsFilePath, "#configurations:\n#- kustomizeconfig.yaml", `#`)
		if err != nil {
			hasWebHookUncommented, err := pluginutil.HasFileContentWith(crdKustomizationsFilePath,
				"configurations:\n- kustomizeconfig.yaml")
			if !hasWebHookUncommented || err != nil {
				log.Warningf("Unable to find the target(s) configurations.kustomizeconfig.yaml "+
					"to uncomment in the file "+
					"%s.", crdKustomizationsFilePath)
			}
		}
	}

	return nil
}

// Deprecated: remove it when go/v4 and/or kustomize/v2 be removed
// validateScaffoldedProject will output a message to help users fix their scaffold
func validateScaffoldedProject() {
	kustomizeFilePath := "config/default/kustomization.yaml"
	hasCertManagerPatch, _ := pluginutil.HasFileContentWith(kustomizeFilePath,
		"crdkustomizecainjectionpatch")

	if hasCertManagerPatch {
		log.Warning(`

1. **Remove the CERTMANAGER Section from config/crd/kustomization.yaml:**

   Delete the CERTMANAGER section to prevent unintended CA injection patches for CRDs.
   Ensure the following lines are removed or commented out:

   # [CERTMANAGER] To enable cert-manager, uncomment all the sections with [CERTMANAGER] prefix.
   # patches here are for enabling the CA injection for each CRD
   #- path: patches/cainjection_in_firstmates.yaml
   # +kubebuilder:scaffold:crdkustomizecainjectionpatch

2. **Ensure CA Injection Configuration in config/default/kustomization.yaml:**

   Under the [CERTMANAGER] replacement in config/default/kustomization.yaml,
   add the following code for proper CA injection generation:

   **NOTE:** You must ensure that the code contains the following target markers:
   - +kubebuilder:scaffold:crdkustomizecainjectionns
   - +kubebuilder:scaffold:crdkustomizecainjectioname

   # - source: # Uncomment the following block if you have a ConversionWebhook (--conversion)
   #     kind: Certificate
   #     group: cert-manager.io
   #     version: v1
   #     name: serving-cert # This name should match the one in certificate.yaml
   #     fieldPath: .metadata.namespace # Namespace of the certificate CR
   #   targets: # Do not remove or uncomment the following scaffold marker; required to generate code for target CRD.
   # +kubebuilder:scaffold:crdkustomizecainjectionns
   # - source:
   #     kind: Certificate
   #     group: cert-manager.io
   #     version: v1
   #     name: serving-cert # This name should match the one in certificate.yaml
   #     fieldPath: .metadata.name
   #   targets: # Do not remove or uncomment the following scaffold marker; required to generate code for target CRD.
   # +kubebuilder:scaffold:crdkustomizecainjectioname

3. **Ensure Only Conversion Webhook Patches in config/crd/patches:**

   The config/crd/patches directory and the corresponding entries in config/crd/kustomization.yaml should only 
   contain files for conversion webhooks. Previously, a bug caused the patch file to be generated for any webhook, 
   but only patches for webhooks created with the --conversion option should be included.

For further guidance, you can refer to examples in the testdata/ directory in the Kubebuilder repository.

**Alternatively**: You can use the 'alpha generate' command to re-generate the project from scratch using the latest
release available. Afterward, you can re-add only your code implementation on top to ensure your project includes all
the latest bug fixes and enhancements.

`)
	}
}

const allowWebhookTrafficFragment = `
- allow-webhook-traffic.yaml`
