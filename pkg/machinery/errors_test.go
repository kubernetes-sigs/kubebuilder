/*
Copyright 2020 The Kubernetes Authors.

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
	"errors"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Errors", func() {
	var (
		path    string
		testErr error
	)

	BeforeEach(func() {
		path = filepath.Join("path", "to", "file")
		testErr = errors.New("test error")
	})

	DescribeTable("should contain the wrapped error",
		func(getErr func() error) {
			err := getErr()
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, testErr)).To(BeTrue())
		},
		Entry("for validate errors", func() error { return ValidateError{testErr} }),
		Entry("for set template defaults errors", func() error { return SetTemplateDefaultsError{testErr} }),
		Entry("for file existence errors", func() error { return ExistsFileError{testErr} }),
		Entry("for file opening errors", func() error { return OpenFileError{testErr} }),
		Entry("for directory creation errors", func() error { return CreateDirectoryError{testErr} }),
		Entry("for file creation errors", func() error { return CreateFileError{testErr} }),
		Entry("for file reading errors", func() error { return ReadFileError{testErr} }),
		Entry("for file writing errors", func() error { return WriteFileError{testErr} }),
		Entry("for file closing errors", func() error { return CloseFileError{testErr} }),
	)

	// NOTE: the following test increases coverage
	It("should print a descriptive error message", func() {
		Expect(ModelAlreadyExistsError{path}.Error()).To(ContainSubstring("model already exists"))
		Expect(UnknownIfExistsActionError{path, -1}.Error()).To(ContainSubstring("unknown behavior if file exists"))
		Expect(FileAlreadyExistsError{path}.Error()).To(ContainSubstring("file already exists"))
	})
})
