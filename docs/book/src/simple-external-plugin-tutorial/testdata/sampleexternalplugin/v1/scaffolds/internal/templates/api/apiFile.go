package api

import "fmt"

// ApiFile represents the apiFile.txt
type ApiFile struct {
	Name     string
	Contents string
	number   int
}

// ApiFileOptions is a way to set configurable options for the API file
type ApiFileOptions func(af *ApiFile)

// WithNumber sets the number to be used in the resulting ApiFile
func WithNumber(number int) ApiFileOptions {
	return func(af *ApiFile) {
		af.number = number
	}
}

// NewApiFile returns a new ApiFile with
func NewApiFile(opts ...ApiFileOptions) *ApiFile {
	apiFile := &ApiFile{
		Name: "apiFile.txt",
	}

	for _, opt := range opts {
		opt(apiFile)
	}

	apiFile.Contents = fmt.Sprintf(apiFileTemplate, apiFile.number)

	return apiFile
}

const apiFileTemplate = "A simple text file created with the `create api` subcommand\nNUMBER: %d"
