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

package v4

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	pluginutil "sigs.k8s.io/kubebuilder/v4/pkg/plugin/util"

	//nolint:golint
	// nolint:revive
	//nolint:golint
	// nolint:revive
	"sigs.k8s.io/kubebuilder/v4/test/e2e/utils"
)

// GenerateV4 implements a go/v4 plugin project defined by a TestContext.
func GenerateV4(kbc *utils.TestContext) {
	initingTheProject(kbc)
	creatingAPI(kbc)

	By("scaffolding mutating and validating webhooks")
	err := kbc.CreateWebhook(
		"--group", kbc.Group,
		"--version", kbc.Version,
		"--kind", kbc.Kind,
		"--defaulting",
		"--programmatic-validation",
	)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())

	By("implementing the mutating and validating webhooks")
	webhookFilePath := filepath.Join(
		kbc.Dir, "internal/webhook", kbc.Version,
		fmt.Sprintf("%s_webhook.go", strings.ToLower(kbc.Kind)))
	err = utils.ImplementWebhooks(webhookFilePath, strings.ToLower(kbc.Kind))
	ExpectWithOffset(1, err).NotTo(HaveOccurred())

	scaffoldConversionWebhook(kbc)

	ExpectWithOffset(1, pluginutil.UncommentCode(
		filepath.Join(kbc.Dir, "config", "default", "kustomization.yaml"),
		"#- ../certmanager", "#")).To(Succeed())
	ExpectWithOffset(1, pluginutil.UncommentCode(
		filepath.Join(kbc.Dir, "config", "default", "kustomization.yaml"),
		"#- ../prometheus", "#")).To(Succeed())
	ExpectWithOffset(1, pluginutil.UncommentCode(filepath.Join(kbc.Dir, "config", "default", "kustomization.yaml"),
		certManagerTarget, "#")).To(Succeed())
	ExpectWithOffset(1, pluginutil.UncommentCode(
		filepath.Join(kbc.Dir, "config", "prometheus", "kustomization.yaml"),
		monitorTlsPatch, "#")).To(Succeed())
	ExpectWithOffset(1, pluginutil.UncommentCode(
		filepath.Join(kbc.Dir, "config", "default", "kustomization.yaml"),
		`#- path: certmanager_metrics_manager_patch.yaml`, "#")).To(Succeed())
	ExpectWithOffset(1, pluginutil.UncommentCode(
		filepath.Join(kbc.Dir, "cmd", "main.go"),
		tlsConfigManager, "// ")).To(Succeed())
}

// GenerateV4WithoutMetrics implements a go/v4 plugin project defined by a TestContext.
func GenerateV4WithoutMetrics(kbc *utils.TestContext) {
	initingTheProject(kbc)
	creatingAPI(kbc)

	By("scaffolding mutating and validating webhooks")
	err := kbc.CreateWebhook(
		"--group", kbc.Group,
		"--version", kbc.Version,
		"--kind", kbc.Kind,
		"--defaulting",
		"--programmatic-validation",
	)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())

	By("implementing the mutating and validating webhooks")
	webhookFilePath := filepath.Join(
		kbc.Dir, "internal/webhook", kbc.Version,
		fmt.Sprintf("%s_webhook.go", strings.ToLower(kbc.Kind)))
	err = utils.ImplementWebhooks(webhookFilePath, strings.ToLower(kbc.Kind))
	ExpectWithOffset(1, err).NotTo(HaveOccurred())

	scaffoldConversionWebhook(kbc)

	ExpectWithOffset(1, pluginutil.UncommentCode(
		filepath.Join(kbc.Dir, "config", "default", "kustomization.yaml"),
		"#- ../certmanager", "#")).To(Succeed())
	ExpectWithOffset(1, pluginutil.UncommentCode(
		filepath.Join(kbc.Dir, "config", "default", "kustomization.yaml"),
		"#- ../prometheus", "#")).To(Succeed())
	ExpectWithOffset(1, pluginutil.UncommentCode(filepath.Join(kbc.Dir, "config", "default", "kustomization.yaml"),
		certManagerTarget, "#")).To(Succeed())
	// Disable metrics
	ExpectWithOffset(1, pluginutil.CommentCode(
		filepath.Join(kbc.Dir, "config", "default", "kustomization.yaml"),
		"- metrics_service.yaml", "#")).To(Succeed())
	ExpectWithOffset(1, pluginutil.CommentCode(
		filepath.Join(kbc.Dir, "config", "default", "kustomization.yaml"),
		metricsTarget, "#")).To(Succeed())
}

