/*
Copyright 2020 The Kubernetes Authors.

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
	"fmt"
	"strings"

	"github.com/spf13/afero"

	"sigs.k8s.io/kubebuilder/v3/pkg/config"
	"sigs.k8s.io/kubebuilder/v3/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v3/pkg/plugins"
)

var _ plugins.Scaffolder = &editScaffolder{}

type editScaffolder struct {
	config     config.Config
	multigroup bool

	// fs is the filesystem that will be used by the scaffolder
	fs machinery.Filesystem
}

// NewEditScaffolder returns a new Scaffolder for configuration edit operations
func NewEditScaffolder(config config.Config, multigroup bool) plugins.Scaffolder {
	return &editScaffolder{
		config:     config,
		multigroup: multigroup,
	}
}

// InjectFS implements cmdutil.Scaffolder
func (s *editScaffolder) InjectFS(fs machinery.Filesystem) {
	s.fs = fs
}

// Scaffold implements cmdutil.Scaffolder
func (s *editScaffolder) Scaffold() error {
	filename := "Dockerfile"
	bs, err := afero.ReadFile(s.fs.FS, filename)
	if err != nil {
		return err
	}
	str := string(bs)

	// update dockerfile
	if s.multigroup {
		str, err = ensureExistAndReplace(
			str,
			"COPY api/ api/",
			`COPY apis/ apis/`)

	} else {
		str, err = ensureExistAndReplace(
			str,
			"COPY apis/ apis/",
			`COPY api/ api/`)
	}

	// Ignore the error encountered, if the file is already in desired format.
	if err != nil && s.multigroup != s.config.IsMultiGroup() {
		return err
	}

	if s.multigroup {
		_ = s.config.SetMultiGroup()
	} else {
		_ = s.config.ClearMultiGroup()
	}

	// Check if the str is not empty, because when the file is already in desired format it will return empty string
	// because there is nothing to replace.
	if str != "" {
		// TODO: instead of writing it directly, we should use the scaffolding machinery for consistency
		return afero.WriteFile(s.fs.FS, filename, []byte(str), 0644)
	}

	return nil
}

func ensureExistAndReplace(input, match, replace string) (string, error) {
	if !strings.Contains(input, match) {
		return "", fmt.Errorf("can't find %q", match)
	}
	return strings.Replace(input, match, replace, -1), nil
}
