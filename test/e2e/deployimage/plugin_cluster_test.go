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
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"sigs.k8s.io/kubebuilder/v3/pkg/plugin/util"

	//nolint:golint
	//nolint:revive
	. "github.com/onsi/ginkgo/v2"

	//nolint:golint
	//nolint:revive
	. "github.com/onsi/gomega"

	"sigs.k8s.io/kubebuilder/v3/test/e2e/utils"
)

var _ = Describe("kubebuilder", func() {
	Context("deploy image plugin", func() {
		var kbc *utils.TestContext

		BeforeEach(func() {
			var err error
			kbc, err = utils.NewTestContext(util.KubebuilderBinName, "GO111MODULE=on")
			Expect(err).NotTo(HaveOccurred())
			Expect(kbc.Prepare()).To(Succeed())

			By("installing prometheus operator")
			Expect(kbc.InstallPrometheusOperManager()).To(Succeed())

			By("creating manager namespace")
			err = kbc.CreateManagerNamespace()
			ExpectWithOffset(1, err).NotTo(HaveOccurred())

			By("labeling all namespaces to warn about restricted")
			err = kbc.LabelAllNamespacesToWarnAboutRestricted()
			ExpectWithOffset(1, err).NotTo(HaveOccurred())

			By("enforce that namespace where the sample will be applied can only run restricted containers")
			_, err = kbc.Kubectl.Command("label", "--overwrite", "ns", kbc.Kubectl.Namespace,
				"pod-security.kubernetes.io/audit=restricted",
				"pod-security.kubernetes.io/enforce-version=v1.24",
				"pod-security.kubernetes.io/enforce=restricted")
			Expect(err).To(Not(HaveOccurred()))
		})

		AfterEach(func() {
			By("clean up API objects created during the test")
			kbc.CleanupManifests(filepath.Join("config", "default"))

			By("uninstalling the Prometheus manager bundle")
			kbc.UninstallPrometheusOperManager()

			By("removing controller image and working dir")
			kbc.Destroy()
		})

		It("should generate a runnable project with deploy-image/v1-alpha options ", func() {
			var err error

			By("initializing a project with go/v3")
			err = kbc.Init(
				"--plugins", "go/v4",
				"--project-version", "3",
				"--domain", kbc.Domain,
			)
			ExpectWithOffset(1, err).NotTo(HaveOccurred())

			By("creating API definition with deploy-image/v1-alpha plugin")
			err = kbc.CreateAPI(
				"--group", kbc.Group,
				"--version", kbc.Version,
				"--kind", kbc.Kind,
				"--plugins", "deploy-image/v1-alpha",
				"--image", "memcached:1.4.36-alpine",
				"--image-container-port", "11211",
				"--image-container-command", "memcached,-m=64,-o,modern,-v",
				"--run-as-user", "1001",
				"--make=false",
				"--manifests=false",
			)
			ExpectWithOffset(1, err).NotTo(HaveOccurred())

			By("uncomment kustomization.yaml to enable prometheus")
			ExpectWithOffset(1, util.UncommentCode(
				filepath.Join(kbc.Dir, "config", "default", "kustomization.yaml"),
				"#- ../prometheus", "#")).To(Succeed())

			By("uncomment kustomize files to ensure that pods are restricted")
			uncommentPodStandards(kbc)

			Run(kbc)
		})

		It("should generate a runnable project with deploy-image/v1-alpha without options ", func() {
			var err error

			By("initializing a project with go/v4")
			err = kbc.Init(
				"--plugins", "go/v4",
				"--project-version", "3",
				"--domain", kbc.Domain,
			)
			ExpectWithOffset(1, err).NotTo(HaveOccurred())

			By("creating API definition with deploy-image/v1-alpha plugin")
			err = kbc.CreateAPI(
				"--group", kbc.Group,
				"--version", kbc.Version,
				"--kind", kbc.Kind,
				"--plugins", "deploy-image/v1-alpha",
				"--image", "busybox:1.28",
				"--make=false",
				"--manifests=false",
			)
			ExpectWithOffset(1, err).NotTo(HaveOccurred())

			By("uncomment kustomization.yaml to enable prometheus")
			ExpectWithOffset(1, util.UncommentCode(
				filepath.Join(kbc.Dir, "config", "default", "kustomization.yaml"),
				"#- ../prometheus", "#")).To(Succeed())

			By("uncomment kustomize files to ensure that pods are restricted")
			uncommentPodStandards(kbc)

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

	By("run make manifests")
	err = kbc.Make("manifests")
	ExpectWithOffset(1, err).NotTo(HaveOccurred())

	By("run make generate")
	err = kbc.Make("generate")
	ExpectWithOffset(1, err).NotTo(HaveOccurred())

	By("run make all")
	err = kbc.Make("all")
	ExpectWithOffset(1, err).NotTo(HaveOccurred())

	By("run make install")
	err = kbc.Make("install")
	ExpectWithOffset(1, err).NotTo(HaveOccurred())

	By("building the controller image")
	err = kbc.Make("docker-build", "IMG="+kbc.ImageName)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())

	By("loading the controller docker image into the kind cluster")
	err = kbc.LoadImageToKindCluster()
	ExpectWithOffset(1, err).NotTo(HaveOccurred())

	By("deploying the controller-manager")
	cmd := exec.Command("make", "deploy", "IMG="+kbc.ImageName)
	outputMake, err := kbc.Run(cmd)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())

	By("validating that manager Pod/container(s) are restricted")
	ExpectWithOffset(1, outputMake).NotTo(ContainSubstring("Warning: would violate PodSecurity"))

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
	sampleFile := filepath.Join("config", "samples",
		fmt.Sprintf("%s_%s_%s.yaml", kbc.Group, kbc.Version, strings.ToLower(kbc.Kind)))

	sampleFilePath, err := filepath.Abs(filepath.Join(fmt.Sprintf("e2e-%s", kbc.TestSuffix), sampleFile))
	Expect(err).To(Not(HaveOccurred()))

	EventuallyWithOffset(1, func() error {
		_, err = kbc.Kubectl.Apply(true, "-f", sampleFilePath)
		return err
	}, time.Minute, time.Second).Should(Succeed())

	By("validating that pod(s) status.phase=Running")
	getMemcachedPodStatus := func() error {
		status, err := kbc.Kubectl.Get(true, "pods", "-l",
			fmt.Sprintf("app.kubernetes.io/name=%s", kbc.Kind),
			"-o", "jsonpath={.items[*].status}",
		)
		ExpectWithOffset(2, err).NotTo(HaveOccurred())
		if !strings.Contains(status, "\"phase\":\"Running\"") {
			return err
		}
		return nil
	}
	EventuallyWithOffset(1, getMemcachedPodStatus, time.Minute, time.Second).Should(Succeed())

	By("validating that the status of the custom resource created is updated or not")
	var status string
	getStatus := func() error {
		status, err = kbc.Kubectl.Get(true, strings.ToLower(kbc.Kind),
			strings.ToLower(kbc.Kind)+"-sample",
			"-o", "jsonpath={.status.conditions}")
		ExpectWithOffset(2, err).NotTo(HaveOccurred())
		if !strings.Contains(status, "Available") {
			return errors.New(`status condition with type "Available" should be set`)
		}
		return nil
	}
	Eventually(getStatus, time.Minute, time.Second).Should(Succeed())

	// Testing the finalizer
	EventuallyWithOffset(1, func() error {
		_, err = kbc.Kubectl.Delete(true, "-f", sampleFilePath)
		return err
	}, time.Minute, time.Second).Should(Succeed())

	EventuallyWithOffset(1, func() error {
		events, err := kbc.Kubectl.Get(true, "events", "--field-selector=type=Warning",
			"-o", "jsonpath={.items[*].message}",
		)
		ExpectWithOffset(2, err).NotTo(HaveOccurred())
		if !strings.Contains(events, "is being deleted from the namespace") {
			return err
		}
		return nil
	}, time.Minute, time.Second).Should(Succeed())
}

func uncommentPodStandards(kbc *utils.TestContext) {
	configManager := filepath.Join(kbc.Dir, "config", "manager", "manager.yaml")

	//nolint:lll
	if err := util.ReplaceInFile(configManager, `# TODO(user): For common cases that do not require escalating privileges
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
