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

package controllers

import (
	"fmt"
	"path/filepath"

	"sigs.k8s.io/kubebuilder/v2/pkg/model/file"
)

var _ file.Template = &Controller{}

// Controller scaffolds the file that defines the controller for a CRD or a builtin resource
// nolint:maligned
type Controller struct {
	file.TemplateMixin
	file.MultiGroupMixin
	file.BoilerplateMixin
	file.ResourceMixin

	ControllerRuntimeVersion string

	// WireResource defines the api resources are generated or not.
	WireResource bool
}

// SetTemplateDefaults implements file.Template
func (f *Controller) SetTemplateDefaults() error {
	if f.Path == "" {
		if f.MultiGroup && f.Resource.Group != "" {
			f.Path = filepath.Join("controllers", "%[group]", "%[kind]_controller.go")
		} else {
			f.Path = filepath.Join("controllers", "%[kind]_controller.go")
		}
	}
	f.Path = f.Resource.Replacer().Replace(f.Path)
	fmt.Println(f.Path)

	f.TemplateBody = controllerTemplate

	f.IfExistsAction = file.Error

	return nil
}

//nolint:lll
const controllerTemplate = `{{ .Boilerplate }}

{{if and .MultiGroup .Resource.Group }}
package {{ .Resource.GroupPackageName }}
{{else}}
package controllers
{{end}}

import (
	"context"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	{{ if .WireResource -}}
	{{ .Resource.ImportAlias }} "{{ .Resource.Package }}"
	{{- end }}
)

// {{ .Resource.Kind }}Reconciler reconciles a {{ .Resource.Kind }} object
type {{ .Resource.Kind }}Reconciler struct {
	client.Client
	Log logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups={{ .Resource.Domain }},resources={{ .Resource.Plural }},verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups={{ .Resource.Domain }},resources={{ .Resource.Plural }}/status,verbs=get;update;patch
// +kubebuilder:rbac:groups={{ .Resource.Domain }},resources={{ .Resource.Plural }}/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the {{ .Resource.Kind }} object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@{{ .ControllerRuntimeVersion }}/pkg/reconcile
func (r *{{ .Resource.Kind }}Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = r.Log.WithValues("{{ .Resource.Kind | lower }}", req.NamespacedName)

	// your logic here

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *{{ .Resource.Kind }}Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		{{ if .WireResource -}}
		For(&{{ .Resource.ImportAlias }}.{{ .Resource.Kind }}{}).
		{{- else -}}
		// Uncomment the following line adding a pointer to an instance of the controlled resource as an argument
		// For().
		{{- end }}
		Complete(r)
}
`
