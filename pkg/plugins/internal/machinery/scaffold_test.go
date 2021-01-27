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
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"github.com/spf13/afero"

	"sigs.k8s.io/kubebuilder/v3/pkg/model"
	"sigs.k8s.io/kubebuilder/v3/pkg/model/file"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugins/internal/filesystem"
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
				si = NewScaffold(afero.NewMemMapFs())
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
				si = NewScaffold(afero.NewMemMapFs(), fakePlugin{})
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
				si = NewScaffold(afero.NewMemMapFs(), fakePlugin{}, fakePlugin{}, fakePlugin{})
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
			output  bytes.Buffer
			testErr = errors.New("error text")
		)

		BeforeEach(func() {
			output.Reset()
		})

		DescribeTable("successes",
			func(expected string, files ...file.Builder) {
				s := &scaffold{
					fs: filesystem.NewMock(
						filesystem.MockOutput(&output),
					),
				}

				Expect(s.Execute(model.NewUniverse(), files...)).To(Succeed())
				Expect(output.String()).To(Equal(expected))
			},
			Entry("should write the file",
				fileContent,
				fakeTemplate{body: fileContent},
			),
			Entry("should skip optional models if already have one",
				fileContent,
				fakeTemplate{body: fileContent},
				fakeTemplate{},
			),
			Entry("should overwrite required models if already have one",
				fileContent,
				fakeTemplate{},
				fakeTemplate{fakeBuilder: fakeBuilder{ifExistsAction: file.Overwrite}, body: fileContent},
			),
			Entry("should format a go file",
				"package file\n",
				fakeTemplate{fakeBuilder: fakeBuilder{path: "file.go"}, body: "package    file"},
			),
		)

		DescribeTable("file builders related errors",
			func(f func(error) bool, files ...file.Builder) {
				s := &scaffold{fs: filesystem.NewMock()}

				Expect(f(s.Execute(model.NewUniverse(), files...))).To(BeTrue())
			},
			Entry("should fail if unable to validate a file builder",
				file.IsValidateError,
				fakeRequiresValidation{validateErr: testErr},
			),
			Entry("should fail if unable to set default values for a template",
				file.IsSetTemplateDefaultsError,
				fakeTemplate{err: testErr},
			),
			Entry("should fail if an unexpected previous model is found",
				IsModelAlreadyExistsError,
				fakeTemplate{fakeBuilder: fakeBuilder{path: "filename"}},
				fakeTemplate{fakeBuilder: fakeBuilder{path: "filename", ifExistsAction: file.Error}},
			),
			Entry("should fail if behavior if file exists is not defined",
				IsUnknownIfExistsActionError,
				fakeTemplate{fakeBuilder: fakeBuilder{path: "filename"}},
				fakeTemplate{fakeBuilder: fakeBuilder{path: "filename", ifExistsAction: -1}},
			),
		)

		// Following errors are unwrapped, so we need to check for substrings
		DescribeTable("template related errors",
			func(errMsg string, files ...file.Builder) {
				s := &scaffold{fs: filesystem.NewMock()}

				err := s.Execute(model.NewUniverse(), files...)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(errMsg))
			},
			Entry("should fail if a template is broken",
				"template: ",
				fakeTemplate{body: "{{ .Field }"},
			),
			Entry("should fail if a template params aren't provided",
				"template: ",
				fakeTemplate{body: "{{ .Field }}"},
			),
			Entry("should fail if unable to format a go file",
				"expected 'package', found ",
				fakeTemplate{fakeBuilder: fakeBuilder{path: "file.go"}, body: fileContent},
			),
		)

		DescribeTable("insert strings",
			func(input, expected string, files ...file.Builder) {
				s := &scaffold{
					fs: filesystem.NewMock(
						filesystem.MockInput(bytes.NewBufferString(input)),
						filesystem.MockOutput(&output),
						filesystem.MockExists(func(_ string) bool { return len(input) != 0 }),
					),
				}

				Expect(s.Execute(model.NewUniverse(), files...)).To(Succeed())
				Expect(output.String()).To(Equal(expected))
			},
			Entry("should insert lines for go files",
				`
//+kubebuilder:scaffold:-
`,
				`
1
2
//+kubebuilder:scaffold:-
`,
				fakeInserter{codeFragments: file.CodeFragmentsMap{
					file.NewMarkerFor("file.go", "-"): {"1\n", "2\n"}},
				},
			),
			Entry("should insert lines for yaml files",
				`
#+kubebuilder:scaffold:-
`,
				`
1
2
#+kubebuilder:scaffold:-
`,
				fakeInserter{codeFragments: file.CodeFragmentsMap{
					file.NewMarkerFor("file.yaml", "-"): {"1\n", "2\n"}},
				},
			),
			Entry("should use models if there is no file",
				"",
				`
1
2
//+kubebuilder:scaffold:-
`,
				fakeTemplate{fakeBuilder: fakeBuilder{ifExistsAction: file.Overwrite}, body: `
//+kubebuilder:scaffold:-
`},
				fakeInserter{codeFragments: file.CodeFragmentsMap{
					file.NewMarkerFor("file.go", "-"): {"1\n", "2\n"}},
				},
			),
			Entry("should use required models over files",
				fileContent,
				`
1
2
//+kubebuilder:scaffold:-
`,
				fakeTemplate{fakeBuilder: fakeBuilder{ifExistsAction: file.Overwrite}, body: `
//+kubebuilder:scaffold:-
`},
				fakeInserter{codeFragments: file.CodeFragmentsMap{
					file.NewMarkerFor("file.go", "-"): {"1\n", "2\n"}},
				},
			),
			Entry("should use files over optional models",
				`
//+kubebuilder:scaffold:-
`,
				`
1
2
//+kubebuilder:scaffold:-
`,
				fakeTemplate{body: fileContent},
				fakeInserter{
					codeFragments: file.CodeFragmentsMap{
						file.NewMarkerFor("file.go", "-"): {"1\n", "2\n"},
					},
				},
			),
			Entry("should filter invalid markers",
				`
//+kubebuilder:scaffold:-
//+kubebuilder:scaffold:*
`,
				`
1
2
//+kubebuilder:scaffold:-
//+kubebuilder:scaffold:*
`,
				fakeInserter{
					markers: []file.Marker{file.NewMarkerFor("file.go", "-")},
					codeFragments: file.CodeFragmentsMap{
						file.NewMarkerFor("file.go", "-"): {"1\n", "2\n"},
						file.NewMarkerFor("file.go", "*"): {"3\n", "4\n"},
					},
				},
			),
			Entry("should filter already existing one-line code fragments",
				`
1
//+kubebuilder:scaffold:-
3
4
//+kubebuilder:scaffold:*
`,
				`
1
2
//+kubebuilder:scaffold:-
3
4
//+kubebuilder:scaffold:*
`,
				fakeInserter{
					codeFragments: file.CodeFragmentsMap{
						file.NewMarkerFor("file.go", "-"): {"1\n", "2\n"},
						file.NewMarkerFor("file.go", "*"): {"3\n", "4\n"},
					},
				},
			),
			Entry("should not insert anything if no code fragment",
				"", // input is provided through a template as mock fs doesn't copy it to the output buffer if no-op
				`
//+kubebuilder:scaffold:-
`,
				fakeTemplate{body: `
//+kubebuilder:scaffold:-
`},
				fakeInserter{
					codeFragments: file.CodeFragmentsMap{
						file.NewMarkerFor("file.go", "-"): {},
					},
				},
			),
		)

		DescribeTable("insert strings related errors",
			func(f func(error) bool, files ...file.Builder) {
				s := &scaffold{
					fs: filesystem.NewMock(
						filesystem.MockExists(func(_ string) bool { return true }),
					),
				}

				err := s.Execute(model.NewUniverse(), files...)
				Expect(err).To(HaveOccurred())
				Expect(f(err)).To(BeTrue())
			},
			Entry("should fail if inserting into a model that fails when a file exists and it does exist",
				IsFileAlreadyExistsError,
				fakeTemplate{fakeBuilder: fakeBuilder{path: "filename", ifExistsAction: file.Error}},
				fakeInserter{fakeBuilder: fakeBuilder{path: "filename"}},
			),
			Entry("should fail if inserting into a model with unknown behavior if the file exists and it does exist",
				IsUnknownIfExistsActionError,
				fakeTemplate{fakeBuilder: fakeBuilder{path: "filename", ifExistsAction: -1}},
				fakeInserter{fakeBuilder: fakeBuilder{path: "filename"}},
			),
		)

		It("should fail if a plugin fails", func() {
			s := &scaffold{
				fs:      filesystem.NewMock(),
				plugins: []model.Plugin{fakePlugin{err: testErr}},
			}

			err := s.Execute(
				model.NewUniverse(),
				fakeTemplate{},
			)
			Expect(err).To(MatchError(testErr))
			Expect(model.IsPluginError(err)).To(BeTrue())
		})

		Context("write when the file already exists", func() {
			var s Scaffold

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
					fakeTemplate{body: fileContent},
				)).To(Succeed())
				Expect(output.String()).To(BeEmpty())
			})

			It("should write the file if configured to do so", func() {
				Expect(s.Execute(
					model.NewUniverse(),
					fakeTemplate{fakeBuilder: fakeBuilder{ifExistsAction: file.Overwrite}, body: fileContent},
				)).To(Succeed())
				Expect(output.String()).To(Equal(fileContent))
			})

			It("should error if configured to do so", func() {
				err := s.Execute(
					model.NewUniverse(),
					fakeTemplate{fakeBuilder: fakeBuilder{path: "filename", ifExistsAction: file.Error}, body: fileContent},
				)
				Expect(err).To(HaveOccurred())
				Expect(IsFileAlreadyExistsError(err)).To(BeTrue())
				Expect(output.String()).To(BeEmpty())
			})
		})

		DescribeTable("filesystem errors",
			func(
				mockErrorF func(error) filesystem.MockOptions,
				checkErrorF func(error) bool,
				files ...file.Builder,
			) {
				s := &scaffold{
					fs: filesystem.NewMock(
						mockErrorF(testErr),
					),
				}

				err := s.Execute(model.NewUniverse(), files...)
				Expect(err).To(HaveOccurred())
				Expect(checkErrorF(err)).To(BeTrue())
			},
			Entry("should fail if fs.Exists failed (at file writing)",
				filesystem.MockExistsError, filesystem.IsFileExistsError,
				fakeTemplate{},
			),
			Entry("should fail if fs.Exists failed (at model updating)",
				filesystem.MockExistsError, filesystem.IsFileExistsError,
				fakeTemplate{},
				fakeInserter{},
			),
			Entry("should fail if fs.Open was unable to open the file",
				filesystem.MockOpenFileError, filesystem.IsOpenFileError,
				fakeInserter{},
			),
			Entry("should fail if fs.Open().Read was unable to read the file",
				filesystem.MockReadFileError, filesystem.IsReadFileError,
				fakeInserter{},
			),
			Entry("should fail if fs.Open().Close was unable to close the file",
				filesystem.MockCloseFileError, filesystem.IsCloseFileError,
				fakeInserter{},
			),
			Entry("should fail if fs.Create was unable to create the directory",
				filesystem.MockCreateDirError, filesystem.IsCreateDirectoryError,
				fakeTemplate{},
			),
			Entry("should fail if fs.Create was unable to create the file",
				filesystem.MockCreateFileError, filesystem.IsCreateFileError,
				fakeTemplate{},
			),
			Entry("should fail if fs.Create().Write was unable to write the file",
				filesystem.MockWriteFileError, filesystem.IsWriteFileError,
				fakeTemplate{},
			),
			Entry("should fail if fs.Create().Write was unable to close the file",
				filesystem.MockCloseFileError, filesystem.IsCloseFileError,
				fakeTemplate{},
			),
		)
	})
})

