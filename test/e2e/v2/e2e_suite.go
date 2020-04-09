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

package v2

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	. "github.com/onsi/ginkgo" //nolint:golint
	. "github.com/onsi/gomega" //nolint:golint

	"sigs.k8s.io/kubebuilder/test/e2e/utils"
)

var _ = Describe("kubebuilder", func() {
	Context("with v2 scaffolding", func() {
		var kbc *utils.KBTestContext
		BeforeEach(func() {
			var err error
			kbc, err = utils.TestContext("GO111MODULE=on")
			Expect(err).NotTo(HaveOccurred())
			Expect(kbc.Prepare()).To(Succeed())

			By("installing cert manager bundle")
			Expect(kbc.InstallCertManager()).To(Succeed())

			By("installing prometheus operator")
			Expect(kbc.InstallPrometheusOperManager()).To(Succeed())
		})

		AfterEach(func() {
			By("clean up created API objects during test process")
			kbc.CleanupManifests(filepath.Join("config", "default"))

			By("uninstalling prometheus manager bundle")
			kbc.UninstallPrometheusOperManager()

			By("uninstalling cert manager bundle")
			kbc.UninstallCertManager()

			By("remove container image and work dir")
			kbc.Destroy()
		})

		It("should generate a runnable project", func() {
			var controllerPodName string
			By("init v2 project")
			err := kbc.Init(
				"--project-version", "2",
				"--domain", kbc.Domain,
				"--dep=false")
			Expect(err).Should(Succeed())

			By("creating api definition")
			err = kbc.CreateAPI(
				"--group", kbc.Group,
				"--version", kbc.Version,
				"--kind", kbc.Kind,
				"--namespaced",
				"--resource",
				"--controller",
				"--make=false")
			Expect(err).Should(Succeed())

			By("implementing the API")
			Expect(utils.InsertCode(
				filepath.Join(kbc.Dir, "api", kbc.Version, fmt.Sprintf("%s_types.go", strings.ToLower(kbc.Kind))),
				fmt.Sprintf(`type %sSpec struct {
`, kbc.Kind),
				`	// +optional
	Count int `+"`"+`json:"count,omitempty"`+"`"+`
`)).Should(Succeed())

			By("scaffolding mutating and validating webhook")
			err = kbc.CreateWebhook(
				"--group", kbc.Group,
				"--version", kbc.Version,
				"--kind", kbc.Kind,
				"--defaulting",
				"--programmatic-validation")
			Expect(err).Should(Succeed())

			By("implementing the mutating and validating webhooks")
			err = implementWebhooks(filepath.Join(
				kbc.Dir, "api", kbc.Version,
				fmt.Sprintf("%s_webhook.go", strings.ToLower(kbc.Kind))))
			Expect(err).Should(Succeed())

			By("uncomment kustomization.yaml to enable webhook and ca injection")
			Expect(utils.UncommentCode(
				filepath.Join(kbc.Dir, "config", "default", "kustomization.yaml"),
				"#- ../webhook", "#")).To(Succeed())
			Expect(utils.UncommentCode(
				filepath.Join(kbc.Dir, "config", "default", "kustomization.yaml"),
				"#- ../certmanager", "#")).To(Succeed())
			Expect(utils.UncommentCode(
				filepath.Join(kbc.Dir, "config", "default", "kustomization.yaml"),
				"#- ../prometheus", "#")).To(Succeed())
			Expect(utils.UncommentCode(
				filepath.Join(kbc.Dir, "config", "default", "kustomization.yaml"),
				"#- manager_webhook_patch.yaml", "#")).To(Succeed())
			Expect(utils.UncommentCode(
				filepath.Join(kbc.Dir, "config", "default", "kustomization.yaml"),
				"#- webhookcainjection_patch.yaml", "#")).To(Succeed())
			Expect(utils.UncommentCode(filepath.Join(kbc.Dir, "config", "default", "kustomization.yaml"),
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

			By("building image")
			err = kbc.Make("docker-build", "IMG="+kbc.ImageName)
			Expect(err).Should(Succeed())

			By("loading docker image into kind cluster")
			err = kbc.LoadImageToKindCluster()
			Expect(err).Should(Succeed())

			// NOTE: If you want to run the test against a GKE cluster, you will need to grant yourself permission.
			// Otherwise, you may see "... is forbidden: attempt to grant extra privileges"
			// $ kubectl create clusterrolebinding myname-cluster-admin-binding \
			// --clusterrole=cluster-admin --user=myname@mycompany.com
			// https://cloud.google.com/kubernetes-engine/docs/how-to/role-based-access-control
			By("deploying controller manager")
			err = kbc.Make("deploy", "IMG="+kbc.ImageName)
			Expect(err).Should(Succeed())

			By("validate the controller-manager pod running as expected")
			verifyControllerUp := func() error {
				// Get pod name
				podOutput, err := kbc.Kubectl.Get(
					true,
					"pods", "-l", "control-plane=controller-manager",
					"-o", "go-template={{ range .items }}{{ if not .metadata.deletionTimestamp }}{{ .metadata.name }}"+
						"{{ \"\\n\" }}{{ end }}{{ end }}")
				Expect(err).NotTo(HaveOccurred())
				podNames := utils.GetNonEmptyLines(podOutput)
				if len(podNames) != 1 {
					return fmt.Errorf("expect 1 controller pods running, but got %d", len(podNames))
				}
				controllerPodName = podNames[0]
				Expect(controllerPodName).Should(ContainSubstring("controller-manager"))

				// Validate pod status
				status, err := kbc.Kubectl.Get(
					true,
					"pods", controllerPodName, "-o", "jsonpath={.status.phase}")
				Expect(err).NotTo(HaveOccurred())
				if status != "Running" {
					return fmt.Errorf("controller pod in %s status", status)
				}
				return nil
			}
			Eventually(verifyControllerUp, time.Minute, time.Second).Should(Succeed())

			By("granting permissions to access the metrics and read the token")
			_, err = kbc.Kubectl.Command(
				"create", "clusterrolebinding", fmt.Sprintf("metrics-%s", kbc.TestSuffix),
				fmt.Sprintf("--clusterrole=e2e-%s-metrics-reader", kbc.TestSuffix),
				fmt.Sprintf("--serviceaccount=%s:default", kbc.Kubectl.Namespace))
			Expect(err).NotTo(HaveOccurred())

			b64Token, err := kbc.Kubectl.Get(true, "secrets", "-o=jsonpath={.items[0].data.token}")
			Expect(err).NotTo(HaveOccurred())
			token, err := base64.StdEncoding.DecodeString(strings.TrimSpace(b64Token))
			Expect(err).NotTo(HaveOccurred())
			Expect(len(token)).To(BeNumerically(">", 0))

			By("creating a pod with curl image")
			cmdOpts := []string{
				"run", "--generator=run-pod/v1", "curl", "--image=curlimages/curl:7.68.0", "--restart=OnFailure", "--",
				"curl", "-v", "-k", "-H", fmt.Sprintf(`Authorization: Bearer %s`, token),
				fmt.Sprintf("https://e2e-%v-controller-manager-metrics-service.e2e-%v-system.svc:8443/metrics",
					kbc.TestSuffix, kbc.TestSuffix),
			}
			_, err = kbc.Kubectl.CommandInNamespace(cmdOpts...)
			Expect(err).NotTo(HaveOccurred())

			By("validating the curl pod running as expected")
			verifyCurlUp := func() error {
				// Validate pod status
				status, err := kbc.Kubectl.Get(
					true,
					"pods", "curl", "-o", "jsonpath={.status.phase}")
				Expect(err).NotTo(HaveOccurred())
				if status != "Completed" && status != "Succeeded" {
					return fmt.Errorf("curl pod in %s status", status)
				}
				return nil
			}
			Eventually(verifyCurlUp, 30*time.Second, time.Second).Should(Succeed())

			By("validating the metrics endpoint is serving as expected")
			getCurlLogs := func() string {
				logOutput, err := kbc.Kubectl.Logs("curl")
				Expect(err).NotTo(HaveOccurred())
				return logOutput
			}
			Eventually(getCurlLogs, 10*time.Second, time.Second).Should(ContainSubstring("< HTTP/2 200"))

			By("validate cert manager has provisioned the certificate secret")
			Eventually(func() error {
				_, err := kbc.Kubectl.Get(
					true,
					"secrets", "webhook-server-cert")
				return err
			}, time.Minute, time.Second).Should(Succeed())

			By("validate prometheus manager has provisioned the Service")
			Eventually(func() error {
				_, err := kbc.Kubectl.Get(
					false,
					"Service", "prometheus-operator")
				return err
			}, time.Minute, time.Second).Should(Succeed())

			By("validate Service Monitor for Prometheus is applied in the namespace")
			_, err = kbc.Kubectl.Get(
				true,
				"ServiceMonitor")
			Expect(err).NotTo(HaveOccurred())

			By("validate the mutating|validating webhooks have the CA injected")
			verifyCAInjection := func() error {
				mwhOutput, err := kbc.Kubectl.Get(
					false,
					"mutatingwebhookconfigurations.admissionregistration.k8s.io",
					fmt.Sprintf("e2e-%s-mutating-webhook-configuration", kbc.TestSuffix),
					"-o", "go-template={{ range .webhooks }}{{ .clientConfig.caBundle }}{{ end }}")
				Expect(err).NotTo(HaveOccurred())
				// sanity check that ca should be long enough, because there may be a place holder "\n"
				Expect(len(mwhOutput)).To(BeNumerically(">", 10))

				vwhOutput, err := kbc.Kubectl.Get(
					false,
					"validatingwebhookconfigurations.admissionregistration.k8s.io",
					fmt.Sprintf("e2e-%s-validating-webhook-configuration", kbc.TestSuffix),
					"-o", "go-template={{ range .webhooks }}{{ .clientConfig.caBundle }}{{ end }}")
				Expect(err).NotTo(HaveOccurred())
				// sanity check that ca should be long enough, because there may be a place holder "\n"
				Expect(len(vwhOutput)).To(BeNumerically(">", 10))

				return nil
			}
			Eventually(verifyCAInjection, time.Minute, time.Second).Should(Succeed())

			By("creating an instance of CR")
			// currently controller-runtime doesn't provide a readiness probe, we retry a few times
			// we can change it to probe the readiness endpoint after CR supports it.
			sampleFile := filepath.Join("config", "samples",
				fmt.Sprintf("%s_%s_%s.yaml", kbc.Group, kbc.Version, strings.ToLower(kbc.Kind)))
			Eventually(func() error {
				_, err = kbc.Kubectl.Apply(true, "-f", sampleFile)
				return err
			}, time.Minute, time.Second).Should(Succeed())

			By("applying CRD Editor Role")
			crdEditorRole := filepath.Join("config", "rbac",
				fmt.Sprintf("%s_editor_role.yaml", strings.ToLower(kbc.Kind)))
			Eventually(func() error {
				_, err = kbc.Kubectl.Apply(true, "-f", crdEditorRole)
				return err
			}, time.Minute, time.Second).Should(Succeed())

			By("applying CRD Viewer Role")
			crdViewerRole := filepath.Join("config", "rbac", fmt.Sprintf("%s_viewer_role.yaml", strings.ToLower(kbc.Kind)))
			Eventually(func() error {
				_, err = kbc.Kubectl.Apply(true, "-f", crdViewerRole)
				return err
			}, time.Minute, time.Second).Should(Succeed())

			By("validate the created resource object gets reconciled in controller")
			managerContainerLogs := func() string {
				logOutput, err := kbc.Kubectl.Logs(controllerPodName, "-c", "manager")
				Expect(err).NotTo(HaveOccurred())
				return logOutput
			}
			Eventually(managerContainerLogs, time.Minute, time.Second).Should(ContainSubstring("Successfully Reconciled"))

			By("validate mutating and validating webhooks are working fine")
			cnt, err := kbc.Kubectl.Get(
				true,
				"-f", sampleFile,
				"-o", "go-template={{ .spec.count }}")
			Expect(err).NotTo(HaveOccurred())
			count, err := strconv.Atoi(cnt)
			Expect(err).NotTo(HaveOccurred())
			Expect(count).To(BeNumerically("==", 5))
		})
	})
})

func implementWebhooks(filename string) error {
	bs, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}
	str := string(bs)

	str, err = ensureExistAndReplace(
		str,
		"import (",
		`import (
	"errors"`)
	if err != nil {
		return err
	}

	// implement defaulting webhook logic
	str, err = ensureExistAndReplace(
		str,
		"// TODO(user): fill in your defaulting logic.",
		`if r.Spec.Count == 0 {
		r.Spec.Count = 5
	}`)
	if err != nil {
		return err
	}

	// implement validation webhook logic
	str, err = ensureExistAndReplace(
		str,
		"// TODO(user): fill in your validation logic upon object creation.",
		`if r.Spec.Count < 0 {
		return errors.New(".spec.count must >= 0")
	}`)
	if err != nil {
		return err
	}
	str, err = ensureExistAndReplace(
		str,
		"// TODO(user): fill in your validation logic upon object update.",
		`if r.Spec.Count < 0 {
		return errors.New(".spec.count must >= 0")
	}`)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(filename, []byte(str), 0644)
}

func ensureExistAndReplace(input, match, replace string) (string, error) {
	if !strings.Contains(input, match) {
		return "", fmt.Errorf("can't find %q", match)
	}
	return strings.Replace(input, match, replace, -1), nil
}
