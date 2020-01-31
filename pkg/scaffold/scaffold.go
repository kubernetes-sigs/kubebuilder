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

package scaffold

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"golang.org/x/tools/imports"

	"sigs.k8s.io/kubebuilder/pkg/model"
	"sigs.k8s.io/kubebuilder/pkg/model/file"
)

var options = imports.Options{
	Comments:   true,
	TabIndent:  true,
	TabWidth:   8,
	FormatOnly: true,
}

// Scaffold writes Templates to scaffold new files
type Scaffold struct {
	// plugins is the list of plugins we should allow to transform our generated scaffolding
	plugins []Plugin

	GetWriter func(path string) (io.Writer, error)

	FileExists func(path string) bool
}

// NewScaffold creates a new Scaffold
func NewScaffold(plugins ...Plugin) *Scaffold {
	return &Scaffold{plugins: plugins}
}

// Plugin is the interface that a plugin must implement
// We will (later) have an ExecPlugin that implements this by exec-ing a binary
type Plugin interface {
	// Pipe is the core plugin interface, that transforms a UniverseModel
	Pipe(universe *model.Universe) error
}

// Execute executes scaffolding the for files
func (s *Scaffold) Execute(
	universe *model.Universe,
	files ...file.Template,
) error {
	if s.GetWriter == nil {
		s.GetWriter = (&FileWriter{}).WriteCloser
	}
	if s.FileExists == nil {
		s.FileExists = func(path string) bool {
			_, err := os.Stat(path)
			return err == nil
		}
	}

	// Set the repo as the local prefix so that it knows how to group imports
	imports.LocalPrefix = universe.Config.Repo

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
		Path: i.Path,
	}

	b, err := doTemplate(i, t)
	if err != nil {
		return nil, err
	}
	m.Contents = string(b)

	return m, nil
}

func (s *Scaffold) writeFile(f *file.File) error {
	// Check if the file to write already exists
	if s.FileExists(f.Path) {
		switch f.IfExistsAction {
		case file.Overwrite:
		case file.Skip:
			return nil
		case file.Error:
			return fmt.Errorf("%s already exists", f.Path)
		}
	}

	writer, err := s.GetWriter(f.Path)
	if err != nil {
		return err
	}
	if c, ok := writer.(io.Closer); ok {
		defer func() {
			if err := c.Close(); err != nil {
				log.Fatal(err)
			}
		}()
	}

	_, err = writer.Write([]byte(f.Contents))

	return err
}

// doTemplate executes the template for a file using the input
func doTemplate(i file.Input, e file.Template) ([]byte, error) {
	temp, err := newTemplate(e).Parse(i.TemplateBody)
	if err != nil {
		return nil, err
	}

	out := &bytes.Buffer{}
	err = temp.Execute(out, e)
	if err != nil {
		return nil, err
	}
	b := out.Bytes()

	// gofmt the imports
	if filepath.Ext(i.Path) == ".go" {
		b, err = imports.Process(i.Path, b, &options)
		if err != nil {
			fmt.Printf("%s\n", out.Bytes())
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
