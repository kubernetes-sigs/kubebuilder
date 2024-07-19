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

package utils

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugin/util"

	. "github.com/onsi/ginkgo/v2"
)

const (
	certmanagerVersion        = "v1.14.4"
	certmanagerURLTmpl        = "https://github.com/cert-manager/cert-manager/releases/download/%s/cert-manager.yaml"
	prometheusOperatorVersion = "0.51"
	prometheusOperatorURL     = "https://raw.githubusercontent.com/prometheus-operator/" +
		"prometheus-operator/release-%s/bundle.yaml"
)

// TestContext specified to run e2e tests
type TestContext struct {
	*CmdContext
	TestSuffix   string
	Domain       string
	Group        string
	Version      string
	Kind         string
	Resources    string
	ImageName    string
	BinaryName   string
	Kubectl      *Kubectl
	K8sVersion   *KubernetesVersion
	IsRestricted bool
}

// NewTestContext init with a random suffix for test TestContext stuff,
// to avoid conflict when running tests synchronously.
func NewTestContext(binaryName string, env ...string) (*TestContext, error) {
	testSuffix, err := util.RandomSuffix()
	if err != nil {
		return nil, err
	}

	cc := &CmdContext{
		Env: env,
	}

	// Use kubectl to get Kubernetes client and cluster version.
	kubectl := &Kubectl{
		Namespace:      fmt.Sprintf("e2e-%s-system", testSuffix),
		ServiceAccount: fmt.Sprintf("e2e-%s-controller-manager", testSuffix),
		CmdContext:     cc,
	}
	k8sVersion, err := kubectl.Version()
	if err != nil {
		return nil, err
	}

	// Set CmdContext.Dir after running Kubectl.Version() because dir does not exist yet.
	if cc.Dir, err = filepath.Abs("e2e-" + testSuffix); err != nil {
		return nil, err
	}

	return &TestContext{
		TestSuffix: testSuffix,
		Domain:     "example.com" + testSuffix,
		Group:      "bar" + testSuffix,
		Version:    "v1alpha1",
		Kind:       "Foo" + testSuffix,
		Resources:  "foo" + testSuffix + "s",
		ImageName:  "e2e-test/controller-manager:" + testSuffix,
		CmdContext: cc,
		Kubectl:    kubectl,
		K8sVersion: &k8sVersion,
		BinaryName: binaryName,
	}, nil
}

func warnError(err error) {
	_, _ = fmt.Fprintf(GinkgoWriter, "warning: %v\n", err)
}

// Prepare prepares the test environment.
func (t *TestContext) Prepare() error {
	// Remove tools used by projects in the environment so the correct version is downloaded for each test.
	_, _ = fmt.Fprintln(GinkgoWriter, "cleaning up tools")
	for _, toolName := range []string{"controller-gen", "kustomize"} {
		if toolPath, err := exec.LookPath(toolName); err == nil {
			if err := os.RemoveAll(toolPath); err != nil {
				return err
			}
		}
	}

	_, _ = fmt.Fprintf(GinkgoWriter, "preparing testing directory: %s\n", t.Dir)
	return os.MkdirAll(t.Dir, 0o755)
}

// makeCertManagerURL returns a kubectl-able URL for the cert-manager bundle.
func (t *TestContext) makeCertManagerURL() string {
	return fmt.Sprintf(certmanagerURLTmpl, certmanagerVersion)
}

func (t *TestContext) makePrometheusOperatorURL() string {
	return fmt.Sprintf(prometheusOperatorURL, prometheusOperatorVersion)
}

// InstallCertManager installs the cert manager bundle. If hasv1beta1CRs is true,
// the legacy version (which uses v1alpha2 CRs) is installed.
func (t *TestContext) InstallCertManager() error {
	url := t.makeCertManagerURL()
	if _, err := t.Kubectl.Apply(false, "-f", url, "--validate=false"); err != nil {
		return err
	}
	// Wait for cert-manager-webhook to be ready, which can take time if cert-manager
	// was re-installed after uninstalling on a cluster.
	_, err := t.Kubectl.Wait(false, "deployment.apps/cert-manager-webhook",
		"--for", "condition=Available",
		"--namespace", "cert-manager",
		"--timeout", "5m",
	)
	return err
}

// UninstallCertManager uninstalls the cert manager bundle.
func (t *TestContext) UninstallCertManager() {
	url := t.makeCertManagerURL()
	if _, err := t.Kubectl.Delete(false, "-f", url); err != nil {
		warnError(err)
	}
}

// InstallPrometheusOperManager installs the prometheus manager bundle.
func (t *TestContext) InstallPrometheusOperManager() error {
	url := t.makePrometheusOperatorURL()
	_, err := t.Kubectl.Apply(false, "-f", url)
	return err
}

// UninstallPrometheusOperManager uninstalls the prometheus manager bundle.
func (t *TestContext) UninstallPrometheusOperManager() {
	url := t.makePrometheusOperatorURL()
	if _, err := t.Kubectl.Delete(false, "-f", url); err != nil {
		warnError(err)
	}
}

// Init is for running `kubebuilder init`
func (t *TestContext) Init(initOptions ...string) error {
	initOptions = append([]string{"init"}, initOptions...)
	//nolint:gosec
	cmd := exec.Command(t.BinaryName, initOptions...)
	_, err := t.Run(cmd)
	return err
}

// Edit is for running `kubebuilder edit`
func (t *TestContext) Edit(editOptions ...string) error {
	editOptions = append([]string{"edit"}, editOptions...)
	//nolint:gosec
	cmd := exec.Command(t.BinaryName, editOptions...)
	_, err := t.Run(cmd)
	return err
}

