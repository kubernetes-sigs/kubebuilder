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

package templates

import (
	"fmt"
	"path/filepath"

	"sigs.k8s.io/kubebuilder/v3/pkg/machinery"
)

const (
	dockerfileMarker = "module-dependency-layers"
)
const defaultDockerfilePath = "Dockerfile"

var _ machinery.Template = &Dockerfile{}

// Dockerfile scaffolds a file that defines the containerized build process
type Dockerfile struct {
	machinery.TemplateMixin
	UseWorkspaces bool
}

// SetTemplateDefaults implements file.Template
func (f *Dockerfile) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = defaultDockerfilePath
	}

	f.TemplateBody = fmt.Sprintf(dockerfileTemplate,
		machinery.NewMarkerFor(f.Path, dockerfileMarker),
	)

	return nil
}

const dockerfileTemplate = `# Build the manager binary
FROM golang:1.18 as builder

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum

{{- if .UseWorkspaces }}

# Copy all Submodules from the workspace
%s

# Copy the Go Workspace manifests
COPY go.work go.work
COPY go.work.sum go.work.sum
RUN go work sync
{{- end }}

# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY main.go main.go
COPY api/ api/
COPY controllers/ controllers/

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o manager main.go

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /workspace/manager .
USER 65532:65532

ENTRYPOINT ["/manager"]
`

// ########################################################
// GoWorkUpdater
// ########################################################

const (
	dockerfileCodeFragment = `
COPY %s/go.mod %s/go.mod
COPY %s/go.sum %s/go.sum
`
)

var _ machinery.Inserter = &DockerfileUpdater{}

type DockerfileUpdater struct {
	machinery.RepositoryMixin
	machinery.TemplateMixin
	machinery.ResourceMixin
	UseWorkspaces bool
}

// SetTemplateDefaults implements file.Template
func (f *DockerfileUpdater) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = defaultDockerfilePath
	}

	f.TemplateBody = fmt.Sprintf(dockerfileTemplate,
		machinery.NewMarkerFor(f.Path, dockerfileMarker),
	)

	f.IfExistsAction = machinery.OverwriteFile

	return nil
}

// GetMarkers implements file.Inserter
func (f *DockerfileUpdater) GetMarkers() []machinery.Marker {
	return []machinery.Marker{
		machinery.NewMarkerFor(f.Path, dockerfileMarker),
	}
}

// GetCodeFragments implements file.Inserter
func (f *DockerfileUpdater) GetCodeFragments() machinery.CodeFragmentsMap {
	fragments := make(machinery.CodeFragmentsMap, 1)

	// If resource is not being provided we are creating the file, not updating it
	if f.Resource == nil {
		return fragments
	}

	// Generate require code fragments
	uses := make([]string, 0)

	moduleRelativePath, err := filepath.Rel(f.Repo, f.Resource.Path)
	if err != nil {
		return fragments
	}
	dep := fmt.Sprintf(dockerfileCodeFragment,
		moduleRelativePath, moduleRelativePath, moduleRelativePath, moduleRelativePath)
	uses = append(uses, dep)

	// Only store code fragments in the map if the slices are non-empty
	if len(uses) != 0 {
		fragments[machinery.NewMarkerFor(f.Path, dockerfileMarker)] = uses
	}

	return fragments
}
