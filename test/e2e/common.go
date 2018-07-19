/*
Copyright 2018 The Kubernetes Authors.

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

package e2e

import (
	"log"
	"os"
	"path/filepath"

	"github.com/kubernetes-sigs/kubebuilder/test/e2e/framework"
	e2einternal "github.com/kubernetes-sigs/kubebuilder/test/internal/e2e"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func prepare(workDir string) {
	By("create a path under given project dir, as the test work dir")
	err := os.MkdirAll(workDir, 0755)
	Expect(err).NotTo(HaveOccurred())
}

func cleanupv0(builderTest *e2einternal.KubebuilderTest, workDir string, imageName string) {
	By("clean up created API objects during test process")
	inputFile := filepath.Join(workDir, "hack", "install.yaml")
	deleteOptions := []string{"delete", "-f", inputFile}
	builderTest.RunKubectlCommand(framework.GetKubectlArgs(deleteOptions))

	By("remove container image created during test")
	builderTest.CleanupImage([]string{imageName})

	By("remove test work dir")
	os.RemoveAll(workDir)
}

func cleanupv1(builderTest *e2einternal.KubebuilderTest, workDir string, imageName string) {
	By("clean up created API objects during test process")

	kustomizeOptions := []string{"build", filepath.Join("config", "default")}
	resources, err := builderTest.RunKustomizeCommand(kustomizeOptions)
	if err != nil {
		log.Printf("error when runing kustomize build during cleaning up: %v", err)
	}

	deleteOptions := []string{"delete", "--recursive", "-f", "-"}
	_, err = builderTest.RunKubectlCommandWithInput(framework.GetKubectlArgs(deleteOptions), resources)
	if err != nil {
		log.Printf("error when runing kubectl delete during cleaning up: %v", err)
	}

	deleteOptions = []string{"delete", "--recursive", "-f", filepath.Join("config", "crds")}
	_, err = builderTest.RunKubectlCommand(framework.GetKubectlArgs(deleteOptions))

	By("remove container image created during test")
	builderTest.CleanupImage([]string{imageName})

	By("remove test work dir")
	os.RemoveAll(workDir)
}
