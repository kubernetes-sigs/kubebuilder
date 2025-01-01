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
	cronjob "sigs.k8s.io/kubebuilder/v4/hack/docs/internal/cronjob-tutorial"
	gettingstarted "sigs.k8s.io/kubebuilder/v4/hack/docs/internal/getting-started"
	multiversion "sigs.k8s.io/kubebuilder/v4/hack/docs/internal/multiversion-tutorial"
)

// KubebuilderBinName make sure executing `build_kb` to generate kb executable from the source code
const KubebuilderBinName = "/tmp/kubebuilder/bin/kubebuilder"

type tutorialGenerator interface {
	Prepare()
	GenerateSampleProject()
	UpdateTutorial()
	CodeGen()
}

func main() {
	type generator func()

	tutorials := map[string]generator{
		"cronjob":         updateCronjobTutorial,
		"getting-started": updateGettingStarted,
		"multiversion":    updateMultiversionTutorial,
	}

	log.SetFormatter(&log.TextFormatter{DisableTimestamp: true})
	log.Println("Generating documents...")

	for tutorial, updater := range tutorials {
		log.Printf("Generating %s tutorial\n", tutorial)
		updater()
	}
}

func updateTutorial(generator tutorialGenerator) {
	generator.Prepare()
	generator.GenerateSampleProject()
	generator.UpdateTutorial()
	generator.CodeGen()
}

func updateCronjobTutorial() {
	samplePath := "docs/book/src/cronjob-tutorial/testdata/project/"
	sp := cronjob.NewSample(KubebuilderBinName, samplePath)
	updateTutorial(&sp)
}

func updateGettingStarted() {
	samplePath := "docs/book/src/getting-started/testdata/project"
	sp := gettingstarted.NewSample(KubebuilderBinName, samplePath)
	updateTutorial(&sp)
}

func updateMultiversionTutorial() {
	samplePath := "docs/book/src/multiversion-tutorial/testdata/project"
	sp := cronjob.NewSample(KubebuilderBinName, samplePath)
	updateTutorial(&sp)
	multi := multiversion.NewSample(KubebuilderBinName, samplePath)
	updateTutorial(&multi)
}
