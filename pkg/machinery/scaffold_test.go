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
	"errors"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/afero"

	cfgv3 "sigs.k8s.io/kubebuilder/v4/pkg/config/v3"
	"sigs.k8s.io/kubebuilder/v4/pkg/model/resource"
)

var _ = Describe("Scaffold", func() {
	Describe("NewScaffold", func() {
		It("should succeed for no option", func() {
			s := NewScaffold(Filesystem{FS: afero.NewMemMapFs()})
			Expect(s.fs).NotTo(BeNil())
			Expect(s.dirPerm).To(Equal(defaultDirectoryPermission))
			Expect(s.filePerm).To(Equal(defaultFilePermission))
			Expect(s.injector.config).To(BeNil())
			Expect(s.injector.boilerplate).To(Equal(""))
			Expect(s.injector.resource).To(BeNil())
		})

		It("should succeed with directory permissions option", func() {
			const dirPermissions os.FileMode = 0o755

			s := NewScaffold(Filesystem{FS: afero.NewMemMapFs()}, WithDirectoryPermissions(dirPermissions))
			Expect(s.fs).NotTo(BeNil())
			Expect(s.dirPerm).To(Equal(dirPermissions))
			Expect(s.filePerm).To(Equal(defaultFilePermission))
			Expect(s.injector.config).To(BeNil())
			Expect(s.injector.boilerplate).To(Equal(""))
			Expect(s.injector.resource).To(BeNil())
		})

		It("should succeed with file permissions option", func() {
			const filePermissions os.FileMode = 0o755

			s := NewScaffold(Filesystem{FS: afero.NewMemMapFs()}, WithFilePermissions(filePermissions))
			Expect(s.fs).NotTo(BeNil())
			Expect(s.dirPerm).To(Equal(defaultDirectoryPermission))
			Expect(s.filePerm).To(Equal(filePermissions))
			Expect(s.injector.config).To(BeNil())
			Expect(s.injector.boilerplate).To(Equal(""))
			Expect(s.injector.resource).To(BeNil())
		})

		It("should succeed with config option", func() {
			cfg := cfgv3.New()

			s := NewScaffold(Filesystem{FS: afero.NewMemMapFs()}, WithConfig(cfg))
			Expect(s.fs).NotTo(BeNil())
			Expect(s.dirPerm).To(Equal(defaultDirectoryPermission))
			Expect(s.filePerm).To(Equal(defaultFilePermission))
			Expect(s.injector.config).NotTo(BeNil())
			Expect(s.injector.config.GetVersion().Compare(cfgv3.Version)).To(Equal(0))
			Expect(s.injector.boilerplate).To(Equal(""))
			Expect(s.injector.resource).To(BeNil())
		})

		It("should succeed with boilerplate option", func() {
			const boilerplate = "Copyright"

			s := NewScaffold(Filesystem{FS: afero.NewMemMapFs()}, WithBoilerplate(boilerplate))
			Expect(s.fs).NotTo(BeNil())
			Expect(s.dirPerm).To(Equal(defaultDirectoryPermission))
			Expect(s.filePerm).To(Equal(defaultFilePermission))
			Expect(s.injector.config).To(BeNil())
			Expect(s.injector.boilerplate).To(Equal(boilerplate))
			Expect(s.injector.resource).To(BeNil())
		})

		It("should succeed with resource option", func() {
			res := &resource.Resource{GVK: resource.GVK{
				Group:   "group",
				Domain:  "my.domain",
				Version: "v1",
				Kind:    "Kind",
			}}

			s := NewScaffold(Filesystem{FS: afero.NewMemMapFs()}, WithResource(res))
			Expect(s.fs).NotTo(BeNil())
			Expect(s.dirPerm).To(Equal(defaultDirectoryPermission))
			Expect(s.filePerm).To(Equal(defaultFilePermission))
			Expect(s.injector.config).To(BeNil())
			Expect(s.injector.boilerplate).To(Equal(""))
			Expect(s.injector.resource).NotTo(BeNil())
			Expect(s.injector.resource.GVK.IsEqualTo(res.GVK)).To(BeTrue())
		})
	})

	Describe("Scaffold.Execute", func() {
		const (
			path     = "filename"
			pathGo   = path + ".go"
			pathYaml = path + ".yaml"
			content  = "Hello world!"
		)

		var (
			testErr = errors.New("error text")

			s *Scaffold
		)

		BeforeEach(func() {
			s = &Scaffold{fs: afero.NewMemMapFs()}
		})

		DescribeTable("successes",
			func(path, expected string, files ...Builder) {
				Expect(s.Execute(files...)).To(Succeed())

				b, err := afero.ReadFile(s.fs, path)
				Expect(err).NotTo(HaveOccurred())
				Expect(string(b)).To(Equal(expected))
			},
			Entry("should write the file",
				path, content,
				&fakeTemplate{fakeBuilder: fakeBuilder{path: path}, body: content},
			),
			Entry("should skip optional models if already have one",
				path, content,
				&fakeTemplate{fakeBuilder: fakeBuilder{path: path}, body: content},
				&fakeTemplate{fakeBuilder: fakeBuilder{path: path}},
			),
			Entry("should overwrite required models if already have one",
				path, content,
				&fakeTemplate{fakeBuilder: fakeBuilder{path: path}},
				&fakeTemplate{fakeBuilder: fakeBuilder{path: path, ifExistsAction: OverwriteFile}, body: content},
			),
			Entry("should format a go file",
				pathGo, "package file\n",
				&fakeTemplate{fakeBuilder: fakeBuilder{path: pathGo}, body: "package    file"},
			),

			Entry("should render actions correctly",
				path, "package testValue",
				&fakeTemplate{fakeBuilder: fakeBuilder{path: path, TestField: "testValue"}, body: "package {{.TestField}}"},
			),

			Entry("should render actions with alternative delimiters correctly",
				path, "package testValue",
				&fakeTemplate{fakeBuilder: fakeBuilder{path: path, TestField: "testValue"},
					body: "package [[.TestField]]", parseDelimLeft: "[[", parseDelimRight: "]]"},
			),
		)

		DescribeTable("file builders related errors",
			func(errType interface{}, files ...Builder) {
				err := s.Execute(files...)
				Expect(err).To(HaveOccurred())
				Expect(errors.As(err, errType)).To(BeTrue())
			},
			Entry("should fail if unable to validate a file builder",
				&ValidateError{},
				fakeRequiresValidation{validateErr: testErr},
			),
			Entry("should fail if unable to set default values for a template",
				&SetTemplateDefaultsError{},
				&fakeTemplate{err: testErr},
			),
			Entry("should fail if an unexpected previous model is found",
				&ModelAlreadyExistsError{},
				&fakeTemplate{fakeBuilder: fakeBuilder{path: path}},
				&fakeTemplate{fakeBuilder: fakeBuilder{path: path, ifExistsAction: Error}},
			),
			Entry("should fail if behavior if-exists-action is not defined",
				&UnknownIfExistsActionError{},
				&fakeTemplate{fakeBuilder: fakeBuilder{path: path}},
				&fakeTemplate{fakeBuilder: fakeBuilder{path: path, ifExistsAction: -1}},
			),
		)

		// Following errors are unwrapped, so we need to check for substrings
		DescribeTable("template related errors",
			func(errMsg string, files ...Builder) {
				err := s.Execute(files...)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(errMsg))
			},
			Entry("should fail if a template is broken",
				"template: ",
				&fakeTemplate{body: "{{ .Field }"},
			),
			Entry("should fail if a template params aren't provided",
				"template: ",
				&fakeTemplate{body: "{{ .Field }}"},
			),
			Entry("should fail if unable to format a go file",
				"expected 'package', found ",
				&fakeTemplate{fakeBuilder: fakeBuilder{path: pathGo}, body: content},
			),
		)

		DescribeTable("insert strings",
			func(path, input, expected string, files ...Builder) {
				Expect(afero.WriteFile(s.fs, path, []byte(input), 0o666)).To(Succeed())

				Expect(s.Execute(files...)).To(Succeed())

				b, err := afero.ReadFile(s.fs, path)
				Expect(err).NotTo(HaveOccurred())
				Expect(string(b)).To(Equal(expected))
			},
			Entry("should insert lines for go files",
				pathGo,
				`package test

// +kubebuilder:scaffold:-
`,
				`package test

var a int
var b int

// +kubebuilder:scaffold:-
`,
				fakeInserter{
					fakeBuilder: fakeBuilder{path: pathGo},
					codeFragments: CodeFragmentsMap{
						NewMarkerFor(pathGo, "-"): {"var a int\n", "var b int\n"},
					},
				},
			),
			Entry("should insert lines for yaml files",
				pathYaml,
				`
# +kubebuilder:scaffold:-
`,
				`
1
2
# +kubebuilder:scaffold:-
`,
				fakeInserter{
					fakeBuilder: fakeBuilder{path: pathYaml},
					codeFragments: CodeFragmentsMap{
						NewMarkerFor(pathYaml, "-"): {"1\n", "2\n"},
					},
				},
			),
			Entry("should use models if there is no file",
				pathYaml,
				"",
				`
1
2
# +kubebuilder:scaffold:-
`,
				&fakeTemplate{fakeBuilder: fakeBuilder{path: pathYaml, ifExistsAction: OverwriteFile}, body: `
# +kubebuilder:scaffold:-
`},
				fakeInserter{
					fakeBuilder: fakeBuilder{path: pathYaml},
					codeFragments: CodeFragmentsMap{
						NewMarkerFor(pathYaml, "-"): {"1\n", "2\n"},
					},
				},
			),
			Entry("should use required models over files",
				pathYaml,
				content,
				`
1
2
# +kubebuilder:scaffold:-
`,
				&fakeTemplate{fakeBuilder: fakeBuilder{path: pathYaml, ifExistsAction: OverwriteFile}, body: `
# +kubebuilder:scaffold:-
`},
				fakeInserter{
					fakeBuilder: fakeBuilder{path: pathYaml},
					codeFragments: CodeFragmentsMap{
						NewMarkerFor(pathYaml, "-"): {"1\n", "2\n"},
					},
				},
			),
			Entry("should use files over optional models",
				pathYaml,
				`
# +kubebuilder:scaffold:-
`,
				`
1
2
# +kubebuilder:scaffold:-
`,
				&fakeTemplate{fakeBuilder: fakeBuilder{path: pathYaml}, body: content},
				fakeInserter{
					fakeBuilder: fakeBuilder{path: pathYaml},
					codeFragments: CodeFragmentsMap{
						NewMarkerFor(pathYaml, "-"): {"1\n", "2\n"},
					},
				},
			),
			Entry("should filter invalid markers",
				pathYaml,
				`
# +kubebuilder:scaffold:-
# +kubebuilder:scaffold:*
`,
				`
1
2
# +kubebuilder:scaffold:-
# +kubebuilder:scaffold:*
`,
				fakeInserter{
					fakeBuilder: fakeBuilder{path: pathYaml},
					markers:     []Marker{NewMarkerFor(pathYaml, "-")},
					codeFragments: CodeFragmentsMap{
						NewMarkerFor(pathYaml, "-"): {"1\n", "2\n"},
						NewMarkerFor(pathYaml, "*"): {"3\n", "4\n"},
					},
				},
			),
			Entry("should filter already existing one-line code fragments",
				pathYaml,
				`
1
# +kubebuilder:scaffold:-
3
4
# +kubebuilder:scaffold:*
`,
				`
1
2
# +kubebuilder:scaffold:-
3
4
# +kubebuilder:scaffold:*
`,
				fakeInserter{
					fakeBuilder: fakeBuilder{path: pathYaml},
					codeFragments: CodeFragmentsMap{
						NewMarkerFor(pathYaml, "-"): {"1\n", "2\n"},
						NewMarkerFor(pathYaml, "*"): {"3\n", "4\n"},
					},
				},
			),
			Entry("should filter already existing multi-line indented code fragments",
				pathGo,
				`package test

func init() {
	if err := something(); err != nil {
		return err
	}
	
	// +kubebuilder:scaffold:-
}
`,
				`package test

func init() {
	if err := something(); err != nil {
		return err
	}
	
	// +kubebuilder:scaffold:-
}
`,
				fakeInserter{
					fakeBuilder: fakeBuilder{path: pathGo},
					codeFragments: CodeFragmentsMap{
						NewMarkerFor(pathGo, "-"): {"if err := something(); err != nil {\n\treturn err\n}\n\n"},
					},
				},
			),
			Entry("should not insert anything if no code fragment",
				pathYaml,
				`
# +kubebuilder:scaffold:-
`,
				`
# +kubebuilder:scaffold:-
`,
				fakeInserter{
					fakeBuilder: fakeBuilder{path: pathYaml},
					codeFragments: CodeFragmentsMap{
						NewMarkerFor(pathYaml, "-"): {},
					},
				},
			),
		)

		DescribeTable("insert strings related errors",
			func(errType interface{}, files ...Builder) {
				Expect(afero.WriteFile(s.fs, path, []byte{}, 0o666)).To(Succeed())

				err := s.Execute(files...)
				Expect(err).To(HaveOccurred())
				Expect(errors.As(err, errType)).To(BeTrue())
			},
			Entry("should fail if inserting into a model that fails when a file exists and it does exist",
				&FileAlreadyExistsError{},
				&fakeTemplate{fakeBuilder: fakeBuilder{path: "filename", ifExistsAction: Error}},
				fakeInserter{fakeBuilder: fakeBuilder{path: "filename"}},
			),
			Entry("should fail if inserting into a model with unknown behavior if the file exists and it does exist",
				&UnknownIfExistsActionError{},
				&fakeTemplate{fakeBuilder: fakeBuilder{path: "filename", ifExistsAction: -1}},
				fakeInserter{fakeBuilder: fakeBuilder{path: "filename"}},
			),
		)

		Context("write when the file already exists", func() {
			BeforeEach(func() {
				_ = afero.WriteFile(s.fs, path, []byte{}, 0o666)
			})

			It("should skip the file by default", func() {
				Expect(s.Execute(&fakeTemplate{
					fakeBuilder: fakeBuilder{path: path},
					body:        content,
				})).To(Succeed())

				b, err := afero.ReadFile(s.fs, path)
				Expect(err).NotTo(HaveOccurred())
				Expect(string(b)).To(BeEmpty())
			})

			It("should write the file if configured to do so", func() {
				Expect(s.Execute(&fakeTemplate{
					fakeBuilder: fakeBuilder{path: path, ifExistsAction: OverwriteFile},
					body:        content,
				})).To(Succeed())

				b, err := afero.ReadFile(s.fs, path)
				Expect(err).NotTo(HaveOccurred())
				Expect(string(b)).To(Equal(content))
			})

			It("should error if configured to do so", func() {
				err := s.Execute(&fakeTemplate{
					fakeBuilder: fakeBuilder{path: path, ifExistsAction: Error},
					body:        content,
				})
				Expect(err).To(HaveOccurred())
				Expect(errors.As(err, &FileAlreadyExistsError{})).To(BeTrue())
			})
		})
	})
})