// GenerateV4WithoutMetrics implements a go/v4 plugin project defined by a TestContext.
func GenerateV4WithNetworkPoliciesWithoutWebhooks(kbc *utils.TestContext) {
	initingTheProject(kbc)
	creatingAPI(kbc)

	ExpectWithOffset(1, pluginutil.UncommentCode(
		filepath.Join(kbc.Dir, "config", "default", "kustomization.yaml"),
		"#- ../prometheus", "#")).To(Succeed())
	ExpectWithOffset(1, pluginutil.UncommentCode(
		filepath.Join(kbc.Dir, "config", "default", "kustomization.yaml"),
		metricsTarget, "#")).To(Succeed())
	By("uncomment kustomization.yaml to enable network policy")
	ExpectWithOffset(1, pluginutil.UncommentCode(
		filepath.Join(kbc.Dir, "config", "default", "kustomization.yaml"),
		"#- ../network-policy", "#")).To(Succeed())
}

// GenerateV4WithNetworkPolicies implements a go/v4 plugin project defined by a TestContext.
func GenerateV4WithNetworkPolicies(kbc *utils.TestContext) {
	initingTheProject(kbc)
	creatingAPI(kbc)

	By("scaffolding mutating and validating webhooks")
	err := kbc.CreateWebhook(
		"--group", kbc.Group,
		"--version", kbc.Version,
		"--kind", kbc.Kind,
		"--defaulting",
		"--programmatic-validation",
	)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())

	By("implementing the mutating and validating webhooks")
	webhookFilePath := filepath.Join(
		kbc.Dir, "internal/webhook", kbc.Version,
		fmt.Sprintf("%s_webhook.go", strings.ToLower(kbc.Kind)))
	err = utils.ImplementWebhooks(webhookFilePath, strings.ToLower(kbc.Kind))
	ExpectWithOffset(1, err).NotTo(HaveOccurred())

	scaffoldConversionWebhook(kbc)

	ExpectWithOffset(1, pluginutil.UncommentCode(
		filepath.Join(kbc.Dir, "config", "default", "kustomization.yaml"),
		"#- ../certmanager", "#")).To(Succeed())
	ExpectWithOffset(1, pluginutil.UncommentCode(
		filepath.Join(kbc.Dir, "config", "default", "kustomization.yaml"),
		"#- ../prometheus", "#")).To(Succeed())
	ExpectWithOffset(1, pluginutil.UncommentCode(
		filepath.Join(kbc.Dir, "config", "default", "kustomization.yaml"),
		metricsTarget, "#")).To(Succeed())
	ExpectWithOffset(1, pluginutil.UncommentCode(
		filepath.Join(kbc.Dir, "config", "default", "kustomization.yaml"),
		`#- path: certmanager_metrics_manager_patch.yaml`, "#")).To(Succeed())
	ExpectWithOffset(1, pluginutil.UncommentCode(
		filepath.Join(kbc.Dir, "config", "prometheus", "kustomization.yaml"),
		monitorTlsPatch, "#")).To(Succeed())
	ExpectWithOffset(1, pluginutil.UncommentCode(
		filepath.Join(kbc.Dir, "cmd", "main.go"),
		tlsConfigManager, "// ")).To(Succeed())
	By("uncomment kustomization.yaml to enable network policy")
	ExpectWithOffset(1, pluginutil.UncommentCode(
		filepath.Join(kbc.Dir, "config", "default", "kustomization.yaml"),
		"#- ../network-policy", "#")).To(Succeed())

	ExpectWithOffset(1, pluginutil.UncommentCode(filepath.Join(kbc.Dir, "config", "default", "kustomization.yaml"),
		certManagerTarget, "#")).To(Succeed())
}

// GenerateV4WithoutWebhooks implements a go/v4 plugin with APIs and enable Prometheus and CertManager
func GenerateV4WithoutWebhooks(kbc *utils.TestContext) {
	initingTheProject(kbc)
	creatingAPI(kbc)

	ExpectWithOffset(1, pluginutil.UncommentCode(
		filepath.Join(kbc.Dir, "config", "default", "kustomization.yaml"),
		"#- ../prometheus", "#")).To(Succeed())
}

