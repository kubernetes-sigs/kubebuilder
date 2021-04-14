/*
Copyright 2021 The Kubernetes Authors.

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
	"errors"
	"fmt"
	"os"
	"path"

	"sigs.k8s.io/kubebuilder/v3/pkg/config"
	"sigs.k8s.io/kubebuilder/v3/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v3/pkg/model/resource"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugins"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugins/common/config-gen/v1/scaffolds/internal/templates/config/configgen"
)

var _ plugins.Scaffolder = &apiScaffolder{}

// apiScaffolder contains configuration for generating scaffolding for Go type
// representing the API and controller that implements the behavior for the API.
type apiScaffolder struct {
	config   config.Config
	resource *resource.Resource

	// fs is the filesystem that will be used by the scaffolder
	fs machinery.Filesystem

	withKustomize bool
}

// NewAPIScaffolder returns a new Scaffolder for API/controller creation operations
func NewAPIScaffolder(config config.Config, res *resource.Resource, withKustomize bool) plugins.Scaffolder {
	return &apiScaffolder{
		config:        config,
		resource:      res,
		withKustomize: withKustomize,
	}
}

func (s *apiScaffolder) InjectFS(fs machinery.Filesystem) { s.fs = fs }

func (s *apiScaffolder) Scaffold() error {

	// Initialize the machinery.Scaffold that will write the files to disk
	scaffold := machinery.NewScaffold(s.fs,
		machinery.WithConfig(s.config),
		machinery.WithResource(s.resource),
	)

	if s.resource.HasAPI() {
		fmt.Println("Writing config-gen manifests for you to edit...")

		if err := scaffold.Execute(
			&configgen.ConfigGenUpdater{},
		); err != nil {
			return err
		}

		if s.withKustomize {
			sampleFileName := s.resource.Replacer().Replace("%[group]_%[version]_%[kind].yaml")
			oldSamplePath := path.Join("config", "samples", sampleFileName)
			newSamplePath := path.Join("samples", sampleFileName)
			if err := mv(s.fs, oldSamplePath, newSamplePath); err != nil {
				return err
			}
		}

	}

	return nil
}

func mv(fs machinery.Filesystem, oldPath, newPath string) error {
	if _, err := fs.FS.Stat(oldPath); err == nil {
		if dir := path.Dir(newPath); dir != "" {
			if err = os.MkdirAll(dir, 0755); err != nil { //nolint:gosec
				return err
			}
		}
		if err = fs.FS.Rename(oldPath, newPath); err != nil {
			return err
		}
	} else if errors.Is(err, os.ErrExist) {
		// Debug log
		fmt.Println("File does not exist:", oldPath)
	} else {
		return err
	}

	return nil
}
