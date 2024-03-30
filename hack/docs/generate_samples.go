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
	gettingstarted "sigs.k8s.io/kubebuilder/v3/hack/docs/internal/getting-started"
)

// Make sure executing `build_kb` to generate kb executable from the source code
const KubebuilderBinName = "/tmp/kubebuilder/bin/kubebuilder"

type tutorial_generator interface {
	Prepare()
	GenerateSampleProject()
	UpdateTutorial()
	CodeGen()
}

func main() {
	type generator func()

	// TODO: Generate multiversion-tutorial
	tutorials := map[string]generator{
		"component-config": UpdateComponentConfigTutorial,
		"cronjob":          UpdateCronjobTutorial,
		"getting-started":  UpdateGettingStarted,
	}

	log.SetFormatter(&log.TextFormatter{DisableTimestamp: true})
	log.Println("Generating documents...")

	for tutorial, updater := range tutorials {
		log.Printf("Generating %s tutorial\n", tutorial)
		updater()
	}
}

func updateTutorial(generator tutorial_generator) {
	generator.Prepare()

	generator.GenerateSampleProject()

	generator.UpdateTutorial()

	generator.CodeGen()
}

func UpdateComponentConfigTutorial() {
	samplePath := "docs/book/src/component-config-tutorial/testdata/project/"
	sp := componentconfig.NewSample(KubebuilderBinName, samplePath)
	updateTutorial(&sp)
}

func UpdateCronjobTutorial() {
	samplePath := "docs/book/src/cronjob-tutorial/testdata/project/"
	sp := cronjob.NewSample(KubebuilderBinName, samplePath)
	updateTutorial(&sp)
}

func UpdateGettingStarted() {
	samplePath := "docs/book/src/getting-started/testdata/project/"
	sp := gettingstarted.NewSample(KubebuilderBinName, samplePath)
	updateTutorial(&sp)
}
