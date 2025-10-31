/*
Copyright 2025 The Kubernetes Authors.

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

package v4

import (
	"errors"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"sigs.k8s.io/kubebuilder/v4/pkg/plugin/util"
	"sigs.k8s.io/kubebuilder/v4/test/e2e/utils"
)

var _ = Describe("checkDir", func() {
	var (
		testCtx *utils.TestContext
		oldDir  string
	)

	BeforeEach(func() {
		var err error
		testCtx, err = utils.NewTestContext(util.KubebuilderBinName, "GO111MODULE=on")
		Expect(err).NotTo(HaveOccurred())
		Expect(testCtx.Prepare()).To(Succeed())

		oldDir, err = os.Getwd()
		Expect(err).NotTo(HaveOccurred())
		Expect(os.Chdir(testCtx.Dir)).To(Succeed())
	})

	AfterEach(func() {
		if oldDir != "" {
			Expect(os.Chdir(oldDir)).To(Succeed())
		}
		if testCtx != nil {
			testCtx.ImageName = ""
			testCtx.Destroy()
		}
	})

	When("directory contains a mise.toml configuration", func() {
		BeforeEach(func() {
			Expect(os.WriteFile("mise.toml", []byte("[tool]\n"), 0o644)).To(Succeed())
		})

		It("does not return an error", func() {
			Expect(checkDir()).To(Succeed())
		})
	})

	When("directory contains files that are not explicitly disallowed", func() {
		BeforeEach(func() {
			Expect(os.WriteFile("unexpected.txt", []byte("temporary content"), 0o644)).To(Succeed())
		})

		It("allows the scaffold to proceed", func() {
			Expect(checkDir()).To(Succeed())
		})
	})

	When("directory contains hidden entries", func() {
		BeforeEach(func() {
			Expect(os.WriteFile(".env", []byte("KB_ENV=true"), 0o644)).To(Succeed())
			Expect(os.MkdirAll(filepath.Join(".cache", "nested"), 0o755)).To(Succeed())
			Expect(os.WriteFile(filepath.Join(".cache", "nested", "config.yaml"), []byte("kind: Test"), 0o644)).To(Succeed())
		})

		It("ignores hidden files and directories", func() {
			Expect(checkDir()).To(Succeed())
		})
	})

	DescribeTable("disallows files with extensions that kubebuilder will overwrite",
		func(filename string) {
			Expect(os.WriteFile(filename, []byte("content"), 0o644)).To(Succeed())

			err := checkDir()
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, os.ErrNotExist)).To(BeFalse())
			Expect(err.Error()).To(ContainSubstring(filename))
		},
		Entry("Go sources", "main.go"),
		Entry("YAML manifests", "config.yaml"),
		Entry("go.mod file", "go.mod"),
		Entry("go.sum file", "go.sum"),
	)
})
