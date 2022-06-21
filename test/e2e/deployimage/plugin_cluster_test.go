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
	"time"

	"sigs.k8s.io/kubebuilder/v3/pkg/plugin/util"

	//nolint:golint
	//nolint:revive
	. "github.com/onsi/ginkgo"

	//nolint:golint
	//nolint:revive
	. "github.com/onsi/gomega"

	"sigs.k8s.io/kubebuilder/v3/test/e2e/utils"
)

var _ = Describe("kubebuilder", func() {
	Context("deploy image plugin 3", func() {
		var (
			kbc *utils.TestContext
		)

		BeforeEach(func() {
			var err error
			kbc, err = utils.NewTestContext(util.KubebuilderBinName, "GO111MODULE=on")
			Expect(err).NotTo(HaveOccurred())
			Expect(kbc.Prepare()).To(Succeed())
		})

		AfterEach(func() {
			By("clean up API objects created during the test")
			kbc.CleanupManifests(filepath.Join("config", "default"))

			By("removing controller image and working dir")
			kbc.Destroy()
		})

		It("should generate a runnable project go/v3 with v1 CRDs", func() {
			// Skip if cluster version < 1.16, when v1 CRDs and webhooks did not exist.
			if srvVer := kbc.K8sVersion.ServerVersion; srvVer.GetMajorInt() <= 1 && srvVer.GetMinorInt() < 16 {
				Skip(fmt.Sprintf("cluster version %s does not support v1 CRDs or webhooks",
					srvVer.GitVersion))
			}

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
				"--image-container-port=", "11211",
				"--image-container-command=", "memcached -m=64 modern -v",
			)
			ExpectWithOffset(1, err).NotTo(HaveOccurred())

			Run(kbc)
		})
	})
})

// Run runs a set of e2e tests for a scaffolded project defined by a TestContext.
func Run(kbc *utils.TestContext) {
	var controllerPodName string
	var err error

	By("updating the go.mod")
	err = kbc.Tidy()
	ExpectWithOffset(1, err).NotTo(HaveOccurred())

	By("building the controller image")
	err = kbc.Make("docker-build", "IMG="+kbc.ImageName)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())

	By("loading the controller docker image into the kind cluster")
	err = kbc.LoadImageToKindCluster()
	ExpectWithOffset(1, err).NotTo(HaveOccurred())

	// NOTE: If you want to run the test against a GKE cluster, you will need to grant yourself permission.
	// Otherwise, you may see "... is forbidden: attempt to grant extra privileges"
	// $ kubectl create clusterrolebinding myname-cluster-admin-binding \
	// --clusterrole=cluster-admin --user=myname@mycompany.com
	// https://cloud.google.com/kubernetes-engine/docs/how-to/role-based-access-control
	By("deploying the controller-manager")
	err = kbc.Make("deploy", "IMG="+kbc.ImageName)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())

	By("validating that the controller-manager pod is running as expected")
	verifyControllerUp := func() error {
		// Get pod name
		podOutput, err := kbc.Kubectl.Get(
			true,
			"pods", "-l", "control-plane=controller-manager",
			"-o", "go-template={{ range .items }}{{ if not .metadata.deletionTimestamp }}{{ .metadata.name }}"+
				"{{ \"\\n\" }}{{ end }}{{ end }}")
		ExpectWithOffset(2, err).NotTo(HaveOccurred())
		podNames := util.GetNonEmptyLines(podOutput)
		if len(podNames) != 1 {
			return fmt.Errorf("expect 1 controller pods running, but got %d", len(podNames))
		}
		controllerPodName = podNames[0]
		ExpectWithOffset(2, controllerPodName).Should(ContainSubstring("controller-manager"))

		// Validate pod status
		status, err := kbc.Kubectl.Get(
			true,
			"pods", controllerPodName, "-o", "jsonpath={.status.phase}")
		ExpectWithOffset(2, err).NotTo(HaveOccurred())
		if status != "Running" {
			return fmt.Errorf("controller pod in %s status", status)
		}
		return nil
	}
	defer func() {
		out, err := kbc.Kubectl.CommandInNamespace("describe", "all")
		ExpectWithOffset(1, err).NotTo(HaveOccurred())
		fmt.Fprintln(GinkgoWriter, out)
	}()
	EventuallyWithOffset(1, verifyControllerUp, time.Minute, time.Second).Should(Succeed())

	By("creating an instance of the CR")
	// currently controller-runtime doesn't provide a readiness probe, we retry a few times
	// we can change it to probe the readiness endpoint after CR supports it.
	sampleFile := filepath.Join("config", "samples",
		fmt.Sprintf("%s_%s_%s.yaml", kbc.Group, kbc.Version, strings.ToLower(kbc.Kind)))

	sampleFilePath, err := filepath.Abs(filepath.Join(fmt.Sprintf("e2e-%s", kbc.TestSuffix), sampleFile))
	Expect(err).To(Not(HaveOccurred()))
	EventuallyWithOffset(1, func() error {
		_, err = kbc.Kubectl.Apply(true, "-f", sampleFilePath)
		return err
	}, time.Minute, time.Second).Should(Succeed())

	By("applying the CRD Editor Role")
	crdEditorRole := filepath.Join("config", "rbac",
		fmt.Sprintf("%s_editor_role.yaml", strings.ToLower(kbc.Kind)))
	EventuallyWithOffset(1, func() error {
		_, err = kbc.Kubectl.Apply(true, "-f", crdEditorRole)
		return err
	}, time.Minute, time.Second).Should(Succeed())

	By("applying the CRD Viewer Role")
	crdViewerRole := filepath.Join("config", "rbac", fmt.Sprintf("%s_viewer_role.yaml", strings.ToLower(kbc.Kind)))
	EventuallyWithOffset(1, func() error {
		_, err = kbc.Kubectl.Apply(true, "-f", crdViewerRole)
		return err
	}, time.Minute, time.Second).Should(Succeed())

	//TODO: Add test to check if the deployment with the memcached was create successfully
}
