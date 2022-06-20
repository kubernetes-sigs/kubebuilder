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

package deployimage

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
