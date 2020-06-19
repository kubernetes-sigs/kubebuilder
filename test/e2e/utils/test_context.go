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

	. "github.com/onsi/ginkgo" //nolint:golint
)

const certmanagerVersion = "v0.11.0"
const prometheusOperatorVersion = "0.33"
const certmanagerURL = "https://github.com/jetstack/cert-manager/releases/download/%s/cert-manager.yaml"
const prometheusOperatorURL = "https://raw.githubusercontent.com/coreos/prometheus-operator/release-%s/bundle.yaml"

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
	Kubectl    *Kubectl
	BinaryName string
}

// NewTestContext init with a random suffix for test TestContext stuff,
// to avoid conflict when running tests synchronously.
func NewTestContext(binaryName string, env ...string) (*TestContext, error) {
	testSuffix, err := RandomSuffix()
	if err != nil {
		return nil, err
	}

	testGroup := "bar" + testSuffix
	path, err := filepath.Abs("e2e-" + testSuffix)
	if err != nil {
		return nil, err
	}

	cc := &CmdContext{
		Env: env,
		Dir: path,
	}

	return &TestContext{
		TestSuffix: testSuffix,
		Domain:     "example.com" + testSuffix,
		Group:      testGroup,
		Version:    "v1alpha1",
		Kind:       "Foo" + testSuffix,
		Resources:  "foo" + testSuffix + "s",
		ImageName:  "e2e-test/controller-manager:" + testSuffix,
		CmdContext: cc,
		Kubectl: &Kubectl{
			Namespace:  fmt.Sprintf("e2e-%s-system", testSuffix),
			CmdContext: cc,
		},
		BinaryName: binaryName,
	}, nil
}

// Prepare prepare a work directory for testing
func (t *TestContext) Prepare() error {
	fmt.Fprintf(GinkgoWriter, "preparing testing directory: %s\n", t.Dir)
	return os.MkdirAll(t.Dir, 0755)
}

// InstallCertManager installs the cert manager bundle.
func (t *TestContext) InstallCertManager() error {
	if _, err := t.Kubectl.Command("create", "namespace", "cert-manager"); err != nil {
		return err
	}
	url := fmt.Sprintf(certmanagerURL, certmanagerVersion)
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

// InstallPrometheusOperManager installs the prometheus manager bundle.
func (t *TestContext) InstallPrometheusOperManager() error {
	url := fmt.Sprintf(prometheusOperatorURL, prometheusOperatorVersion)
	_, err := t.Kubectl.Apply(false, "-f", url)
	return err
}

// UninstallPrometheusOperManager uninstalls the prometheus manager bundle.
func (t *TestContext) UninstallPrometheusOperManager() {
	url := fmt.Sprintf(prometheusOperatorURL, prometheusOperatorVersion)
	if _, err := t.Kubectl.Delete(false, "-f", url); err != nil {
		fmt.Fprintf(GinkgoWriter, "error when running kubectl delete during cleaning up prometheus bundle: %v\n", err)
	}
}

// UninstallCertManager uninstalls the cert manager bundle.
func (t *TestContext) UninstallCertManager() {
	url := fmt.Sprintf(certmanagerURL, certmanagerVersion)
	if _, err := t.Kubectl.Delete(false, "-f", url); err != nil {
		fmt.Fprintf(GinkgoWriter,
			"warning: error when running kubectl delete during cleaning up cert manager: %v\n", err)
	}
	if _, err := t.Kubectl.Delete(false, "namespace", "cert-manager"); err != nil {
		fmt.Fprintf(GinkgoWriter, "warning: error when cleaning up the cert manager namespace: %v\n", err)
	}
}

// CleanupManifests is a helper func to run kustomize build and pipe the output to kubectl delete -f -
func (t *TestContext) CleanupManifests(dir string) {
	cmd := exec.Command("kustomize", "build", dir)
	output, err := t.Run(cmd)
	if err != nil {
		fmt.Fprintf(GinkgoWriter, "warning: error when running kustomize build: %v\n", err)
	}
	if _, err := t.Kubectl.WithInput(string(output)).Command("delete", "-f", "-"); err != nil {
		fmt.Fprintf(GinkgoWriter, "warning: error when running kubectl delete -f -: %v\n", err)
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

// Destroy is for cleaning up the docker images for testing
func (t *TestContext) Destroy() {
	//nolint:gosec
	cmd := exec.Command("docker", "rmi", "-f", t.ImageName)
	if _, err := t.Run(cmd); err != nil {
		fmt.Fprintf(GinkgoWriter, "warning: error when removing the local image: %v\n", err)
	}
	if err := os.RemoveAll(t.Dir); err != nil {
		fmt.Fprintf(GinkgoWriter, "warning: error when removing the word dir: %v\n", err)
	}
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
		return output, fmt.Errorf("%s failed with error: %s", command, string(output))
	}

	return output, nil
}
