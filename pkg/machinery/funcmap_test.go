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

package machinery

import (
	"bytes"
	"text/template"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("funcmap functions", func() {
	Context("isEmptyString", func() {
		It("should return true for empty strings", func() {
			Expect(isEmptyString("")).To(BeTrue())
		})

		DescribeTable("should return false for any other string",
			func(str string) { Expect(isEmptyString(str)).To(BeFalse()) },
			Entry(`for "a"`, "a"),
			Entry(`for "1"`, "1"),
			Entry(`for "-"`, "-"),
			Entry(`for "."`, "."),
		)
	})

	Context("title", func() {
		DescribeTable("should upper-case the first letter of each word",
			func(str, expected string) { Expect(title(str)).To(Equal(expected)) },
			Entry(`for "foo"`, "foo", "Foo"),
			Entry(`for "foo bar"`, "foo bar", "Foo Bar"),
			Entry(`for "fooBar"`, "fooBar", "FooBar"),
			Entry(`for ""`, "", ""),
		)

		It("should be usable from a template", func() {
			tmpl, err := template.New("test").Funcs(DefaultFuncMap()).Parse(`{{ title "first mate" }}`)
			Expect(err).NotTo(HaveOccurred())
			out := &bytes.Buffer{}
			Expect(tmpl.Execute(out, nil)).To(Succeed())
			Expect(out.String()).To(Equal("First Mate"))
		})
	})

	Context("hashFNV", func() {
		It("should hash the input", func() {
			Expect(hashFNV("test")).To(Equal("afd071e5"))
		})
	})
})
