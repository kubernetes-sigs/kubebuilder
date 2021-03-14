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
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/spf13/afero"
	"golang.org/x/tools/imports"

	"sigs.k8s.io/kubebuilder/v3/pkg/config"
	"sigs.k8s.io/kubebuilder/v3/pkg/model"
	"sigs.k8s.io/kubebuilder/v3/pkg/model/file"
	"sigs.k8s.io/kubebuilder/v3/pkg/model/resource"
)

const (
	createOrUpdate = os.O_WRONLY | os.O_CREATE | os.O_TRUNC

	defaultDirectoryPermission os.FileMode = 0700
	defaultFilePermission      os.FileMode = 0600
)

var options = imports.Options{
	Comments:   true,
	TabIndent:  true,
	TabWidth:   8,
	FormatOnly: true,
}

// Scaffold uses templates to scaffold new files
type Scaffold struct {
	// plugins is the list of plugins we should allow to transform our generated scaffolding
	plugins []model.Plugin

	// fs allows to mock the file system for tests
	fs afero.Fs

	// permissions for new directories and files
	dirPerm  os.FileMode
	filePerm os.FileMode

	// fields to create the universe
	config      config.Config
	boilerplate string
	resource    *resource.Resource
}

// ScaffoldOption allows to provide optional arguments to the Scaffold
type ScaffoldOption func(*Scaffold)

// NewScaffold returns a new Scaffold with the provided plugins
func NewScaffold(fs Filesystem, options ...ScaffoldOption) *Scaffold {
	s := &Scaffold{
		fs:       fs.FS,
		dirPerm:  defaultDirectoryPermission,
		filePerm: defaultFilePermission,
	}

	for _, option := range options {
		option(s)
	}

	return s
}

// WithPlugins sets the plugins to be used
func WithPlugins(plugins ...model.Plugin) ScaffoldOption {
	return func(s *Scaffold) {
		s.plugins = append(s.plugins, plugins...)
	}
}

// WithDirectoryPermissions sets the permissions for new directories
func WithDirectoryPermissions(dirPerm os.FileMode) ScaffoldOption {
	return func(s *Scaffold) {
		s.dirPerm = dirPerm
	}
}

// WithFilePermissions sets the permissions for new files
func WithFilePermissions(filePerm os.FileMode) ScaffoldOption {
	return func(s *Scaffold) {
		s.filePerm = filePerm
	}
}

// WithConfig provides the project configuration to the Scaffold
func WithConfig(cfg config.Config) ScaffoldOption {
	return func(s *Scaffold) {
		s.config = cfg

		if cfg != nil && cfg.GetRepository() != "" {
			imports.LocalPrefix = cfg.GetRepository()
		}
	}
}

// WithBoilerplate provides the boilerplate to the Scaffold
func WithBoilerplate(boilerplate string) ScaffoldOption {
	return func(s *Scaffold) {
		s.boilerplate = boilerplate
	}
}

// WithResource provides the resource to the Scaffold
func WithResource(resource *resource.Resource) ScaffoldOption {
	return func(s *Scaffold) {
		s.resource = resource
	}
}

// Execute writes to disk the provided files
func (s *Scaffold) Execute(files ...file.Builder) error {
	// Initialize the universe
	universe := &model.Universe{
		Config:      s.config,
		Boilerplate: s.boilerplate,
		Resource:    s.resource,
		Files:       make(map[string]*file.File, len(files)),
	}

	// Set the repo as the local prefix so that it knows how to group imports
	if universe.Config != nil {
		imports.LocalPrefix = universe.Config.GetRepository()
	}

	for _, f := range files {
		// Inject common fields
		universe.InjectInto(f)

		// Validate file builders
		if reqValFile, requiresValidation := f.(file.RequiresValidation); requiresValidation {
			if err := reqValFile.Validate(); err != nil {
				return ValidateError{err}
			}
		}

		// Build models for Template builders
		if t, isTemplate := f.(file.Template); isTemplate {
			if err := s.buildFileModel(t, universe.Files); err != nil {
				return err
			}
		}

		// Build models for Inserter builders
		if i, isInserter := f.(file.Inserter); isInserter {
			if err := s.updateFileModel(i, universe.Files); err != nil {
				return err
			}
		}
	}

	// Execute plugins
	for _, plugin := range s.plugins {
		if err := plugin.Pipe(universe); err != nil {
			return PluginError{err}
		}
	}

	// Persist the files to disk
	for _, f := range universe.Files {
		if err := s.writeFile(f); err != nil {
			return err
		}
	}

	return nil
}

