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
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"golang.org/x/tools/imports"

	internalconfig "sigs.k8s.io/kubebuilder/internal/config"
	"sigs.k8s.io/kubebuilder/pkg/model"
	"sigs.k8s.io/kubebuilder/pkg/model/config"
	"sigs.k8s.io/kubebuilder/pkg/scaffold/input"
)

var options = imports.Options{
	Comments:   true,
	TabIndent:  true,
	TabWidth:   8,
	FormatOnly: true,
}

// Scaffold writes Templates to scaffold new files
type Scaffold struct {
	// BoilerplatePath is the path to the boilerplate file
	BoilerplatePath string

	// Boilerplate is the contents of the boilerplate file for code generation
	Boilerplate string

	// Config is the project configuration
	Config *config.Config

	// ConfigPath is the relative path to the project root
	ConfigPath string

	GetWriter func(path string) (io.Writer, error)

	// Plugins is the list of plugins we should allow to transform our generated scaffolding
	Plugins []Plugin

	FileExists func(path string) bool

	// BoilerplateOptional, if true, skips errors reading the Boilerplate file
	BoilerplateOptional bool

	// ConfigOptional, if true, skips errors reading the project configuration
	ConfigOptional bool
}

// Plugin is the interface that a plugin must implement
// We will (later) have an ExecPlugin that implements this by exec-ing a binary
type Plugin interface {
	// Pipe is the core plugin interface, that transforms a UniverseModel
	Pipe(universe *model.Universe) error
}

func (s *Scaffold) setFields(t input.File) {
	// Inject project configuration into file templates
	if s.Config != nil {
		if b, ok := t.(input.Domain); ok {
			b.SetDomain(s.Config.Domain)
		}
		if b, ok := t.(input.Version); ok {
			b.SetVersion(s.Config.Version)
		}
		if b, ok := t.(input.Repo); ok {
			b.SetRepo(s.Config.Repo)
		}
		if b, ok := t.(input.ProjecPath); ok {
			b.SetProjectPath(s.ConfigPath)
		}
		if b, ok := t.(input.MultiGroup); ok {
			b.SetMultiGroup(s.Config.MultiGroup)
		}
	}
	// Inject boilerplate into file templates
	if s.BoilerplatePath != "" {
		if b, ok := t.(input.BoilerplatePath); ok {
			b.SetBoilerplatePath(s.BoilerplatePath)
		}
	}
	if s.Boilerplate != "" {
		if b, ok := t.(input.Boilerplate); ok {
			b.SetBoilerplate(s.Boilerplate)
		}
	}
}

func validate(file input.File) error {
	if reqValFile, ok := file.(input.RequiresValidation); ok {
		return reqValFile.Validate()
	}

	return nil
}

func (s *Scaffold) defaultOptions(options *input.Options) error {
	// Use the default Boilerplate path if unset
	if options.BoilerplatePath == "" {
		options.BoilerplatePath = filepath.Join("hack", "boilerplate.go.txt")
	}

	// Use the default Project path if unset
	if options.ProjectPath == "" {
		options.ProjectPath = internalconfig.DefaultPath
	}

	s.BoilerplatePath = options.BoilerplatePath

	var err error
	s.Config, err = internalconfig.ReadFrom(options.ProjectPath)
	if !s.ConfigOptional && err != nil {
		return err
	}

	var boilerplateBytes []byte
	boilerplateBytes, err = ioutil.ReadFile(options.BoilerplatePath) // nolint:gosec
	if !s.BoilerplateOptional && err != nil {
		return err
	}
	s.Boilerplate = string(boilerplateBytes)

	return nil
}

func (s *Scaffold) universeDefaults(universe *model.Universe, files int) {
	if universe.Config == nil {
		universe.Config = s.Config
	}

	if universe.Boilerplate == "" {
		universe.Boilerplate = s.Boilerplate
	}

	universe.Files = make([]*model.File, 0, files)
}

// Execute executes scaffolding the for files
func (s *Scaffold) Execute(
	universe *model.Universe,
	options input.Options,
	files ...input.File,
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

	if err := s.defaultOptions(&options); err != nil {
		return err
	}

	s.universeDefaults(universe, len(files))

	// Set the repo as the local prefix so that it knows how to group imports
	imports.LocalPrefix = universe.Config.Repo

	for _, f := range files {
		m, err := s.buildFileModel(f)
		if err != nil {
			return err
		}
		universe.Files = append(universe.Files, m)
	}

	for _, plugin := range s.Plugins {
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

// doFile scaffolds a single file
func (s *Scaffold) buildFileModel(e input.File) (*model.File, error) {
	// Set common fields
	s.setFields(e)

	// Validate the file scaffold
	if err := validate(e); err != nil {
		return nil, err
	}

	// Get the template input params
	i, err := e.GetInput()
	if err != nil {
		return nil, err
	}

	m := &model.File{
		Path: i.Path,
	}

	b, err := doTemplate(i, e)
	if err != nil {
		return nil, err
	}
	m.Contents = string(b)

	return m, nil
}

func (s *Scaffold) writeFile(file *model.File) error {
	// Check if the file to write already exists
	if s.FileExists(file.Path) {
		switch file.IfExistsAction {
		case input.Overwrite:
		case input.Skip:
			return nil
		case input.Error:
			return fmt.Errorf("%s already exists", file.Path)
		}
	}

	f, err := s.GetWriter(file.Path)
	if err != nil {
		return err
	}
	if c, ok := f.(io.Closer); ok {
		defer func() {
			if err := c.Close(); err != nil {
				log.Fatal(err)
			}
		}()
	}

	_, err = f.Write([]byte(file.Contents))

	return err
}

// doTemplate executes the template for a file using the input
func doTemplate(i input.Input, e input.File) ([]byte, error) {
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
func newTemplate(t input.File) *template.Template {
	return template.New(fmt.Sprintf("%T", t)).Funcs(template.FuncMap{
		"title": strings.Title,
		"lower": strings.ToLower,
	})
}
