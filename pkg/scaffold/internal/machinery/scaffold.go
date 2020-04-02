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
	"io/ioutil"
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
	// Execute writes to disk the provided files
	Execute(*model.Universe, ...file.Builder) error
}

// scaffold implements Scaffold interface
type scaffold struct {
	// plugins is the list of plugins we should allow to transform our generated scaffolding
	plugins []model.Plugin

	// fs allows to mock the file system for tests
	fs filesystem.FileSystem
}

// NewScaffold returns a new Scaffold with the provided plugins
func NewScaffold(plugins ...model.Plugin) Scaffold {
	return &scaffold{
		plugins: plugins,
		fs:      filesystem.New(),
	}
}

// Execute implements Scaffold.Execute
func (s *scaffold) Execute(universe *model.Universe, files ...file.Builder) error {
	// Initialize the universe files
	universe.Files = make(map[string]*file.File, len(files))

	// Set the repo as the local prefix so that it knows how to group imports
	if universe.Config != nil {
		imports.LocalPrefix = universe.Config.Repo
	}

	for _, f := range files {
		// Inject common fields
		universe.InjectInto(f)

		// Validate file builders
		if reqValFile, requiresValidation := f.(file.RequiresValidation); requiresValidation {
			if err := reqValFile.Validate(); err != nil {
				return file.NewValidateError(err)
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
			return model.NewPluginError(err)
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
func (scaffold) buildFileModel(t file.Template, models map[string]*file.File) error {
	// Set the template default values
	err := t.SetTemplateDefaults()
	if err != nil {
		return file.NewSetTemplateDefaultsError(err)
	}

	// Handle already existing models
	if _, found := models[t.GetPath()]; found {
		switch t.GetIfExistsAction() {
		case file.Skip:
			return nil
		case file.Error:
			return modelAlreadyExistsError{t.GetPath()}
		case file.Overwrite:
		default:
			return unknownIfExistsActionError{t.GetPath(), t.GetIfExistsAction()}
		}
	}

	m := &file.File{
		Path:           t.GetPath(),
		IfExistsAction: t.GetIfExistsAction(),
	}

	b, err := doTemplate(t)
	if err != nil {
		return err
	}
	m.Contents = string(b)

	models[m.Path] = m
	return nil
}

// doTemplate executes the template for a file using the input
func doTemplate(t file.Template) ([]byte, error) {
	temp, err := newTemplate(t).Parse(t.GetBody())
	if err != nil {
		return nil, err
	}

	out := &bytes.Buffer{}
	err = temp.Execute(out, t)
	if err != nil {
		return nil, err
	}
	b := out.Bytes()

	// TODO(adirio): move go-formatting to write step
	// gofmt the imports
	if filepath.Ext(t.GetPath()) == ".go" {
		b, err = imports.Process(t.GetPath(), b, &options)
		if err != nil {
			return nil, err
		}
	}

	return b, nil
}

// newTemplate a new template with common functions
func newTemplate(t file.Template) *template.Template {
	fm := file.DefaultFuncMap()
	useFM, ok := t.(file.UseCustomFuncMap)
	if ok {
		fm = useFM.GetFuncMap()
	}
	return template.New(fmt.Sprintf("%T", t)).Funcs(fm)
}

// updateFileModel updates a single file
func (s scaffold) updateFileModel(i file.Inserter, models map[string]*file.File) error {
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
func (s scaffold) loadPreviousModel(i file.Inserter, models map[string]*file.File) (*file.File, error) {
	// Lets see if we already have a model for this file
	if m, found := models[i.GetPath()]; found {
		// Check if there is already an scaffolded file
		exists, err := s.fs.Exists(i.GetPath())
		if err != nil {
			return nil, err
		}

		// If there is a model but no scaffolded file we return the model
		if !exists {
			return m, nil
		}

		// If both a model and a file are found, check which has preference
		switch m.IfExistsAction {
		case file.Skip:
			// File has preference
			fromFile, err := s.loadModelFromFile(i.GetPath())
			if err != nil {
				return m, nil
			}
			return fromFile, nil
		case file.Error:
			// Writing will result in an error, so we can return error now
			return nil, fileAlreadyExistsError{i.GetPath()}
		case file.Overwrite:
			// Model has preference
			return m, nil
		default:
			return nil, unknownIfExistsActionError{i.GetPath(), m.IfExistsAction}
		}
	}

	// There was no model
	return s.loadModelFromFile(i.GetPath())
}

// loadModelFromFile gets the previous model from the actual file
func (s scaffold) loadModelFromFile(path string) (f *file.File, err error) {
	reader, err := s.fs.Open(path)
	if err != nil {
		return
	}
	defer func() {
		closeErr := reader.Close()
		if err == nil {
			err = closeErr
		}
	}()

	content, err := ioutil.ReadAll(reader)
	if err != nil {
		return
	}

	f = &file.File{Path: path, Contents: string(content)}
	return
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
			if strings.TrimSpace(line) == strings.TrimSpace(marker.String()) {
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

func (s scaffold) writeFile(f *file.File) error {
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
			return fileAlreadyExistsError{f.Path}
		}
	}

	writer, err := s.fs.Create(f.Path)
	if err != nil {
		return err
	}

	_, err = writer.Write([]byte(f.Contents))

	return err
}