// buildFileModel scaffolds a single file
func (Scaffold) buildFileModel(t file.Template, models map[string]*file.File) error {
	// Set the template default values
	if err := t.SetTemplateDefaults(); err != nil {
		return SetTemplateDefaultsError{err}
	}

	path := t.GetPath()

	// Handle already existing models
	if _, found := models[path]; found {
		switch t.GetIfExistsAction() {
		case file.Skip:
			return nil
		case file.Error:
			return ModelAlreadyExistsError{path}
		case file.Overwrite:
		default:
			return UnknownIfExistsActionError{path, t.GetIfExistsAction()}
		}
	}

	b, err := doTemplate(t)
	if err != nil {
		return err
	}

	models[path] = &file.File{
		Path:           path,
		Contents:       string(b),
		IfExistsAction: t.GetIfExistsAction(),
	}
	return nil
}

// doTemplate executes the template for a file using the input
func doTemplate(t file.Template) ([]byte, error) {
	// Create a new template.Template using the type of the Template as the name
	temp := template.New(fmt.Sprintf("%T", t))

	// Set the function map to be used
	fm := file.DefaultFuncMap()
	if templateWithFuncMap, hasCustomFuncMap := t.(file.UseCustomFuncMap); hasCustomFuncMap {
		fm = templateWithFuncMap.GetFuncMap()
	}
	temp.Funcs(fm)

	// Set the template body
	if _, err := temp.Parse(t.GetBody()); err != nil {
		return nil, err
	}

	out := &bytes.Buffer{}
	if err := temp.Execute(out, t); err != nil {
		return nil, err
	}
	b := out.Bytes()

	// TODO(adirio): move go-formatting to write step
	// gofmt the imports
	if filepath.Ext(t.GetPath()) == ".go" {
		var err error
		if b, err = imports.Process(t.GetPath(), b, &options); err != nil {
			return nil, err
		}
	}

	return b, nil
}

// updateFileModel updates a single file
func (s Scaffold) updateFileModel(i file.Inserter, models map[string]*file.File) error {
	m, err := s.loadPreviousModel(i, models)
	if err != nil {
		return err
	}

	// Get valid code fragments
	codeFragments := getValidCodeFragments(i)

	// Remove code fragments that already were applied
	err = filterExistingValues(m.Contents, codeFragments)
	if err != nil {
		return err
	}

	// If no code fragment to insert, we are done
	if len(codeFragments) == 0 {
		return nil
	}

	content, err := insertStrings(m.Contents, codeFragments)
	if err != nil {
		return err
	}

	// TODO(adirio): move go-formatting to write step
	formattedContent := content
	if ext := filepath.Ext(i.GetPath()); ext == ".go" {
		formattedContent, err = imports.Process(i.GetPath(), content, nil)
		if err != nil {
			return err
		}
	}

	m.Contents = string(formattedContent)
	m.IfExistsAction = file.Overwrite
	models[m.Path] = m
	return nil
}

