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
	networkpolicy "sigs.k8s.io/kubebuilder/v4/pkg/plugins/common/kustomize/v2/scaffolds/internal/templates/config/network-policy"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/common/kustomize/v2/scaffolds/internal/templates/config/webhook"
)

var _ plugins.Scaffolder = &webhookScaffolder{}

const (
	kustomizeFilePath    = "config/default/kustomization.yaml"
	kustomizeCRDFilePath = "config/crd/kustomization.yaml"
)

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
		&networkpolicy.PolicyAllowWebhooks{},
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
		return fmt.Errorf("error scaffolding kustomize webhook manifests: %w", err)
	}

	// Apply project-specific customizations:
	// - Add reference to allow-webhook-traffic.yaml in network policy configuration.
	// - Enable all webhook-related sections in config/default/kustomization.yaml.
	addNetworkPoliciesForWebhooks()
	// enableWebhookDefaults ensures all necessary components for webhook functionality
	// are enabled in config/default/kustomization.yaml, including:
	// - webhook and cert-manager directories
	// - manager patches
	// - replacements for certificate injection
	enableWebhookDefaults()
	if s.resource.HasValidationWebhook() {
		uncommentCodeForValidationWebhooks()
	}
	if s.resource.HasDefaultingWebhook() {
		uncommentCodeForDefaultWebhooks()
	}
	if s.resource.HasConversionWebhook() {
		uncommentCodeForConversionWebhooks(s.resource)
	}

	const helmPluginKey = "helm.kubebuilder.io/v1-alpha"
	var helmPlugin interface{}
	err := s.config.DecodePluginConfig(helmPluginKey, &helmPlugin)
	if !errors.As(err, &config.PluginKeyNotFoundError{}) {
		testChartPath := ".github/workflows/test-chart.yml"
		//nolint:lll
		_ = pluginutil.UncommentCode(
			testChartPath, `#      - name: Install cert-manager via Helm
#        run: |
#          helm repo add jetstack https://charts.jetstack.io
#          helm repo update
#          helm install cert-manager jetstack/cert-manager --namespace cert-manager --create-namespace --set installCRDs=true
#
#      - name: Wait for cert-manager to be ready
#        run: |
#          kubectl wait --namespace cert-manager --for=condition=available --timeout=300s deployment/cert-manager
#          kubectl wait --namespace cert-manager --for=condition=available --timeout=300s deployment/cert-manager-cainjector
#          kubectl wait --namespace cert-manager --for=condition=available --timeout=300s deployment/cert-manager-webhook
`, "#",
		)

		_ = pluginutil.ReplaceInFile(testChartPath, "# TODO: Uncomment if cert-manager is enabled", "")
	}

	return nil
}

// uncommentCodeForConversionWebhooks enables CA injection logic in Kustomize manifests
// for ConversionWebhooks by uncommenting certificate sources and CRD annotation targets.
// This is required to make cert-manager correctly inject the CA bundle into CRDs.
func uncommentCodeForConversionWebhooks(r resource.Resource) {
	crdName := fmt.Sprintf("%s.%s", r.Plural, r.QualifiedGroup())
	err := pluginutil.UncommentCode(
		kustomizeFilePath,
		fmt.Sprintf(`# - source: # Uncomment the following block if you have a ConversionWebhook (--conversion)
#     kind: Certificate
#     group: cert-manager.io
#     version: v1
#     name: serving-cert
#     fieldPath: .metadata.namespace # Namespace of the certificate CR
#   targets: # Do not remove or uncomment the following scaffold marker; required to generate code for target CRD.
#     - select:
#         kind: CustomResourceDefinition
#         name: %s
#       fieldPaths:
#         - .metadata.annotations.[cert-manager.io/inject-ca-from]
#       options:
#         delimiter: '/'
#         index: 0
#         create: true`, crdName),
		"#",
	)
	if err != nil {
		log.Warningf("Unable to find the certificate namespace replacement for "+
			"CRD %s to uncomment in %s. Conversion webhooks require this replacement "+
			"to inject the CA properly.",
			crdName, kustomizeFilePath)
	}
	err = pluginutil.UncommentCode(
		kustomizeFilePath,
		fmt.Sprintf(`# - source:
#     kind: Certificate
#     group: cert-manager.io
#     version: v1
#     name: serving-cert
#     fieldPath: .metadata.name
#   targets: # Do not remove or uncomment the following scaffold marker; required to generate code for target CRD.
#     - select:
#         kind: CustomResourceDefinition
#         name: %s
#       fieldPaths:
#         - .metadata.annotations.[cert-manager.io/inject-ca-from]
#       options:
#         delimiter: '/'
#         index: 1
#         create: true`, crdName),
		"#",
	)
	if err != nil {
		log.Warningf("Unable to find the certificate name replacement for CRD %s "+
			"to uncomment in %s. Conversion webhooks require this replacement to inject "+
			"the CA properly.",
			crdName, kustomizeFilePath)
	}

	err = pluginutil.UncommentCode(kustomizeCRDFilePath, `#configurations:
#- kustomizeconfig.yaml`, `#`)
	if err != nil {
		hasWebHookUncommented, errCheck := pluginutil.HasFileContentWith(kustomizeCRDFilePath,
			`configurations:
- kustomizeconfig.yaml`)
		if !hasWebHookUncommented || errCheck != nil {
			log.Warningf("Unable to find the target configurations with kustomizeconfig.yaml"+
				"to uncomment in the file %s. ConverstionWebhooks requires this configuration "+
				"to be uncommented to inject CA", kustomizeCRDFilePath)
		}
	}
}

