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

package v3

import (
	"fmt"
	"path/filepath"
	"strings"

	pluginutil "sigs.k8s.io/kubebuilder/v3/pkg/plugin/util"

	//nolint:golint
	//nolint:revive
	. "github.com/onsi/ginkgo"

	//nolint:golint
	//nolint:revive
	. "github.com/onsi/gomega"

	"sigs.k8s.io/kubebuilder/v3/test/e2e/utils"
)

//nolint:dupl
// GenerateV2 implements a go/v2 plugin project defined by a TestContext.
func GenerateV2(kbc *utils.TestContext) {
	var err error

	By("initializing a project")
	err = kbc.Init(
		"--plugins", "go/v2",
		"--project-version", "3",
		"--domain", kbc.Domain,
		"--fetch-deps=false",
	)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())

	By("creating API definition")
	err = kbc.CreateAPI(
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

	By("scaffolding mutating and validating webhooks")
	err = kbc.CreateWebhook(
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

	By("uncomment kustomization.yaml to enable webhook and ca injection")
	ExpectWithOffset(1, pluginutil.UncommentCode(
		filepath.Join(kbc.Dir, "config", "default", "kustomization.yaml"),
		"#- ../webhook", "#")).To(Succeed())
	ExpectWithOffset(1, pluginutil.UncommentCode(
		filepath.Join(kbc.Dir, "config", "default", "kustomization.yaml"),
		"#- ../certmanager", "#")).To(Succeed())
	ExpectWithOffset(1, pluginutil.UncommentCode(
		filepath.Join(kbc.Dir, "config", "default", "kustomization.yaml"),
		"#- ../prometheus", "#")).To(Succeed())
	ExpectWithOffset(1, pluginutil.UncommentCode(
		filepath.Join(kbc.Dir, "config", "default", "kustomization.yaml"),
		"#- manager_webhook_patch.yaml", "#")).To(Succeed())
	ExpectWithOffset(1, pluginutil.UncommentCode(
		filepath.Join(kbc.Dir, "config", "default", "kustomization.yaml"),
		"#- webhookcainjection_patch.yaml", "#")).To(Succeed())
	ExpectWithOffset(1, pluginutil.UncommentCode(filepath.Join(kbc.Dir, "config", "default", "kustomization.yaml"),
		`#- name: CERTIFICATE_NAMESPACE # namespace of the certificate CR
#  objref:
#    kind: Certificate
#    group: cert-manager.io
#    version: v1alpha2
#    name: serving-cert # this name should match the one in certificate.yaml
#  fieldref:
#    fieldpath: metadata.namespace
#- name: CERTIFICATE_NAME
#  objref:
#    kind: Certificate
#    group: cert-manager.io
#    version: v1alpha2
#    name: serving-cert # this name should match the one in certificate.yaml
#- name: SERVICE_NAMESPACE # namespace of the service
#  objref:
#    kind: Service
#    version: v1
#    name: webhook-service
#  fieldref:
#    fieldpath: metadata.namespace
#- name: SERVICE_NAME
#  objref:
#    kind: Service
#    version: v1
#    name: webhook-service`, "#")).To(Succeed())
}

//nolint:dupl
// GenerateV3 implements a go/v3(-alpha) plugin project defined by a TestContext.
func GenerateV3(kbc *utils.TestContext, crdAndWebhookVersion string) {
	var err error

	By("initializing a project")
	err = kbc.Init(
		"--plugins", "go/v3",
		"--project-version", "3",
		"--domain", kbc.Domain,
		"--fetch-deps=false",
	)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())

	By("creating API definition")
	err = kbc.CreateAPI(
		"--group", kbc.Group,
		"--version", kbc.Version,
		"--kind", kbc.Kind,
		"--namespaced",
		"--resource",
		"--controller",
		"--make=false",
		"--crd-version", crdAndWebhookVersion,
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

	By("scaffolding mutating and validating webhooks")
	err = kbc.CreateWebhook(
		"--group", kbc.Group,
		"--version", kbc.Version,
		"--kind", kbc.Kind,
		"--defaulting",
		"--programmatic-validation",
		"--webhook-version", crdAndWebhookVersion,
	)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())

	By("implementing the mutating and validating webhooks")
	err = pluginutil.ImplementWebhooks(filepath.Join(
		kbc.Dir, "api", kbc.Version,
		fmt.Sprintf("%s_webhook.go", strings.ToLower(kbc.Kind))))
	ExpectWithOffset(1, err).NotTo(HaveOccurred())

	By("uncomment kustomization.yaml to enable webhook and ca injection")
	ExpectWithOffset(1, pluginutil.UncommentCode(
		filepath.Join(kbc.Dir, "config", "default", "kustomization.yaml"),
		"#- ../webhook", "#")).To(Succeed())
	ExpectWithOffset(1, pluginutil.UncommentCode(
		filepath.Join(kbc.Dir, "config", "default", "kustomization.yaml"),
		"#- ../certmanager", "#")).To(Succeed())
	ExpectWithOffset(1, pluginutil.UncommentCode(
		filepath.Join(kbc.Dir, "config", "default", "kustomization.yaml"),
		"#- ../prometheus", "#")).To(Succeed())
	ExpectWithOffset(1, pluginutil.UncommentCode(
		filepath.Join(kbc.Dir, "config", "default", "kustomization.yaml"),
		"#- manager_webhook_patch.yaml", "#")).To(Succeed())
	ExpectWithOffset(1, pluginutil.UncommentCode(
		filepath.Join(kbc.Dir, "config", "default", "kustomization.yaml"),
		"#- webhookcainjection_patch.yaml", "#")).To(Succeed())
	ExpectWithOffset(1, pluginutil.UncommentCode(filepath.Join(kbc.Dir, "config", "default", "kustomization.yaml"),
		`#- name: CERTIFICATE_NAMESPACE # namespace of the certificate CR
#  objref:
#    kind: Certificate
#    group: cert-manager.io
#    version: v1
#    name: serving-cert # this name should match the one in certificate.yaml
#  fieldref:
#    fieldpath: metadata.namespace
#- name: CERTIFICATE_NAME
#  objref:
#    kind: Certificate
#    group: cert-manager.io
#    version: v1
#    name: serving-cert # this name should match the one in certificate.yaml
#- name: SERVICE_NAMESPACE # namespace of the service
#  objref:
#    kind: Service
#    version: v1
#    name: webhook-service
#  fieldref:
#    fieldpath: metadata.namespace
#- name: SERVICE_NAME
#  objref:
#    kind: Service
#    version: v1
#    name: webhook-service`, "#")).To(Succeed())

	if crdAndWebhookVersion == "v1beta1" {
		_ = pluginutil.RunCmd("Update dependencies", "go", "mod", "tidy")
	}
}

