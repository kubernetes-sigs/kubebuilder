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

package v1alpha1

import (
	"fmt"
	"path/filepath"

	"sigs.k8s.io/kubebuilder/v3/pkg/plugin/util"
)

// insertModUpdatesInDockerfile will add channels staging required for declarative plugin
func insertModUpdatesInDockerfile(apiPath string) error {
	fmt.Println("updating Dockerfile to add module in the image")
	dockerfile := filepath.Join("Dockerfile")

	goModPath := filepath.Join(apiPath, "go.mod")
	goSumPath := filepath.Join(apiPath, "go.sum")

	// nolint:lll
	err := insertCodeIfDoesNotExist(dockerfile,
		"COPY go.sum go.sum",
		fmt.Sprintf("\n# Copy the Go Sub-Module manifests"+
			"\nCOPY %s %s"+
			"\nCOPY %s %s",
			goModPath, goModPath, goSumPath, goSumPath))
	if err != nil {
		return err
	}

	return nil
}

func removeModUpdatesInDockerfile(apiPath string) error {
	fmt.Println("updating Dockerfile to remove module in the image")
	dockerfile := filepath.Join("Dockerfile")

	goModPath := filepath.Join(apiPath, "go.mod")
	goSumPath := filepath.Join(apiPath, "go.sum")

	toRemove := fmt.Sprintf("# Copy the Go Sub-Module manifests"+
		"\nCOPY %s %s"+
		"\nCOPY %s %s",
		goModPath, goModPath, goSumPath, goSumPath)

	return removeCodeIfExist(dockerfile, toRemove)
}

// insertCodeIfDoesNotExist insert code if it does not already exists
func insertCodeIfDoesNotExist(filename, target, code string) error {
	if err := util.InsertCode(filename, target, code); err != nil && err != util.ErrContentNotFound {
		return err
	}
	return nil
}

// removeCodeIfExist removes code if it already exists
func removeCodeIfExist(filename, target string) error {
	if err := util.ReplaceInFile(filename, target, ""); err != nil && err != util.ErrContentNotFound {
		return err
	}
	return nil
}