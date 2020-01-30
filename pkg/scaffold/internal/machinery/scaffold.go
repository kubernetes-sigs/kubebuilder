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

package machinery

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strings"
	"text/template"

	"golang.org/x/tools/imports"

	"sigs.k8s.io/kubebuilder/pkg/model"
	"sigs.k8s.io/kubebuilder/pkg/model/file"
	"sigs.k8s.io/kubebuilder/pkg/scaffold/internal/filesystem"
)

var options = imports.Options{
	Comments:   true,
	TabIndent:  true,
	TabWidth:   8,
	FormatOnly: true,
}

// Scaffold uses templates to scaffold new files
type Scaffold interface {
	// Execute writes to disk the provided templates
	Execute(*model.Universe, ...file.Template) error
}

// scaffold implements Scaffold interface
type scaffold struct {
	// plugins is the list of plugins we should allow to transform our generated scaffolding
	plugins []model.Plugin

	// fs allows to mock the file system for tests
	fs filesystem.FileSystem
}

func NewScaffold(plugins ...model.Plugin) Scaffold {
	return &scaffold{
		plugins: plugins,
		fs:      filesystem.New(),
	}
}

// Execute implements Scaffold.Execute
func (s *scaffold) Execute(universe *model.Universe, files ...file.Template) error {
	// Set the repo as the local prefix so that it knows how to group imports
	if universe.Config != nil {
		imports.LocalPrefix = universe.Config.Repo
	}

	for _, f := range files {
		m, err := buildFileModel(universe, f)
		if err != nil {
			return err
		}
		universe.Files = append(universe.Files, m)
	}

	for _, plugin := range s.plugins {
		if err := plugin.Pipe(universe); err != nil {
			return err
		}
	}

	for _, f := range universe.Files {
		if err := s.writeFile(f); err != nil {
			return err
		}
	}

	return nil
}

// buildFileModel scaffolds a single file
func buildFileModel(universe *model.Universe, t file.Template) (*file.File, error) {
	// Inject common fields
	universe.InjectInto(t)

	// Validate the file scaffold
	if reqValFile, ok := t.(file.RequiresValidation); ok {
		if err := reqValFile.Validate(); err != nil {
			return nil, err
		}
	}

	// Get the template input params
	i, err := t.GetInput()
	if err != nil {
		return nil, err
	}

	m := &file.File{
		Path:           i.Path,
		IfExistsAction: i.IfExistsAction,
	}

	b, err := doTemplate(i, t)
	if err != nil {
		return nil, err
	}
	m.Contents = string(b)

	return m, nil
}

func (s *scaffold) writeFile(f *file.File) error {
	// Check if the file to write already exists
	exists, err := s.fs.Exists(f.Path)
	if err != nil {
		return err
	}
	if exists {
		switch f.IfExistsAction {
		case file.Overwrite:
			// By not returning, the file is written as if it didn't exist
		case file.Skip:
			// By returning nil, the file is not written but the process will carry on
			return nil
		case file.Error:
			// By returning an error, the file is not written and the process will fail
			return fmt.Errorf("failed to create %s: file already exists", f.Path)
		}
	}

	writer, err := s.fs.Create(f.Path)
	if err != nil {
		return err
	}

	_, err = writer.Write([]byte(f.Contents))

	return err
}

// doTemplate executes the template for a file using the input
func doTemplate(i file.Input, t file.Template) ([]byte, error) {
	temp, err := newTemplate(t).Parse(i.TemplateBody)
	if err != nil {
		return nil, err
	}

	out := &bytes.Buffer{}
	err = temp.Execute(out, t)
	if err != nil {
		return nil, err
	}
	b := out.Bytes()

	// gofmt the imports
	if filepath.Ext(i.Path) == ".go" {
		b, err = imports.Process(i.Path, b, &options)
		if err != nil {
			return nil, err
		}
	}

	return b, nil
}

// newTemplate a new template with common functions
func newTemplate(t file.Template) *template.Template {
	return template.New(fmt.Sprintf("%T", t)).Funcs(template.FuncMap{
		"title": strings.Title,
		"lower": strings.ToLower,
	})
}
