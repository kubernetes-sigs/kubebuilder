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

package filesystem

import (
	"bytes"
	"io"
)

// mockFileSystem implements FileSystem
type mockFileSystem struct {
	path            string
	exists          func(path string) bool
	existsError     error
	openFileError   error
	createDirError  error
	createFileError error
	input           *bytes.Buffer
	readFileError   error
	output          *bytes.Buffer
	writeFileError  error
	closeFileError  error
}

// NewMock returns a new FileSystem
func NewMock(options ...MockOptions) FileSystem {
	// Default values
	fs := mockFileSystem{
		exists: func(_ string) bool { return false },
		output: new(bytes.Buffer),
	}

	// Apply options
	for _, option := range options {
		option(&fs)
	}

	return fs
}

// MockOptions configure FileSystem
type MockOptions func(system *mockFileSystem)

// MockPath ensures that the file created with this scaffold is at path
func MockPath(path string) MockOptions {
	return func(fs *mockFileSystem) {
		fs.path = path
	}
}

// MockExists makes FileSystem.Exists use the provided function to check if the file exists
func MockExists(exists func(path string) bool) MockOptions {
	return func(fs *mockFileSystem) {
		fs.exists = exists
	}
}

// MockExistsError makes FileSystem.Exists return err
func MockExistsError(err error) MockOptions {
	return func(fs *mockFileSystem) {
		fs.existsError = err
	}
}

// MockOpenFileError makes FileSystem.Open return err
func MockOpenFileError(err error) MockOptions {
	return func(fs *mockFileSystem) {
		fs.openFileError = err
	}
}

// MockCreateDirError makes FileSystem.Create return err
func MockCreateDirError(err error) MockOptions {
	return func(fs *mockFileSystem) {
		fs.createDirError = err
	}
}

// MockCreateFileError makes FileSystem.Create return err
func MockCreateFileError(err error) MockOptions {
	return func(fs *mockFileSystem) {
		fs.createFileError = err
	}
}

// MockInput provides a buffer where the content will be read from
func MockInput(input *bytes.Buffer) MockOptions {
	return func(fs *mockFileSystem) {
		fs.input = input
	}
}

// MockReadFileError  makes the Read method (of the io.Reader returned by FileSystem.Open) return err
func MockReadFileError(err error) MockOptions {
	return func(fs *mockFileSystem) {
		fs.readFileError = err
	}
}

// MockOutput provides a buffer where the content will be written
func MockOutput(output *bytes.Buffer) MockOptions {
	return func(fs *mockFileSystem) {
		fs.output = output
	}
}

// MockWriteFileError makes the Write method (of the io.Writer returned by FileSystem.Create) return err
func MockWriteFileError(err error) MockOptions {
	return func(fs *mockFileSystem) {
		fs.writeFileError = err
	}
}

// MockCloseFileError makes the Write method (of the io.Writer returned by FileSystem.Create) return err
func MockCloseFileError(err error) MockOptions {
	return func(fs *mockFileSystem) {
		fs.closeFileError = err
	}
}

// Exists implements FileSystem.Exists
func (fs mockFileSystem) Exists(path string) (bool, error) {
	if fs.existsError != nil {
		return false, fileExistsError{path, fs.existsError}
	}

	return fs.exists(path), nil
}

// Open implements FileSystem.Open
func (fs mockFileSystem) Open(path string) (io.ReadCloser, error) {
	if fs.openFileError != nil {
		return nil, openFileError{path, fs.openFileError}
	}

	if fs.input == nil {
		fs.input = bytes.NewBufferString("Hello world!")
	}

	return &mockReadFile{path, fs.input, fs.readFileError, fs.closeFileError}, nil
}

// Create implements FileSystem.Create
func (fs mockFileSystem) Create(path string) (io.Writer, error) {
	if fs.createDirError != nil {
		return nil, createDirectoryError{path, fs.createDirError}
	}

	if fs.createFileError != nil {
		return nil, createFileError{path, fs.createFileError}
	}

	return &mockWriteFile{path, fs.output, fs.writeFileError, fs.closeFileError}, nil
}

// mockReadFile implements io.Reader mocking a readFile for tests
type mockReadFile struct {
	path           string
	input          *bytes.Buffer
	readFileError  error
	closeFileError error
}

// Read implements io.Reader.ReadCloser
func (f *mockReadFile) Read(content []byte) (n int, err error) {
	if f.readFileError != nil {
		return 0, readFileError{path: f.path, err: f.readFileError}
	}

	return f.input.Read(content)
}

// Read implements io.Reader.ReadCloser
func (f *mockReadFile) Close() error {
	if f.closeFileError != nil {
		return closeFileError{path: f.path, err: f.closeFileError}
	}

	return nil
}

// mockWriteFile implements io.Writer mocking a writeFile for tests
type mockWriteFile struct {
	path           string
	content        *bytes.Buffer
	writeFileError error
	closeFileError error
}

// Write implements io.Writer.Write
func (f *mockWriteFile) Write(content []byte) (n int, err error) {
	defer func() {
		if err == nil && f.closeFileError != nil {
			err = closeFileError{f.path, f.closeFileError}
		}
	}()

	if f.writeFileError != nil {
		return 0, writeFileError{f.path, f.writeFileError}
	}

	return f.content.Write(content)
}
