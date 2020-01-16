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

package model

import (
	"sigs.k8s.io/kubebuilder/pkg/scaffold/input"
)

// File describes a file that will be written
type File struct {
	// Path is the file to write
	Path string `json:"path,omitempty"`

	// Contents is the generated output
	Contents string `json:"contents,omitempty"`

	// TODO: Move input.IfExistsAction into model
	// IfExistsAction determines what to do if the file exists
	IfExistsAction input.IfExistsAction `json:"ifExistsAction,omitempty"`
}