func creatingAPI(kbc *utils.TestContext) {
	By("creating API definition")
	err := kbc.CreateAPI(
		"--group", kbc.Group,
		"--version", kbc.Version,
		"--kind", kbc.Kind,
		"--namespaced",
		"--resource",
		"--controller",
		"--make=false",
	)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())

	By("implementing the API")
	ExpectWithOffset(1, pluginutil.InsertCode(
		filepath.Join(kbc.Dir, "api", kbc.Version, fmt.Sprintf("%s_types.go", strings.ToLower(kbc.Kind))),
		fmt.Sprintf(`type %sSpec struct {
`, kbc.Kind),
		`	// +optional
Count int `+"`"+`json:"count,omitempty"`+"`"+`
`)).Should(Succeed())
}

func initingTheProject(kbc *utils.TestContext) {
	By("initializing a project")
	err := kbc.Init(
		"--plugins", "go/v4",
		"--project-version", "3",
		"--domain", kbc.Domain,
	)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
}

const metricsTarget = `- path: manager_metrics_patch.yaml
  target:
    kind: Deployment`

//nolint:lll
const certManagerTarget = `#replacements:
# - source: # Uncomment the following block if you have any webhook
#     kind: Service
#     version: v1
#     name: webhook-service
#     fieldPath: .metadata.name # Name of the service
#   targets:
#     - select:
#         kind: Certificate
#         group: cert-manager.io
#         version: v1
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
#       fieldPaths:
#         - .spec.dnsNames.0
#         - .spec.dnsNames.1
#       options:
#         delimiter: '.'
#         index: 1
#         create: true
#
# - source: # Uncomment the following block if you have a ValidatingWebhook (--programmatic-validation)
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
#     name: serving-cert # This name should match the one in certificate.yaml
#     fieldPath: .metadata.name
#   targets:
#     - select:
#         kind: ValidatingWebhookConfiguration
#       fieldPaths:
#         - .metadata.annotations.[cert-manager.io/inject-ca-from]
#       options:
#         delimiter: '/'
#         index: 1
#         create: true
#
# - source: # Uncomment the following block if you have a DefaultingWebhook (--defaulting )
#     kind: Certificate
#     group: cert-manager.io
#     version: v1
#     name: serving-cert # This name should match the one in certificate.yaml
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
#     name: serving-cert # This name should match the one in certificate.yaml
#     fieldPath: .metadata.name
#   targets:
#     - select:
#         kind: MutatingWebhookConfiguration
#       fieldPaths:
#         - .metadata.annotations.[cert-manager.io/inject-ca-from]
#       options:
#         delimiter: '/'
#         index: 1
#         create: true
#
# - source: # Uncomment the following block if you have a ConversionWebhook (--conversion)
#     kind: Certificate
#     group: cert-manager.io
#     version: v1
#     name: serving-cert # This name should match the one in certificate.yaml
#     fieldPath: .metadata.namespace # Namespace of the certificate CR
#   targets:
#     - select:
#         kind: CustomResourceDefinition
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
#     name: serving-cert # This name should match the one in certificate.yaml
#     fieldPath: .metadata.name
#   targets:
#     - select:
#         kind: CustomResourceDefinition
#       fieldPaths:
#         - .metadata.annotations.[cert-manager.io/inject-ca-from]
#       options:
#         delimiter: '/'
#         index: 1
#         create: true`

