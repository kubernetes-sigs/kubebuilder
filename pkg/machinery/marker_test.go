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

var _ = Describe("NewMarkerFor", func() {
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
			Entry("for go files", Marker{prefix: kbPrefix, comment: "//", value: "test"},
				"// +kubebuilder:scaffold:test"),
			Entry("for yaml files", Marker{prefix: kbPrefix, comment: "#", value: "test"},
				"# +kubebuilder:scaffold:test"),
		)
	})
})

var _ = Describe("NewMarkerFor", func() {
	Context("String", func() {
		DescribeTable("should return the right string representation",
			func(marker Marker, str string) { Expect(marker.String()).To(Equal(str)) },
			Entry("for yaml files", NewMarkerFor("test.yaml", "test"),
				"# +kubebuilder:scaffold:test"),
		)
	})
})

var _ = Describe("NewMarkerForImports", func() {
	Context("String", func() {
		DescribeTable("should return the correct string representation for import markers",
			func(marker Marker, str string) { Expect(marker.String()).To(Equal(str)) },
			Entry("for go import marker", NewMarkerFor("test.go", "import \"my/package\""),
				"// +kubebuilder:scaffold:import \"my/package\""),
			Entry("for go import marker with alias", NewMarkerFor("test.go",
				"import alias \"my/package\""), "// +kubebuilder:scaffold:import alias \"my/package\""),
			Entry("for multiline go import marker", NewMarkerFor("test.go",
				"import (\n\"my/package\"\n)"), "// +kubebuilder:scaffold:import (\n\"my/package\"\n)"),
			Entry("for multiline go import marker with alias", NewMarkerFor("test.go",
				"import (\nalias \"my/package\"\n)"), "// +kubebuilder:scaffold:import (\nalias \"my/package\"\n)"),
		)
	})

	It("should detect import in Go file", func() {
		line := "// +kubebuilder:scaffold:import \"my/package\""
		marker := NewMarkerFor("test.go", "import \"my/package\"")
		Expect(marker.EqualsLine(line)).To(BeTrue())
	})

	It("should detect import with alias in Go file", func() {
		line := "// +kubebuilder:scaffold:import alias \"my/package\""
		marker := NewMarkerFor("test.go", "import alias \"my/package\"")
		Expect(marker.EqualsLine(line)).To(BeTrue())
	})

	It("should detect multiline import in Go file", func() {
		line := "// +kubebuilder:scaffold:import (\n\"my/package\"\n)"
		marker := NewMarkerFor("test.go", "import (\n\"my/package\"\n)")
		Expect(marker.EqualsLine(line)).To(BeTrue())
	})

	It("should detect multiline import with alias in Go file", func() {
		line := "// +kubebuilder:scaffold:import (\nalias \"my/package\"\n)"
		marker := NewMarkerFor("test.go",
			"import (\nalias \"my/package\"\n)")
		Expect(marker.EqualsLine(line)).To(BeTrue())
	})
})

var _ = Describe("NewMarkerForImports with different formatting", func() {
	Context("String", func() {
		DescribeTable("should handle variations in spacing and formatting for import markers",
			func(marker Marker, str string) { Expect(marker.String()).To(Equal(str)) },
			Entry("go import marker with extra spaces",
				NewMarkerFor("test.go", "import  \"my/package\""),
				"// +kubebuilder:scaffold:import  \"my/package\""),
			Entry("go import marker with spaces around alias",
				NewMarkerFor("test.go", "import  alias   \"my/package\""),
				"// +kubebuilder:scaffold:import  alias   \"my/package\""),
			Entry("go import marker with newline",
				NewMarkerFor("test.go", "import \n\"my/package\""),
				"// +kubebuilder:scaffold:import \n\"my/package\""),
		)
	})

	It("should detect import with spaces in Go file", func() {
		line := "// +kubebuilder:scaffold:import  \"my/package\""
		marker := NewMarkerFor("test.go", "import  \"my/package\"")
		Expect(marker.EqualsLine(line)).To(BeTrue())
	})

	It("should detect import with alias and spaces in Go file", func() {
		line := "// +kubebuilder:scaffold:import  alias   \"my/package\""
		marker := NewMarkerFor("test.go",
			"import  alias   \"my/package\"")
		Expect(marker.EqualsLine(line)).To(BeTrue())
	})
})

var _ = Describe("NewMarkerWithPrefixFor", func() {
	Context("String", func() {
		DescribeTable("should return the right string representation",
			func(marker Marker, str string) { Expect(marker.String()).To(Equal(str)) },

			Entry("for yaml files",
				NewMarkerWithPrefixFor("custom:scaffold",
					"test.yaml", "test"), "# +custom:scaffold:test"),
			Entry("for yaml files",
				NewMarkerWithPrefixFor("+custom:scaffold",
					"test.yaml", "test"), "# +custom:scaffold:test"),
			Entry("for yaml files",
				NewMarkerWithPrefixFor("custom:scaffold:",
					"test.yaml", "test"), "# +custom:scaffold:test"),
			Entry("for yaml files",
				NewMarkerWithPrefixFor("+custom:scaffold:",
					"test.yaml", "test"), "# +custom:scaffold:test"),
			Entry("for yaml files",
				NewMarkerWithPrefixFor(" +custom:scaffold: ",
					"test.yaml", "test"), "# +custom:scaffold:test"),

			Entry("for go files",
				NewMarkerWithPrefixFor("custom:scaffold",
					"test.go", "test"), "// +custom:scaffold:test"),
			Entry("for go files",
				NewMarkerWithPrefixFor("+custom:scaffold",
					"test.go", "test"), "// +custom:scaffold:test"),
			Entry("for go files",
				NewMarkerWithPrefixFor("custom:scaffold:",
					"test.go", "test"), "// +custom:scaffold:test"),
			Entry("for go files",
				NewMarkerWithPrefixFor("+custom:scaffold:",
					"test.go", "test"), "// +custom:scaffold:test"),
			Entry("for go files",
				NewMarkerWithPrefixFor(" +custom:scaffold: ",
					"test.go", "test"), "// +custom:scaffold:test"),
		)
	})
})

var _ = Describe("NewMarkerFor with unsupported extensions", func() {
	It("should panic for unsupported extensions", func() {
		Expect(func() { NewMarkerFor("file.txt", "test") }).To(Panic())
		Expect(func() { NewMarkerFor("file.md", "test") }).To(Panic())
	})
})
