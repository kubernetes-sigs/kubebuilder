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

package manager

import (
	"sigs.k8s.io/kubebuilder/pkg/model/file"
)

var _ file.Template = &Dockerfile{}

// Dockerfile scaffolds a Dockerfile for building a main
type Dockerfile struct {
	file.TemplateMixin
	file.RepositoryMixin
}

// GetTemplateMixin implements input.Template
func (f *Dockerfile) GetTemplateMixin() (file.TemplateMixin, error) {
	if f.Path == "" {
		f.Path = "Dockerfile"
	}
	f.TemplateBody = dockerfileTemplate
	return f.TemplateMixin, nil
}

const dockerfileTemplate = `# Build the manager binary
FROM golang:1.10.3 as builder

# Copy in the go src
WORKDIR /go/src/{{ .Repo }}
COPY cmd/    cmd/
COPY vendor/ vendor/
COPY pkg/    pkg/

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o manager {{ .Repo }}/cmd/manager

# Copy the controller-manager into a thin image
FROM ubuntu:latest
WORKDIR /
COPY --from=builder /go/src/{{ .Repo }}/manager .
ENTRYPOINT ["/manager"]
`
