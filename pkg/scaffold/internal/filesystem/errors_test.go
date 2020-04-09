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

package filesystem

import (
	"errors"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

func TestErrors(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Error suite")
}

var _ = Describe("Errors", func() {
	var (
		path               = filepath.Join("path", "to", "file")
		err                = errors.New("test error")
		fileExistsErr      = fileExistsError{path, err}
		openFileErr        = openFileError{path, err}
		createDirectoryErr = createDirectoryError{path, err}
		createFileErr      = createFileError{path, err}
		readFileErr        = readFileError{path, err}
		writeFileErr       = writeFileError{path, err}
		closeFileErr       = closeFileError{path, err}
	)

	DescribeTable("IsXxxxError should return true for themselves and false for the rest",
		func(f func(error) bool, itself error, rest ...error) {
			Expect(f(itself)).To(BeTrue())
			for _, err := range rest {
				Expect(f(err)).To(BeFalse())
			}
		},
		Entry("file exists", IsFileExistsError, fileExistsErr,
			openFileErr, createDirectoryErr, createFileErr, readFileErr, writeFileErr, closeFileErr),
		Entry("open file", IsOpenFileError, openFileErr,
			fileExistsErr, createDirectoryErr, createFileErr, readFileErr, writeFileErr, closeFileErr),
		Entry("create directory", IsCreateDirectoryError, createDirectoryErr,
			fileExistsErr, openFileErr, createFileErr, readFileErr, writeFileErr, closeFileErr),
		Entry("create file", IsCreateFileError, createFileErr,
			fileExistsErr, openFileErr, createDirectoryErr, readFileErr, writeFileErr, closeFileErr),
		Entry("read file", IsReadFileError, readFileErr,
			fileExistsErr, openFileErr, createDirectoryErr, createFileErr, writeFileErr, closeFileErr),
		Entry("write file", IsWriteFileError, writeFileErr,
			fileExistsErr, openFileErr, createDirectoryErr, createFileErr, readFileErr, closeFileErr),
		Entry("close file", IsCloseFileError, closeFileErr,
			fileExistsErr, openFileErr, createDirectoryErr, createFileErr, readFileErr, writeFileErr),
	)

	DescribeTable("should contain the wrapped error and error message",
		func(err error) {
			Expect(err).To(MatchError(err))
			Expect(err.Error()).To(ContainSubstring(err.Error()))
		},
		Entry("file exists", fileExistsErr),
		Entry("open file", openFileErr),
		Entry("create directory", createDirectoryErr),
		Entry("create file", createFileErr),
		Entry("read file", readFileErr),
		Entry("write file", writeFileErr),
		Entry("close file", closeFileErr),
	)
})
