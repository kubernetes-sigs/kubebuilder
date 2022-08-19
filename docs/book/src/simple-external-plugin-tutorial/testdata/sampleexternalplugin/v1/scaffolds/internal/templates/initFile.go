package templates

import "fmt"

// InitFile represents the InitFile.txt
type InitFile struct {
	Name     string
	Contents string
	domain   string
}

// InitFileOptions is a way to set configurable options for the Init file
type InitFileOptions func(inf *InitFile)

// WithDomain sets the number to be used in the resulting InitFile
func WithDomain(domain string) InitFileOptions {
	return func(inf *InitFile) {
		inf.domain = domain
	}
}

// NewInitFile returns a new InitFile with
func NewInitFile(opts ...InitFileOptions) *InitFile {
	initFile := &InitFile{
		Name: "initFile.txt",
	}

	for _, opt := range opts {
		opt(initFile)
	}

	initFile.Contents = fmt.Sprintf(initFileTemplate, initFile.domain)

	return initFile
}

const initFileTemplate = "A simple text file created with the `init` subcommand\nDOMAIN: %s"
