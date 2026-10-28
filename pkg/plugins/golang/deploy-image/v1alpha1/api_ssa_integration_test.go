//go:build integration

/*
Copyright 2026 The Kubernetes Authors.

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

package v1alpha1

import (
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	pluginutil "sigs.k8s.io/kubebuilder/v4/pkg/plugin/util"
	"sigs.k8s.io/kubebuilder/v4/test/e2e/utils"
)

var _ = Describe("deploy-image plugin chained after go/v4 with Server-Side Apply (--ssa)", func() {
	var kbc *utils.TestContext

	BeforeEach(func() {
		var err error
		kbc, err = utils.NewTestContext(pluginutil.KubebuilderBinName, "GO111MODULE=on")
		Expect(err).NotTo(HaveOccurred())
		Expect(kbc.Prepare()).To(Succeed())
	})

	AfterEach(func() {
		By("removing working directory")
		kbc.Destroy()
	})

	// projectFiles reads every regular file of the project, keyed by its path in the project.
	projectFiles := func() map[string]string {
		files := map[string]string{}
		Expect(filepath.WalkDir(kbc.Dir, func(path string, d fs.DirEntry, err error) error {
			Expect(err).NotTo(HaveOccurred())
			if !d.Type().IsRegular() {
				return nil
			}
			content, err := os.ReadFile(path)
			Expect(err).NotTo(HaveOccurred())
			rel, err := filepath.Rel(kbc.Dir, path)
			Expect(err).NotTo(HaveOccurred())
			files[rel] = string(content)
			return nil
		})).To(Succeed())
		return files
	}

	expectRejected := func(createAPIArgs ...string) {
		before := projectFiles()

		By("creating an API with --ssa")
		args := append([]string{
			"create", "api",
			"--group", "crew",
			"--version", "v1",
			"--kind", "Memcached",
			"--image=memcached:1.6.15-alpine",
			"--resource", "--controller",
			"--ssa",
			"--make=false",
		}, createAPIArgs...)
		output, err := kbc.Run(exec.Command(kbc.BinaryName, args...))
		Expect(err).To(HaveOccurred(), "create api must fail")
		Expect(string(output)).To(ContainSubstring(
			"'--ssa' is not supported by the deploy-image.go.kubebuilder.io/v1-alpha plugin"))

		By("verifying the project was left as it was")
		Expect(projectFiles()).To(Equal(before))
	}

	It("should reject --ssa in a project initialized with go/v4 and deploy-image", func() {
		Expect(kbc.Init(
			"--plugins", "go/v4,deploy-image/v1-alpha",
			"--domain", "test.io",
			"--repo", "test.io/ssatest",
		)).To(Succeed())

		expectRejected()
	})

	It("should reject --ssa with --plugins=go/v4,deploy-image/v1-alpha in a go/v4 project", func() {
		Expect(kbc.Init(
			"--domain", "test.io",
			"--repo", "test.io/ssatest",
		)).To(Succeed())

		expectRejected("--plugins", "go/v4,deploy-image/v1-alpha")
	})
})
