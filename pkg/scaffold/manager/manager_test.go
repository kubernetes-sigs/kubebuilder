package manager

import (
	"fmt"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/kubebuilder/pkg/scaffold/input"
	"sigs.k8s.io/kubebuilder/pkg/scaffold/scaffoldtest"
)

var _ = Describe("Manager", func() {
	Describe(fmt.Sprintf("scaffolding Manager"), func() {
		files := []struct {
			instance input.File
			file     string
		}{
			{
				file:     filepath.Join("pkg", "apis", "apis.go"),
				instance: &APIs{},
			},
			{
				file:     filepath.Join("cmd", "manager", "main.go"),
				instance: &Cmd{},
			},
			{
				file:     filepath.Join("config", "manager", "manager.yaml"),
				instance: &Config{Image: "controller:latest"},
			},
			{
				file:     filepath.Join("pkg", "controller", "controller.go"),
				instance: &Controller{},
			},
			{
				file:     filepath.Join("pkg", "webhook", "webhook.go"),
				instance: &Webhook{},
			},
			{
				file:     filepath.Join("Dockerfile"),
				instance: &Dockerfile{},
			},
		}

		for j := range files {
			f := files[j]
			Context(f.file, func() {
				It("should write a file matching the golden file", func() {
					s, result := scaffoldtest.NewTestScaffold(f.file, f.file)
					Expect(s.Execute(scaffoldtest.Options(), f.instance)).To(Succeed())
					Expect(result.Actual.String()).To(Equal(result.Golden), result.Actual.String())
				})
			})
		}
	})

	Describe(fmt.Sprintf("scaffolding Manager"), func() {
		Context("APIs", func() {
			It("should return an error if the relative path cannot be calculated", func() {
				instance := &APIs{}
				s, _ := scaffoldtest.NewTestScaffold(filepath.Join("pkg", "apis", "apis.go"), "")
				s.ProjectPath = "."
				err := s.Execute(scaffoldtest.Options(), instance)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Rel: can't make"))
			})
		})
	})
})
