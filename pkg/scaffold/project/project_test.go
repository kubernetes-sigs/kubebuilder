package project

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/kubebuilder/pkg/scaffold"
	"sigs.k8s.io/kubebuilder/pkg/scaffold/input"
	"sigs.k8s.io/kubebuilder/pkg/scaffold/scaffoldtest"
)

var _ = Describe("Project", func() {
	var result *scaffoldtest.TestResult
	var writeToPath, goldenPath string
	var s *scaffold.Scaffold

	JustBeforeEach(func() {
		s, result = scaffoldtest.NewTestScaffold(writeToPath, goldenPath)
		s.BoilerplateOptional = true
		s.ProjectOptional = true
	})

	Describe("scaffolding a boilerplate file", func() {
		BeforeEach(func() {
			goldenPath = filepath.Join("hack", "boilerplate.go.txt")
			writeToPath = goldenPath
		})

		It("should match the golden file", func() {
			instance := &Boilerplate{Year: "2018", License: "apache2", Owner: "The Kubernetes authors"}
			Expect(s.Execute(input.Options{}, instance)).NotTo(HaveOccurred())
			Expect(result.Actual.String()).To(BeEquivalentTo(result.Golden))
		})

		It("should skip writing boilerplate if the file exists", func() {
			i, err := (&Boilerplate{}).GetInput()
			Expect(err).NotTo(HaveOccurred())
			Expect(i.IfExistsAction).To(Equal(input.Skip))
		})

		Context("for apache2", func() {
			It("should write the apache2 boilerplate with specified owners", func() {
				instance := &Boilerplate{Year: "2018", Owner: "Example Owners"}
				Expect(s.Execute(input.Options{}, instance)).NotTo(HaveOccurred())
				e := strings.Replace(
					result.Golden, "The Kubernetes authors", "Example Owners", -1)
				Expect(result.Actual.String()).To(BeEquivalentTo(e))
			})

			It("should use apache2 as the default", func() {
				instance := &Boilerplate{Year: "2018", Owner: "The Kubernetes authors"}
				Expect(s.Execute(input.Options{}, instance)).NotTo(HaveOccurred())
				Expect(result.Actual.String()).To(BeEquivalentTo(result.Golden))
			})
		})

		Context("for none", func() {
			It("should write the empty boilerplate", func() {
				// Scaffold a boilerplate file
				instance := &Boilerplate{Year: "2019", License: "none", Owner: "Example Owners"}
				Expect(s.Execute(input.Options{}, instance)).NotTo(HaveOccurred())
				Expect(result.Actual.String()).To(BeEquivalentTo(`/*
Copyright 2019 Example Owners.
*/`))
			})
		})

		Context("if the boilerplate is given", func() {
			It("should skip writing Gopkg.toml", func() {
				instance := &Boilerplate{}
				instance.Boilerplate = `/* Hello World */`

				Expect(s.Execute(input.Options{}, instance)).NotTo(HaveOccurred())
				Expect(result.Actual.String()).To(BeEquivalentTo(`/* Hello World */`))
			})
		})
	})

	Describe("scaffolding a Gopkg.toml", func() {
		BeforeEach(func() {
			goldenPath = filepath.Join("Gopkg.toml")
			writeToPath = goldenPath
		})
		Context("with defaults ", func() {
			It("should match the golden file", func() {
				instance := &GopkgToml{}
				Expect(s.Execute(input.Options{}, instance)).NotTo(HaveOccurred())

				// Verify the contents matches the golden file.
				Expect(result.Actual.String()).To(BeEquivalentTo(result.Golden))
			})
		})

		Context("if the file exists without the header", func() {
			var f *os.File
			var err error
			BeforeEach(func() {
				f, err = ioutil.TempFile("", "controller-tools-pkg-scaffold-project")
				Expect(err).NotTo(HaveOccurred())
				writeToPath = f.Name()
			})

			It("should skip writing Gopkg.toml", func() {
				e := strings.Replace(string(result.Golden), DefaultGopkgHeader, "", -1)
				_, err = f.Write([]byte(e))
				Expect(err).NotTo(HaveOccurred())
				Expect(f.Close()).NotTo(HaveOccurred())

				instance := &GopkgToml{}
				instance.Input.Path = f.Name()

				err = s.Execute(input.Options{}, instance)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(
					"skipping modifying Gopkg.toml - file already exists and is unmanaged"))
			})
		})

		Context("if the file exists with existing user content", func() {
			var f *os.File
			var err error
			BeforeEach(func() {
				f, err = ioutil.TempFile("", "controller-tools-pkg-scaffold-project")
				Expect(err).NotTo(HaveOccurred())
				writeToPath = f.Name()
			})

			It("should keep the user content", func() {
				e := strings.Replace(string(result.Golden),
					DefaultGopkgUserContent, "Hello World", -1)
				_, err = f.Write([]byte(e))
				Expect(err).NotTo(HaveOccurred())
				Expect(f.Close()).NotTo(HaveOccurred())

				fmt.Printf("Write\n\n")
				instance := &GopkgToml{}
				instance.Input.Path = f.Name()

				Expect(s.Execute(input.Options{}, instance)).NotTo(HaveOccurred())
				Expect(result.Actual.String()).To(BeEquivalentTo(e))
			})
		})

		Context("if no file exists", func() {
			var f *os.File
			var err error
			BeforeEach(func() {
				f, err = ioutil.TempFile("", "controller-tools-pkg-scaffold-project")
				Expect(err).NotTo(HaveOccurred())
				Expect(os.Remove(f.Name())).NotTo(HaveOccurred())
				writeToPath = f.Name()
			})

			It("should use the default user content", func() {
				instance := &GopkgToml{}
				instance.Input.Path = writeToPath

				err = s.Execute(input.Options{}, instance)
				// Verify the contents matches the golden file.
				Expect(result.Actual.String()).To(BeEquivalentTo(result.Golden))
			})
		})
	})

	Describe("scaffolding a Makefile", func() {
		BeforeEach(func() {
			goldenPath = filepath.Join("Makefile")
			writeToPath = goldenPath
		})
		Context("with defaults ", func() {
			It("should match the golden file", func() {
				instance := &Makefile{Image: "controller:latest", ControllerToolsPath: ".."}
				instance.Repo = "sigs.k8s.io/controller-tools/test"
				Expect(s.Execute(input.Options{}, instance)).NotTo(HaveOccurred())

				// Verify the contents matches the golden file.
				Expect(result.Actual.String()).To(BeEquivalentTo(result.Golden))
			})
		})
	})

	Describe("scaffolding a Kustomization", func() {
		BeforeEach(func() {
			goldenPath = filepath.Join("config", "default", "kustomization.yaml")
			writeToPath = goldenPath
		})
		Context("with defaults ", func() {
			It("should match the golden file", func() {
				instance := &Kustomize{Prefix: "test"}
				instance.Repo = "sigs.k8s.io/controller-tools/test"
				Expect(s.Execute(input.Options{}, instance)).NotTo(HaveOccurred())

				// Verify the contents matches the golden file.
				Expect(result.Actual.String()).To(BeEquivalentTo(result.Golden))
			})
		})
	})

	Describe("scaffolding a Kustomize image patch", func() {
		BeforeEach(func() {
			goldenPath = filepath.Join("config", "default", "manager_image_patch.yaml")
			writeToPath = goldenPath
		})
		Context("with defaults ", func() {
			It("should match the golden file", func() {
				instance := &KustomizeImagePatch{}
				instance.Repo = "sigs.k8s.io/controller-tools/test"
				Expect(s.Execute(input.Options{}, instance)).NotTo(HaveOccurred())

				// Verify the contents matches the golden file.
				Expect(result.Actual.String()).To(BeEquivalentTo(result.Golden))
			})
		})
	})

	Describe("scaffolding a .gitignore", func() {
		BeforeEach(func() {
			goldenPath = filepath.Join(".gitignore")
			writeToPath = goldenPath
		})
		Context("with defaults ", func() {
			It("should match the golden file", func() {
				instance := &GitIgnore{}
				Expect(s.Execute(input.Options{}, instance)).NotTo(HaveOccurred())

				// Verify the contents matches the golden file.
				Expect(result.Actual.String()).To(BeEquivalentTo(result.Golden))
			})
		})
	})

	Describe("scaffolding a PROEJCT", func() {
		BeforeEach(func() {
			goldenPath = filepath.Join("PROJECT")
			writeToPath = goldenPath
		})
		Context("with defaults", func() {
			It("should match the golden file", func() {
				instance := &Project{}
				instance.Version = "2"
				instance.Domain = "testproject.org"
				instance.Repo = "sigs.k8s.io/controller-tools/test"
				Expect(s.Execute(input.Options{}, instance)).NotTo(HaveOccurred())

				// Verify the contents matches the golden file.
				Expect(result.Actual.String()).To(BeEquivalentTo(result.Golden))
			})
		})

		Context("by calling repoFromGopathAndWd", func() {
			It("should return the directory if it is under the gopath", func() {
				instance := &Project{}
				repo, err := instance.repoFromGopathAndWd("/home/fake/go", func() (string, error) {
					return "/home/fake/go/src/kubernetes-sigs/controller-tools", nil
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(repo).To(Equal("kubernetes-sigs/controller-tools"))
			})

			It("should return an error if the wd is not under GOPATH", func() {
				instance := &Project{}
				_, err := instance.repoFromGopathAndWd("/home/fake/go/src", func() (string, error) {
					return "/home/fake", nil
				})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(""))
			})

			It("should return an error if the wd is not under GOPATH", func() {
				instance := &Project{}
				_, err := instance.repoFromGopathAndWd("/home/fake/go/src", func() (string, error) {
					return "/home/fake/go", nil
				})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("working directory must be a project directory"))
			})

			It("should return an error if it cannot get the WD", func() {
				instance := &Project{}
				e := fmt.Errorf("expected error")
				_, err := instance.repoFromGopathAndWd("/home/fake/go/src", func() (string, error) {
					return "", e
				})
				Expect(err).To(HaveOccurred())
				Expect(err).To(Equal(e))
			})

			It("should use the build.Default GOPATH if none is defined", func() {
				instance := &Project{}
				instance.repoFromGopathAndWd("", func() (string, error) {
					return "/home/fake/go/src/project", nil
				})
			})
		})

		Context("by calling GetInput", func() {
			It("should return the Repo from GetInput", func() {
				instance := &Project{}
				i, err := instance.GetInput()
				Expect(err).NotTo(HaveOccurred())
				Expect(i.Path).To(Equal("PROJECT"))
				Expect(i.Repo).To(Equal("sigs.k8s.io/kubebuilder/pkg/scaffold/project"))
			})
		})
	})
})
