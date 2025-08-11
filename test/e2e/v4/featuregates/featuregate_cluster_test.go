/*
Copyright 2025 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed t		It("should respect --feature-gates flag in deployment", func() {
			By("initializing a project with feature gates")
			err := kbc.Init(
				"--plugins", "go.kubebuilder.io/v4",
				"--domain", kbc.Domain,
				"--with-feature-gates",
			)
			Expect(err).NotTo(HaveOccurred())ting, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package featuregates

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"sigs.k8s.io/kubebuilder/v4/pkg/plugin/util"
	"sigs.k8s.io/kubebuilder/v4/test/e2e/utils"
)

var _ = Describe("Feature Gates Cluster E2E", func() {
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
		By("cleaning up created resources")
		_ = kbc.Make("undeploy")
		_ = kbc.Make("uninstall")
		kbc.Destroy()
	})

	Context("when deploying a controller with feature gates", func() {
		It("should validate feature gate behavior in cluster logs", func() {
			By("initializing a project with feature gates")
			err := kbc.Init(
				"--plugins", "go.kubebuilder.io/v4",
				"--domain", kbc.Domain,
				"--with-feature-gates",
			)
			Expect(err).NotTo(HaveOccurred())

			By("creating API with controller")
			err = kbc.CreateAPI(
				"--group", kbc.Group,
				"--version", kbc.Version,
				"--kind", kbc.Kind,
				"--resource", "--controller",
				"--make=false",
			)
			Expect(err).NotTo(HaveOccurred())

			By("adding feature gate markers to the API types")
			typesFile := filepath.Join(kbc.Dir, "api", kbc.Version, strings.ToLower(kbc.Kind)+"_types.go")

			// Add a feature gate field to the API
			experimentalField := `
	// ExperimentalField is a feature-gated field for testing
	// +feature-gate experimental-field
	// +optional
	ExperimentalField *string ` + "`" + `json:"experimentalField,omitempty"` + "`"

			err = util.InsertCode(
				typesFile,
				"Foo *string `json:\"foo,omitempty\"`",
				experimentalField,
			)
			Expect(err).NotTo(HaveOccurred())

			By("regenerating the project to discover feature gates")
			err = kbc.CreateAPI(
				"--group", kbc.Group,
				"--version", kbc.Version,
				"--kind", kbc.Kind,
				"--resource", "--controller",
				"--force",
				"--make=false",
			)
			Expect(err).NotTo(HaveOccurred())

			By("adding feature gate logging to the controller")
			controllerFile := filepath.Join(kbc.Dir, "internal", "controller", strings.ToLower(kbc.Kind)+"_controller.go")

			// Add feature gate validation logging
			featureGateLogic := `
	// Feature gate validation and logging
	if featureGates.IsEnabled("experimental-field") {
		log.Info("experimental feature gate enabled", "feature", "experimental-field")
		if obj.Spec.ExperimentalField != nil {
			log.Info("experimental field found", "value", *obj.Spec.ExperimentalField)
		}
	} else {
		log.Info("experimental feature gate disabled", "feature", "experimental-field")
	}`

			err = util.InsertCode(
				controllerFile,
				"// TODO(user): your logic here.",
				featureGateLogic,
			)
			Expect(err).NotTo(HaveOccurred())

			By("adding feature gates import to controller")
			err = util.InsertCode(
				controllerFile,
				`"sigs.k8s.io/controller-runtime/pkg/reconcile"`,
				`
	"sigs.k8s.io/kubebuilder/v4/internal/featuregates"`,
			)
			Expect(err).NotTo(HaveOccurred())

			By("updating controller to use global feature gates")
			err = util.ReplaceInFile(
				controllerFile,
				"func (r *"+kbc.Kind+"Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {",
				`func (r *`+kbc.Kind+`Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// Get the object
	var obj `+strings.ToLower(kbc.Group)+kbc.Version+`.`+kbc.Kind+`
	if err := r.Get(ctx, req.NamespacedName, &obj); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Get global feature gates (this should be passed from main.go in real implementation)
	featureGates := featuregates.FeatureGates{
		"experimental-field": true, // For testing, enable the feature gate
	}`,
			)
			Expect(err).NotTo(HaveOccurred())

			By("building the project")
			err = kbc.Make("all")
			Expect(err).NotTo(HaveOccurred())

			By("building the controller image")
			err = kbc.Make("docker-build", "IMG="+kbc.ImageName)
			Expect(err).NotTo(HaveOccurred())

			By("loading the controller docker image into the kind cluster")
			err = kbc.LoadImageToKindCluster()
			Expect(err).NotTo(HaveOccurred())

			By("deploying the controller-manager with feature gates enabled")
			cmd := exec.Command("make", "deploy", "IMG="+kbc.ImageName)
			_, err = kbc.Run(cmd)
			Expect(err).NotTo(HaveOccurred())

			By("creating a sample CR to trigger reconciliation")
			sampleFile := filepath.Join("config", "samples", 
				strings.ToLower(kbc.Group)+"_"+kbc.Version+"_"+strings.ToLower(kbc.Kind)+".yaml")
			
			// Update sample to include feature-gated field
			sampleContent := fmt.Sprintf(`apiVersion: %s/%s
kind: %s
metadata:
  labels:
    app.kubernetes.io/name: %s
    app.kubernetes.io/managed-by: kustomize
  name: %s-sample
  namespace: %s-system
spec:
  foo: "test"
  experimentalField: "feature-gate-test-value"
`, kbc.Group+"."+kbc.Domain, kbc.Version, kbc.Kind, 
   strings.ToLower(kbc.Kind), strings.ToLower(kbc.Kind), strings.ToLower(kbc.Kind))

			err = os.WriteFile(sampleFile, []byte(sampleContent), 0644)
			Expect(err).NotTo(HaveOccurred())

			By("applying the sample CR")
			EventuallyWithOffset(1, func() error {
				cmd := exec.Command("kubectl", "apply", "-f", sampleFile)
				_, err := kbc.Run(cmd)
				return err
			}, time.Minute, time.Second).Should(Succeed())

			By("waiting for controller pod to be ready")
			EventuallyWithOffset(1, func() error {
				cmd := exec.Command("kubectl", "get", "pods", "-n", kbc.Kind+"-system", 
					"-l", "control-plane=controller-manager", "--field-selector=status.phase=Running")
				out, err := kbc.Run(cmd)
				if err != nil {
					return err
				}
				if !strings.Contains(string(out), "Running") {
					return fmt.Errorf("controller pod not ready yet")
				}
				return nil
			}, 2*time.Minute, 5*time.Second).Should(Succeed())

			By("checking controller logs for feature gate messages")
			EventuallyWithOffset(1, func() error {
				// Get controller pod name
				cmd := exec.Command("kubectl", "get", "pods", "-n", kbc.Kind+"-system",
					"-l", "control-plane=controller-manager", "-o", "jsonpath={.items[0].metadata.name}")
				out, err := kbc.Run(cmd)
				if err != nil {
					return err
				}
				podName := strings.TrimSpace(string(out))

				// Get logs
				cmd = exec.Command("kubectl", "logs", "-n", kbc.Kind+"-system", podName, "-c", "manager")
				out, err = kbc.Run(cmd)
				if err != nil {
					return err
				}
				logs := string(out)

				// Validate expected log messages
				if !strings.Contains(logs, "experimental feature gate enabled") {
					return fmt.Errorf("expected 'experimental feature gate enabled' not found in logs")
				}

				if !strings.Contains(logs, "experimental field found") {
					return fmt.Errorf("expected 'experimental field found' not found in logs")
				}

				if !strings.Contains(logs, "feature-gate-test-value") {
					return fmt.Errorf("expected feature gate test value not found in logs")
				}

				return nil
			}, 3*time.Minute, 10*time.Second).Should(Succeed())

			By("testing with feature gate disabled")
			// Update the deployment to disable feature gates
			err = util.ReplaceInFile(
				controllerFile,
				`"experimental-field": true, // For testing, enable the feature gate`,
				`"experimental-field": false, // For testing, disable the feature gate`,
			)
			Expect(err).NotTo(HaveOccurred())

			By("rebuilding and redeploying with disabled feature gate")
			err = kbc.Make("docker-build", "IMG="+kbc.ImageName)
			Expect(err).NotTo(HaveOccurred())

			err = kbc.LoadImageToKindCluster()
			Expect(err).NotTo(HaveOccurred())

			cmd = exec.Command("make", "deploy", "IMG="+kbc.ImageName)
			_, err = kbc.Run(cmd)
			Expect(err).NotTo(HaveOccurred())

			By("triggering reconciliation again")
			// Delete and recreate the CR to trigger reconciliation
			cmd = exec.Command("kubectl", "delete", "-f", sampleFile)
			kbc.Run(cmd) // Ignore errors

			EventuallyWithOffset(1, func() error {
				cmd := exec.Command("kubectl", "apply", "-f", sampleFile)
				_, err := kbc.Run(cmd)
				return err
			}, time.Minute, time.Second).Should(Succeed())

			By("checking logs for disabled feature gate message")
			EventuallyWithOffset(1, func() error {
				// Get controller pod name
				cmd := exec.Command("kubectl", "get", "pods", "-n", kbc.Kind+"-system",
					"-l", "control-plane=controller-manager", "-o", "jsonpath={.items[0].metadata.name}")
				out, err := kbc.Run(cmd)
				if err != nil {
					return err
				}
				podName := strings.TrimSpace(string(out))

				// Get recent logs
				cmd = exec.Command("kubectl", "logs", "-n", kbc.Kind+"-system", podName, "-c", "manager", "--tail=50")
				out, err = kbc.Run(cmd)
				if err != nil {
					return err
				}
				logs := string(out)

				// Validate disabled feature gate message
				if !strings.Contains(logs, "experimental feature gate disabled") {
					return fmt.Errorf("expected 'experimental feature gate disabled' not found in recent logs")
				}

				return nil
			}, 2*time.Minute, 10*time.Second).Should(Succeed())
		})
	})

	Context("when deploying with feature gates via command line", func() {
		It("should respect --feature-gates flag in deployment", func() {
			By("initializing a project with feature gates")
			err := kbc.Init(
				"--plugins", "go.kubebuilder.io/v4",
				"--domain", kbc.Domain,
				"--with-feature-gates",
			)
			Expect(err).NotTo(HaveOccurred())

			By("creating API with controller")
			err = kbc.CreateAPI(
				"--group", kbc.Group,
				"--version", kbc.Version,
				"--kind", kbc.Kind,
				"--resource", "--controller",
				"--make=false",
			)
			Expect(err).NotTo(HaveOccurred())

			By("updating main.go to log feature gate initialization")
			mainFile := filepath.Join(kbc.Dir, "cmd", "main.go")
			
			// Add logging after feature gate parsing
			err = util.InsertCode(
				mainFile,
				"setupLog.Error(err, \"invalid feature gates\")",
				`
	setupLog.Info("Feature gates initialized", "gates", parsedFeatureGates.String())`,
			)
			Expect(err).NotTo(HaveOccurred())

			By("updating manager deployment to include feature gates flag")
			managerFile := filepath.Join("config", "manager", "manager.yaml")
			
			err = util.ReplaceInFile(
				managerFile,
				"- --leader-elect",
				`- --leader-elect
        - --feature-gates=experimental-gate=true`,
			)
			Expect(err).NotTo(HaveOccurred())

			By("building and deploying the controller")
			err = kbc.Make("all")
			Expect(err).NotTo(HaveOccurred())

			err = kbc.Make("docker-build", "IMG="+kbc.ImageName)
			Expect(err).NotTo(HaveOccurred())

			err = kbc.LoadImageToKindCluster()
			Expect(err).NotTo(HaveOccurred())

			cmd := exec.Command("make", "deploy", "IMG="+kbc.ImageName)
			_, err = kbc.Run(cmd)
			Expect(err).NotTo(HaveOccurred())

			By("checking that feature gates are initialized in pod logs")
			EventuallyWithOffset(1, func() error {
				// Get controller pod name
				cmd := exec.Command("kubectl", "get", "pods", "-n", kbc.Kind+"-system",
					"-l", "control-plane=controller-manager", "-o", "jsonpath={.items[0].metadata.name}")
				out, err := kbc.Run(cmd)
				if err != nil {
					return err
				}
				podName := strings.TrimSpace(string(out))

				// Get logs
				cmd = exec.Command("kubectl", "logs", "-n", kbc.Kind+"-system", podName, "-c", "manager")
				out, err = kbc.Run(cmd)
				if err != nil {
					return err
				}
				logs := string(out)

				// Validate feature gate initialization log
				if !strings.Contains(logs, "Feature gates initialized") {
					return fmt.Errorf("expected 'Feature gates initialized' not found in logs")
				}

				if !strings.Contains(logs, "experimental-gate") {
					return fmt.Errorf("expected 'experimental-gate' not found in logs")
				}

				return nil
			}, 2*time.Minute, 10*time.Second).Should(Succeed())
		})
	})
})
