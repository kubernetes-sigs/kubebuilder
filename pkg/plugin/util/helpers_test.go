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
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestPlugin(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Plugin Util Suite")
}

var _ = Describe("hasDifferentAPIVersion", func() {
	DescribeTable("should return false",
		func(versions []string) { Expect(hasDifferentAPIVersion(versions, "v1")).To(BeFalse()) },
		Entry("for an empty list of versions", []string{}),
		Entry("for a list of only that version", []string{"v1"}),
	)

	DescribeTable("should return true",
		func(versions []string) { Expect(hasDifferentAPIVersion(versions, "v1")).To(BeTrue()) },
		Entry("for a list of only a different version", []string{"v2"}),
		Entry("for a list of several different versions", []string{"v2", "v3"}),
		Entry("for a list of several versions containing that version", []string{"v1", "v2"}),
	)
})
