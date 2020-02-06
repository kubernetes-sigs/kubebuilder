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

package controller

import (
	"fmt"
	"path/filepath"

	"sigs.k8s.io/kubebuilder/pkg/model/file"
)

var _ file.Template = &Controller{}

// Controller scaffolds a Controller for a Resource
type Controller struct {
	file.TemplateMixin
	file.MultiGroupMixin
	file.BoilerplateMixin
	file.ResourceMixin
}

// SetTemplateDefaults implements input.Template
func (f *Controller) SetTemplateDefaults() error {
	if f.Path == "" {
		if f.MultiGroup {
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

package controllers

import (
	"context"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	{{ .Resource.ImportAlias }} "{{ .Resource.Package }}"
)

// {{ .Resource.Kind }}Reconciler reconciles a {{ .Resource.Kind }} object
type {{ .Resource.Kind }}Reconciler struct {
	client.Client
	Log logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups={{ .Resource.Domain }},resources={{ .Resource.Plural }},verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups={{ .Resource.Domain }},resources={{ .Resource.Plural }}/status,verbs=get;update;patch

func (r *{{ .Resource.Kind }}Reconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()
	_ = r.Log.WithValues("{{ .Resource.Kind | lower }}", req.NamespacedName)

	// your logic here

	return ctrl.Result{}, nil
}

func (r *{{ .Resource.Kind }}Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&{{ .Resource.ImportAlias }}.{{ .Resource.Kind }}{}).
		Complete(r)
}
`
