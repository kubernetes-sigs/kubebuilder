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
	"os"
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

	// NOTE: FileSystem.Exists, FileSystem.Open, FileSystem.Open().Read, FileSystem.Create and FileSystem.Create().Write
	// are hard to test in unitary tests as they deal with actual files
})
