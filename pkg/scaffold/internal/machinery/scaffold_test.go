/*
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
	"errors"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"sigs.k8s.io/kubebuilder/pkg/model"
	"sigs.k8s.io/kubebuilder/pkg/model/config"
	"sigs.k8s.io/kubebuilder/pkg/model/file"
	"sigs.k8s.io/kubebuilder/pkg/scaffold/internal/filesystem"
)

func TestScaffold(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Scaffold suite")
}

var _ = Describe("Scaffold", func() {
	Describe("NewScaffold", func() {
		var (
			si Scaffold
			s  *scaffold
			ok bool
		)

		Context("when using no plugins", func() {
			BeforeEach(func() {
				si = NewScaffold()
				s, ok = si.(*scaffold)
			})

			It("should be a scaffold instance", func() {
				Expect(ok).To(BeTrue())
			})

			It("should not have a nil fs", func() {
				Expect(s.fs).NotTo(BeNil())
			})

			It("should not have any plugin", func() {
				Expect(len(s.plugins)).To(Equal(0))
			})
		})

		Context("when using one plugin", func() {
			BeforeEach(func() {
				si = NewScaffold(fakePlugin{})
				s, ok = si.(*scaffold)
			})

			It("should be a scaffold instance", func() {
				Expect(ok).To(BeTrue())
			})

			It("should not have a nil fs", func() {
				Expect(s.fs).NotTo(BeNil())
			})

			It("should have one plugin", func() {
				Expect(len(s.plugins)).To(Equal(1))
			})
		})

		Context("when using several plugins", func() {
			BeforeEach(func() {
				si = NewScaffold(fakePlugin{}, fakePlugin{}, fakePlugin{})
				s, ok = si.(*scaffold)
			})

			It("should be a scaffold instance", func() {
				Expect(ok).To(BeTrue())
			})

			It("should not have a nil fs", func() {
				Expect(s.fs).NotTo(BeNil())
			})

			It("should have several plugins", func() {
				Expect(len(s.plugins)).To(Equal(3))
			})
		})
	})

	Describe("Scaffold.Execute", func() {
		const fileContent = "Hello world!"

		var (
			s         Scaffold
			output    bytes.Buffer
			testError = errors.New("error text")
		)

		BeforeEach(func() {
			output.Reset()
		})

		It("should write the file", func() {
			s = &scaffold{
				fs: filesystem.NewMock(
					filesystem.MockOutput(&output),
				),
			}

			Expect(s.Execute(
				model.NewUniverse(
					model.WithConfig(&config.Config{}),
				),
				fakeFile{
					body: fileContent,
				},
			)).To(Succeed())
			Expect(output.String()).To(Equal(fileContent))
		})

		It("should fail if a plugin fails", func() {
			s = &scaffold{
				fs: filesystem.NewMock(
					filesystem.MockOutput(&output),
				),
				plugins: []model.Plugin{fakePlugin{err: testError}},
			}

			err := s.Execute(
				model.NewUniverse(),
				fakeFile{
					body: fileContent,
				},
			)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(testError.Error()))
		})

		It("should fail if a template validation fails", func() {
			s = &scaffold{
				fs: filesystem.NewMock(
					filesystem.MockOutput(&output),
				),
			}

			err := s.Execute(
				model.NewUniverse(),
				fakeFile{
					body:          fileContent,
					validateError: testError,
				},
			)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(testError.Error()))
		})

		It("should fail if a template GetTemplateMixin method fails", func() {
			s = &scaffold{
				fs: filesystem.NewMock(
					filesystem.MockOutput(&output),
				),
			}

			err := s.Execute(
				model.NewUniverse(),
				fakeFile{
					body: fileContent,
					err:  testError,
				},
			)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(testError.Error()))
		})

		It("should fail if a template is broken", func() {
			s = &scaffold{
				fs: filesystem.NewMock(
					filesystem.MockOutput(&output),
				),
			}

			err := s.Execute(
				model.NewUniverse(),
				fakeFile{
					body: fileContent + "{{ .Field }",
				},
			)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("template: "))
		})

		It("should fail if a template params aren't provided", func() {
			s = &scaffold{
				fs: filesystem.NewMock(
					filesystem.MockOutput(&output),
				),
			}

			err := s.Execute(
				model.NewUniverse(),
				fakeFile{
					body: fileContent + "{{ .Field }}",
				},
			)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("template: "))
		})

		It("should format a go file", func() {
			s = &scaffold{
				fs: filesystem.NewMock(
					filesystem.MockOutput(&output),
				),
			}

			Expect(s.Execute(
				model.NewUniverse(),
				fakeFile{
					path: "file.go",
					body: "package file",
				},
			)).To(Succeed())
			Expect(output.String()).To(Equal("package file\n"))
		})

		It("should fail if unable to format a go file", func() {
			s = &scaffold{
				fs: filesystem.NewMock(
					filesystem.MockOutput(&output),
				),
			}

			err := s.Execute(
				model.NewUniverse(),
				fakeFile{
					path: "file.go",
					body: fileContent,
				},
			)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("expected 'package', found "))
		})

		Context("when the file already exists", func() {
			BeforeEach(func() {
				s = &scaffold{
					fs: filesystem.NewMock(
						filesystem.MockExists(func(_ string) bool { return true }),
						filesystem.MockOutput(&output),
					),
				}
			})

			It("should skip the file by default", func() {
				Expect(s.Execute(
					model.NewUniverse(),
					fakeFile{
						body: fileContent,
					},
				)).To(Succeed())
				Expect(output.String()).To(BeEmpty())
			})

			It("should write the file if configured to do so", func() {
				Expect(s.Execute(
					model.NewUniverse(),
					fakeFile{
						body:           fileContent,
						ifExistsAction: file.Overwrite,
					},
				)).To(Succeed())
				Expect(output.String()).To(Equal(fileContent))
			})

			It("should error if configured to do so", func() {
				err := s.Execute(
					model.NewUniverse(),
					fakeFile{
						path:           "filename",
						body:           fileContent,
						ifExistsAction: file.Error,
					},
				)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("failed to create filename: file already exists"))
				Expect(output.String()).To(BeEmpty())
			})
		})

		Context("when the filesystem returns an error", func() {

			It("should fail if fs.Exists failed", func() {
				s = &scaffold{
					fs: filesystem.NewMock(
						filesystem.MockExistsError(testError),
					),
				}

				err := s.Execute(
					model.NewUniverse(),
					fakeFile{
						body: fileContent,
					},
				)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(testError.Error()))
			})

			It("should fail if fs.Create was unable to create the directory", func() {
				s = &scaffold{
					fs: filesystem.NewMock(
						filesystem.MockCreateDirError(testError),
					),
				}

				err := s.Execute(
					model.NewUniverse(),
					fakeFile{
						body: fileContent,
					},
				)
				Expect(err).To(HaveOccurred())
				Expect(filesystem.IsCreateDirectoryError(err)).To(BeTrue())
			})

			It("should fail if fs.Create was unable to create the file", func() {
				s = &scaffold{
					fs: filesystem.NewMock(
						filesystem.MockCreateFileError(testError),
					),
				}

				err := s.Execute(
					model.NewUniverse(),
					fakeFile{
						body: fileContent,
					},
				)
				Expect(err).To(HaveOccurred())
				Expect(filesystem.IsCreateFileError(err)).To(BeTrue())
			})

			It("should fail if fs.Create().Write was unable to write the file", func() {
				s = &scaffold{
					fs: filesystem.NewMock(
						filesystem.MockWriteFileError(testError),
					),
				}

				err := s.Execute(
					model.NewUniverse(),
					fakeFile{
						body: fileContent,
					},
				)
				Expect(err).To(HaveOccurred())
				Expect(filesystem.IsWriteFileError(err)).To(BeTrue())
			})

			It("should fail if fs.Create().Write was unable to close the file", func() {
				s = &scaffold{
					fs: filesystem.NewMock(
						filesystem.MockCloseFileError(testError),
					),
				}

				err := s.Execute(
					model.NewUniverse(),
					fakeFile{
						body: fileContent,
					},
				)
				Expect(err).To(HaveOccurred())
				Expect(filesystem.IsCloseFileError(err)).To(BeTrue())
			})
		})
	})
})

// fakePlugin is used to mock a model.Plugin in order to test Scaffold
type fakePlugin struct {
	err error
}

// Pipe implements model.Plugin
func (f fakePlugin) Pipe(_ *model.Universe) error {
	return f.err
}

// fakeFile is used to mock a file.File in order to test Scaffold
type fakeFile struct {
	path           string
	body           string
	ifExistsAction file.IfExistsAction

	err           error
	validateError error
}

// GetTemplateMixin implements file.Template
func (f fakeFile) GetTemplateMixin() (file.TemplateMixin, error) {
	if f.err != nil {
		return file.TemplateMixin{}, f.err
	}

	return file.TemplateMixin{f.path, f.body, f.ifExistsAction}, nil
}

// GetPath implements file.Template
func (f fakeFile) GetPath() string {
	return f.path
}

// GetBody implements file.Template
func (f fakeFile) GetBody() string {
	return f.body
}

// GetIfExistsAction implements file.Template
func (f fakeFile) GetIfExistsAction() file.IfExistsAction {
	return f.ifExistsAction
}

// Validate implements file.RequiresValidation
func (f fakeFile) Validate() error {
	return f.validateError
}
