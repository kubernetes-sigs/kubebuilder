/*
Copyright 2019 The Kubernetes Authors.

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

	"sigs.k8s.io/kubebuilder/pkg/model/file"
)

var _ file.Template = &Dockerfile{}

const (
	defaultDockerfilePath = "Dockerfile"
)

// Dockerfile scaffolds a Dockerfile for building a main
type Dockerfile struct {
	file.TemplateMixin
}

// SetTemplateDefaults implements input.Template
func (f *Dockerfile) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = defaultDockerfilePath
	}

	f.TemplateBody = fmt.Sprintf(dockerfileTemplate,
		file.NewMarkerFor(f.Path, copyMarker),
	)

	return nil
}

var _ file.Inserter = &DockerfileUpdater{}

type DockerfileUpdater struct { //nolint:maligned
	file.MultiGroupMixin
	// Flags to indicate which parts need to be included when updating the file
	HasResource, HasController bool
}

// GetPath implements Builder
func (*DockerfileUpdater) GetPath() string {
	return defaultDockerfilePath
}

// GetIfExistsAction implements Builder
func (*DockerfileUpdater) GetIfExistsAction() file.IfExistsAction {
	return file.Overwrite
}

const (
	copyMarker = "copy"
)

// GetMarkers implements file.Inserter
func (f *DockerfileUpdater) GetMarkers() []file.Marker {
	return []file.Marker{
		file.NewMarkerFor(defaultDockerfilePath, copyMarker),
	}
}

// GetCodeFragments implements file.Inserter
func (f *DockerfileUpdater) GetCodeFragments() file.CodeFragmentsMap {
	var fragment file.CodeFragments

	if f.HasResource && !f.MultiGroup {
		fragment = append(fragment, "COPY api/ api/\n")
	} else if f.HasResource {
		fragment = append(fragment, "COPY apis/ apis/\n")
	}

	if f.HasController {
		fragment = append(fragment, "COPY controllers/ controllers/\n")
	}

	fragments := make(file.CodeFragmentsMap, 1)
	if len(fragment) > 0 {
		fragments[file.NewMarkerFor(defaultDockerfilePath, copyMarker)] = fragment
	}

	return fragments
}

const dockerfileTemplate = `# Build the manager binary
FROM golang:1.13 as builder

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY main.go main.go
%s

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -a -o manager main.go

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /workspace/manager .
USER nonroot:nonroot

ENTRYPOINT ["/manager"]
`
