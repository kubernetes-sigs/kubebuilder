/*
Copyright 2022 The Kubernetes Authors.

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

package util

import (
	"os"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Cover plugin util helpers", func() {
	Describe("RandomSuffix", func() {
		It("should return a string with 4 caracteres", func() {
			suffix, err := RandomSuffix()
			Expect(err).NotTo(HaveOccurred())
			Expect(suffix).To(HaveLen(4))
		})

		It("should return different values when call more than once", func() {
			suffix1, _ := RandomSuffix()
			suffix2, _ := RandomSuffix()
			Expect(suffix1).NotTo(Equal(suffix2))
		})
	})

	Describe("GetNonEmptyLines", func() {
		It("should return non-empty lines", func() {
			output := "text1\n\ntext2\ntext3\n\n"
			lines := GetNonEmptyLines(output)
			Expect(lines).To(Equal([]string{"text1", "text2", "text3"}))
		})

		It("should return an empty when an empty value is passed", func() {
			lines := GetNonEmptyLines("")
			Expect(lines).To(BeEmpty())
		})

		It("should return same string without empty lines", func() {
			output := "noemptylines"
			lines := GetNonEmptyLines(output)
			Expect(lines).To(Equal([]string{"noemptylines"}))
		})
	})

	Describe("InsertCode", Ordered, func() {
		var (
			content []byte
			path    string
		)

		BeforeAll(func() {
			path = filepath.Join("testdata", "exampleFile.txt")

			err := os.MkdirAll("testdata", 0o755)
			Expect(err).NotTo(HaveOccurred())

			if _, err = os.Stat(path); os.IsNotExist(err) {
				err = os.WriteFile(path, []byte("exampleTarget"), 0o644)
				Expect(err).NotTo(HaveOccurred())
			}

			content, err = os.ReadFile(path)
			Expect(err).NotTo(HaveOccurred())
		})

		AfterAll(func() {
			err := os.WriteFile(path, content, 0o644)
			Expect(err).NotTo(HaveOccurred())

			err = os.RemoveAll("testdata")
			Expect(err).NotTo(HaveOccurred())
		})

		DescribeTable("should not succeed",
			func(target string) {
				Expect(InsertCode(path, target, "exampleCode")).ShouldNot(Succeed())
			},
			Entry("target given is not present in file", "randomTarget"),
		)

		DescribeTable("should succeed",
			func(target string) {
				Expect(InsertCode(path, target, "exampleCode")).Should(Succeed())
			},
			Entry("target given is present in file", "exampleTarget"),
		)
	})

	Describe("InsertCodeIfNotExist", Ordered, func() {
		var (
			content []byte
			path    string
		)

		BeforeAll(func() {
			path = filepath.Join("testdata", "exampleFile.txt")

			err := os.MkdirAll("testdata", 0o755)
			Expect(err).NotTo(HaveOccurred())

			err = os.WriteFile(path, []byte("target\n"), 0o644)
			Expect(err).NotTo(HaveOccurred())

			content, err = os.ReadFile(path)
			Expect(err).NotTo(HaveOccurred())
		})

		AfterAll(func() {
			err := os.WriteFile(path, content, 0o644)
			Expect(err).NotTo(HaveOccurred())

			err = os.RemoveAll("testdata")
			Expect(err).NotTo(HaveOccurred())
		})

		It("should insert code if not present", func() {
			Expect(InsertCodeIfNotExist(path, "target", "code\n")).To(Succeed())
			b, err := os.ReadFile(path)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(b)).To(ContainSubstring("code"))
		})

		It("should not insert code if already present", func() {
			Expect(InsertCodeIfNotExist(path, "target", "code\n")).To(Succeed())
			b, err := os.ReadFile(path)
			Expect(err).NotTo(HaveOccurred())

			// Only one "code" should be present
			Expect(strings.Count(string(b), "code")).To(Equal(1))
		})
	})

	Describe("AppendCodeIfNotExist", Ordered, func() {
		var (
			content []byte
			path    string
		)

		BeforeAll(func() {
			path = filepath.Join("testdata", "exampleFile.txt")

			err := os.MkdirAll("testdata", 0o755)
			Expect(err).NotTo(HaveOccurred())

			err = os.WriteFile(path, []byte("foo\n"), 0o644)
			Expect(err).NotTo(HaveOccurred())

			content, err = os.ReadFile(path)
			Expect(err).NotTo(HaveOccurred())
		})

		AfterAll(func() {
			err := os.WriteFile(path, content, 0o644)
			Expect(err).NotTo(HaveOccurred())

			err = os.RemoveAll("testdata")
			Expect(err).NotTo(HaveOccurred())
		})

		It("should append code if not present", func() {
			Expect(AppendCodeIfNotExist(path, "code\n")).To(Succeed())
			b, _ := os.ReadFile(path)
			Expect(string(b)).To(HaveSuffix("code\n"))
		})

		It("should not append code if already present", func() {
			Expect(AppendCodeIfNotExist(path, "code\n")).To(Succeed())
			b, _ := os.ReadFile(path)
			Expect(strings.Count(string(b), "code\n")).To(Equal(1))
		})
	})

	Describe("UncommentCode and CommentCode", Ordered, func() {
		var (
			content []byte
			path    string
		)

		BeforeAll(func() {
			path = filepath.Join("testdata", "exampleFile.txt")

			err := os.MkdirAll("testdata", 0o755)
			Expect(err).NotTo(HaveOccurred())

			// Write a file with commented lines
			err = os.WriteFile(path, []byte("#line1\n#line2\nline3\n"), 0o644)
			Expect(err).NotTo(HaveOccurred())

			content, err = os.ReadFile(path)
			Expect(err).NotTo(HaveOccurred())
		})

		AfterAll(func() {
			err := os.WriteFile(path, content, 0o644)
			Expect(err).NotTo(HaveOccurred())

			err = os.RemoveAll("testdata")
			Expect(err).NotTo(HaveOccurred())
		})

		It("should uncomment code with prefix", func() {
			target := "#line1\n#line2"
			Expect(UncommentCode(path, target, "#")).To(Succeed())
			b, err := os.ReadFile(path)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(b)).To(ContainSubstring("line1\nline2\nline3\n"))
			Expect(string(b)).NotTo(ContainSubstring("#line1"))
		})

		It("should comment code with prefix", func() {
			target := "line1\nline2\n"
			Expect(CommentCode(path, target, "#")).To(Succeed())
			b, err := os.ReadFile(path)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(b)).To(ContainSubstring("#line1\n#line2\n"))
		})

		It("should error if target not found for uncomment", func() {
			Expect(UncommentCode(path, "notfound", "#")).NotTo(Succeed())
		})

		It("should error if target not found for comment", func() {
			Expect(CommentCode(path, "notfound", "#")).NotTo(Succeed())
		})
	})

	Describe("EnsureExistAndReplace", func() {
		Context("Content Exists", func() {
			It("should replace all the matched contents", func() {
				got, err := EnsureExistAndReplace("test", "t", "r")
				Expect(err).NotTo(HaveOccurred())
				Expect(got).To(Equal("resr"))
			})
		})

		Context("Content Not Exists", func() {
			It("should error out", func() {
				got, err := EnsureExistAndReplace("test", "m", "r")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal(`can't find "m"`))
				Expect(got).To(Equal(""))
			})
		})
	})

	Describe("ReplaceInFile", Ordered, func() {
		var (
			content []byte
			path    string
		)

		BeforeAll(func() {
			path = filepath.Join("testdata", "exampleFile.txt")

			err := os.MkdirAll("testdata", 0o755)
			Expect(err).NotTo(HaveOccurred())

			err = os.WriteFile(path, []byte("foo bar foo\nbaz foo\n"), 0o644)
			Expect(err).NotTo(HaveOccurred())

			content, err = os.ReadFile(path)
			Expect(err).NotTo(HaveOccurred())
		})

		AfterAll(func() {
			err := os.WriteFile(path, content, 0o644)
			Expect(err).NotTo(HaveOccurred())

			err = os.RemoveAll("testdata")
			Expect(err).NotTo(HaveOccurred())
		})

		It("should replace all occurrences of a string", func() {
			Expect(ReplaceInFile(path, "foo", "qux")).To(Succeed())
			b, err := os.ReadFile(path)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(b)).To(Equal("qux bar qux\nbaz qux\n"))
		})

		It("should error if oldValue not found", func() {
			Expect(ReplaceInFile(path, "notfound", "something")).NotTo(Succeed())
		})
	})

	Describe("ReplaceRegexInFile", Ordered, func() {
		var (
			content []byte
			path    string
		)

		BeforeAll(func() {
			path = filepath.Join("testdata", "exampleFile.txt")

			err := os.MkdirAll("testdata", 0o755)
			Expect(err).NotTo(HaveOccurred())

			err = os.WriteFile(path, []byte("foo123 bar456 foo789\nbaz000\n"), 0o644)
			Expect(err).NotTo(HaveOccurred())

			content, err = os.ReadFile(path)
			Expect(err).NotTo(HaveOccurred())
		})

		AfterAll(func() {
			err := os.WriteFile(path, content, 0o644)
			Expect(err).NotTo(HaveOccurred())

			err = os.RemoveAll("testdata")
			Expect(err).NotTo(HaveOccurred())
		})

		It("should replace all regex matches", func() {
			Expect(ReplaceRegexInFile(path, `\d+`, "X")).To(Succeed())
			b, err := os.ReadFile(path)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(b)).To(Equal("fooX barX fooX\nbazX\n"))
		})

		It("should error if regex not found", func() {
			Expect(ReplaceRegexInFile(path, `notfound`, "Y")).NotTo(Succeed())
		})

		It("should error if regex is invalid", func() {
			Expect(ReplaceRegexInFile(path, `\K`, "Z")).NotTo(Succeed())
		})
	})

	Describe("HasFileContentWith", Ordered, func() {
		const (
			path    = "testdata/PROJECT"
			content = `# Code generated by tool. DO NOT EDIT.
# This file is used to track the info used to scaffold your project
# and allow the plugins properly work.
# More info: https://book.kubebuilder.io/reference/project-config.html
domain: example.org
layout:
- go.kubebuilder.io/v4
- helm.kubebuilder.io/v1-alpha
plugins:
  helm.kubebuilder.io/v1-alpha: {}
repo: github.com/example/repo
version: "3"
`
		)

		BeforeAll(func() {
			err := os.MkdirAll("testdata", 0o755)
			Expect(err).NotTo(HaveOccurred())

			if _, err = os.Stat(path); os.IsNotExist(err) {
				err = os.WriteFile(path, []byte(content), 0o644)
				Expect(err).NotTo(HaveOccurred())
			}
		})

		AfterAll(func() {
			err := os.RemoveAll("testdata")
			Expect(err).NotTo(HaveOccurred())
		})

		It("should return true when file contains the expected content", func() {
			content := "repo: github.com/example/repo"
			found, err := HasFileContentWith(path, content)
			Expect(err).NotTo(HaveOccurred())
			Expect(found).To(BeTrue())
		})

		It("should return true when file contains multiline expected content", func() {
			content := `plugins:
  helm.kubebuilder.io/v1-alpha: {}`
			found, err := HasFileContentWith(path, content)
			Expect(err).NotTo(HaveOccurred())
			Expect(found).To(BeTrue())
		})

		It("should return false when file does not contain the expected content", func() {
			content := "nonExistentContent"
			found, err := HasFileContentWith(path, content)
			Expect(err).NotTo(HaveOccurred())
			Expect(found).To(BeFalse())
		})
	})
})
