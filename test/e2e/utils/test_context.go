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
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"sigs.k8s.io/kubebuilder/v3/pkg/plugin/util"

	. "github.com/onsi/ginkgo" //nolint:golint,revive
)

// TestContext specified to run e2e tests
type TestContext struct {
	*CmdContext
	TestSuffix string
	Domain     string
	Group      string
	Version    string
	Kind       string
	Resources  string
	ImageName  string
	BinaryName string
	Kubectl    *Kubectl
	K8sVersion *KubernetesVersion
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
	fmt.Fprintf(GinkgoWriter, "warning: %v\n", err)
}

// Prepare prepares the test environment.
func (t *TestContext) Prepare() error {
	// Remove tools used by projects in the environment so the correct version is downloaded for each test.
	fmt.Fprintln(GinkgoWriter, "cleaning up tools")
	for _, toolName := range []string{"controller-gen", "kustomize"} {
		if toolPath, err := exec.LookPath(toolName); err == nil {
			if err := os.RemoveAll(toolPath); err != nil {
				return err
			}
		}
	}

	fmt.Fprintf(GinkgoWriter, "preparing testing directory: %s\n", t.Dir)
	return os.MkdirAll(t.Dir, 0755)
}

const (
	certmanagerVersionWithv1beta2CRs = "v0.11.0"
	certmanagerLegacyVersion         = "v1.0.4"
	certmanagerVersion               = "v1.5.3"

	certmanagerURLTmplLegacy = "https://github.com/jetstack/cert-manager/releases/download/%s/cert-manager-legacy.yaml"
	certmanagerURLTmpl       = "https://github.com/jetstack/cert-manager/releases/download/%s/cert-manager.yaml"
)

// makeCertManagerURL returns a kubectl-able URL for the cert-manager bundle.
func (t *TestContext) makeCertManagerURL(hasv1beta1CRs bool) string {
	// Return a URL for the manifest bundle with v1beta1 CRs.
	if hasv1beta1CRs {
		return fmt.Sprintf(certmanagerURLTmpl, certmanagerVersionWithv1beta2CRs)
	}

	// Determine which URL to use for a manifest bundle with v1 CRs.
	// The most up-to-date bundle uses v1 CRDs, which were introduced in k8s v1.16.
	if ver := t.K8sVersion.ServerVersion; ver.GetMajorInt() <= 1 && ver.GetMinorInt() < 16 {
		return fmt.Sprintf(certmanagerURLTmplLegacy, certmanagerLegacyVersion)
	}
	return fmt.Sprintf(certmanagerURLTmpl, certmanagerVersion)
}

// InstallCertManager installs the cert manager bundle. If hasv1beta1CRs is true,
// the legacy version (which uses v1alpha2 CRs) is installed.
func (t *TestContext) InstallCertManager(hasv1beta1CRs bool) error {
	url := t.makeCertManagerURL(hasv1beta1CRs)
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

// UninstallCertManager uninstalls the cert manager bundle. If hasv1beta1CRs is true,
// the legacy version (which uses v1alpha2 CRs) is installed.
func (t *TestContext) UninstallCertManager(hasv1beta1CRs bool) {
	url := t.makeCertManagerURL(hasv1beta1CRs)
	if _, err := t.Kubectl.Delete(false, "-f", url); err != nil {
		warnError(err)
	}
}

const (
	prometheusOperatorLegacyVersion = "0.33"
	prometheusOperatorLegacyURL     = "https://raw.githubusercontent.com/coreos/prometheus-operator/release-%s/bundle.yaml"
	prometheusOperatorVersion       = "0.51"
	prometheusOperatorURL           = "https://raw.githubusercontent.com/prometheus-operator/" +
		"prometheus-operator/release-%s/bundle.yaml"
)

// InstallPrometheusOperManager installs the prometheus manager bundle.
func (t *TestContext) InstallPrometheusOperManager() error {
	var url string
	if ver := t.K8sVersion.ServerVersion; ver.GetMajorInt() <= 1 && ver.GetMinorInt() < 16 {
		url = fmt.Sprintf(prometheusOperatorLegacyURL, prometheusOperatorLegacyVersion)
	} else {
		url = fmt.Sprintf(prometheusOperatorURL, prometheusOperatorVersion)
	}
	_, err := t.Kubectl.Apply(false, "-f", url)
	return err
}

// UninstallPrometheusOperManager uninstalls the prometheus manager bundle.
func (t *TestContext) UninstallPrometheusOperManager() {
	var url string
	if ver := t.K8sVersion.ServerVersion; ver.GetMajorInt() <= 1 && ver.GetMinorInt() < 16 {
		url = fmt.Sprintf(prometheusOperatorLegacyURL, prometheusOperatorLegacyVersion)
	} else {
		url = fmt.Sprintf(prometheusOperatorURL, prometheusOperatorVersion)
	}
	if _, err := t.Kubectl.Delete(false, "-f", url); err != nil {
		warnError(err)
	}
}

// CleanupManifests is a helper func to run kustomize build and pipe the output to kubectl delete -f -
func (t *TestContext) CleanupManifests(dir string) {
	kustomizePath := filepath.Join(t.Dir, "bin", "kustomize")
	if _, err := os.Stat(kustomizePath); err != nil {
		// Just fail below with an error about kustomize not being installed globally.
		kustomizePath = "kustomize"
	}
	cmd := exec.Command(kustomizePath, "build", dir)
	output, err := t.Run(cmd)
	if err != nil {
		warnError(err)
	}
	if _, err := t.Kubectl.WithInput(string(output)).Command("delete", "-f", "-"); err != nil {
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
	cmd := exec.Command("docker", "rmi", "-f", t.ImageName)
	if _, err := t.Run(cmd); err != nil {
		warnError(err)
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

// LabelAllNamespacesToWarnAboutRestricted will label all namespaces so that we can verify
// if a warning with `Warning: would violate PodSecurity` will be raised when the manifests are applied
func (t *TestContext) LabelAllNamespacesToWarnAboutRestricted() error {
	_, err := t.Kubectl.Command("label", "--overwrite", "ns", "--all",
		"pod-security.kubernetes.io/audit=restricted",
		"pod-security.kubernetes.io/enforce-version=v1.24",
		"pod-security.kubernetes.io/warn=restricted")
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
func (tc TestContext) LoadImageToKindClusterWithName(image string) error {
	cmd := exec.Command("docker", "pull", image)
	_, err := tc.Run(cmd)
	if err != nil {
		return err
	}
	cluster := "kind"
	if v, ok := os.LookupEnv("KIND_CLUSTER"); ok {
		cluster = v
	}
	kindOptions := []string{"load", "docker-image", "--name", cluster, image}
	cmd = exec.Command("kind", kindOptions...)
	_, err = tc.Run(cmd)
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
	fmt.Fprintf(GinkgoWriter, "running: %s\n", command)
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
	projectBytes, err := ioutil.ReadFile(filepath.Join(t.Dir, "PROJECT"))
	if err != nil {
		return err
	}

	projectBytes = append([]byte(multiGroup), projectBytes...)
	err = ioutil.WriteFile(filepath.Join(t.Dir, "PROJECT"), projectBytes, 0644)
	if err != nil {
		return err
	}
	return nil
}
