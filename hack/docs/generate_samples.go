/*
Copyright 2023 The Kubernetes Authors.

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

package main

import (
	log "github.com/sirupsen/logrus"

	componentconfig "sigs.k8s.io/kubebuilder/v3/hack/docs/internal/component-config-tutorial"
	cronjob "sigs.k8s.io/kubebuilder/v3/hack/docs/internal/cronjob-tutorial"
)

// Make sure executing `build_kb` to generate kb executable from the source code
const KubebuilderBinName = "/tmp/kubebuilder/bin/kubebuilder"

func main() {
	log.Println("Generating documents...")

	log.Println("Generating component-config tutorial...")
	UpdateComponentConfigTutorial()

	log.Println("Generating cronjob tutorial...")
	UpdateCronjobTutorial()
	// TODO: Generate multiversion-tutorial
}

func UpdateComponentConfigTutorial() {
	binaryPath := KubebuilderBinName
	samplePath := "docs/book/src/component-config-tutorial/testdata/project/"

	sp := componentconfig.NewSample(binaryPath, samplePath)

	sp.Prepare()

	sp.GenerateSampleProject()

	sp.UpdateTutorial()

	sp.CodeGen()
}

func UpdateCronjobTutorial() {
	binaryPath := KubebuilderBinName
	samplePath := "docs/book/src/cronjob-tutorial/testdata/project/"

	sp := cronjob.NewSample(binaryPath, samplePath)

	sp.Prepare()

	sp.GenerateSampleProject()

	sp.UpdateTutorial()
}
