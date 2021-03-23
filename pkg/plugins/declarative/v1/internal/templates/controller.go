/*
Copyright 2021 The Kubernetes Authors.

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
	"path/filepath"

	"sigs.k8s.io/kubebuilder/v3/pkg/machinery"
)

var _ machinery.Template = &Controller{}

// Controller scaffolds the file that defines the controller for a CRD or a builtin resource
// nolint:maligned
type Controller struct {
	machinery.TemplateMixin
	machinery.MultiGroupMixin
	machinery.BoilerplateMixin
	machinery.ResourceMixin
}

// SetTemplateDefaults implements file.Template
func (f *Controller) SetTemplateDefaults() error {
	if f.Path == "" {
		if f.MultiGroup {
			f.Path = filepath.Join("controllers", "%[group]", "%[kind]_controller.go")
		} else {
			f.Path = filepath.Join("controllers", "%[kind]_controller.go")
		}
	}
	f.Path = f.Resource.Replacer().Replace(f.Path)

	f.TemplateBody = controllerTemplate

	f.IfExistsAction = machinery.OverwriteFile

	return nil
}

//nolint:lll
const controllerTemplate = `{{ .Boilerplate }}

package controllers

import (
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
	"sigs.k8s.io/kubebuilder-declarative-pattern/pkg/patterns/addon"
	"sigs.k8s.io/kubebuilder-declarative-pattern/pkg/patterns/addon/pkg/status"
	"sigs.k8s.io/kubebuilder-declarative-pattern/pkg/patterns/declarative"

	{{ .Resource.ImportAlias }} "{{ .Resource.Path }}"
)

var _ reconcile.Reconciler = &{{ .Resource.Kind }}Reconciler{}

// {{ .Resource.Kind }}Reconciler reconciles a {{ .Resource.Kind }} object
type {{ .Resource.Kind }}Reconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme

	declarative.Reconciler
}

//+kubebuilder:rbac:groups={{ .Resource.QualifiedGroup }},resources={{ .Resource.Plural }},verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups={{ .Resource.QualifiedGroup }},resources={{ .Resource.Plural }}/status,verbs=get;update;patch

// SetupWithManager sets up the controller with the Manager.
func (r *{{ .Resource.Kind }}Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	addon.Init()

	labels := map[string]string{
		"k8s-app": "{{ lower .Resource.Kind }}",
	}

	watchLabels := declarative.SourceLabel(mgr.GetScheme())

	if err := r.Reconciler.Init(mgr, &{{ .Resource.ImportAlias }}.{{ .Resource.Kind }}{},
		declarative.WithObjectTransform(declarative.AddLabels(labels)),
		declarative.WithOwner(declarative.SourceAsOwner),
		declarative.WithLabels(watchLabels),
		declarative.WithStatus(status.NewBasic(mgr.GetClient())),
		// TODO: add an application to your manifest:  declarative.WithObjectTransform(addon.TransformApplicationFromStatus),
		// TODO: add an application to your manifest:  declarative.WithManagedApplication(watchLabels),
		declarative.WithObjectTransform(addon.ApplyPatches),
	); err != nil {
		return err
	}

	c, err := controller.New("{{ lower .Resource.Kind }}-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to {{ .Resource.Kind }}
	err = c.Watch(&source.Kind{Type: &{{ .Resource.ImportAlias }}.{{ .Resource.Kind }}{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// Watch for changes to deployed objects
	_, err = declarative.WatchAll(mgr.GetConfig(), c, r, watchLabels)
	if err != nil {
		return err
	}

	return nil
}
`