var _ model.Plugin = fakePlugin{}

// fakePlugin is used to mock a model.Plugin in order to test Scaffold
type fakePlugin struct {
	err error
}

// Pipe implements model.Plugin
func (f fakePlugin) Pipe(_ *model.Universe) error {
	return f.err
}

var _ file.Builder = fakeBuilder{}

// fakeBuilder is used to mock a file.Builder
type fakeBuilder struct {
	path           string
	ifExistsAction file.IfExistsAction
}

// GetPath implements file.Builder
func (f fakeBuilder) GetPath() string {
	return f.path
}

// GetIfExistsAction implements file.Builder
func (f fakeBuilder) GetIfExistsAction() file.IfExistsAction {
	return f.ifExistsAction
}

var _ file.RequiresValidation = fakeRequiresValidation{}

// fakeRequiresValidation is used to mock a file.RequiresValidation in order to test Scaffold
type fakeRequiresValidation struct {
	fakeBuilder

	validateErr error
}

// Validate implements file.RequiresValidation
func (f fakeRequiresValidation) Validate() error {
	return f.validateErr
}

var _ file.Template = fakeTemplate{}

// fakeTemplate is used to mock a file.File in order to test Scaffold
type fakeTemplate struct {
	fakeBuilder

	body string
	err  error
}

// GetBody implements file.Template
func (f fakeTemplate) GetBody() string {
	return f.body
}

// SetTemplateDefaults implements file.Template
func (f fakeTemplate) SetTemplateDefaults() error {
	if f.err != nil {
		return f.err
	}

	return nil
}

type fakeInserter struct {
	fakeBuilder

	markers       []file.Marker
	codeFragments file.CodeFragmentsMap
}

// GetMarkers implements file.UpdatableTemplate
func (f fakeInserter) GetMarkers() []file.Marker {
	if f.markers != nil {
		return f.markers
	}

	markers := make([]file.Marker, 0, len(f.codeFragments))
	for marker := range f.codeFragments {
		markers = append(markers, marker)
	}
	return markers
}

// GetCodeFragments implements file.UpdatableTemplate
func (f fakeInserter) GetCodeFragments() file.CodeFragmentsMap {
	return f.codeFragments
}
