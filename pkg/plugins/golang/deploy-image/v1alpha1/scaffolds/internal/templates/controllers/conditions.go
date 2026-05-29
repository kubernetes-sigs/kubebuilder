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

package controllers

import (
	log "log/slog"
	"path/filepath"

	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
)

var _ machinery.Template = &Conditions{}

// Conditions scaffolds shared metav1.Condition constants for deploy-image controllers in this package.
type Conditions struct {
	machinery.TemplateMixin
	machinery.MultiGroupMixin
	machinery.BoilerplateMixin
	machinery.ResourceMixin
}

// SetTemplateDefaults implements machinery.Template
func (f *Conditions) SetTemplateDefaults() error {
	if f.Path == "" {
		if f.MultiGroup && f.Resource.Group != "" {
			f.Path = filepath.Join("internal", "controller", "%[group]", "conditions.go")
		} else {
			f.Path = filepath.Join("internal", "controller", "conditions.go")
		}
	}
	f.Path = f.Resource.Replacer().Replace(f.Path)
	log.Info(f.Path)

	f.TemplateBody = conditionsTemplate
	f.IfExistsAction = machinery.SkipFile

	return nil
}

const conditionsTemplate = `{{ .Boilerplate }}

{{if and .MultiGroup .Resource.Group }}
package {{ .Resource.PackageName }}
{{else}}
package controller
{{end}}

const (
	reasonReconciling = "Reconciling"
	reasonFinalizing  = "Finalizing"
)
`
