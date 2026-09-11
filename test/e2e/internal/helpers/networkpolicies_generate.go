/*
Copyright 2026 The Kubernetes Authors.

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

package helpers

import (
	"fmt"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck
	. "github.com/onsi/gomega"    //nolint:staticcheck

	pluginutil "sigs.k8s.io/kubebuilder/v4/pkg/plugin/util"
	"sigs.k8s.io/kubebuilder/v4/test/e2e/utils"
)

// Project generators used only by the NetworkPolicy specs in feature_networkpolicy_test.go.

// GenerateV4WithNetworkPoliciesWithoutMetrics implements a go/v4 plugin project with webhooks
// and network policies, but without exposing metrics. The Helm chart then renders only the
// webhook policy, while kustomize still applies both; the manager must stay isolated either way.
func GenerateV4WithNetworkPoliciesWithoutMetrics(kbc *utils.TestContext) {
	GenerateV4WithoutMetrics(kbc)
	enableNetworkPolicies(kbc)
}

func enableNetworkPolicies(kbc *utils.TestContext) {
	By("uncommenting kustomization.yaml to enable network policies")
	ExpectWithOffset(1, pluginutil.UncommentCode(
		filepath.Join(kbc.Dir, "config", "default", "kustomization.yaml"),
		"#- ../network-policy", "#")).To(Succeed())
}

// GenerateV4WithNetworkPoliciesWithoutWebhooks implements a go/v4 plugin project with metrics
// and network policies, but without webhooks.
func GenerateV4WithNetworkPoliciesWithoutWebhooks(kbc *utils.TestContext) {
	initingTheProject(kbc)
	creatingAPI(kbc)

	ExpectWithOffset(1, pluginutil.UncommentCode(
		filepath.Join(kbc.Dir, "config", "default", "kustomization.yaml"),
		"#- ../prometheus", "#")).To(Succeed())
	ExpectWithOffset(1, pluginutil.UncommentCode(
		filepath.Join(kbc.Dir, "config", "default", "kustomization.yaml"),
		metricsTarget, "#")).To(Succeed())
	enableNetworkPolicies(kbc)
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
		"--make=false",
	)
	Expect(err).NotTo(HaveOccurred(), "Failed to scaffolding mutating webhook")

	By("implementing the mutating and validating webhooks")
	webhookFilePath := filepath.Join(
		kbc.Dir, "internal/webhook", kbc.Version,
		fmt.Sprintf("%s_webhook.go", strings.ToLower(kbc.Kind)))
	err = utils.ImplementWebhooks(webhookFilePath, strings.ToLower(kbc.Kind))
	Expect(err).NotTo(HaveOccurred(), "Failed to implement webhooks")

	scaffoldConversionWebhook(kbc)
	ExpectWithOffset(1, pluginutil.UncommentCode(
		filepath.Join(kbc.Dir, "config", "default", "kustomization.yaml"),
		"#- ../prometheus", "#")).To(Succeed())
	ExpectWithOffset(1, pluginutil.UncommentCode(
		filepath.Join(kbc.Dir, "config", "default", "kustomization.yaml"),
		metricsTarget, "#")).To(Succeed())
	ExpectWithOffset(1, pluginutil.UncommentCode(
		filepath.Join(kbc.Dir, "config", "default", "kustomization.yaml"),
		metricsCertPatch, "#")).To(Succeed())
	ExpectWithOffset(1, pluginutil.UncommentCode(
		filepath.Join(kbc.Dir, "config", "default", "kustomization.yaml"),
		metricsCertReplaces, "#")).To(Succeed())
	ExpectWithOffset(1, pluginutil.UncommentCode(
		filepath.Join(kbc.Dir, "config", "prometheus", "kustomization.yaml"),
		monitorTLSPatch, "#")).To(Succeed())

	enableNetworkPolicies(kbc)
}

// EnableWebhookNamespaceGating scopes the admission webhooks to namespaces labeled
// 'webhook: enabled'. It uses the controller-tools patch marker, so `make manifests`
// emits the namespaceSelector into the generated webhook configurations.
func EnableWebhookNamespaceGating(kbc *utils.TestContext) {
	By("adding the patch marker so the webhook configurations only admit resources from labeled namespaces")
	const namespaceSelectorPatch = ",admissionReviewVersions=v1," +
		"patch=`{\"namespaceSelector\":{\"matchLabels\":{\"webhook\":\"enabled\"}}}`"
	webhookFilePath := filepath.Join(
		kbc.Dir, "internal/webhook", kbc.Version,
		fmt.Sprintf("%s_webhook.go", strings.ToLower(kbc.Kind)))
	ExpectWithOffset(1, pluginutil.ReplaceInFile(
		webhookFilePath, ",admissionReviewVersions=v1", namespaceSelectorPatch)).To(Succeed())
}
