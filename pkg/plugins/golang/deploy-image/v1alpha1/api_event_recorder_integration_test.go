//go:build integration

/*
Copyright 2026 The Kubernetes Authors.

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

package v1alpha1

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	pluginutil "sigs.k8s.io/kubebuilder/v4/pkg/plugin/util"
	"sigs.k8s.io/kubebuilder/v4/test/e2e/utils"
)

var _ = Describe("deploy-image event recorder", func() {
	var project *utils.TestContext

	BeforeEach(func() {
		var err error
		project, err = utils.NewTestContext(pluginutil.KubebuilderBinName, "GO111MODULE=on")
		Expect(err).NotTo(HaveOccurred())
		Expect(project.Prepare()).To(Succeed())
	})

	AfterEach(func() {
		project.Destroy()
	})

	DescribeTable("should attach the recorder to the Deploy Image controller",
		func(existingGroup, existingKind string, multigroup bool) {
			initArgs := []string{"--domain", "example.com", "--repo", "example.com/crew"}
			if multigroup {
				initArgs = append(initArgs, "--multigroup")
			}
			Expect(project.Init(initArgs...)).To(Succeed())

			Expect(project.CreateAPI("--group", existingGroup, "--version", "v1", "--kind", existingKind,
				"--resource", "--controller", "--make=false")).To(Succeed())
			Expect(project.CreateAPI("--group", "crew", "--version", "v1", "--kind", "Captain",
				"--image=busybox:1.36", "--plugins=deploy-image/v1-alpha",
				"--make=false", "--manifests=false")).To(Succeed())

			Expect(project.Make("build")).To(Succeed())

			main, err := os.ReadFile(filepath.Join(project.Dir, "cmd", "main.go"))
			Expect(err).NotTo(HaveOccurred())
			Expect(string(main)).To(ContainSubstring(`mgr.GetEventRecorder("captain-controller")`))
		},
		Entry("when a controller name ends with the Deploy Image kind", "crew", "NightlyCaptain", false),
		Entry("when another controller package has the same kind", "fleet", "Captain", true),
	)
})