var _ Builder = fakeBuilder{}

// fakeBuilder is used to mock a Builder
type fakeBuilder struct {
	path           string
	ifExistsAction IfExistsAction
	TestField      string // test go template actions
}

// GetPath implements Builder
func (f fakeBuilder) GetPath() string {
	return f.path
}

// GetIfExistsAction implements Builder
func (f fakeBuilder) GetIfExistsAction() IfExistsAction {
	return f.ifExistsAction
}

var _ RequiresValidation = fakeRequiresValidation{}

// fakeRequiresValidation is used to mock a RequiresValidation in order to test Scaffold
type fakeRequiresValidation struct {
	fakeBuilder

	validateErr error
}

// Validate implements RequiresValidation
func (f fakeRequiresValidation) Validate() error {
	return f.validateErr
}

var _ Template = &fakeTemplate{}

// fakeTemplate is used to mock a File in order to test Scaffold
type fakeTemplate struct {
	fakeBuilder

	body            string
	err             error
	parseDelimLeft  string
	parseDelimRight string
}

func (f *fakeTemplate) SetDelim(left, right string) {
	f.parseDelimLeft = left
	f.parseDelimRight = right
}

func (f *fakeTemplate) GetDelim() (string, string) {
	return f.parseDelimLeft, f.parseDelimRight
}

// GetBody implements Template
func (f *fakeTemplate) GetBody() string {
	return f.body
}

// SetTemplateDefaults implements Template
func (f *fakeTemplate) SetTemplateDefaults() error {
	if f.err != nil {
		return f.err
	}

	return nil
}

type fakeInserter struct {
	fakeBuilder

	markers       []Marker
	codeFragments CodeFragmentsMap
}

// GetMarkers implements Inserter
func (f fakeInserter) GetMarkers() []Marker {
	if f.markers != nil {
		return f.markers
	}

	markers := make([]Marker, 0, len(f.codeFragments))
	for marker := range f.codeFragments {
		markers = append(markers, marker)
	}
	return markers
}

// GetCodeFragments implements Inserter
func (f fakeInserter) GetCodeFragments() CodeFragmentsMap {
	return f.codeFragments
}
