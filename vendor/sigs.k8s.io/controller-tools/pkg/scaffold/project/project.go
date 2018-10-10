/*
Copyright 2018 The Kubernetes Authors.

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

package project

import (
	"fmt"
	"go/build"
	"os"
	"path/filepath"
	"strings"

	yaml "gopkg.in/yaml.v2"
	"sigs.k8s.io/controller-tools/pkg/scaffold/input"
)

var _ input.File = &Project{}

// Project scaffolds the PROJECT file with project metadata
type Project struct {
	// Path is the output file location - defaults to PROJECT
	Path string

	input.ProjectFile
}

// GetInput implements input.File
func (c *Project) GetInput() (input.Input, error) {
	if c.Path == "" {
		c.Path = "PROJECT"
	}
	if c.Repo == "" {
		r, err := c.repoFromGopathAndWd(os.Getenv("GOPATH"), os.Getwd)
		if err != nil {
			return input.Input{}, err
		}
		c.Repo = r
	}

	out, err := yaml.Marshal(c.ProjectFile)
	if err != nil {
		return input.Input{}, err
	}

	return input.Input{
		Path:           c.Path,
		TemplateBody:   string(out),
		Repo:           c.Repo,
		Version:        c.Version,
		Domain:         c.Domain,
		IfExistsAction: input.Error,
	}, nil
}

func (Project) repoFromGopathAndWd(gopath string, getwd func() (string, error)) (string, error) {
	// Assume the working dir is the root of the repo
	wd, err := getwd()
	if err != nil {
		return "", err
	}

	// Strip the GOPATH from the working dir to get the go package of the repo
	if len(gopath) == 0 {
		gopath = build.Default.GOPATH
	}
	goSrc := filepath.Join(gopath, "src")

	// Make sure the GOPATH is set and the working dir is under the GOPATH
	if !strings.HasPrefix(filepath.Dir(wd), goSrc) {
		return "", fmt.Errorf("working directory must be a project directory under "+
			"$GOPATH/src/<project-package>\n- GOPATH=%s\n- WD=%s", gopath, wd)
	}

	// Figure out the repo name by removing $GOPATH/src from the working directory - e.g.
	// '$GOPATH/src/kubernetes-sigs/controller-tools' becomes 'kubernetes-sigs/controller-tools'
	repo := ""
	for wd != goSrc {
		repo = filepath.Join(filepath.Base(wd), repo)
		wd = filepath.Dir(wd)
	}
	return repo, nil
}
