/*
Copyright 2025 The Kubernetes Authors.

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

package scaffolds

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/afero"

	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
)

var _ = Describe("Init Scaffolder", func() {
	var (
		fs         machinery.Filesystem
		scaffolder *initScaffolder
		tmpDir     string
	)

	BeforeEach(func() {
		var err error
		tmpDir, err = os.MkdirTemp("", "grafana-init-test-*")
		Expect(err).NotTo(HaveOccurred())

		// Change to tmpDir so relative paths work correctly
		err = os.Chdir(tmpDir)
		Expect(err).NotTo(HaveOccurred())

		fs = machinery.Filesystem{
			FS: afero.NewBasePathFs(afero.NewOsFs(), tmpDir),
		}
		scaffolder = &initScaffolder{}
		scaffolder.InjectFS(fs)
	})

	AfterEach(func() {
		if tmpDir != "" {
			_ = os.RemoveAll(tmpDir)
		}
	})

	Describe("Scaffold", func() {
		It("should handle re-scaffolding gracefully", func() {
			By("running scaffolder first time")
			err := scaffolder.Scaffold()
			Expect(err).NotTo(HaveOccurred())

			By("verifying files were created")
			runtimePath := filepath.Join("grafana", "controller-runtime-metrics.json")
			Expect(fileExistsInit(runtimePath)).To(BeTrue())

			By("running scaffolder second time")
			scaffolder2 := &initScaffolder{}
			scaffolder2.InjectFS(fs)
			err = scaffolder2.Scaffold()
			Expect(err).NotTo(HaveOccurred())

			By("verifying files still exist")
			Expect(fileExistsInit(runtimePath)).To(BeTrue())
		})
	})
})

func fileExistsInit(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
