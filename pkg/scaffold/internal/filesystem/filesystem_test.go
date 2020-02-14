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
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestFileSystem(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "FileSystem suite")
}

var _ = Describe("FileSystem", func() {
	Describe("New", func() {
		const (
			dirPerm  os.FileMode = 0777
			filePerm os.FileMode = 0666
		)

		var (
			fsi FileSystem
			fs  fileSystem
			ok  bool
		)

		Context("when using no options", func() {
			BeforeEach(func() {
				fsi = New()
				fs, ok = fsi.(fileSystem)
			})

			It("should be a fileSystem instance", func() {
				Expect(ok).To(BeTrue())
			})

			It("should not have a nil fs", func() {
				Expect(fs.fs).NotTo(BeNil())
			})

			It("should use default directory permission", func() {
				Expect(fs.dirPerm).To(Equal(defaultDirectoryPermission))
			})

			It("should use default file permission", func() {
				Expect(fs.filePerm).To(Equal(defaultFilePermission))
			})

			It("should use default file mode", func() {
				Expect(fs.fileMode).To(Equal(createOrUpdate))
			})
		})

		Context("when using directory permission option", func() {
			BeforeEach(func() {
				fsi = New(DirectoryPermissions(dirPerm))
				fs, ok = fsi.(fileSystem)
			})

			It("should be a fileSystem instance", func() {
				Expect(ok).To(BeTrue())
			})

			It("should not have a nil fs", func() {
				Expect(fs.fs).NotTo(BeNil())
			})

			It("should use provided directory permission", func() {
				Expect(fs.dirPerm).To(Equal(dirPerm))
			})

			It("should use default file permission", func() {
				Expect(fs.filePerm).To(Equal(defaultFilePermission))
			})

			It("should use default file mode", func() {
				Expect(fs.fileMode).To(Equal(createOrUpdate))
			})
		})

		Context("when using file permission option", func() {
			BeforeEach(func() {
				fsi = New(FilePermissions(filePerm))
				fs, ok = fsi.(fileSystem)
			})

			It("should be a fileSystem instance", func() {
				Expect(ok).To(BeTrue())
			})

			It("should not have a nil fs", func() {
				Expect(fs.fs).NotTo(BeNil())
			})

			It("should use default directory permission", func() {
				Expect(fs.dirPerm).To(Equal(defaultDirectoryPermission))
			})

			It("should use provided file permission", func() {
				Expect(fs.filePerm).To(Equal(filePerm))
			})

			It("should use default file mode", func() {
				Expect(fs.fileMode).To(Equal(createOrUpdate))
			})
		})

		Context("when using both directory and file permission options", func() {
			BeforeEach(func() {
				fsi = New(DirectoryPermissions(dirPerm), FilePermissions(filePerm))
				fs, ok = fsi.(fileSystem)
			})

			It("should be a fileSystem instance", func() {
				Expect(ok).To(BeTrue())
			})

			It("should not have a nil fs", func() {
				Expect(fs.fs).NotTo(BeNil())
			})

			It("should use provided directory permission", func() {
				Expect(fs.dirPerm).To(Equal(dirPerm))
			})

			It("should use provided file permission", func() {
				Expect(fs.filePerm).To(Equal(filePerm))
			})

			It("should use default file mode", func() {
				Expect(fs.fileMode).To(Equal(createOrUpdate))
			})
		})
	})

	Describe("Errors", func() {
		var (
			err                = errors.New("test error")
			path               = filepath.Join("path", "to", "file")
			fileExistsErr      = fileExistsError{path, err}
			openFileErr        = openFileError{path, err}
			createDirectoryErr = createDirectoryError{path, err}
			createFileErr      = createFileError{path, err}
			readFileErr        = readFileError{path, err}
			writeFileErr       = writeFileError{path, err}
			closeFileErr       = closeFileError{path, err}
		)

		Context("IsFileExistsError", func() {
			It("should return true for file exists errors", func() {
				Expect(IsFileExistsError(fileExistsErr)).To(BeTrue())
			})

			It("should return false for any other error", func() {
				Expect(IsFileExistsError(err)).To(BeFalse())
				Expect(IsFileExistsError(openFileErr)).To(BeFalse())
				Expect(IsFileExistsError(createDirectoryErr)).To(BeFalse())
				Expect(IsFileExistsError(createFileErr)).To(BeFalse())
				Expect(IsFileExistsError(readFileErr)).To(BeFalse())
				Expect(IsFileExistsError(writeFileErr)).To(BeFalse())
				Expect(IsFileExistsError(closeFileErr)).To(BeFalse())
			})
		})

		Context("IsOpenFileError", func() {
			It("should return true for open file errors", func() {
				Expect(IsOpenFileError(openFileErr)).To(BeTrue())
			})

			It("should return false for any other error", func() {
				Expect(IsOpenFileError(err)).To(BeFalse())
				Expect(IsOpenFileError(fileExistsErr)).To(BeFalse())
				Expect(IsOpenFileError(createDirectoryErr)).To(BeFalse())
				Expect(IsOpenFileError(createFileErr)).To(BeFalse())
				Expect(IsOpenFileError(readFileErr)).To(BeFalse())
				Expect(IsOpenFileError(writeFileErr)).To(BeFalse())
				Expect(IsOpenFileError(closeFileErr)).To(BeFalse())
			})
		})

		Context("IsCreateDirectoryError", func() {
			It("should return true for create directory errors", func() {
				Expect(IsCreateDirectoryError(createDirectoryErr)).To(BeTrue())
			})

			It("should return false for any other error", func() {
				Expect(IsCreateDirectoryError(err)).To(BeFalse())
				Expect(IsCreateDirectoryError(fileExistsErr)).To(BeFalse())
				Expect(IsCreateDirectoryError(openFileErr)).To(BeFalse())
				Expect(IsCreateDirectoryError(createFileErr)).To(BeFalse())
				Expect(IsCreateDirectoryError(readFileErr)).To(BeFalse())
				Expect(IsCreateDirectoryError(writeFileErr)).To(BeFalse())
				Expect(IsCreateDirectoryError(closeFileErr)).To(BeFalse())
			})
		})

		Context("IsCreateFileError", func() {
			It("should return true for create file errors", func() {
				Expect(IsCreateFileError(createFileErr)).To(BeTrue())
			})

			It("should return false for any other error", func() {
				Expect(IsCreateFileError(err)).To(BeFalse())
				Expect(IsCreateFileError(fileExistsErr)).To(BeFalse())
				Expect(IsCreateFileError(openFileErr)).To(BeFalse())
				Expect(IsCreateFileError(createDirectoryErr)).To(BeFalse())
				Expect(IsCreateFileError(readFileErr)).To(BeFalse())
				Expect(IsCreateFileError(writeFileErr)).To(BeFalse())
				Expect(IsCreateFileError(closeFileErr)).To(BeFalse())
			})
		})

		Context("IsReadFileError", func() {
			It("should return true for read file errors", func() {
				Expect(IsReadFileError(readFileErr)).To(BeTrue())
			})

			It("should return false for any other error", func() {
				Expect(IsReadFileError(err)).To(BeFalse())
				Expect(IsReadFileError(fileExistsErr)).To(BeFalse())
				Expect(IsReadFileError(openFileErr)).To(BeFalse())
				Expect(IsReadFileError(createDirectoryErr)).To(BeFalse())
				Expect(IsReadFileError(createFileErr)).To(BeFalse())
				Expect(IsReadFileError(writeFileErr)).To(BeFalse())
				Expect(IsReadFileError(closeFileErr)).To(BeFalse())
			})
		})

		Context("IsWriteFileError", func() {
			It("should return true for write file errors", func() {
				Expect(IsWriteFileError(writeFileErr)).To(BeTrue())
			})

			It("should return false for any other error", func() {
				Expect(IsWriteFileError(err)).To(BeFalse())
				Expect(IsWriteFileError(fileExistsErr)).To(BeFalse())
				Expect(IsWriteFileError(openFileErr)).To(BeFalse())
				Expect(IsWriteFileError(createDirectoryErr)).To(BeFalse())
				Expect(IsWriteFileError(createFileErr)).To(BeFalse())
				Expect(IsWriteFileError(readFileErr)).To(BeFalse())
				Expect(IsWriteFileError(closeFileErr)).To(BeFalse())
			})
		})

		Context("IsCloseFileError", func() {
			It("should return true for close file errors", func() {
				Expect(IsCloseFileError(closeFileErr)).To(BeTrue())
			})

			It("should return false for any other error", func() {
				Expect(IsCloseFileError(err)).To(BeFalse())
				Expect(IsCloseFileError(fileExistsErr)).To(BeFalse())
				Expect(IsCloseFileError(openFileErr)).To(BeFalse())
				Expect(IsCloseFileError(createDirectoryErr)).To(BeFalse())
				Expect(IsCloseFileError(createFileErr)).To(BeFalse())
				Expect(IsCloseFileError(readFileErr)).To(BeFalse())
				Expect(IsCloseFileError(writeFileErr)).To(BeFalse())
			})
		})

		Describe("error messages", func() {
			It("should contain the wrapped err", func() {
				Expect(fileExistsErr.Error()).To(ContainSubstring(err.Error()))
				Expect(openFileErr.Error()).To(ContainSubstring(err.Error()))
				Expect(createDirectoryErr.Error()).To(ContainSubstring(err.Error()))
				Expect(createFileErr.Error()).To(ContainSubstring(err.Error()))
				Expect(readFileErr.Error()).To(ContainSubstring(err.Error()))
				Expect(writeFileErr.Error()).To(ContainSubstring(err.Error()))
				Expect(closeFileErr.Error()).To(ContainSubstring(err.Error()))
			})
		})
	})

	// NOTE: FileSystem.Exists, FileSystem.Open, FileSystem.Open().Read, FileSystem.Create and FileSystem.Create().Write
	// are hard to test in unitary tests as they deal with actual files
})
