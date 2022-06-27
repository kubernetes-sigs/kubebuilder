/*
Copyright 2022 The Kubernetes Authors.

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

	ControllerRuntimeVersion string

	Image string
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

	fmt.Println("creating import for %", f.Resource.Path)
	f.TemplateBody = controllerTemplate

	// This one is to overwrite the controller if it exist
	f.IfExistsAction = machinery.OverwriteFile

	return nil
}

//nolint:lll
const controllerTemplate = `{{ .Boilerplate }}

package {{ if and .MultiGroup .Resource.Group }}{{ .Resource.PackageName }}{{ else }}controllers{{ end }}

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"context"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	
	{{ if not (isEmptyStr .Resource.Path) -}}
	{{ .Resource.ImportAlias }} "{{ .Resource.Path }}"
	{{- end }}
)

// {{ .Resource.Kind }}Reconciler reconciles a {{ .Resource.Kind }} object
type {{ .Resource.Kind }}Reconciler struct {
	client.Client
	Scheme *runtime.Scheme
}
// The following markers are used to generate the rules permissions on config/rbac using controller-gen
// when the command <make manifests> is executed. 
// To know more about markers see: https://book.kubebuilder.io/reference/markers.html

//+kubebuilder:rbac:groups={{ .Resource.QualifiedGroup }},resources={{ .Resource.Plural }},verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups={{ .Resource.QualifiedGroup }},resources={{ .Resource.Plural }}/status,verbs=get;update;patch
//+kubebuilder:rbac:groups={{ .Resource.QualifiedGroup }},resources={{ .Resource.Plural }}/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.

// Note: It is essential for the controller's reconciliation loop to be idempotent. By following the Operator 
// pattern(https://kubernetes.io/docs/concepts/extend-kubernetes/operator/) you will create
// Controllers(https://kubernetes.io/docs/concepts/architecture/controller/) which provide a reconcile function
// responsible for synchronizing resources until the desired state is reached on the cluster. Breaking this
// recommendation goes against the design principles of Controller-runtime(https://github.com/kubernetes-sigs/controller-runtime) 
// and may lead to unforeseen consequences such as resources becoming stuck and requiring manual intervention.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@{{ .ControllerRuntimeVersion }}/pkg/reconcile
func (r *{{ .Resource.Kind }}Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// Fetch the {{ .Resource.Kind }} instance
	// The purpose is check if the Custom Resource for the Kind {{ .Resource.Kind }}
	// is applied on the cluster if not we return nill to stop the reconciliation
	{{ lower .Resource.Kind }} := &{{ .Resource.ImportAlias }}.{{ .Resource.Kind }}{}
	err := r.Get(ctx, req.NamespacedName, {{ lower .Resource.Kind }})
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			log.Info("{{ lower .Resource.Kind }} resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to get {{ lower .Resource.Kind }} }}")
		return ctrl.Result{}, err
	}

	// Check if the deployment already exists, if not create a new one
	found := &appsv1.Deployment{}
	err = r.Get(ctx, types.NamespacedName{Name: {{ lower .Resource.Kind }}.Name, Namespace: {{ lower .Resource.Kind }}.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		// Define a new deployment
		dep := r.deploymentFor{{ .Resource.Kind }}({{ lower .Resource.Kind }})
		log.Info("Creating a new Deployment", "Deployment.Namespace", dep.Namespace, "Deployment.Name", dep.Name)
		err = r.Create(ctx, dep)
		if err != nil {
			log.Error(err, "Failed to create new Deployment", "Deployment.Namespace", dep.Namespace, "Deployment.Name", dep.Name)
			return ctrl.Result{}, err
		}
		// Deployment created successfully 
		// We will requeue the reconciliation so that we can ensure the state
		// and move forward for the next operations
		return ctrl.Result{RequeueAfter: time.Minute}, nil
	} else if err != nil {
		log.Error(err, "Failed to get Deployment")
		// Let's return the error for the reconciliation be re-trigged again 
		return ctrl.Result{}, err
	}

	// The API is defining that the {{ .Resource.Kind }} type, have a {{ .Resource.Kind }}Spec.Size field to set the quantity of {{ .Resource.Kind }} instances (CRs) to be deployed. 
	// The following code ensure the deployment size is the same as the spec
	size := {{ lower .Resource.Kind }}.Spec.Size
	if *found.Spec.Replicas != size {
		found.Spec.Replicas = &size
		err = r.Update(ctx, found)
		if err != nil {
			log.Error(err, "Failed to update Deployment", "Deployment.Namespace", found.Namespace, "Deployment.Name", found.Name)
			return ctrl.Result{}, err
		}
		// Since it fails we want to re-queue the reconciliation
		// The reconciliation will only stop when we be able to ensure 
		// the desired state on the cluster
		return ctrl.Result{Requeue: true}, nil
	}
	return ctrl.Result{}, nil
}

// deploymentFor{{ .Resource.Kind }} returns a {{ .Resource.Kind }} Deployment object
func (r *{{ .Resource.Kind }}Reconciler) deploymentFor{{ .Resource.Kind }}(m *{{ .Resource.ImportAlias }}.{{ .Resource.Kind }}) *appsv1.Deployment {
	ls := labelsFor{{ .Resource.Kind }}(m.Name)
	replicas := m.Spec.Size

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.Name,
			Namespace: m.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: ls,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: ls,
				},
				Spec: corev1.PodSpec{
					SecurityContext: &corev1.PodSecurityContext{
						RunAsNonRoot: &[]bool{true}[0],
						// IMPORTANT: seccomProfile was introduced with Kubernetes 1.19
						// If you are looking for to produce solutions to be supported
						// on lower versions you must remove this option.
						SeccompProfile: &corev1.SeccompProfile{
							Type: corev1.SeccompProfileTypeRuntimeDefault,
						},
					},
					//TODO: scaffold container,
				},
			},
		},
	}
	// Set {{ .Resource.Kind }} instance as the owner and controller
	// You should use the method ctrl.SetControllerReference for all resources
	// which are created by your controller so that when the Custom Resource be deleted
	// all resources owned by it (child) will also be deleted.
	// To know more about it see: https://kubernetes.io/docs/tasks/administer-cluster/use-cascading-deletion/
	ctrl.SetControllerReference(m, dep, r.Scheme)
	return dep
}

// labelsFor{{ .Resource.Kind }} returns the labels for selecting the resources
// belonging to the given  {{ .Resource.Kind }} CR name.
func labelsFor{{ .Resource.Kind }}(name string) map[string]string {
	return map[string]string{"type": "{{ lower .Resource.Kind }}", "{{ lower .Resource.Kind }}_cr": name}
}

// SetupWithManager sets up the controller with the Manager.
// The following code specifies how the controller is built to watch a CR 
// and other resources that are owned and managed by that controller.
// In this way, the reconciliation can be re-trigged when the CR and/or the Deployment
// be created/edit/delete.
func (r *{{ .Resource.Kind }}Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		{{ if not (isEmptyStr .Resource.Path) -}}
		For(&{{ .Resource.ImportAlias }}.{{ .Resource.Kind }}{}).
		{{- else -}}
		// Uncomment the following line adding a pointer to an instance of the controlled resource as an argument
		// For().
		{{- end }}
		Owns(&appsv1.Deployment{}).
		Complete(r)
}
`
