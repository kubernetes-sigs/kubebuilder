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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("NerMarkerFor", func() {
	DescribeTable("should create valid markers for known extensions",
		func(path, comment string) { Expect(NewMarkerFor(path, "").comment).To(Equal(comment)) },
		Entry("for go files", "file.go", "//"),
		Entry("for yaml files", "file.yaml", "#"),
		Entry("for yaml files (short version)", "file.yml", "#"),
	)

	It("should panic for unknown extensions", func() {
		// testing panics require to use a function with no arguments
		Expect(func() { NewMarkerFor("file.unkownext", "") }).To(Panic())
	})
})

var _ = Describe("Marker", func() {
	Context("String", func() {
		DescribeTable("should return the right string representation",
			func(marker Marker, str string) { Expect(marker.String()).To(Equal(str)) },
			Entry("for go files", Marker{comment: "//", value: "test"}, "//+kubebuilder:scaffold:test"),
			Entry("for yaml files", Marker{comment: "#", value: "test"}, "#+kubebuilder:scaffold:test"),
		)
	})
})
