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

package v1

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	. "github.com/onsi/ginkgo" //nolint:golint
	. "github.com/onsi/gomega" //nolint:golint

	"sigs.k8s.io/kubebuilder/v2/test/e2e/utils"
)

var _ = Describe("kubebuilder", func() {
	Context("with v1 scaffolding", func() {
		var kbc *utils.KBTestContext
		BeforeEach(func() {
			var err error
			kbc, err = utils.TestContext("GO111MODULE=off")
			Expect(err).NotTo(HaveOccurred())
			Expect(kbc.Prepare()).To(Succeed())
		})

		AfterEach(func() {
			By("clean up created API objects during test process")
			kbc.CleanupManifests(filepath.Join("config", "default"))
			if _, err := kbc.Kubectl.Command(
				"delete", "--recursive",
				"-f", filepath.Join("config", "crds"),
			); err != nil {
				fmt.Fprintf(GinkgoWriter, "error when running kubectl delete during cleaning up crd: %v\n", err)
			}

			By("remove container image and work dir")
			kbc.Destroy()
		})

		It("should generate a runnable project", func() {
			// prepare v1 vendor
			By("downloading the vendor tarball")
			cmd := exec.Command("wget",
				"https://storage.googleapis.com/kubebuilder-vendor/vendor.v1.tgz",
				"-O", "/tmp/vendor.v1.tgz")
			_, err := kbc.Run(cmd)
			Expect(err).Should(Succeed())

			By("untar the vendor tarball")
			cmd = exec.Command("tar", "-zxf", "/tmp/vendor.v1.tgz")
			_, err = kbc.Run(cmd)
			Expect(err).Should(Succeed())

			var controllerPodName string

			By("init v1 project")
			err = kbc.Init(
				"--project-version", "1",
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

			By("creating core-type resource controller")
			err = kbc.CreateAPI(
				"--group", "apps",
				"--version", "v1",
				"--kind", "Deployment",
				"--namespaced",
				"--resource=false",
				"--controller",
				"--make=false")
			Expect(err).Should(Succeed())

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
			err = kbc.Make("deploy")
			Expect(err).Should(Succeed())

			By("validate the controller-manager pod running as expected")
			verifyControllerUp := func() error {
				// Get pod name
				podOutput, err := kbc.Kubectl.Get(
					true,
					"pods", "-l", "control-plane=controller-manager",
					"-o", "go-template={{ range .items }}{{ if not .metadata.deletionTimestamp }}{{ .metadata.name }}"+
						"{{ \"\\n\" }}{{ end }}{{ end }}",
				)
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
					"pods", controllerPodName, "-o", "jsonpath={.status.phase}",
				)
				Expect(err).NotTo(HaveOccurred())
				if status != "Running" {
					return fmt.Errorf("controller pod in %s status", status)
				}

				return nil
			}
			Eventually(verifyControllerUp, 2*time.Minute, time.Second).Should(Succeed())

			By("creating an instance of CR")
			inputFile := filepath.Join("config", "samples",
				fmt.Sprintf("%s_%s_%s.yaml", kbc.Group, kbc.Version, strings.ToLower(kbc.Kind)))
			_, err = kbc.Kubectl.Apply(false, "-f", inputFile)
			Expect(err).NotTo(HaveOccurred())

			By("validate the created resource object gets reconciled in controller")
			controllerContainerLogs := func() string {
				// Check container log to validate that the created resource object gets reconciled in controller
				logOutput, err := kbc.Kubectl.Logs(controllerPodName, "-c", "manager")
				Expect(err).NotTo(HaveOccurred())

				return logOutput
			}
			Eventually(controllerContainerLogs, 2*time.Minute, time.Second).Should(ContainSubstring("Updating"))
		})
	})
})