// loadPreviousModel gets the previous model from the models map or the actual file
func (s Scaffold) loadPreviousModel(i file.Inserter, models map[string]*file.File) (*file.File, error) {
	path := i.GetPath()

	// Lets see if we already have a model for this file
	if m, found := models[path]; found {
		// Check if there is already an scaffolded file
		exists, err := afero.Exists(s.fs, path)
		if err != nil {
			return nil, ExistsFileError{err}
		}

		// If there is a model but no scaffolded file we return the model
		if !exists {
			return m, nil
		}

		// If both a model and a file are found, check which has preference
		switch m.IfExistsAction {
		case file.Skip:
			// File has preference
			fromFile, err := s.loadModelFromFile(path)
			if err != nil {
				return m, nil
			}
			return fromFile, nil
		case file.Error:
			// Writing will result in an error, so we can return error now
			return nil, FileAlreadyExistsError{path}
		case file.Overwrite:
			// Model has preference
			return m, nil
		default:
			return nil, UnknownIfExistsActionError{path, m.IfExistsAction}
		}
	}

	// There was no model
	return s.loadModelFromFile(path)
}

// loadModelFromFile gets the previous model from the actual file
func (s Scaffold) loadModelFromFile(path string) (f *file.File, err error) {
	reader, err := s.fs.Open(path)
	if err != nil {
		return nil, OpenFileError{err}
	}
	defer func() {
		if closeErr := reader.Close(); err == nil && closeErr != nil {
			err = CloseFileError{closeErr}
		}
	}()

	content, err := afero.ReadAll(reader)
	if err != nil {
		return nil, ReadFileError{err}
	}

	return &file.File{Path: path, Contents: string(content)}, nil
}

// getValidCodeFragments obtains the code fragments from a file.Inserter
func getValidCodeFragments(i file.Inserter) file.CodeFragmentsMap {
	// Get the code fragments
	codeFragments := i.GetCodeFragments()

	// Validate the code fragments
	validMarkers := i.GetMarkers()
	for marker := range codeFragments {
		valid := false
		for _, validMarker := range validMarkers {
			if marker == validMarker {
				valid = true
				break
			}
		}
		if !valid {
			delete(codeFragments, marker)
		}
	}

	return codeFragments
}

// filterExistingValues removes the single-line values that already exists
// TODO: Add support for multi-line duplicate values
func filterExistingValues(content string, codeFragmentsMap file.CodeFragmentsMap) error {
	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := scanner.Text()
		for marker, codeFragments := range codeFragmentsMap {
			for i, codeFragment := range codeFragments {
				if strings.TrimSpace(line) == strings.TrimSpace(codeFragment) {
					codeFragmentsMap[marker] = append(codeFragments[:i], codeFragments[i+1:]...)
				}
			}
			if len(codeFragmentsMap[marker]) == 0 {
				delete(codeFragmentsMap, marker)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	return nil
}

func insertStrings(content string, codeFragmentsMap file.CodeFragmentsMap) ([]byte, error) {
	out := new(bytes.Buffer)

	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := scanner.Text()

		for marker, codeFragments := range codeFragmentsMap {
			if marker.EqualsLine(line) {
				for _, codeFragment := range codeFragments {
					_, _ = out.WriteString(codeFragment) // bytes.Buffer.WriteString always returns nil errors
				}
			}
		}

		_, _ = out.WriteString(line + "\n") // bytes.Buffer.WriteString always returns nil errors
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return out.Bytes(), nil
}

func (s Scaffold) writeFile(f *file.File) (err error) {
	// Check if the file to write already exists
	exists, err := afero.Exists(s.fs, f.Path)
	if err != nil {
		return ExistsFileError{err}
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
			return FileAlreadyExistsError{f.Path}
		}
	}

	// Create the directory if needed
	if err := s.fs.MkdirAll(filepath.Dir(f.Path), s.dirPerm); err != nil {
		return CreateDirectoryError{err}
	}

	// Create or truncate the file
	writer, err := s.fs.OpenFile(f.Path, createOrUpdate, s.filePerm)
	if err != nil {
		return CreateFileError{err}
	}
	defer func() {
		if closeErr := writer.Close(); err == nil && closeErr != nil {
			err = CloseFileError{err}
		}
	}()

	if _, err := writer.Write([]byte(f.Contents)); err != nil {
		return WriteFileError{err}
	}

	return nil
}
