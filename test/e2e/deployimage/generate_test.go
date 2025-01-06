/*
Copyright 2020 The Kubernetes Authors.

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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"sigs.k8s.io/kubebuilder/v4/test/e2e/utils"
)

// GenerateDeployImageWithOptions implements a go/v4 plugin project and scaffold an API using the image options
func GenerateDeployImageWithOptions(kbc *utils.TestContext) {
	initTheProject(kbc)
	creatingAPIWithOptions(kbc)
}

// GenerateDeployImage implements a go/v4 plugin project and scaffold an API using the deploy image plugin
func GenerateDeployImage(kbc *utils.TestContext) {
	initTheProject(kbc)
	creatingAPI(kbc)
}

func creatingAPI(kbc *utils.TestContext) {
	By("creating API definition with deploy-image/v1-alpha plugin")
	err := kbc.CreateAPI(
		"--group", kbc.Group,
		"--version", kbc.Version,
		"--kind", kbc.Kind,
		"--plugins", "deploy-image/v1-alpha",
		"--image", "busybox:1.36.1",
		"--run-as-user", "1001",
		"--make=false",
		"--manifests=false",
	)
	Expect(err).NotTo(HaveOccurred(), "Failed to create API definition")
}

func creatingAPIWithOptions(kbc *utils.TestContext) {
	var err error
	By("creating API definition with deploy-image/v1-alpha plugin")
	err = kbc.CreateAPI(
		"--group", kbc.Group,
		"--version", kbc.Version,
		"--kind", kbc.Kind,
		"--plugins", "deploy-image/v1-alpha",
		"--image", "memcached:1.6.26-alpine3.19",
		"--image-container-port", "11211",
		"--image-container-command", "memcached,--memory-limit=64,-o,modern,-v",
		"--run-as-user", "1001",
		"--make=false",
		"--manifests=false",
	)
	Expect(err).NotTo(HaveOccurred(), "Failed to create API definition with deploy-image/v1-alpha")
}

func initTheProject(kbc *utils.TestContext) {
	By("initializing a project")
	err := kbc.Init(
		"--plugins", "go/v4",
		"--project-version", "3",
		"--domain", kbc.Domain,
	)
	Expect(err).NotTo(HaveOccurred(), "Failed to initialize project")
}
