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

package internal

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("generate: cleanup-helpers", func() {
	var tmpDir string

	BeforeEach(func() {
		var err error
		tmpDir, err = os.MkdirTemp("", "kubebuilder-clean-output-")
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		Expect(os.RemoveAll(tmpDir)).To(Succeed())
	})

	entryNames := func(dir string) []string {
		GinkgoHelper()
		entries, err := os.ReadDir(dir)
		Expect(err).NotTo(HaveOccurred())
		names := make([]string, 0, len(entries))
		for _, e := range entries {
			names = append(names, e.Name())
		}
		return names
	}

	Context("when the output directory contains mixed project entries", func() {
		BeforeEach(func() {
			Expect(os.Mkdir(filepath.Join(tmpDir, ".git"), 0o755)).To(Succeed())
			Expect(os.WriteFile(filepath.Join(tmpDir, ".git", "HEAD"), []byte("ref: refs/heads/main\n"), 0o644)).To(Succeed())
			Expect(os.WriteFile(filepath.Join(tmpDir, "PROJECT"), []byte("version: 3\n"), 0o644)).To(Succeed())
			Expect(os.WriteFile(filepath.Join(tmpDir, "Makefile"), []byte("all:\n"), 0o644)).To(Succeed())
			Expect(os.WriteFile(filepath.Join(tmpDir, ".gitignore"), []byte("bin/\n"), 0o644)).To(Succeed())
			Expect(os.Mkdir(filepath.Join(tmpDir, ".github"), 0o755)).To(Succeed())
			Expect(os.WriteFile(filepath.Join(tmpDir, ".github", "workflows.yml"), []byte("name: ci\n"), 0o644)).To(Succeed())
			Expect(os.Mkdir(filepath.Join(tmpDir, "cmd"), 0o755)).To(Succeed())
			Expect(os.WriteFile(filepath.Join(tmpDir, "cmd", "main.go"), []byte("package main\n"), 0o644)).To(Succeed())
			// Names with shell metacharacters must be removed via Go APIs, never a shell.
			Expect(os.WriteFile(filepath.Join(tmpDir, "evil$(id).txt"), []byte("x"), 0o644)).To(Succeed())
			Expect(os.WriteFile(filepath.Join(tmpDir, "file;rm -rf.x"), []byte("y"), 0o644)).To(Succeed())
		})

		It("preserves .git contents and PROJECT while removing other top-level entries", func() {
			Expect(cleanOutputDirPreservingGitAndProject(tmpDir)).To(Succeed())
			Expect(entryNames(tmpDir)).To(ConsistOf(".git", "PROJECT"))

			head, err := os.ReadFile(filepath.Join(tmpDir, ".git", "HEAD"))
			Expect(err).NotTo(HaveOccurred())
			Expect(string(head)).To(Equal("ref: refs/heads/main\n"))

			project, err := os.ReadFile(filepath.Join(tmpDir, "PROJECT"))
			Expect(err).NotTo(HaveOccurred())
			Expect(string(project)).To(Equal("version: 3\n"))

			Expect(filepath.Join(tmpDir, "cmd")).NotTo(BeADirectory())
			Expect(filepath.Join(tmpDir, "Makefile")).NotTo(BeAnExistingFile())
			Expect(filepath.Join(tmpDir, ".gitignore")).NotTo(BeAnExistingFile())
			Expect(filepath.Join(tmpDir, ".github")).NotTo(BeADirectory())
			Expect(filepath.Join(tmpDir, "evil$(id).txt")).NotTo(BeAnExistingFile())
			Expect(filepath.Join(tmpDir, "file;rm -rf.x")).NotTo(BeAnExistingFile())
		})
	})

	Context("when the output directory only has preserved entries", func() {
		It("succeeds and leaves .git and PROJECT untouched when both exist", func() {
			Expect(os.Mkdir(filepath.Join(tmpDir, ".git"), 0o755)).To(Succeed())
			Expect(os.WriteFile(filepath.Join(tmpDir, ".git", "HEAD"), []byte("ref: refs/heads/main\n"), 0o644)).To(Succeed())
			Expect(os.WriteFile(filepath.Join(tmpDir, "PROJECT"), []byte("version: 3\n"), 0o644)).To(Succeed())

			Expect(cleanOutputDirPreservingGitAndProject(tmpDir)).To(Succeed())
			Expect(entryNames(tmpDir)).To(ConsistOf(".git", "PROJECT"))
			head, err := os.ReadFile(filepath.Join(tmpDir, ".git", "HEAD"))
			Expect(err).NotTo(HaveOccurred())
			Expect(string(head)).To(Equal("ref: refs/heads/main\n"))
			content, err := os.ReadFile(filepath.Join(tmpDir, "PROJECT"))
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).To(Equal("version: 3\n"))
		})

		It("succeeds when only PROJECT exists", func() {
			Expect(os.WriteFile(filepath.Join(tmpDir, "PROJECT"), []byte("version: 3\n"), 0o644)).To(Succeed())
			Expect(cleanOutputDirPreservingGitAndProject(tmpDir)).To(Succeed())
			Expect(entryNames(tmpDir)).To(ConsistOf("PROJECT"))
		})

		It("succeeds when only .git exists", func() {
			Expect(os.Mkdir(filepath.Join(tmpDir, ".git"), 0o755)).To(Succeed())
			Expect(cleanOutputDirPreservingGitAndProject(tmpDir)).To(Succeed())
			Expect(entryNames(tmpDir)).To(ConsistOf(".git"))
		})
	})

	Context("when the output directory is empty", func() {
		It("succeeds without error", func() {
			Expect(cleanOutputDirPreservingGitAndProject(tmpDir)).To(Succeed())
			Expect(entryNames(tmpDir)).To(BeEmpty())
		})
	})

	Context("when nested content exists under a removable directory", func() {
		It("removes the directory tree with RemoveAll", func() {
			nested := filepath.Join(tmpDir, "api", "v1", "types.go")
			Expect(os.MkdirAll(filepath.Dir(nested), 0o755)).To(Succeed())
			Expect(os.WriteFile(nested, []byte("package v1\n"), 0o644)).To(Succeed())
			Expect(os.WriteFile(filepath.Join(tmpDir, "PROJECT"), []byte("version: 3\n"), 0o644)).To(Succeed())

			Expect(cleanOutputDirPreservingGitAndProject(tmpDir)).To(Succeed())
			Expect(entryNames(tmpDir)).To(ConsistOf("PROJECT"))
			Expect(filepath.Join(tmpDir, "api")).NotTo(BeADirectory())
		})
	})

	Context("when the output directory cannot be read", func() {
		It("returns an error for a non-existent path", func() {
			missing := filepath.Join(tmpDir, "does-not-exist")
			err := cleanOutputDirPreservingGitAndProject(missing)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("read output directory"))
			Expect(err.Error()).To(ContainSubstring(missing))
		})
	})
})
