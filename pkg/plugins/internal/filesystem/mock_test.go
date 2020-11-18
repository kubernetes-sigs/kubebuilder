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
	"bytes"
	"errors"
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestMockFileSystem(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "MockFileSystem suite")
}

//nolint:dupl
var _ = Describe("MockFileSystem", func() {
	var (
		fsi     FileSystem
		fs      mockFileSystem
		ok      bool
		options []MockOptions
		testErr = errors.New("test error")
	)

	JustBeforeEach(func() {
		fsi = NewMock(options...)
		fs, ok = fsi.(mockFileSystem)
	})

	Context("when using no options", func() {
		BeforeEach(func() {
			options = make([]MockOptions, 0)
		})

		It("should be a mockFileSystem instance", func() {
			Expect(ok).To(BeTrue())
		})

		It("should claim that files don't exist", func() {
			exists, err := fsi.Exists("")
			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeFalse())
		})

		It("should open readable files", func() {
			f, err := fsi.Open("")
			Expect(err).NotTo(HaveOccurred())

			_, err = f.Read([]byte(""))
			Expect(err).NotTo(HaveOccurred())
		})

		It("should create writable files", func() {
			f, err := fsi.Create("")
			Expect(err).NotTo(HaveOccurred())

			_, err = f.Write([]byte(""))
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("when using MockPath", func() {
		var filePath = filepath.Join("path", "to", "file")

		BeforeEach(func() {
			options = []MockOptions{MockPath(filePath)}
		})

		It("should be a mockFileSystem instance", func() {
			Expect(ok).To(BeTrue())
		})

		It("should claim that files don't exist", func() {
			exists, err := fsi.Exists("")
			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeFalse())
		})

		It("should open readable files", func() {
			f, err := fsi.Open("")
			Expect(err).NotTo(HaveOccurred())

			_, err = f.Read([]byte(""))
			Expect(err).NotTo(HaveOccurred())
		})

		It("should create writable files", func() {
			f, err := fsi.Create("")
			Expect(err).NotTo(HaveOccurred())

			_, err = f.Write([]byte(""))
			Expect(err).NotTo(HaveOccurred())
		})

		It("should save the provided path", func() {
			Expect(fs.path).To(Equal(filePath))
		})
	})

	Context("when using MockExists", func() {
		BeforeEach(func() {
			options = []MockOptions{MockExists(func(_ string) bool { return true })}
		})

		It("should be a mockFileSystem instance", func() {
			Expect(ok).To(BeTrue())
		})

		It("should claim that files exist", func() {
			exists, err := fsi.Exists("")
			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeTrue())
		})

		It("should open readable files", func() {
			f, err := fsi.Open("")
			Expect(err).NotTo(HaveOccurred())

			_, err = f.Read([]byte(""))
			Expect(err).NotTo(HaveOccurred())
		})

		It("should create writable files", func() {
			f, err := fsi.Create("")
			Expect(err).NotTo(HaveOccurred())

			_, err = f.Write([]byte(""))
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("when using MockExistsError", func() {
		BeforeEach(func() {
			options = []MockOptions{MockExistsError(testErr)}
		})

		It("should be a mockFileSystem instance", func() {
			Expect(ok).To(BeTrue())
		})

		It("should error when calling Exists", func() {
			_, err := fsi.Exists("")
			Expect(err).To(MatchError(testErr))
			Expect(IsFileExistsError(err)).To(BeTrue())
		})

		It("should open readable files", func() {
			f, err := fsi.Open("")
			Expect(err).NotTo(HaveOccurred())

			_, err = f.Read([]byte(""))
			Expect(err).NotTo(HaveOccurred())
		})

		It("should create writable files", func() {
			f, err := fsi.Create("")
			Expect(err).NotTo(HaveOccurred())

			_, err = f.Write([]byte(""))
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("when using MockOpenFileError", func() {
		BeforeEach(func() {
			options = []MockOptions{MockOpenFileError(testErr)}
		})

		It("should be a mockFileSystem instance", func() {
			Expect(ok).To(BeTrue())
		})

		It("should claim that files don't exist", func() {
			exists, err := fsi.Exists("")
			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeFalse())
		})

		It("should error when calling Open", func() {
			_, err := fsi.Open("")
			Expect(err).To(MatchError(testErr))
			Expect(IsOpenFileError(err)).To(BeTrue())
		})

		It("should create writable files", func() {
			f, err := fsi.Create("")
			Expect(err).NotTo(HaveOccurred())

			_, err = f.Write([]byte(""))
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("when using MockCreateDirError", func() {
		BeforeEach(func() {
			options = []MockOptions{MockCreateDirError(testErr)}
		})

		It("should be a mockFileSystem instance", func() {
			Expect(ok).To(BeTrue())
		})

		It("should claim that files don't exist", func() {
			exists, err := fsi.Exists("")
			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeFalse())
		})

		It("should open readable files", func() {
			f, err := fsi.Open("")
			Expect(err).NotTo(HaveOccurred())

			_, err = f.Read([]byte(""))
			Expect(err).NotTo(HaveOccurred())
		})

		It("should error when calling Create", func() {
			_, err := fsi.Create("")
			Expect(err).To(MatchError(testErr))
			Expect(IsCreateDirectoryError(err)).To(BeTrue())
		})
	})

	Context("when using MockCreateFileError", func() {
		BeforeEach(func() {
			options = []MockOptions{MockCreateFileError(testErr)}
		})

		It("should be a mockFileSystem instance", func() {
			Expect(ok).To(BeTrue())
		})

		It("should claim that files don't exist", func() {
			exists, err := fsi.Exists("")
			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeFalse())
		})

		It("should open readable files", func() {
			f, err := fsi.Open("")
			Expect(err).NotTo(HaveOccurred())

			_, err = f.Read([]byte(""))
			Expect(err).NotTo(HaveOccurred())
		})

		It("should error when calling Create", func() {
			_, err := fsi.Create("")
			Expect(err).To(MatchError(testErr))
			Expect(IsCreateFileError(err)).To(BeTrue())
		})
	})

	Context("when using MockInput", func() {
		var (
			input       *bytes.Buffer
			fileContent = []byte("Hello world!")
		)

		BeforeEach(func() {
			input = bytes.NewBufferString("Hello world!")
			options = []MockOptions{MockInput(input)}
		})

		It("should be a mockFileSystem instance", func() {
			Expect(ok).To(BeTrue())
		})

		It("should claim that files don't exist", func() {
			exists, err := fsi.Exists("")
			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeFalse())
		})

		It("should open readable files and the content to be accessible", func() {
			f, err := fsi.Open("")
			Expect(err).NotTo(HaveOccurred())

			output := make([]byte, len(fileContent))
			n, err := f.Read(output)
			Expect(err).NotTo(HaveOccurred())
			Expect(n).To(Equal(len(fileContent)))
			Expect(output).To(Equal(fileContent))
		})

		It("should create writable files", func() {
			f, err := fsi.Create("")
			Expect(err).NotTo(HaveOccurred())

			_, err = f.Write([]byte(""))
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("when using MockReadFileError", func() {
		BeforeEach(func() {
			options = []MockOptions{MockReadFileError(testErr)}
		})

		It("should be a mockFileSystem instance", func() {
			Expect(ok).To(BeTrue())
		})

		It("should claim that files don't exist", func() {
			exists, err := fsi.Exists("")
			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeFalse())
		})

		It("should error when calling Open().Read", func() {
			f, err := fsi.Open("")
			Expect(err).NotTo(HaveOccurred())

			output := make([]byte, 0)
			_, err = f.Read(output)
			Expect(err).To(MatchError(testErr))
			Expect(IsReadFileError(err)).To(BeTrue())
		})

		It("should create writable files", func() {
			f, err := fsi.Create("")
			Expect(err).NotTo(HaveOccurred())

			_, err = f.Write([]byte(""))
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("when using MockOutput", func() {
		var (
			output      bytes.Buffer
			fileContent = []byte("Hello world!")
		)

		BeforeEach(func() {
			options = []MockOptions{MockOutput(&output)}
			output.Reset()
		})

		It("should be a mockFileSystem instance", func() {
			Expect(ok).To(BeTrue())
		})

		It("should claim that files don't exist", func() {
			exists, err := fsi.Exists("")
			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeFalse())
		})

		It("should open readable files", func() {
			f, err := fsi.Open("")
			Expect(err).NotTo(HaveOccurred())

			_, err = f.Read([]byte(""))
			Expect(err).NotTo(HaveOccurred())
		})

		It("should create writable files and the content should be accesible", func() {
			f, err := fsi.Create("")
			Expect(err).NotTo(HaveOccurred())

			n, err := f.Write(fileContent)
			Expect(err).NotTo(HaveOccurred())
			Expect(n).To(Equal(len(fileContent)))
			Expect(output.Bytes()).To(Equal(fileContent))
		})
	})

	Context("when using MockWriteFileError", func() {
		BeforeEach(func() {
			options = []MockOptions{MockWriteFileError(testErr)}
		})

		It("should be a mockFileSystem instance", func() {
			Expect(ok).To(BeTrue())
		})

		It("should claim that files don't exist", func() {
			exists, err := fsi.Exists("")
			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeFalse())
		})

		It("should open readable files", func() {
			f, err := fsi.Open("")
			Expect(err).NotTo(HaveOccurred())

			_, err = f.Read([]byte(""))
			Expect(err).NotTo(HaveOccurred())
		})

		It("should error when calling Create().Write", func() {
			f, err := fsi.Create("")
			Expect(err).NotTo(HaveOccurred())

			_, err = f.Write([]byte(""))
			Expect(err).To(MatchError(testErr))
			Expect(IsWriteFileError(err)).To(BeTrue())
		})
	})

	Context("when using MockCloseFileError", func() {
		BeforeEach(func() {
			options = []MockOptions{MockCloseFileError(testErr)}
		})

		It("should be a mockFileSystem instance", func() {
			Expect(ok).To(BeTrue())
		})

		It("should claim that files don't exist", func() {
			exists, err := fsi.Exists("")
			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeFalse())
		})

		It("should error when calling Open().Close", func() {
			f, err := fsi.Open("")
			Expect(err).NotTo(HaveOccurred())

			err = f.Close()
			Expect(err).To(MatchError(testErr))
			Expect(IsCloseFileError(err)).To(BeTrue())
		})

		It("should error when calling Create().Write", func() {
			f, err := fsi.Create("")
			Expect(err).NotTo(HaveOccurred())

			_, err = f.Write([]byte(""))
			Expect(err).To(MatchError(testErr))
			Expect(IsCloseFileError(err)).To(BeTrue())
		})
	})
})