func uncommentCodeForDefaultWebhooks() {
	err := pluginutil.UncommentCode(
		kustomizeFilePath,
		`# - source: # Uncomment the following block if you have a DefaultingWebhook (--defaulting )
#     kind: Certificate
#     group: cert-manager.io
#     version: v1
#     name: serving-cert
#     fieldPath: .metadata.namespace # Namespace of the certificate CR
#   targets:
#     - select:
#         kind: MutatingWebhookConfiguration
#       fieldPaths:
#         - .metadata.annotations.[cert-manager.io/inject-ca-from]
#       options:
#         delimiter: '/'
#         index: 0
#         create: true
# - source:
#     kind: Certificate
#     group: cert-manager.io
#     version: v1
#     name: serving-cert
#     fieldPath: .metadata.name
#   targets:
#     - select:
#         kind: MutatingWebhookConfiguration
#       fieldPaths:
#         - .metadata.annotations.[cert-manager.io/inject-ca-from]
#       options:
#         delimiter: '/'
#         index: 1
#         create: true`,
		"#",
	)
	if err != nil {
		hasWebHookUncommented, errCheck := pluginutil.HasFileContentWith(kustomizeFilePath,
			`   targets:
     - select:
         kind: MutatingWebhookConfiguration`)
		if !hasWebHookUncommented || errCheck != nil {
			log.Warningf("Unable to find the MutatingWebhookConfiguration section "+
				"to uncomment in %s. Webhooks scaffolded with '--defaulting' require "+
				"this configuration for CA injection.",
				kustomizeFilePath)
		}
	}
}

func uncommentCodeForValidationWebhooks() {
	err := pluginutil.UncommentCode(
		kustomizeFilePath,
		`# - source: # Uncomment the following block if you have a ValidatingWebhook (--programmatic-validation)
#     kind: Certificate
#     group: cert-manager.io
#     version: v1
#     name: serving-cert # This name should match the one in certificate.yaml
#     fieldPath: .metadata.namespace # Namespace of the certificate CR
#   targets:
#     - select:
#         kind: ValidatingWebhookConfiguration
#       fieldPaths:
#         - .metadata.annotations.[cert-manager.io/inject-ca-from]
#       options:
#         delimiter: '/'
#         index: 0
#         create: true
# - source:
#     kind: Certificate
#     group: cert-manager.io
#     version: v1
#     name: serving-cert
#     fieldPath: .metadata.name
#   targets:
#     - select:
#         kind: ValidatingWebhookConfiguration
#       fieldPaths:
#         - .metadata.annotations.[cert-manager.io/inject-ca-from]
#       options:
#         delimiter: '/'
#         index: 1
#         create: true`,
		"#",
	)
	if err != nil {
		hasWebHookUncommented, errCheck := pluginutil.HasFileContentWith(kustomizeFilePath,
			`   targets:
     - select:
         kind: ValidatingWebhookConfiguration`)
		if !hasWebHookUncommented || errCheck != nil {
			log.Warningf("Unable to find the ValidatingWebhookConfiguration section "+
				"to uncomment in %s. Webhooks scaffolded with '--programmatic-validation' "+
				"require this configuration for CA injection.",
				kustomizeFilePath)
		}
	}
}

