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
package internal

import (
	"os"
	"path/filepath"
	"runtime"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"sigs.k8s.io/kubebuilder/v4/pkg/config/store/yaml"
)

var _ = Describe("Update.loadConfigFile", func() {
	var opts Update

	BeforeEach(func() {
		opts = Update{}

		originalDir, err := os.Getwd()
		Expect(err).NotTo(HaveOccurred())
		Expect(os.Chdir(GinkgoT().TempDir())).To(Succeed())
		DeferCleanup(func() { Expect(os.Chdir(originalDir)).To(Succeed()) })
	})

	It("should load an existing PROJECT file", func() {
		Expect(os.WriteFile(yaml.DefaultPath, []byte("version: \"3\"\n"), 0o644)).To(Succeed())

		_, err := opts.loadConfigFile()
		Expect(err).NotTo(HaveOccurred())
	})

	It("should report a missing PROJECT file", func() {
		_, err := opts.loadConfigFile()
		Expect(err).To(MatchError(ContainSubstring("no PROJECT file found")))
	})

	It("should report what occupies the path when PROJECT is a directory", func() {
		Expect(os.Mkdir(yaml.DefaultPath, 0o755)).To(Succeed())

		_, err := opts.loadConfigFile()
		Expect(err).To(MatchError(ContainSubstring(`"PROJECT" is a directory`)))
	})

	It("should report a dangling symbolic link as a link and not as a missing file", func() {
		if runtime.GOOS == "windows" {
			Skip("symlink creation requires elevated privileges on Windows")
		}
		target := filepath.Join(GinkgoT().TempDir(), "stolen.yaml")
		Expect(os.Symlink(target, yaml.DefaultPath)).To(Succeed())

		_, err := opts.loadConfigFile()
		Expect(err).To(MatchError(ContainSubstring(`"PROJECT" is a symbolic link`)))
		Expect(target).NotTo(BeAnExistingFile())
	})

	It("should load a PROJECT file through a symbolic link to a regular file", func() {
		if runtime.GOOS == "windows" {
			Skip("symlink creation requires elevated privileges on Windows")
		}
		target := filepath.Join(GinkgoT().TempDir(), "project.yaml")
		Expect(os.WriteFile(target, []byte("version: \"3\"\n"), 0o644)).To(Succeed())
		Expect(os.Symlink(target, yaml.DefaultPath)).To(Succeed())

		_, err := opts.loadConfigFile()
		Expect(err).NotTo(HaveOccurred())
	})
})
