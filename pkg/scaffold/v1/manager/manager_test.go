package manager_test

import (
	"fmt"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/kubebuilder/pkg/model"
	"sigs.k8s.io/kubebuilder/pkg/scaffold/input"
	"sigs.k8s.io/kubebuilder/pkg/scaffold/scaffoldtest"
	"sigs.k8s.io/kubebuilder/pkg/scaffold/v1/manager"
)

var _ = Describe("Manager", func() {
	Describe(fmt.Sprintf("scaffolding Manager"), func() {
		files := []struct {
			instance input.File
			file     string
		}{
			{
				file:     filepath.Join("pkg", "apis", "apis.go"),
				instance: &manager.APIs{},
			},
			{
				file:     filepath.Join("cmd", "manager", "main.go"),
				instance: &manager.Cmd{},
			},
			{
				file:     filepath.Join("config", "manager", "manager.yaml"),
				instance: &manager.Config{Image: "controller:latest"},
			},
			{
				file:     filepath.Join("pkg", "controller", "controller.go"),
				instance: &manager.Controller{},
			},
			{
				file:     filepath.Join("pkg", "webhook", "webhook.go"),
				instance: &manager.Webhook{},
			},
			{
				file:     filepath.Join("Dockerfile"),
				instance: &manager.Dockerfile{},
			},
		}

		for j := range files {
			f := files[j]
			Context(f.file, func() {
				It(fmt.Sprintf("should write a file matching the golden file %s", f.file), func() {
					s, result := scaffoldtest.NewTestScaffold(f.file, f.file)
					Expect(s.Execute(&model.Universe{}, scaffoldtest.Options(), f.instance)).To(Succeed())
					Expect(result.Actual.String()).To(Equal(result.Golden), result.Actual.String())
				})
			})
		}
	})

	Describe(fmt.Sprintf("scaffolding Manager"), func() {
		Context("APIs", func() {
			It("should return an error if the relative path cannot be calculated", func() {
				instance := &manager.APIs{}
				s, _ := scaffoldtest.NewTestScaffold(filepath.Join("pkg", "apis", "apis.go"), "")
				s.ProjectPath = "."
				err := s.Execute(&model.Universe{}, scaffoldtest.Options(), instance)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Rel: can't make"))
			})
		})
	})
})
