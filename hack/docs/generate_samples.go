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
	"fmt"

	componentconfig "sigs.k8s.io/kubebuilder/v3/hack/docs/internal"
)

func main() {
	fmt.Println("Generating documents...")

	fmt.Println("Generating component-config tutorial...")
	UpdateComponentConfigTutorial()

	// TODO: Generate cronjob-tutorial

	// TODO: Generate multiversion-tutorial
}

func UpdateComponentConfigTutorial() {
	binaryPath := "/tmp/kubebuilder/bin/kubebuilder"
	samplePath := "docs/book/src/component-config-tutorial/testdata/project/"

	sp := componentconfig.NewSample(binaryPath, samplePath)

	sp.Prepare()

	sp.GenerateSampleProject()

	sp.UpdateTutorial()

	sp.CodeGen()
}
