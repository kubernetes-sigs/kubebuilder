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

package common

import (
	"fmt"
	"os"

	"github.com/spf13/afero"

	"sigs.k8s.io/kubebuilder/v4/pkg/config/store"
	"sigs.k8s.io/kubebuilder/v4/pkg/config/store/yaml"
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
)

// LoadProjectConfig load the project config.
func LoadProjectConfig(inputDir string) (store.Store, error) {
	projectConfig := yaml.New(machinery.Filesystem{FS: afero.NewOsFs()})
	if err := projectConfig.LoadFrom(fmt.Sprintf("%s/%s", inputDir, yaml.DefaultPath)); err != nil {
		return nil, fmt.Errorf("failed to load PROJECT file: %w", err)
	}
	return projectConfig, nil
}

// GetInputPath will return the input path for the project.
func GetInputPath(inputPath string) (string, error) {
	if inputPath == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("failed to get working directory: %w", err)
		}
		inputPath = cwd
	}
	projectPath := fmt.Sprintf("%s/%s", inputPath, yaml.DefaultPath)
	if _, err := os.Stat(projectPath); os.IsNotExist(err) {
		return "", fmt.Errorf("project path %q does not exist: %w", projectPath, err)
	}
	return inputPath, nil
}
