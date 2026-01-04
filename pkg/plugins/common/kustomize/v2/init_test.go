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

package v2

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"sigs.k8s.io/kubebuilder/v4/pkg/config"
	cfgv3 "sigs.k8s.io/kubebuilder/v4/pkg/config/v3"
)

const testDomain = "example.com"

var _ = Describe("initSubcommand", func() {
	var (
		subCmd *initSubcommand
		cfg    config.Config
	)

	BeforeEach(func() {
		subCmd = &initSubcommand{}
		cfg = cfgv3.New()
	})

	It("should set domain and project name", func() {
		subCmd.domain = testDomain
		subCmd.name = "my-project"
		err := subCmd.InjectConfig(cfg)

		Expect(err).NotTo(HaveOccurred())
		Expect(cfg.GetDomain()).To(Equal(testDomain))
		Expect(cfg.GetProjectName()).To(Equal("my-project"))
	})

	It("should derive project name from directory when not provided", func() {
		originalDir, err := os.Getwd()
		Expect(err).NotTo(HaveOccurred())
		defer func() { _ = os.Chdir(originalDir) }()

		tmpDir, err := os.MkdirTemp("", "test-project-name")
		Expect(err).NotTo(HaveOccurred())
		defer func() { _ = os.RemoveAll(tmpDir) }()

		err = os.Chdir(tmpDir)
		Expect(err).NotTo(HaveOccurred())

		subCmd.domain = testDomain
		subCmd.name = ""
		err = subCmd.InjectConfig(cfg)

		Expect(err).NotTo(HaveOccurred())
		Expect(cfg.GetProjectName()).To(Equal(filepath.Base(tmpDir)))
	})

	It("should reject invalid DNS 1123 label project names", func() {
		subCmd.domain = testDomain
		subCmd.name = "Invalid_Project"

		err := subCmd.InjectConfig(cfg)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("is invalid"))
	})
})