func enableWebhookDefaults() {
	err := pluginutil.UncommentCode(kustomizeFilePath, "#- ../webhook", `#`)
	if err != nil {
		hasWebHookUncommented, errCheck := pluginutil.HasFileContentWith(kustomizeFilePath, "- ../webhook")
		if !hasWebHookUncommented || errCheck != nil {
			log.Warningf("Unable to find the target #- ../webhook to uncomment in the file "+
				"%s.", kustomizeFilePath)
		}
	}

	err = pluginutil.UncommentCode(kustomizeFilePath, "#patches:", `#`)
	if err != nil {
		hasWebHookUncommented, errCheck := pluginutil.HasFileContentWith(kustomizeFilePath, "patches:")
		if !hasWebHookUncommented || errCheck != nil {
			log.Warningf("Unable to find the line '#patches:' to uncomment in the file "+
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
			log.Warningf("Unable to find the target #- path: manager_webhook_patch.yaml to uncomment in the file "+
				"%s.", kustomizeFilePath)
		}
	}

	err = pluginutil.UncommentCode(kustomizeFilePath, `#- ../certmanager`, `#`)
	if err != nil {
		hasWebHookUncommented, errCheck := pluginutil.HasFileContentWith(kustomizeFilePath,
			"../certmanager")
		if !hasWebHookUncommented || errCheck != nil {
			log.Warningf("Unable to find the '../certmanager' section to uncomment in %s. "+
				"Projects that use webhooks must enable certificate management."+
				"Please ensure cert-manager integration is enabled.",
				kustomizeFilePath)
		}
	}

	err = pluginutil.UncommentCode(kustomizeFilePath, `#replacements:`, `#`)
	if err != nil {
		hasWebHookUncommented, errCheck := pluginutil.HasFileContentWith(kustomizeFilePath,
			"replacements:")
		if !hasWebHookUncommented || errCheck != nil {
			log.Warningf("Unable to find the '#replacements:' section to uncomment in %s."+
				"Projects using webhooks must enable cert-manager CA injection by uncommenting"+
				"the required replacements.",
				kustomizeFilePath)
		}
	}

	err = pluginutil.UncommentCode(
		kustomizeFilePath,
		`# - source: # Uncomment the following block if you have any webhook
#     kind: Service
#     version: v1
#     name: webhook-service
#     fieldPath: .metadata.name # Name of the service
#   targets:
#     - select:
#         kind: Certificate
#         group: cert-manager.io
#         version: v1
#         name: serving-cert
#       fieldPaths:
#         - .spec.dnsNames.0
#         - .spec.dnsNames.1
#       options:
#         delimiter: '.'
#         index: 0
#         create: true
# - source:
#     kind: Service
#     version: v1
#     name: webhook-service
#     fieldPath: .metadata.namespace # Namespace of the service
#   targets:
#     - select:
#         kind: Certificate
#         group: cert-manager.io
#         version: v1
#         name: serving-cert
#       fieldPaths:
#         - .spec.dnsNames.0
#         - .spec.dnsNames.1
#       options:
#         delimiter: '.'
#         index: 1
#         create: true`,
		"#",
	)
	if err != nil {
		hasWebHookUncommented, errCheck := pluginutil.HasFileContentWith(kustomizeFilePath,
			`     kind: Service
     version: v1
     name: webhook-service
     fieldPath: .metadata.name`)
		if !hasWebHookUncommented || errCheck != nil {
			log.Warningf("Unable to find the '#- source: # Uncomment the following block if you have any webhook' "+
				"section to uncomment in %s. "+
				"Projects with webhooks must enable certificates via cert-manager.",
				kustomizeFilePath)
		}
	}
}

func addNetworkPoliciesForWebhooks() {
	policyKustomizeFilePath := "config/network-policy/kustomization.yaml"
	err := pluginutil.InsertCodeIfNotExist(policyKustomizeFilePath,
		"resources:", allowWebhookTrafficFragment)
	if err != nil {
		log.Errorf("Unable to add the line '- allow-webhook-traffic.yaml' at the end of the file"+
			"%s to allow webhook traffic.", policyKustomizeFilePath)
	}
}

// Deprecated: remove it when go/v4 and/or kustomize/v2 be removed
// validateScaffoldedProject will output a message to help users fix their scaffold
func validateScaffoldedProject() {
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
