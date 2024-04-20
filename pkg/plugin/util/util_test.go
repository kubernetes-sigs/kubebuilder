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

var _ = Describe("InsertCode", Ordered, func() {
	path := filepath.Join("testdata", "exampleFile.txt")
	var originalContent []byte

	BeforeAll(func() {
		var err error
		originalContent, err = os.ReadFile(path)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterAll(func() {
		err := os.WriteFile(path, originalContent, 0644)
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
