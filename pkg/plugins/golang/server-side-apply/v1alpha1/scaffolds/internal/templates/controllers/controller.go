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

var _ machinery.Template = &Controller{}

// Controller scaffolds a controller using Server-Side Apply patterns
type Controller struct {
	machinery.TemplateMixin
	machinery.MultiGroupMixin
	machinery.BoilerplateMixin
	machinery.ResourceMixin
	machinery.ProjectNameMixin
	machinery.RepositoryMixin
	machinery.NamespacedMixin

	ControllerRuntimeVersion string

	Force bool
}

// SetTemplateDefaults implements machinery.Template
func (f *Controller) SetTemplateDefaults() error {
	if f.Path == "" {
		if f.MultiGroup && f.Resource.Group != "" {
			f.Path = filepath.Join("internal", "controller", "%[group]", "%[kind]_controller.go")
		} else {
			f.Path = filepath.Join("internal", "controller", "%[kind]_controller.go")
		}
	}

	f.Path = f.Resource.Replacer().Replace(f.Path)
	log.Info(f.Path)

	f.TemplateBody = controllerTemplate

	// Always overwrite because go/v4 creates the controller first in the plugin chain
	f.IfExistsAction = machinery.OverwriteFile

	return nil
}

//nolint:lll
const controllerTemplate = `{{ .Boilerplate }}

package {{ if and .MultiGroup .Resource.Group }}{{ .Resource.PackageName }}{{ else }}controller{{ end }}

import (
	"context"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	{{ if not (isEmptyStr .Resource.Path) -}}
	{{ .Resource.ImportAlias }} "{{ .Resource.Path }}"
	// TODO(user): Uncomment the following imports after running 'make generate'
	// metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	// {{ .Resource.ImportAlias }}apply "{{ .Resource.Path }}/applyconfiguration"
	{{- end }}
)

// {{ .Resource.Kind }}Reconciler reconciles a {{ .Resource.Kind }} object using Server-Side Apply
type {{ .Resource.Kind }}Reconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

{{ if .Namespaced -}}
// +kubebuilder:rbac:groups={{ .Resource.QualifiedGroup }},namespace={{ .ProjectName }}-system,resources={{ .Resource.Plural }},verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups={{ .Resource.QualifiedGroup }},namespace={{ .ProjectName }}-system,resources={{ .Resource.Plural }}/status,verbs=get;update;patch
// +kubebuilder:rbac:groups={{ .Resource.QualifiedGroup }},namespace={{ .ProjectName }}-system,resources={{ .Resource.Plural }}/finalizers,verbs=update
{{- else -}}
// +kubebuilder:rbac:groups={{ .Resource.QualifiedGroup }},resources={{ .Resource.Plural }},verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups={{ .Resource.QualifiedGroup }},resources={{ .Resource.Plural }}/status,verbs=get;update;patch
// +kubebuilder:rbac:groups={{ .Resource.QualifiedGroup }},resources={{ .Resource.Plural }}/finalizers,verbs=update
{{- end }}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// This controller uses Server-Side Apply to manage resources. Server-Side Apply provides:
// - Declarative field ownership tracking
// - Conflict detection when multiple controllers manage the same resource
// - Safer field management when resources are shared with users
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@{{ .ControllerRuntimeVersion }}/pkg/reconcile
func (r *{{ .Resource.Kind }}Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// Fetch the {{ .Resource.Kind }} instance
	// This is optional - you might build the apply configuration without fetching first
	{{ if not (isEmptyStr .Resource.Path) -}}
	var {{ lower .Resource.Kind }} {{ .Resource.ImportAlias }}.{{ .Resource.Kind }}
	if err := r.Get(ctx, req.NamespacedName, &{{ lower .Resource.Kind }}); err != nil {
		if errors.IsNotFound(err) {
			// Resource not found - might have been deleted
			log.Info("{{ .Resource.Kind }} resource not found, ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get {{ .Resource.Kind }}")
		return ctrl.Result{}, err
	}
	{{- end }}

	// TODO(user): Implement Server-Side Apply logic
	// 1. Run 'make generate' to create apply configuration types
	// 2. Uncomment the import above for {{ .Resource.ImportAlias }}apply
	// 3. Uncomment and customize the code below
	//
	// Build the desired state using apply configuration
	// Only specify the fields you want this controller to manage
	// User customizations (labels, annotations, other fields) will be preserved
	{{ if not (isEmptyStr .Resource.Path) -}}
	//
	// {{ lower .Resource.Kind }}Apply := {{ .Resource.ImportAlias }}apply.{{ .Resource.Kind }}(req.Name, req.Namespace)
	// // Add the fields you want to manage, for example:
	// // {{ lower .Resource.Kind }}Apply = {{ lower .Resource.Kind }}Apply.WithSpec(
	// //     {{ .Resource.ImportAlias }}apply.{{ .Resource.Kind }}Spec().
	// //         WithYourField("value"))
	//
	// // Apply the desired state using Server-Side Apply
	// // The FieldOwner identifies this controller in the managed fields
	// if err := r.Apply(ctx, {{ lower .Resource.Kind }}Apply, client.ForceOwnership, client.FieldOwner("{{ lower .Resource.Kind }}-controller")); err != nil {
	//     log.Error(err, "Failed to apply {{ .Resource.Kind }}")
	//     return ctrl.Result{}, err
	// }
	//
	// log.Info("Successfully applied {{ .Resource.Kind }}")
	{{- end }}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *{{ .Resource.Kind }}Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		{{ if not (isEmptyStr .Resource.Path) -}}
		For(&{{ .Resource.ImportAlias }}.{{ .Resource.Kind }}{}).
		{{- else -}}
		// Uncomment the following line adding a pointer to an instance of the controlled resource as an argument
		// For().
		{{- end }}
		{{- if and (.MultiGroup) (not (isEmptyStr .Resource.Group)) }}
		Named("{{ lower .Resource.Group }}-{{ lower .Resource.Kind }}").
		{{- else }}
		Named("{{ lower .Resource.Kind }}").
		{{- end }}
		Complete(r)
}
`
