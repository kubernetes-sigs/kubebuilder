package project_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/kubebuilder/pkg/model"
	"sigs.k8s.io/kubebuilder/pkg/scaffold"
	"sigs.k8s.io/kubebuilder/pkg/scaffold/files"
	filesv1 "sigs.k8s.io/kubebuilder/pkg/scaffold/files/v1"
	. "sigs.k8s.io/kubebuilder/pkg/scaffold/files/v1/project"
	metricsauthv1 "sigs.k8s.io/kubebuilder/pkg/scaffold/files/v1/metricsauth"
	"sigs.k8s.io/kubebuilder/pkg/scaffold/input"
	"sigs.k8s.io/kubebuilder/pkg/scaffold/scaffoldtest"
)

var _ = Describe("Project", func() {
	var result *scaffoldtest.TestResult
	var writeToPath, goldenPath string
	var s *scaffold.Scaffold
	var year string

	JustBeforeEach(func() {
		s, result = scaffoldtest.NewTestScaffold(writeToPath, goldenPath)
		s.BoilerplateOptional = true
		s.ProjectOptional = true
		year = strconv.Itoa(time.Now().Year())
	})

	Describe("scaffolding a boilerplate file", func() {
		BeforeEach(func() {
			goldenPath = filepath.Join("hack", "boilerplate.go.txt")
			writeToPath = goldenPath
		})

		It("should match the golden file", func() {
			instance := &files.Boilerplate{Year: year, License: "apache2", Owner: "The Kubernetes authors"}
			Expect(s.Execute(&model.Universe{}, input.Options{}, instance)).NotTo(HaveOccurred())
			Expect(result.Actual.String()).To(BeEquivalentTo(result.Golden))
		})

		It("should skip writing boilerplate if the file exists", func() {
			i, err := (&files.Boilerplate{}).GetInput()
			Expect(err).NotTo(HaveOccurred())
			Expect(i.IfExistsAction).To(Equal(input.Skip))
		})

		Context("for apache2", func() {
			It("should write the apache2 boilerplate with specified owners", func() {
				instance := &files.Boilerplate{Year: year, Owner: "Example Owners"}
				Expect(s.Execute(&model.Universe{}, input.Options{}, instance)).NotTo(HaveOccurred())
				e := strings.Replace(
					result.Golden, "The Kubernetes authors", "Example Owners", -1)
				Expect(result.Actual.String()).To(BeEquivalentTo(e))
			})

			It("should use apache2 as the default", func() {
				instance := &files.Boilerplate{Year: year, Owner: "The Kubernetes authors"}
				Expect(s.Execute(&model.Universe{}, input.Options{}, instance)).NotTo(HaveOccurred())
				Expect(result.Actual.String()).To(BeEquivalentTo(result.Golden))
			})
		})

		Context("for none", func() {
			It("should write the empty boilerplate", func() {
				// Scaffold a boilerplate file
				instance := &files.Boilerplate{Year: year, License: "none", Owner: "Example Owners"}
				Expect(s.Execute(&model.Universe{}, input.Options{}, instance)).NotTo(HaveOccurred())
				Expect(result.Actual.String()).To(BeEquivalentTo(fmt.Sprintf(`/*
Copyright %s Example Owners.
*/`, year)))
			})
		})

		Context("if the boilerplate is given", func() {
			It("should skip writing Gopkg.toml", func() {
				instance := &files.Boilerplate{}
				instance.Boilerplate = `/* Hello World */`

				Expect(s.Execute(&model.Universe{}, input.Options{}, instance)).NotTo(HaveOccurred())
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
				Expect(s.Execute(&model.Universe{}, input.Options{}, instance)).NotTo(HaveOccurred())

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

				err = s.Execute(&model.Universe{}, input.Options{}, instance)
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

				Expect(s.Execute(&model.Universe{}, input.Options{}, instance)).NotTo(HaveOccurred())
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

				err = s.Execute(&model.Universe{}, input.Options{}, instance)
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
				instance := &Makefile{Image: "controller:latest"}
				instance.Repo = "project"
				Expect(s.Execute(&model.Universe{}, input.Options{}, instance)).NotTo(HaveOccurred())

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
				instance := &Kustomize{Prefix: "project"}
				instance.Repo = "project"
				Expect(s.Execute(&model.Universe{}, input.Options{}, instance)).NotTo(HaveOccurred())

				// Verify the contents matches the golden file.
				Expect(result.Actual.String()).To(BeEquivalentTo(result.Golden))
			})
		})
	})

	Describe("scaffolding a RBAC Kustomization", func() {
		BeforeEach(func() {
			goldenPath = filepath.Join("config", "rbac", "kustomization.yaml")
			writeToPath = goldenPath
		})
		Context("with rbac", func() {
			It("should match the golden file", func() {
				instance := &KustomizeRBAC{}
				instance.Repo = "project"
				Expect(s.Execute(&model.Universe{}, input.Options{}, instance)).NotTo(HaveOccurred())

				// Verify the contents matches the golden file.
				Expect(result.Actual.String()).To(BeEquivalentTo(result.Golden))
			})
		})
	})

	Describe("scaffolding a manager Kustomization", func() {
		BeforeEach(func() {
			goldenPath = filepath.Join("config", "manager", "kustomization.yaml")
			writeToPath = goldenPath
		})
		Context("with manager", func() {
			It("should match the golden file", func() {
				instance := &KustomizeManager{}
				instance.Repo = "project"
				Expect(s.Execute(&model.Universe{}, input.Options{}, instance)).NotTo(HaveOccurred())

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
				instance := &filesv1.KustomizeImagePatch{}
				instance.Repo = "project"
				Expect(s.Execute(&model.Universe{}, input.Options{}, instance)).NotTo(HaveOccurred())

				// Verify the contents matches the golden file.
				Expect(result.Actual.String()).To(BeEquivalentTo(result.Golden))
			})
		})
	})

	Describe("scaffolding a Kustomize prometheus metrics patch", func() {
		BeforeEach(func() {
			goldenPath = filepath.Join("config", "default", "manager_prometheus_metrics_patch.yaml")
			writeToPath = goldenPath
		})
		Context("with defaults ", func() {
			It("should match the golden file", func() {
				instance := &metricsauthv1.KustomizePrometheusMetricsPatch{}
				instance.Repo = "project"
				Expect(s.Execute(&model.Universe{}, input.Options{}, instance)).NotTo(HaveOccurred())

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
				Expect(s.Execute(&model.Universe{}, input.Options{}, instance)).NotTo(HaveOccurred())

				// Verify the contents matches the golden file.
				Expect(result.Actual.String()).To(BeEquivalentTo(result.Golden))
			})
		})
	})

	Describe("scaffolding a PROJECT", func() {
		BeforeEach(func() {
			goldenPath = filepath.Join("PROJECT")
			writeToPath = goldenPath
		})
		Context("with defaults", func() {
			It("should match the golden file", func() {
				instance := &files.Project{}
				instance.Version = "1"
				instance.Domain = "testproject.org"
				instance.Repo = "project"
				Expect(s.Execute(&model.Universe{}, input.Options{}, instance)).NotTo(HaveOccurred())

				// Verify the contents matches the golden file.
				Expect(result.Actual.String()).To(BeEquivalentTo(result.Golden))
			})
		})
	})
})
