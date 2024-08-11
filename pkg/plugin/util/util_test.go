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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Cover plugin util helpers", func() {
	Describe("InsertCode", Ordered, func() {
		path := filepath.Join("testdata", "exampleFile.txt")
		var content []byte

		BeforeAll(func() {
			err := os.MkdirAll("testdata", 0755)
			Expect(err).NotTo(HaveOccurred())

			if _, err := os.Stat(path); os.IsNotExist(err) {
				err = os.WriteFile(path, []byte("exampleTarget"), 0644)
				Expect(err).NotTo(HaveOccurred())
			}

			content, err = os.ReadFile(path)
			Expect(err).NotTo(HaveOccurred())
		})

		AfterAll(func() {
			err := os.WriteFile(path, content, 0644)
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
})
