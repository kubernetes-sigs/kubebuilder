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
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	pluginutil "sigs.k8s.io/kubebuilder/v3/pkg/plugin/util"

	//nolint:golint
	//nolint:revive
	//nolint:golint
	//nolint:revive
	"sigs.k8s.io/kubebuilder/v3/test/e2e/utils"
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
	err = pluginutil.ImplementWebhooks(filepath.Join(
		kbc.Dir, "api", kbc.Version,
		fmt.Sprintf("%s_webhook.go", strings.ToLower(kbc.Kind))))
	ExpectWithOffset(1, err).NotTo(HaveOccurred())

	ExpectWithOffset(1, pluginutil.UncommentCode(
		filepath.Join(kbc.Dir, "config", "default", "kustomization.yaml"),
		"#- ../certmanager", "#")).To(Succeed())
	ExpectWithOffset(1, pluginutil.UncommentCode(
		filepath.Join(kbc.Dir, "config", "default", "kustomization.yaml"),
		"#- ../prometheus", "#")).To(Succeed())
	ExpectWithOffset(1, pluginutil.UncommentCode(
		filepath.Join(kbc.Dir, "config", "default", "kustomization.yaml"),
		"#- path: webhookcainjection_patch.yaml", "#")).To(Succeed())
	ExpectWithOffset(1, pluginutil.UncommentCode(filepath.Join(kbc.Dir, "config", "default", "kustomization.yaml"),
		certManagerTarget, "#")).To(Succeed())

	if kbc.IsRestricted {
		By("uncomment kustomize files to ensure that pods are restricted")
		uncommentPodStandards(kbc)
	}
}

// GenerateV4WithoutWebhooks implements a go/v4 plugin with APIs and enable Prometheus and CertManager
func GenerateV4WithoutWebhooks(kbc *utils.TestContext) {
	initingTheProject(kbc)
	creatingAPI(kbc)

	ExpectWithOffset(1, pluginutil.UncommentCode(
		filepath.Join(kbc.Dir, "config", "default", "kustomization.yaml"),
		"#- ../prometheus", "#")).To(Succeed())

	if kbc.IsRestricted {
		By("uncomment kustomize files to ensure that pods are restricted")
		uncommentPodStandards(kbc)
	}
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

//nolint:lll
const certManagerTarget = `#replacements:
#  - source: # Add cert-manager annotation to ValidatingWebhookConfiguration, MutatingWebhookConfiguration and CRDs
#      kind: Certificate
#      group: cert-manager.io
#      version: v1
#      name: serving-cert # this name should match the one in certificate.yaml
#      fieldPath: .metadata.namespace # namespace of the certificate CR
#    targets:
#      - select:
#          kind: ValidatingWebhookConfiguration
#        fieldPaths:
#          - .metadata.annotations.[cert-manager.io/inject-ca-from]
#        options:
#          delimiter: '/'
#          index: 0
#          create: true
#      - select:
#          kind: MutatingWebhookConfiguration
#        fieldPaths:
#          - .metadata.annotations.[cert-manager.io/inject-ca-from]
#        options:
#          delimiter: '/'
#          index: 0
#          create: true
#      - select:
#          kind: CustomResourceDefinition
#        fieldPaths:
#          - .metadata.annotations.[cert-manager.io/inject-ca-from]
#        options:
#          delimiter: '/'
#          index: 0
#          create: true
#  - source:
#      kind: Certificate
#      group: cert-manager.io
#      version: v1
#      name: serving-cert # this name should match the one in certificate.yaml
#      fieldPath: .metadata.name
#    targets:
#      - select:
#          kind: ValidatingWebhookConfiguration
#        fieldPaths:
#          - .metadata.annotations.[cert-manager.io/inject-ca-from]
#        options:
#          delimiter: '/'
#          index: 1
#          create: true
#      - select:
#          kind: MutatingWebhookConfiguration
#        fieldPaths:
#          - .metadata.annotations.[cert-manager.io/inject-ca-from]
#        options:
#          delimiter: '/'
#          index: 1
#          create: true
#      - select:
#          kind: CustomResourceDefinition
#        fieldPaths:
#          - .metadata.annotations.[cert-manager.io/inject-ca-from]
#        options:
#          delimiter: '/'
#          index: 1
#          create: true
#  - source: # Add cert-manager annotation to the webhook Service
#      kind: Service
#      version: v1
#      name: webhook-service
#      fieldPath: .metadata.name # namespace of the service
#    targets:
#      - select:
#          kind: Certificate
#          group: cert-manager.io
#          version: v1
#        fieldPaths:
#          - .spec.dnsNames.0
#          - .spec.dnsNames.1
#        options:
#          delimiter: '.'
#          index: 0
#          create: true
#  - source:
#      kind: Service
#      version: v1
#      name: webhook-service
#      fieldPath: .metadata.namespace # namespace of the service
#    targets:
#      - select:
#          kind: Certificate
#          group: cert-manager.io
#          version: v1
#        fieldPaths:
#          - .spec.dnsNames.0
#          - .spec.dnsNames.1
#        options:
#          delimiter: '.'
#          index: 1
#          create: true`

func uncommentPodStandards(kbc *utils.TestContext) {
	configManager := filepath.Join(kbc.Dir, "config", "manager", "manager.yaml")

	//nolint:lll
	if err := pluginutil.ReplaceInFile(configManager, `# TODO(user): For common cases that do not require escalating privileges
        # it is recommended to ensure that all your Pods/Containers are restrictive.
        # More info: https://kubernetes.io/docs/concepts/security/pod-security-standards/#restricted
        # Please uncomment the following code if your project does NOT have to work on old Kubernetes
        # versions < 1.19 or on vendors versions which do NOT support this field by default (i.e. Openshift < 4.11 ).
        # seccompProfile:
        #   type: RuntimeDefault`, `seccompProfile:
          type: RuntimeDefault`); err == nil {
		ExpectWithOffset(1, err).NotTo(HaveOccurred())
	}
}
