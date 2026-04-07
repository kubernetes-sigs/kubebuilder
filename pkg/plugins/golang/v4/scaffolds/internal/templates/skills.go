/*
Copyright 2026 The Kubernetes Authors.

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
	"strings"

	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
)

var _ machinery.Template = &Skills{}

// Skills scaffolds a SKILLS.MD file.
type Skills struct {
	machinery.TemplateMixin
	machinery.ProjectNameMixin

	// CommandName stores the name of the bin used.
	CommandName string
	// IsKubebuilderCLI indicates if kubebuilder CLI is being used (vs operator-sdk, etc).
	IsKubebuilderCLI bool
}

// SetTemplateDefaults implements machinery.Template.
func (f *Skills) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = "SKILLS.MD"
	}

	if f.CommandName != "" {
		f.IsKubebuilderCLI = strings.Contains(f.CommandName, "kubebuilder")
	}

	f.TemplateBody = agentsFileTemplate

	return nil
}
