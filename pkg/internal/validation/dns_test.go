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

package validation

import (
	"fmt"
	"strings"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestValidation(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Validation Suite")
}

var _ = Describe("IsDNS1123Label", func() {
	It("should return no error", func() {
		for _, value := range []string{
			"a", "ab", "abc", "a1", "a-1", "a--1--2--b",
			"0", "01", "012", "1a", "1-a", "1--a--b--2",
			strings.Repeat("a", 56),
		} {
			By(fmt.Sprintf("for %s", value))
			Expect(len(IsDNS1123Label(value))).To(Equal(0))
		}
	})

	It("should return at least one error", func() {
		for _, value := range []string{
			"", "A", "ABC", "aBc", "A1", "A-1", "1-A",
			"-", "a-", "-a", "1-", "-1",
			"_", "a_", "_a", "a_b", "1_", "_1", "1_2",
			".", "a.", ".a", "a.b", "1.", ".1", "1.2",
			" ", "a ", " a", "a b", "1 ", " 1", "1 2",
			strings.Repeat("a", 57),
		} {
			By(fmt.Sprintf("for %s", value))
			Expect(len(IsDNS1123Label(value))).NotTo(Equal(0))
		}
	})
})

var _ = Describe("IsDNS1123Subdomain", func() {
	It("should return no error", func() {
		for _, value := range []string{
			"a", "ab", "abc", "a1", "a-1", "a--1--2--b",
			"0", "01", "012", "1a", "1-a", "1--a--b--2",
			"a.a", "ab.a", "abc.a", "a1.a", "a-1.a", "a--1--2--b.a",
			"a.1", "ab.1", "abc.1", "a1.1", "a-1.1", "a--1--2--b.1",
			"0.a", "01.a", "012.a", "1a.a", "1-a.a", "1--a--b--2",
			"0.1", "01.1", "012.1", "1a.1", "1-a.1", "1--a--b--2.1",
			"a.b.c.d.e", "aa.bb.cc.dd.ee", "1.2.3.4.5", "11.22.33.44.55",
			strings.Repeat("a", 253),
		} {
			By(fmt.Sprintf("for %s", value))
			Expect(len(IsDNS1123Subdomain(value))).To(Equal(0))
		}
	})

	It("should return at least one error", func() {
		for _, value := range []string{
			"", "A", "ABC", "aBc", "A1", "A-1", "1-A",
			"-", "a-", "-a", "1-", "-1",
			"_", "a_", "_a", "a_b", "1_", "_1", "1_2",
			".", "a.", ".a", "a..b", "1.", ".1", "1..2",
			" ", "a ", " a", "a b", "1 ", " 1", "1 2",
			"A.a", "aB.a", "ab.A", "A1.a", "a1.A",
			"A.1", "aB.1", "A1.1", "1A.1",
			"0.A", "01.A", "012.A", "1A.a", "1a.A",
			"A.B.C.D.E", "AA.BB.CC.DD.EE", "a.B.c.d.e", "aa.bB.cc.dd.ee",
			"a@b", "a,b", "a_b", "a;b",
			"a:b", "a%b", "a?b", "a$b",
			strings.Repeat("a", 254),
		} {
			By(fmt.Sprintf("for %s", value))
			Expect(len(IsDNS1123Subdomain(value))).NotTo(Equal(0))
		}
	})
})

var _ = Describe("IsDNS1035Label", func() {
	It("should return no error", func() {
		for _, value := range []string{
			"a", "ab", "abc", "a1", "a-1", "a--1--2--b",
			strings.Repeat("a", 63),
		} {
			By(fmt.Sprintf("for %s", value))
			Expect(len(IsDNS1035Label(value))).To(Equal(0))
		}
	})

	It("should return at least one error", func() {
		for _, value := range []string{
			"0", "01", "012", "1a", "1-a", "1--a--b--2",
			"", "A", "ABC", "aBc", "A1", "A-1", "1-A",
			"-", "a-", "-a", "1-", "-1",
			"_", "a_", "_a", "a_b", "1_", "_1", "1_2",
			".", "a.", ".a", "a.b", "1.", ".1", "1.2",
			" ", "a ", " a", "a b", "1 ", " 1", "1 2",
			strings.Repeat("a", 64),
		} {
			By(fmt.Sprintf("for %s", value))
			Expect(len(IsDNS1035Label(value))).NotTo(Equal(0))
		}
	})
})

// This test provides coverage to the conditional that handles no examples
// as they are not being used in any check
var _ = Describe("regexError", func() {
	It("should work without examples", func() {
		Expect(regexError("$", "_")).To(Equal("$ (regex used for validation is '_')"))
	})
})