//nolint:dupl
// GenerateV3WithDeployImage implements a go/v3 plugin and the deployImage one
func GenerateV3WithDeployImage(kbc *utils.TestContext) {
	var err error

	By("initializing a project with go/v3")
	err = kbc.Init(
		"--plugins", "go/v3",
		"--project-version", "3",
		"--domain", kbc.Domain,
		"--fetch-deps=false",
	)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())

	By("creating API definition with deploy-image/v1-alpha plugin")
	err = kbc.CreateAPI(
		"--group", kbc.Group,
		"--version", kbc.Version,
		"--kind", kbc.Kind,
		"--plugins", "deploy-image/v1-alpha",
		"--image", "memcached:1.6.15-alpine",
	)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())

	By("implementing the API without the plugin")
	ExpectWithOffset(1, pluginutil.InsertCode(
		filepath.Join(kbc.Dir, "api", kbc.Version, fmt.Sprintf("%s_types.go", strings.ToLower(kbc.Kind))),
		fmt.Sprintf(`type %sSpec struct {
`, kbc.Kind),
		`	// +optional
Count int `+"`"+`json:"count,omitempty"`+"`"+`
`)).Should(Succeed())

	By("scaffolding mutating and validating webhooks")
	err = kbc.CreateWebhook(
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

	By("uncomment kustomization.yaml to enable webhook and ca injection")
	ExpectWithOffset(1, pluginutil.UncommentCode(
		filepath.Join(kbc.Dir, "config", "default", "kustomization.yaml"),
		"#- ../webhook", "#")).To(Succeed())
	ExpectWithOffset(1, pluginutil.UncommentCode(
		filepath.Join(kbc.Dir, "config", "default", "kustomization.yaml"),
		"#- ../certmanager", "#")).To(Succeed())
	ExpectWithOffset(1, pluginutil.UncommentCode(
		filepath.Join(kbc.Dir, "config", "default", "kustomization.yaml"),
		"#- ../prometheus", "#")).To(Succeed())
	ExpectWithOffset(1, pluginutil.UncommentCode(
		filepath.Join(kbc.Dir, "config", "default", "kustomization.yaml"),
		"#- manager_webhook_patch.yaml", "#")).To(Succeed())
	ExpectWithOffset(1, pluginutil.UncommentCode(
		filepath.Join(kbc.Dir, "config", "default", "kustomization.yaml"),
		"#- webhookcainjection_patch.yaml", "#")).To(Succeed())
	ExpectWithOffset(1, pluginutil.UncommentCode(filepath.Join(kbc.Dir, "config", "default", "kustomization.yaml"),
		`#- name: CERTIFICATE_NAMESPACE # namespace of the certificate CR
#  objref:
#    kind: Certificate
#    group: cert-manager.io
#    version: v1
#    name: serving-cert # this name should match the one in certificate.yaml
#  fieldref:
#    fieldpath: metadata.namespace
#- name: CERTIFICATE_NAME
#  objref:
#    kind: Certificate
#    group: cert-manager.io
#    version: v1
#    name: serving-cert # this name should match the one in certificate.yaml
#- name: SERVICE_NAMESPACE # namespace of the service
#  objref:
#    kind: Service
#    version: v1
#    name: webhook-service
#  fieldref:
#    fieldpath: metadata.namespace
#- name: SERVICE_NAME
#  objref:
#    kind: Service
#    version: v1
#    name: webhook-service`, "#")).To(Succeed())

}

