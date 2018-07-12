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
	yaml "gopkg.in/yaml.v2"
	"sigs.k8s.io/controller-tools/pkg/scaffold/input"
	"sigs.k8s.io/controller-tools/pkg/util"
)

// Scaffold writes Templates to scaffold new files
type Scaffold struct {
	// BoilerplatePath is the path to the boilerplate file
	BoilerplatePath string

	// Boilerplate is the contents of the boilerplate file for code generation
	Boilerplate string

	BoilerplateOptional bool

	// Project is the project
	Project input.ProjectFile

	ProjectOptional bool

	// ProjectPath is the relative path to the project root
	ProjectPath string

	GetWriter func(path string) (io.Writer, error)
}

func (s *Scaffold) setFieldsAndValidate(t input.File) error {
	// Set boilerplate on templates
	if b, ok := t.(input.BoilerplatePath); ok {
		b.SetBoilerplatePath(s.BoilerplatePath)
	}
	if b, ok := t.(input.Boilerplate); ok {
		b.SetBoilerplate(s.Boilerplate)
	}
	if b, ok := t.(input.Domain); ok {
		b.SetDomain(s.Project.Domain)
	}
	if b, ok := t.(input.Version); ok {
		b.SetVersion(s.Project.Version)
	}
	if b, ok := t.(input.Repo); ok {
		b.SetRepo(s.Project.Repo)
	}
	if b, ok := t.(input.ProjecPath); ok {
		b.SetProjectPath(s.ProjectPath)
	}

	// Validate the template is ok
	if v, ok := t.(input.Validate); ok {
		if err := v.Validate(); err != nil {
			return err
		}
	}
	return nil
}

// GetProject reads the project file and deserializes it into a Project
func getProject(path string) (input.ProjectFile, error) {
	in, err := ioutil.ReadFile(path)
	if err != nil {
		return input.ProjectFile{}, err
	}
	p := input.ProjectFile{}
	err = yaml.Unmarshal(in, &p)
	if err != nil {
		return input.ProjectFile{}, err
	}
	return p, nil
}

// GetBoilerplate reads the boilerplate file
func getBoilerplate(path string) (string, error) {
	b, err := ioutil.ReadFile(path)
	return string(b), err
}

func (s *Scaffold) defaultOptions(options *input.Options) error {
	// Use the default Boilerplate path if unset
	if options.BoilerplatePath == "" {
		options.BoilerplatePath = filepath.Join("hack", "boilerplate.go.txt")
	}

	// Use the default Project path if unset
	if options.ProjectPath == "" {
		options.ProjectPath = "PROJECT"
	}

	s.BoilerplatePath = options.BoilerplatePath

	var err error
	s.Boilerplate, err = getBoilerplate(options.BoilerplatePath)
	if !s.BoilerplateOptional && err != nil {
		return err
	}

	s.Project, err = getProject(options.ProjectPath)
	if !s.ProjectOptional && err != nil {
		return err
	}
	return nil
}

// Execute executes scaffolding the Files
func (s *Scaffold) Execute(options input.Options, files ...input.File) error {
	if s.GetWriter == nil {
		s.GetWriter = util.NewWriteCloser
	}

	if err := s.defaultOptions(&options); err != nil {
		return err
	}
	for _, f := range files {
		if err := s.doFile(f); err != nil {
			return err
		}
	}
	return nil
}

// doFile scaffolds a single file
func (s *Scaffold) doFile(e input.File) error {
	// Set common fields
	err := s.setFieldsAndValidate(e)
	if err != nil {
		return err
	}

	// Get the template input params
	i, err := e.GetInput()
	if err != nil {
		return err
	}

	// Check if the file to write already exists
	if _, err := os.Stat(i.Path); err == nil {
		switch i.IfExistsAction {
		case input.Overwrite:
		case input.Skip:
			return nil
		case input.Error:
			return fmt.Errorf("%s already exists", i.Path)
		}
	}

	if err := s.doTemplate(i, e); err != nil {
		return err
	}
	return nil
}

// doTemplate executes the template for a file using the input
func (s *Scaffold) doTemplate(i input.Input, e input.File) error {
	temp, err := newTemplate(e).Parse(i.TemplateBody)
	if err != nil {
		return err
	}
	f, err := s.GetWriter(i.Path)
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

	out := &bytes.Buffer{}
	err = temp.Execute(out, e)
	if err != nil {
		return err
	}
	b := out.Bytes()

	// gofmt the imports
	if filepath.Ext(i.Path) == ".go" {
		b, err = imports.Process(i.Path, b, nil)
		if err != nil {
			fmt.Printf("%s\n", out.Bytes())
			return err
		}
	}

	_, err = f.Write(b)
	return err
}

// newTemplate a new template with common functions
func newTemplate(t input.File) *template.Template {
	return template.New(fmt.Sprintf("%T", t)).Funcs(template.FuncMap{
		"title": strings.Title,
		"lower": strings.ToLower,
	})
}
