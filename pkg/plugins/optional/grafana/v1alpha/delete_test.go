/*
Copyright 2022 The Kubernetes Authors.

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

package v1alpha

import (
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/spf13/afero"

	cfgv3 "sigs.k8s.io/kubebuilder/v4/pkg/config/v3"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
)

var _ = Describe("deleteSubcommand", func() {
	var (
		sub *deleteSubcommand
		fs  machinery.Filesystem
	)

	BeforeEach(func() {
		sub = &deleteSubcommand{}
		fs = machinery.Filesystem{FS: afero.NewMemMapFs()}
		cfg := cfgv3.New()
		Expect(sub.InjectConfig(cfg)).To(Succeed())
	})

	It("should succeed when no Grafana files exist", func() {
		Expect(sub.Scaffold(fs)).To(Succeed())
	})

	It("should delete all scaffolded Grafana files", func() {
		grafanaFiles := []string{
			filepath.Join("grafana", "controller-runtime-metrics.json"),
			filepath.Join("grafana", "custom-metrics", "config.yaml"),
			filepath.Join("grafana", "custom-metrics", "custom-metrics-dashboard.json"),
		}

		for _, f := range grafanaFiles {
			Expect(fs.FS.MkdirAll(filepath.Dir(f), 0o755)).To(Succeed())
			Expect(afero.WriteFile(fs.FS, f, []byte("{}"), 0o644)).To(Succeed())
		}

		Expect(sub.Scaffold(fs)).To(Succeed())

		for _, f := range grafanaFiles {
			exists, err := afero.Exists(fs.FS, f)
			Expect(err).NotTo(HaveOccurred())
			Expect(exists).To(BeFalse(), "expected %q to be deleted", f)
		}

		dirExists, err := afero.DirExists(fs.FS, "grafana")
		Expect(err).NotTo(HaveOccurred())
		Expect(dirExists).To(BeFalse())
	})
})
