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
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/afero"
)

const (
	createOrUpdate = os.O_WRONLY | os.O_CREATE | os.O_TRUNC

	defaultDirectoryPermission os.FileMode = 0700
	defaultFilePermission      os.FileMode = 0600
)

// FileSystem is an IO wrapper to create files
type FileSystem interface {
	// Exists checks if the file exists
	Exists(path string) (bool, error)

	// Open opens the file and returns a self-closing io.Reader.
	Open(path string) (io.ReadCloser, error)

	// Create creates the directory and file and returns a self-closing
	// io.Writer pointing to that file. If the file exists, it truncates it.
	Create(path string) (io.Writer, error)
}

// fileSystem implements FileSystem
type fileSystem struct {
	fs       afero.Fs
	dirPerm  os.FileMode
	filePerm os.FileMode
	fileMode int
}

// New returns a new FileSystem
func New(options ...Options) FileSystem {
	// Default values
	fs := fileSystem{
		fs:       afero.NewOsFs(),
		dirPerm:  defaultDirectoryPermission,
		filePerm: defaultFilePermission,
		fileMode: createOrUpdate,
	}

	// Apply options
	for _, option := range options {
		option(&fs)
	}

	return fs
}

// Options configure FileSystem
type Options func(system *fileSystem)

// DirectoryPermissions makes FileSystem.Create use the provided directory
// permissions
func DirectoryPermissions(dirPerm os.FileMode) Options {
	return func(fs *fileSystem) {
		fs.dirPerm = dirPerm
	}
}

// FilePermissions makes FileSystem.Create use the provided file permissions
func FilePermissions(filePerm os.FileMode) Options {
	return func(fs *fileSystem) {
		fs.filePerm = filePerm
	}
}

// Exists implements FileSystem.Exists
func (fs fileSystem) Exists(path string) (bool, error) {
	exists, err := afero.Exists(fs.fs, path)
	if err != nil {
		return exists, fileExistsError{path, err}
	}

	return exists, nil
}

// Open implements FileSystem.Open
func (fs fileSystem) Open(path string) (io.ReadCloser, error) {
	rc, err := fs.fs.Open(path)
	if err != nil {
		return nil, openFileError{path, err}
	}

	return &readFile{path, rc}, nil
}

// Create implements FileSystem.Create
func (fs fileSystem) Create(path string) (io.Writer, error) {
	// Create the directory if needed
	if err := fs.fs.MkdirAll(filepath.Dir(path), fs.dirPerm); err != nil {
		return nil, createDirectoryError{path, err}
	}

	// Create or truncate the file
	wc, err := fs.fs.OpenFile(path, fs.fileMode, fs.filePerm)
	if err != nil {
		return nil, createFileError{path, err}
	}

	return &writeFile{path, wc}, nil
}

var _ io.ReadCloser = &readFile{}

// readFile implements io.Reader
type readFile struct {
	path string
	io.ReadCloser
}

// Read implements io.Reader.ReadCloser
func (f *readFile) Read(content []byte) (n int, err error) {
	// Read the content
	n, err = f.ReadCloser.Read(content)
	// EOF is a special case error that we can't wrap
	if err == io.EOF {
		return
	}
	if err != nil {
		return n, readFileError{f.path, err}
	}

	return n, nil
}

// Close implements io.Reader.ReadCloser
func (f *readFile) Close() error {
	if err := f.ReadCloser.Close(); err != nil {
		return closeFileError{f.path, err}
	}

	return nil
}

// writeFile implements io.Writer
type writeFile struct {
	path string
	io.WriteCloser
}

// Write implements io.Writer.Write
func (f *writeFile) Write(content []byte) (n int, err error) {
	// Close the file when we end writing
	defer func() {
		if closeErr := f.Close(); err == nil && closeErr != nil {
			err = closeFileError{f.path, err}
		}
	}()

	// Write the content
	n, err = f.WriteCloser.Write(content)
	if err != nil {
		return n, writeFileError{f.path, err}
	}

	return n, nil
}
