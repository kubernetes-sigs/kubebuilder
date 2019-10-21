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
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo"
)

const certmanagerVersion = "v0.11.0"
const prometheusOperatorVersion = "0.33"

// KBTestContext specified to run e2e tests
type KBTestContext struct {
	*CmdContext
	TestSuffix string
	Domain     string
	Group      string
	Version    string
	Kind       string
	Resources  string
	ImageName  string
	Kubectl    *Kubectl
}

// TestContext init with a random suffix for test KBTestContext stuff,
// to avoid conflict when running tests synchronously.
func TestContext(env ...string) (*KBTestContext, error) {
	testSuffix, err := randomSuffix()
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

	return &KBTestContext{
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
	}, nil
}

// Prepare prepare a work directory for testing
func (kc *KBTestContext) Prepare() error {
	fmt.Fprintf(GinkgoWriter, "preparing testing directory: %s\n", kc.Dir)
	return os.MkdirAll(kc.Dir, 0755)
}

// InstallCertManager installs the cert manager bundle.
func (kc *KBTestContext) InstallCertManager() error {
	if _, err := kc.Kubectl.Command("create", "namespace", "cert-manager"); err != nil {
		return err
	}
	_, err := kc.Kubectl.Apply(false, "-f", fmt.Sprintf("https://github.com/jetstack/cert-manager/releases/download/%s/cert-manager.yaml", certmanagerVersion), "--validate=false")
	return err
}

// InstallPrometheusOperManager installs the prometheus manager bundle.
func (kc *KBTestContext) InstallPrometheusOperManager() error {
	_, err := kc.Kubectl.Apply(false, "-f", fmt.Sprintf("https://raw.githubusercontent.com/coreos/prometheus-operator/release-%s/bundle.yaml", prometheusOperatorVersion))
	return err
}

// UninstallPrometheusOperManager uninstalls the prometheus manager bundle.
func (kc *KBTestContext) UninstallPrometheusOperManager() {
	if _, err := kc.Kubectl.Delete(false, "-f", fmt.Sprintf("https://github.com/coreos/prometheus-operator/blob/release-%s/bundle.yaml", prometheusOperatorVersion)); err != nil {
		fmt.Fprintf(GinkgoWriter, "error when running kubectl delete during cleaning up prometheus bundle: %v\n", err)
	}
}

// UninstallCertManager uninstalls the cert manager bundle.
func (kc *KBTestContext) UninstallCertManager() {
	if _, err := kc.Kubectl.Delete(false, "-f", fmt.Sprintf("https://github.com/jetstack/cert-manager/releases/download/%s/cert-manager.yaml", certmanagerVersion)); err != nil {
		fmt.Fprintf(GinkgoWriter, "warning: error when running kubectl delete during cleaning up cert manager: %v\n", err)
	}
	if _, err := kc.Kubectl.Delete(false, "namespace", "cert-manager"); err != nil {
		fmt.Fprintf(GinkgoWriter, "warning: error when cleaning up the cert manager namespace: %v\n", err)
	}
}

// CleanupManifests is a helper func to run kustomize build and pipe the output to kubectl delete -f -
func (kc *KBTestContext) CleanupManifests(dir string) {
	cmd := exec.Command("kustomize", "build", dir)
	output, err := kc.Run(cmd)
	if err != nil {
		fmt.Fprintf(GinkgoWriter, "warning: error when running kustomize build: %v\n", err)
	}
	if _, err := kc.Kubectl.CommandWithInput(string(output), "delete", "-f", "-"); err != nil {
		fmt.Fprintf(GinkgoWriter, "warning: error when running kubectl delete -f -: %v\n", err)
	}
}

// Init is for running `kubebuilder init`
func (kc *KBTestContext) Init(initOptions ...string) error {
	initOptions = append([]string{"init"}, initOptions...)
	cmd := exec.Command("kubebuilder", initOptions...)
	_, err := kc.Run(cmd)
	return err
}

// CreateAPI is for running `kubebuilder create api`
func (kc *KBTestContext) CreateAPI(resourceOptions ...string) error {
	resourceOptions = append([]string{"create", "api"}, resourceOptions...)
	cmd := exec.Command("kubebuilder", resourceOptions...)
	_, err := kc.Run(cmd)
	return err
}

// CreateWebhook is for running `kubebuilder create webhook`
func (kc *KBTestContext) CreateWebhook(resourceOptions ...string) error {
	resourceOptions = append([]string{"create", "webhook"}, resourceOptions...)
	cmd := exec.Command("kubebuilder", resourceOptions...)
	_, err := kc.Run(cmd)
	return err
}

// Make is for running `make` with various targets
func (kc *KBTestContext) Make(makeOptions ...string) error {
	cmd := exec.Command("make", makeOptions...)
	_, err := kc.Run(cmd)
	return err
}

// CleanupImage is for cleaning up the docker images for testing
func (kc *KBTestContext) Destroy() {
	cmd := exec.Command("docker", "rmi", "-f", kc.ImageName)
	if _, err := kc.Run(cmd); err != nil {
		fmt.Fprintf(GinkgoWriter, "warning: error when removing the local image: %v\n", err)
	}
	if err := os.RemoveAll(kc.Dir); err != nil {
		fmt.Fprintf(GinkgoWriter, "warning: error when removing the word dir: %v\n", err)
	}
}

// LoadImageToKindCluster loads a local docker image to the kind cluster
func (kc *KBTestContext) LoadImageToKindCluster() error {
	kindOptions := []string{"load", "docker-image", kc.ImageName}
	cmd := exec.Command("kind", kindOptions...)
	_, err := kc.Run(cmd)
	return err
}

type CmdContext struct {
	// environment variables in k=v format.
	Env []string
	Dir string
}

func (cc *CmdContext) Run(cmd *exec.Cmd) ([]byte, error) {
	cmd.Dir = cc.Dir
	cmd.Env = append(os.Environ(), cc.Env...)
	command := strings.Join(cmd.Args, " ")
	fmt.Fprintf(GinkgoWriter, "running: %s\n", command)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return output, fmt.Errorf("%s failed with error: %s", command, string(output))
	}

	return output, nil
}