// GenerateV3 implements a go/v3(-alpha) plugin project defined by a TestContext.
func GenerateV3WithKustomizeV2(kbc *utils.TestContext, crdAndWebhookVersion string) {
	var err error

	By("initializing a project")
	err = kbc.Init(
		"--plugins", "kustomize/v2-alpha,base.go.kubebuilder.io/v3",
		"--project-version", "3",
		"--domain", kbc.Domain,
		"--fetch-deps=false",
	)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())

	By("creating API definition")
	err = kbc.CreateAPI(
		"--group", kbc.Group,
		"--version", kbc.Version,
		"--kind", kbc.Kind,
		"--namespaced",
		"--resource",
		"--controller",
		"--make=false",
		"--crd-version", crdAndWebhookVersion,
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

	By("scaffolding mutating and validating webhooks")
	err = kbc.CreateWebhook(
		"--group", kbc.Group,
		"--version", kbc.Version,
		"--kind", kbc.Kind,
		"--defaulting",
		"--programmatic-validation",
		"--webhook-version", crdAndWebhookVersion,
	)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())

	By("implementing the mutating and validating webhooks")
	err = pluginutil.ImplementWebhooks(filepath.Join(
		kbc.Dir, "api", kbc.Version,
		fmt.Sprintf("%s_webhook.go", strings.ToLower(kbc.Kind))))
	ExpectWithOffset(1, err).NotTo(HaveOccurred())

	By("uncomment kustomization.yaml to enable webhook and ca injection")
	ExpectWithOffset(1, pluginutil.UncommentCode(
		filepath.Join(kbc.Dir, "config", "default", "kustomization.yaml"),
		"#- ../webhook", "#")).To(Succeed())
	ExpectWithOffset(1, pluginutil.UncommentCode(
		filepath.Join(kbc.Dir, "config", "default", "kustomization.yaml"),
		"#- ../certmanager", "#")).To(Succeed())
	ExpectWithOffset(1, pluginutil.UncommentCode(
		filepath.Join(kbc.Dir, "config", "default", "kustomization.yaml"),
		"#- ../prometheus", "#")).To(Succeed())
	ExpectWithOffset(1, pluginutil.UncommentCode(
		filepath.Join(kbc.Dir, "config", "default", "kustomization.yaml"),
		"#- manager_webhook_patch.yaml", "#")).To(Succeed())
	ExpectWithOffset(1, pluginutil.UncommentCode(
		filepath.Join(kbc.Dir, "config", "default", "kustomization.yaml"),
		"#- webhookcainjection_patch.yaml", "#")).To(Succeed())
	ExpectWithOffset(1, pluginutil.UncommentCode(filepath.Join(kbc.Dir, "config", "default", "kustomization.yaml"),
		`#replacements:
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
#          create: true`, "#")).To(Succeed())

}
