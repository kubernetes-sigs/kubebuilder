/*
Copyright 2019 The Kubernetes Authors.

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

package e2e

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo" //nolint:golint
	. "github.com/onsi/gomega" //nolint:golint

	"sigs.k8s.io/kubebuilder/v3/pkg/config"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugin"
	"sigs.k8s.io/kubebuilder/v3/test/e2e/utils"
)

var (
	pluginV2 = plugin.Version{Number: 2}
	pluginV3 = plugin.Version{Number: 3}

	projectV2 = config.Version{Number: 2}
	projectV3 = config.Version{Number: 3}

	certManagerVersion = map[plugin.Version]string{
		pluginV2: "v1alpha2",
		pluginV3: "v1",
	}
	defaultCRDAndWebhookVersions = map[plugin.Version]string{
		pluginV2: "v1beta1",
		pluginV3: "v1",
	}
)

// GenerateGo implements a go plugin project defined by a TestContext.
func GenerateGo(
	kbc *utils.TestContext,
	pluginVersion plugin.Version,
	projectVersion config.Version, //nolint:interfacer
	crdAndWebhookVersion string,
) {
	var err error

	By(fmt.Sprintf("init go/v2 project with project version %s", projectVersion))
	err = kbc.Init(
		"--plugins", fmt.Sprintf("go/%s", pluginVersion),
		"--project-version", projectVersion.String(),
		"--domain", kbc.Domain,
		"--fetch-deps=false",
	)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())

	var crdVersionOptions, webhookVersionOptions []string
	if crdAndWebhookVersion != defaultCRDAndWebhookVersions[pluginVersion] {
		if pluginVersion.Compare(pluginV2) == 0 {
			panic(fmt.Errorf("go/v2 didn't support %s crd and webhook versions", crdAndWebhookVersion))
		}

		crdVersionOptions = append(crdVersionOptions, "--crd-version", crdAndWebhookVersion)
		webhookVersionOptions = append(webhookVersionOptions, "--webhook-version", crdAndWebhookVersion)

		if pluginVersion.Compare(pluginV3) == 0 {
			// Users have to manually add "crdVersions={non-default-version}" to their Makefile
			// if using a non-default CRD version.
			makefilePath := filepath.Join(kbc.Dir, "Makefile")
			bs, err := ioutil.ReadFile(makefilePath)
			ExpectWithOffset(1, err).NotTo(HaveOccurred())
			content, err := utils.EnsureExistAndReplace(
				string(bs),
				`CRD_OPTIONS ?= "crd:trivialVersions=true,preserveUnknownFields=false"`,
				fmt.Sprintf(`CRD_OPTIONS ?= "crd:crdVersions={%s},trivialVersions=true,preserveUnknownFields=false"`,
					crdAndWebhookVersion),
			)
			ExpectWithOffset(1, err).NotTo(HaveOccurred())
			ExpectWithOffset(1, ioutil.WriteFile(makefilePath, []byte(content), 0600)).To(Succeed())
		}
	}

	By("creating API definition")
	err = kbc.CreateAPI(append([]string{
		"--group", kbc.Group,
		"--version", kbc.Version,
		"--kind", kbc.Kind,
		"--namespaced",
		"--resource",
		"--controller",
		"--make=false",
	}, crdVersionOptions...)...)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())

	By("implementing the API")
	ExpectWithOffset(1, utils.InsertCode(
		filepath.Join(kbc.Dir, "api", kbc.Version, fmt.Sprintf("%s_types.go", strings.ToLower(kbc.Kind))),
		fmt.Sprintf(`type %sSpec struct {
`, kbc.Kind),
		`	// +optional
	Count int `+"`"+`json:"count,omitempty"`+"`"+`
`)).Should(Succeed())

	By("scaffolding mutating and validating webhooks")
	err = kbc.CreateWebhook(append([]string{
		"--group", kbc.Group,
		"--version", kbc.Version,
		"--kind", kbc.Kind,
		"--defaulting",
		"--programmatic-validation",
	}, webhookVersionOptions...)...)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())

	By("implementing the mutating and validating webhooks")
	err = utils.ImplementWebhooks(filepath.Join(
		kbc.Dir, "api", kbc.Version,
		fmt.Sprintf("%s_webhook.go", strings.ToLower(kbc.Kind))))
	ExpectWithOffset(1, err).NotTo(HaveOccurred())

	By("uncomment kustomization.yaml to enable webhook and ca injection")
	ExpectWithOffset(1, utils.UncommentCode(
		filepath.Join(kbc.Dir, "config", "default", "kustomization.yaml"),
		"#- ../webhook", "#")).To(Succeed())
	ExpectWithOffset(1, utils.UncommentCode(
		filepath.Join(kbc.Dir, "config", "default", "kustomization.yaml"),
		"#- ../certmanager", "#")).To(Succeed())
	ExpectWithOffset(1, utils.UncommentCode(
		filepath.Join(kbc.Dir, "config", "default", "kustomization.yaml"),
		"#- ../prometheus", "#")).To(Succeed())
	ExpectWithOffset(1, utils.UncommentCode(
		filepath.Join(kbc.Dir, "config", "default", "kustomization.yaml"),
		"#- manager_webhook_patch.yaml", "#")).To(Succeed())
	ExpectWithOffset(1, utils.UncommentCode(
		filepath.Join(kbc.Dir, "config", "default", "kustomization.yaml"),
		"#- webhookcainjection_patch.yaml", "#")).To(Succeed())
	ExpectWithOffset(1, utils.UncommentCode(
		filepath.Join(kbc.Dir, "config", "default", "kustomization.yaml"),
		fmt.Sprintf(`#- name: CERTIFICATE_NAMESPACE # namespace of the certificate CR
#  objref:
#    kind: Certificate
#    group: cert-manager.io
#    version: %[1]s
#    name: serving-cert # this name should match the one in certificate.yaml
#  fieldref:
#    fieldpath: metadata.namespace
#- name: CERTIFICATE_NAME
#  objref:
#    kind: Certificate
#    group: cert-manager.io
#    version: %[1]s
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
#    name: webhook-service`, certManagerVersion[pluginVersion]), "#")).To(Succeed())
}