// scaffoldConversionWebhook sets up conversion webhooks for testing the ConversionTest API
func scaffoldConversionWebhook(kbc *utils.TestContext) {
	By("scaffolding conversion webhooks for testing ConversionTest v1 to v2 conversion")

	// Create API for v1 (hub) with conversion enabled
	err := kbc.CreateAPI(
		"--group", kbc.Group,
		"--version", "v1",
		"--kind", "ConversionTest",
		"--controller=true",
		"--resource=true",
		"--make=false",
	)
	ExpectWithOffset(1, err).NotTo(HaveOccurred(), "failed to create v1 API for conversion testing")

	// Create API for v2 (spoke) without a controller
	err = kbc.CreateAPI(
		"--group", kbc.Group,
		"--version", "v2",
		"--kind", "ConversionTest",
		"--controller=false",
		"--resource=true",
		"--make=false",
	)
	ExpectWithOffset(1, err).NotTo(HaveOccurred(), "failed to create v2 API for conversion testing")

	// Create the conversion webhook for v1
	By("setting up the conversion webhook for v1")
	err = kbc.CreateWebhook(
		"--group", kbc.Group,
		"--version", "v1",
		"--kind", "ConversionTest",
		"--conversion",
		"--make=false",
	)
	ExpectWithOffset(1, err).NotTo(HaveOccurred(), "failed to create conversion webhook for v1")

	// Insert Size field in v1
	By("implementing the size spec in v1")
	ExpectWithOffset(1, pluginutil.InsertCode(
		filepath.Join(kbc.Dir, "api", "v1", "conversiontest_types.go"),
		"Foo string `json:\"foo,omitempty\"`",
		"\n\tSize int `json:\"size,omitempty\"` // Number of desired instances",
	)).NotTo(HaveOccurred(), "failed to add size spec to conversiontest_types v1")

	// Insert Replicas field in v2
	By("implementing the replicas spec in v2")
	ExpectWithOffset(1, pluginutil.InsertCode(
		filepath.Join(kbc.Dir, "api", "v2", "conversiontest_types.go"),
		"Foo string `json:\"foo,omitempty\"`",
		"\n\tReplicas int `json:\"replicas,omitempty\"` // Number of replicas",
	)).NotTo(HaveOccurred(), "failed to add replicas spec to conversiontest_types v2")

	// TODO: Remove the code bellow when we have hub and spoke scaffolded by
	// Kubebuilder. Intead of create the file we will replace the TODO(user)
	// with the code implementation.
	By("implementing markers")
	ExpectWithOffset(1, pluginutil.InsertCode(
		filepath.Join(kbc.Dir, "api", "v1", "conversiontest_types.go"),
		"// +kubebuilder:object:root=true\n// +kubebuilder:subresource:status",
		"\n// +kubebuilder:storageversion\n// +kubebuilder:conversion:hub\n",
	)).NotTo(HaveOccurred(), "failed to add markers to conversiontest_types v1")

	// Create the hub conversion file in v1
	By("creating the conversion implementation in v1 as hub")
	err = os.WriteFile(filepath.Join(kbc.Dir, "api", "v1", "conversiontest_conversion.go"), []byte(`
package v1

// ConversionTest defines the hub conversion logic.
// Implement the Hub interface to signal that v1 is the hub version.
func (*ConversionTest) Hub() {}
`), 0644)
	ExpectWithOffset(1, err).NotTo(HaveOccurred(), "failed to create hub conversion file in v1")

	// Create the conversion file in v2
	By("creating the conversion implementation in v2")
	err = os.WriteFile(filepath.Join(kbc.Dir, "api", "v2", "conversiontest_conversion.go"), []byte(`
package v2

import (
	"log"

	"sigs.k8s.io/controller-runtime/pkg/conversion"
	v1 "sigs.k8s.io/kubebuilder/v4/api/v1"
)

// ConvertTo converts this ConversionTest to the Hub version (v1).
func (src *ConversionTest) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*v1.ConversionTest)
	log.Printf("Converting from %T to %T", src.APIVersion, dst.APIVersion)

	// Implement conversion logic from v2 to v1
	dst.Spec.Size = src.Spec.Replicas // Convert replicas in v2 to size in v1

	return nil
}

// ConvertFrom converts the Hub version (v1) to this ConversionTest (v2).
func (dst *ConversionTest) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*v1.ConversionTest)
	log.Printf("Converting from %T to %T", src.APIVersion, dst.APIVersion)

	// Implement conversion logic from v1 to v2
	dst.Spec.Replicas = src.Spec.Size // Convert size in v1 to replicas in v2

	return nil
}
`), 0644)
	ExpectWithOffset(1, err).NotTo(HaveOccurred(), "failed to create conversion file in v2")
}

const monitorTlsPatch = `#patches:
#  - path: monitor_tls_patch.yaml
#    target:
#      kind: ServiceMonitor`

const tlsConfigManager = `// metricsServerOptions.CertDir = "/tmp/k8s-metrics-server/metrics-certs"
		// metricsServerOptions.CertName = "tls.crt"
		// metricsServerOptions.KeyName = "tls.key"`
