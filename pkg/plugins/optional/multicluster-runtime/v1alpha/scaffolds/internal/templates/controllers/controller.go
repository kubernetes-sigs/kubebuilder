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
	"path/filepath"

	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v4/pkg/model/resource"
)

var _ machinery.Template = &Controller{}

// Controller scaffolds a multicluster-aware controller file.
type Controller struct {
	machinery.TemplateMixin
	machinery.MultiGroupMixin
	machinery.BoilerplateMixin
	machinery.ResourceMixin
	machinery.ProjectNameMixin

	Force bool

	ControllerName string
}

// SetTemplateDefaults implements machinery.Template.
func (f *Controller) SetTemplateDefaults() error {
	if f.Path == "" {
		fileName := "%[kind]_controller.go"
		if f.ControllerName != "" {
			fileName = resource.NormalizeFileName(f.ControllerName) + "_controller.go"
		}

		if f.MultiGroup && f.Resource.Group != "" {
			f.Path = filepath.Join("internal", "controller", "%[group]", fileName)
		} else {
			f.Path = filepath.Join("internal", "controller", fileName)
		}
	}

	f.Path = f.Resource.Replacer().Replace(f.Path)

	f.TemplateBody = controllerTemplate

	if f.Force {
		f.IfExistsAction = machinery.OverwriteFile
	} else {
		f.IfExistsAction = machinery.Error
	}

	return nil
}

// ReconcilerName returns the name for the reconciler struct.
func (f *Controller) ReconcilerName() string {
	return resource.NormalizeReconcilerName(f.ControllerName, f.Resource.Kind)
}

// ControllerRuntimeName returns the controller runtime name used in Named().
func (f *Controller) ControllerRuntimeName() string {
	return resource.GetControllerName(f.ControllerName, f.Resource.Kind, f.Resource.Group, f.MultiGroup)
}

//nolint:lll
const controllerTemplate = `{{ .Boilerplate }}

package {{ if and .MultiGroup .Resource.Group }}{{ .Resource.PackageName }}{{ else }}controller{{ end }}

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	mcbuilder "sigs.k8s.io/multicluster-runtime/pkg/builder"
	mcmanager "sigs.k8s.io/multicluster-runtime/pkg/manager"
	mcreconcile "sigs.k8s.io/multicluster-runtime/pkg/reconcile"
	{{ if not (isEmptyStr .Resource.Path) -}}
	{{ .Resource.ImportAlias }} "{{ .Resource.Path }}"
	{{- end }}
)

// {{ .ReconcilerName }} reconciles a {{ .Resource.Kind }} object across clusters
type {{ .ReconcilerName }} struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups={{ .Resource.QualifiedGroup }},resources={{ .Resource.Plural }},verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups={{ .Resource.QualifiedGroup }},resources={{ .Resource.Plural }}/status,verbs=get;update;patch
// +kubebuilder:rbac:groups={{ .Resource.QualifiedGroup }},resources={{ .Resource.Plural }}/finalizers,verbs=update

// Reconcile reconciles {{ .Resource.Kind }} objects across all clusters managed by the multicluster provider.
// req.ClusterName identifies which cluster the event originated from.
func (r *{{ .ReconcilerName }}) Reconcile(ctx context.Context, req mcreconcile.Request) (ctrl.Result, error) {
	_ = logf.FromContext(ctx)

	// TODO(user): your logic here

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the multicluster Manager.
func (r *{{ .ReconcilerName }}) SetupWithManager(mgr mcmanager.Manager) error {
	return mcbuilder.ControllerManagedBy(mgr).
		{{ if not (isEmptyStr .Resource.Path) -}}
		For(&{{ .Resource.ImportAlias }}.{{ .Resource.Kind }}{}).
		{{- else -}}
		// Uncomment the following line adding a pointer to an instance of the controlled resource as an argument
		// For().
		{{- end }}
		Named("{{ .ControllerRuntimeName }}").
		Complete(r)
}
`