// CreateAPI is for running `kubebuilder create api`
func (t *TestContext) CreateAPI(resourceOptions ...string) error {
	resourceOptions = append([]string{"create", "api"}, resourceOptions...)
	//nolint:gosec
	cmd := exec.Command(t.BinaryName, resourceOptions...)
	_, err := t.Run(cmd)
	return err
}

// CreateWebhook is for running `kubebuilder create webhook`
func (t *TestContext) CreateWebhook(resourceOptions ...string) error {
	resourceOptions = append([]string{"create", "webhook"}, resourceOptions...)
	//nolint:gosec
	cmd := exec.Command(t.BinaryName, resourceOptions...)
	_, err := t.Run(cmd)
	return err
}

// Regenerate is for running `kubebuilder alpha generate`
func (t *TestContext) Regenerate(resourceOptions ...string) error {
	resourceOptions = append([]string{"alpha", "generate"}, resourceOptions...)
	//nolint:gosec
	cmd := exec.Command(t.BinaryName, resourceOptions...)
	_, err := t.Run(cmd)
	return err
}

// Make is for running `make` with various targets
func (t *TestContext) Make(makeOptions ...string) error {
	cmd := exec.Command("make", makeOptions...)
	_, err := t.Run(cmd)
	return err
}

// Tidy runs `go mod tidy` so that go 1.16 build doesn't fail.
// See https://blog.golang.org/go116-module-changes#TOC_3.
func (t *TestContext) Tidy() error {
	cmd := exec.Command("go", "mod", "tidy")
	_, err := t.Run(cmd)
	return err
}

// Destroy is for cleaning up the docker images for testing
func (t *TestContext) Destroy() {
	//nolint:gosec
	// if image name is not present or not provided skip execution of docker command
	if t.ImageName != "" {
		// Check white space from image name
		if len(strings.TrimSpace(t.ImageName)) == 0 {
			log.Println("Image not set, skip cleaning up of docker image")
		} else {
			cmd := exec.Command("docker", "rmi", "-f", t.ImageName)
			if _, err := t.Run(cmd); err != nil {
				warnError(err)
			}
		}

	}
	if err := os.RemoveAll(t.Dir); err != nil {
		warnError(err)
	}
}

// CreateManagerNamespace will create the namespace where the manager is deployed
func (t *TestContext) CreateManagerNamespace() error {
	_, err := t.Kubectl.Command("create", "ns", t.Kubectl.Namespace)
	return err
}

// LabelNamespacesToWarnAboutRestricted will label all namespaces so that we can verify
// if a warning with `Warning: would violate PodSecurity` will be raised when the manifests are applied
func (t *TestContext) LabelNamespacesToWarnAboutRestricted() error {
	_, err := t.Kubectl.Command("label", "--overwrite", "ns", t.Kubectl.Namespace,
		"pod-security.kubernetes.io/warn=restricted")
	return err
}

// RemoveNamespaceLabelToWarnAboutRestricted will remove the `pod-security.kubernetes.io/warn` label
// from the specified namespace
func (t *TestContext) RemoveNamespaceLabelToWarnAboutRestricted() error {
	_, err := t.Kubectl.Command("label", "ns", t.Kubectl.Namespace, "pod-security.kubernetes.io/warn-")
	return err
}

// LoadImageToKindCluster loads a local docker image to the kind cluster
func (t *TestContext) LoadImageToKindCluster() error {
	cluster := "kind"
	if v, ok := os.LookupEnv("KIND_CLUSTER"); ok {
		cluster = v
	}
	kindOptions := []string{"load", "docker-image", t.ImageName, "--name", cluster}
	cmd := exec.Command("kind", kindOptions...)
	_, err := t.Run(cmd)
	return err
}

// LoadImageToKindClusterWithName loads a local docker image with the name informed to the kind cluster
func (t TestContext) LoadImageToKindClusterWithName(image string) error {
	cluster := "kind"
	if v, ok := os.LookupEnv("KIND_CLUSTER"); ok {
		cluster = v
	}
	kindOptions := []string{"load", "docker-image", "--name", cluster, image}
	cmd := exec.Command("kind", kindOptions...)
	_, err := t.Run(cmd)
	return err
}

// CmdContext provides context for command execution
type CmdContext struct {
	// environment variables in k=v format.
	Env   []string
	Dir   string
	Stdin io.Reader
}

// Run executes the provided command within this context
func (cc *CmdContext) Run(cmd *exec.Cmd) ([]byte, error) {
	cmd.Dir = cc.Dir
	cmd.Env = append(os.Environ(), cc.Env...)
	cmd.Stdin = cc.Stdin
	command := strings.Join(cmd.Args, " ")
	_, _ = fmt.Fprintf(GinkgoWriter, "running: %s\n", command)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return output, fmt.Errorf("%s failed with error: (%v) %s", command, err, string(output))
	}

	return output, nil
}

// AllowProjectBeMultiGroup will update the PROJECT file with the information to allow we scaffold
// apis with different groups. be available.
func (t *TestContext) AllowProjectBeMultiGroup() error {
	const multiGroup = `multigroup: true
`
	projectBytes, err := os.ReadFile(filepath.Join(t.Dir, "PROJECT"))
	if err != nil {
		return err
	}

	projectBytes = append([]byte(multiGroup), projectBytes...)
	err = os.WriteFile(filepath.Join(t.Dir, "PROJECT"), projectBytes, 0o644)
	if err != nil {
		return err
	}
	return nil
}
