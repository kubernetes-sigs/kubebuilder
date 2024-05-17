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
	"path/filepath"

	log "github.com/sirupsen/logrus"

	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
)

var _ machinery.Template = &Controller{}

// Controller scaffolds the file that defines the controller for a CRD or a builtin resource
// nolint:maligned
type Controller struct {
	machinery.TemplateMixin
	machinery.MultiGroupMixin
	machinery.BoilerplateMixin
	machinery.ResourceMixin
	machinery.ProjectNameMixin

	ControllerRuntimeVersion string

	PackageName string
}

// SetTemplateDefaults implements file.Template
func (f *Controller) SetTemplateDefaults() error {
	if f.Path == "" {
		if f.MultiGroup && f.Resource.Group != "" {
			f.Path = filepath.Join("internal", "controller", "%[group]", "%[kind]_controller.go")
		} else {
			f.Path = filepath.Join("internal", "controller", "%[kind]_controller.go")
		}
	}
	f.Path = f.Resource.Replacer().Replace(f.Path)
	log.Println(f.Path)

	f.PackageName = "controller"

	log.Println("creating import for %", f.Resource.Path)
	f.TemplateBody = controllerTemplate

	// This one is to overwrite the controller if it exist
	f.IfExistsAction = machinery.OverwriteFile

	return nil
}

//nolint:lll
const controllerTemplate = `{{ .Boilerplate }}

package {{ if and .MultiGroup .Resource.Group }}{{ .Resource.PackageName }}{{ else }}{{ .PackageName }}{{ end }}

import (
	"context"
	"strings"
	"time"
	"fmt"
	"os"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	
	{{ if not (isEmptyStr .Resource.Path) -}}
	{{ .Resource.ImportAlias }} "{{ .Resource.Path }}"
	{{- end }}
)

const {{ lower .Resource.Kind }}Finalizer = "{{ .Resource.Group }}.{{ .Resource.Domain }}/finalizer"

// Definitions to manage status conditions
const (
	// typeAvailable{{ .Resource.Kind }} represents the status of the Deployment reconciliation
	typeAvailable{{ .Resource.Kind }} = "Available"
	// typeDegraded{{ .Resource.Kind }} represents the status used when the custom resource is deleted and the finalizer operations are yet to occur.
	typeDegraded{{ .Resource.Kind }} = "Degraded"
)

// {{ .Resource.Kind }}Reconciler reconciles a {{ .Resource.Kind }} object
type {{ .Resource.Kind }}Reconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Recorder record.EventRecorder
}

// The following markers are used to generate the rules permissions (RBAC) on config/rbac using controller-gen
// when the command <make manifests> is executed.
// To know more about markers see: https://book.kubebuilder.io/reference/markers.html

// +kubebuilder:rbac:groups={{ .Resource.QualifiedGroup }},resources={{ .Resource.Plural }},verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups={{ .Resource.QualifiedGroup }},resources={{ .Resource.Plural }}/status,verbs=get;update;patch
// +kubebuilder:rbac:groups={{ .Resource.QualifiedGroup }},resources={{ .Resource.Plural }}/finalizers,verbs=update
// +kubebuilder:rbac:groups=core,resources=events,verbs=create;patch
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// It is essential for the controller's reconciliation loop to be idempotent. By following the Operator
// pattern you will create Controllers which provide a reconcile function
// responsible for synchronizing resources until the desired state is reached on the cluster.
// Breaking this recommendation goes against the design principles of controller-runtime.
// and may lead to unforeseen consequences such as resources becoming stuck and requiring manual intervention.
// For further info:
// - About Operator Pattern: https://kubernetes.io/docs/concepts/extend-kubernetes/operator/
// - About Controllers: https://kubernetes.io/docs/concepts/architecture/controller/
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@{{ .ControllerRuntimeVersion }}/pkg/reconcile
func (r *{{ .Resource.Kind }}Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// Fetch the {{ .Resource.Kind }} instance
	// The purpose is check if the Custom Resource for the Kind {{ .Resource.Kind }}
	// is applied on the cluster if not we return nil to stop the reconciliation
	{{ lower .Resource.Kind }} := &{{ .Resource.ImportAlias }}.{{ .Resource.Kind }}{}
	err := r.Get(ctx, req.NamespacedName, {{ lower .Resource.Kind }})
	if err != nil {
		if apierrors.IsNotFound(err) {
			// If the custom resource is not found then it usually means that it was deleted or not created
			// In this way, we will stop the reconciliation
			log.Info("{{ lower .Resource.Kind }} resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to get {{ lower .Resource.Kind }}")
		return ctrl.Result{}, err
	}

	// Let's just set the status as Unknown when no status is available
	if {{ lower .Resource.Kind }}.Status.Conditions == nil || len({{ lower .Resource.Kind }}.Status.Conditions) == 0 {
		meta.SetStatusCondition(&{{ lower .Resource.Kind }}.Status.Conditions, metav1.Condition{Type: typeAvailable{{ .Resource.Kind }}, Status: metav1.ConditionUnknown, Reason: "Reconciling", Message: "Starting reconciliation"})
		if err = r.Status().Update(ctx, {{ lower .Resource.Kind }}); err != nil {
			log.Error(err, "Failed to update {{ .Resource.Kind }} status")
			return ctrl.Result{}, err
		}

		// Let's re-fetch the {{ lower .Resource.Kind }} Custom Resource after updating the status
		// so that we have the latest state of the resource on the cluster and we will avoid
		// raising the error "the object has been modified, please apply
		// your changes to the latest version and try again" which would re-trigger the reconciliation
		// if we try to update it again in the following operations
		if err := r.Get(ctx, req.NamespacedName, {{ lower .Resource.Kind }}); err != nil {
			log.Error(err, "Failed to re-fetch {{ lower .Resource.Kind }}")
			return ctrl.Result{}, err
		}
	}

	// Let's add a finalizer. Then, we can define some operations which should
	// occur before the custom resource is deleted.
	// More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/finalizers
	if !controllerutil.ContainsFinalizer({{ lower .Resource.Kind }}, {{ lower .Resource.Kind }}Finalizer) {
		log.Info("Adding Finalizer for {{ .Resource.Kind }}")
		if ok := controllerutil.AddFinalizer({{ lower .Resource.Kind }}, {{ lower .Resource.Kind }}Finalizer); !ok {
			log.Error(err, "Failed to add finalizer into the custom resource")
			return ctrl.Result{Requeue: true}, nil
		}
		
		if err = r.Update(ctx, {{ lower .Resource.Kind }}); err != nil {
			log.Error(err, "Failed to update custom resource to add finalizer")
			return ctrl.Result{}, err
		}
	}

	// Check if the {{ .Resource.Kind }} instance is marked to be deleted, which is
	// indicated by the deletion timestamp being set.
	is{{ .Resource.Kind }}MarkedToBeDeleted := {{ lower .Resource.Kind }}.GetDeletionTimestamp() != nil
	if is{{ .Resource.Kind }}MarkedToBeDeleted {
		if controllerutil.ContainsFinalizer({{ lower .Resource.Kind }}, {{ lower .Resource.Kind }}Finalizer) {
			log.Info("Performing Finalizer Operations for {{ .Resource.Kind }} before delete CR")

			// Let's add here a status "Downgrade" to reflect that this resource began its process to be terminated.
			meta.SetStatusCondition(&{{ lower .Resource.Kind }}.Status.Conditions, metav1.Condition{Type: typeDegraded{{ .Resource.Kind }},
				Status: metav1.ConditionUnknown, Reason: "Finalizing",
				Message: fmt.Sprintf("Performing finalizer operations for the custom resource: %s ", {{ lower .Resource.Kind }}.Name)})

			if err := r.Status().Update(ctx, {{ lower .Resource.Kind }}); err != nil {
				log.Error(err, "Failed to update {{ .Resource.Kind }} status")
				return ctrl.Result{}, err
			}

			// Perform all operations required before removing the finalizer and allow
			// the Kubernetes API to remove the custom resource.
			r.doFinalizerOperationsFor{{ .Resource.Kind }}({{ lower .Resource.Kind }})

			// TODO(user): If you add operations to the doFinalizerOperationsFor{{ .Resource.Kind }} method
			// then you need to ensure that all worked fine before deleting and updating the Downgrade status
			// otherwise, you should requeue here.

			// Re-fetch the {{ lower .Resource.Kind }} Custom Resource before updating the status
			// so that we have the latest state of the resource on the cluster and we will avoid
			// raising the error "the object has been modified, please apply
			// your changes to the latest version and try again" which would re-trigger the reconciliation
			if err := r.Get(ctx, req.NamespacedName, {{ lower .Resource.Kind }}); err != nil {
				log.Error(err, "Failed to re-fetch {{ lower .Resource.Kind }}")
				return ctrl.Result{}, err
			}

			meta.SetStatusCondition(&{{ lower .Resource.Kind }}.Status.Conditions, metav1.Condition{Type: typeDegraded{{ .Resource.Kind }},
				Status: metav1.ConditionTrue, Reason: "Finalizing",
				Message: fmt.Sprintf("Finalizer operations for custom resource %s name were successfully accomplished", {{ lower .Resource.Kind }}.Name)})
			
			if err := r.Status().Update(ctx, {{ lower .Resource.Kind }}); err != nil {
				log.Error(err, "Failed to update {{ .Resource.Kind }} status")
				return ctrl.Result{}, err
			}

			log.Info("Removing Finalizer for {{ .Resource.Kind }} after successfully perform the operations")
			if ok:= controllerutil.RemoveFinalizer({{ lower .Resource.Kind }}, {{ lower .Resource.Kind }}Finalizer); !ok{
				log.Error(err, "Failed to remove finalizer for {{ .Resource.Kind }}")
				return ctrl.Result{Requeue: true}, nil
			}

			if err := r.Update(ctx, {{ lower .Resource.Kind }}); err != nil {
				log.Error(err, "Failed to remove finalizer for {{ .Resource.Kind }}")
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// Check if the deployment already exists, if not create a new one
	found := &appsv1.Deployment{}
	err = r.Get(ctx, types.NamespacedName{Name: {{ lower .Resource.Kind }}.Name, Namespace: {{ lower .Resource.Kind }}.Namespace}, found)
	if err != nil && apierrors.IsNotFound(err) {
		// Define a new deployment
		dep, err := r.deploymentFor{{ .Resource.Kind }}({{ lower .Resource.Kind }})
		if err != nil {
			log.Error(err, "Failed to define new Deployment resource for {{ .Resource.Kind }}")

			// The following implementation will update the status
			meta.SetStatusCondition(&{{ lower .Resource.Kind }}.Status.Conditions, metav1.Condition{Type: typeAvailable{{ .Resource.Kind }},
				Status: metav1.ConditionFalse, Reason: "Reconciling",
				Message: fmt.Sprintf("Failed to create Deployment for the custom resource (%s): (%s)", {{ lower .Resource.Kind }}.Name, err)})

			if err := r.Status().Update(ctx, {{ lower .Resource.Kind }}); err != nil {
				log.Error(err, "Failed to update {{ .Resource.Kind }} status")
				return ctrl.Result{}, err
			}

			return ctrl.Result{}, err
		}

		log.Info("Creating a new Deployment",
			"Deployment.Namespace", dep.Namespace, "Deployment.Name", dep.Name)
		if err = r.Create(ctx, dep); err != nil {
			log.Error(err, "Failed to create new Deployment",
				"Deployment.Namespace", dep.Namespace, "Deployment.Name", dep.Name)
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

	// The CRD API defines that the {{ .Resource.Kind }} type have a {{ .Resource.Kind }}Spec.Size field
	// to set the quantity of Deployment instances to the desired state on the cluster.
	// Therefore, the following code will ensure the Deployment size is the same as defined
	// via the Size spec of the Custom Resource which we are reconciling.
	size := {{ lower .Resource.Kind }}.Spec.Size
	if *found.Spec.Replicas != size {
		found.Spec.Replicas = &size
		if err = r.Update(ctx, found); err != nil {
			log.Error(err, "Failed to update Deployment",
				"Deployment.Namespace", found.Namespace, "Deployment.Name", found.Name)

			// Re-fetch the {{ lower .Resource.Kind }} Custom Resource before updating the status
			// so that we have the latest state of the resource on the cluster and we will avoid
			// raising the error "the object has been modified, please apply
			// your changes to the latest version and try again" which would re-trigger the reconciliation
			if err := r.Get(ctx, req.NamespacedName, {{ lower .Resource.Kind }}); err != nil {
				log.Error(err, "Failed to re-fetch {{ lower .Resource.Kind }}")
				return ctrl.Result{}, err
			}

			// The following implementation will update the status
			meta.SetStatusCondition(&{{ lower .Resource.Kind }}.Status.Conditions, metav1.Condition{Type: typeAvailable{{ .Resource.Kind }},
				Status: metav1.ConditionFalse, Reason: "Resizing",
				Message: fmt.Sprintf("Failed to update the size for the custom resource (%s): (%s)", {{ lower .Resource.Kind }}.Name, err)})

			if err := r.Status().Update(ctx, {{ lower .Resource.Kind }}); err != nil {
				log.Error(err, "Failed to update {{ .Resource.Kind }} status")
				return ctrl.Result{}, err
			}

			return ctrl.Result{}, err
		}

		// Now, that we update the size we want to requeue the reconciliation
		// so that we can ensure that we have the latest state of the resource before
		// update. Also, it will help ensure the desired state on the cluster
		return ctrl.Result{Requeue: true}, nil
	}

	// The following implementation will update the status
	meta.SetStatusCondition(&{{ lower .Resource.Kind }}.Status.Conditions, metav1.Condition{Type: typeAvailable{{ .Resource.Kind }},
		Status: metav1.ConditionTrue, Reason: "Reconciling",
		Message: fmt.Sprintf("Deployment for custom resource (%s) with %d replicas created successfully", {{ lower .Resource.Kind }}.Name, size)})

	if err := r.Status().Update(ctx, {{ lower .Resource.Kind }}); err != nil {
		log.Error(err, "Failed to update {{ .Resource.Kind }} status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// finalize{{ .Resource.Kind }} will perform the required operations before delete the CR.
func (r *{{ .Resource.Kind }}Reconciler) doFinalizerOperationsFor{{ .Resource.Kind }}(cr *{{ .Resource.ImportAlias }}.{{ .Resource.Kind }}) {
	// TODO(user): Add the cleanup steps that the operator
	// needs to do before the CR can be deleted. Examples
	// of finalizers include performing backups and deleting
	// resources that are not owned by this CR, like a PVC.

	// Note: It is not recommended to use finalizers with the purpose of deleting resources which are
	// created and managed in the reconciliation. These ones, such as the Deployment created on this reconcile,
	// are defined as dependent of the custom resource. See that we use the method ctrl.SetControllerReference.
	// to set the ownerRef which means that the Deployment will be deleted by the Kubernetes API.
	// More info: https://kubernetes.io/docs/tasks/administer-cluster/use-cascading-deletion/

	// The following implementation will raise an event
	r.Recorder.Event(cr, "Warning", "Deleting",
		fmt.Sprintf("Custom Resource %s is being deleted from the namespace %s",
		cr.Name,
		cr.Namespace),)
}

// deploymentFor{{ .Resource.Kind }} returns a {{ .Resource.Kind }} Deployment object
func (r *{{ .Resource.Kind }}Reconciler) deploymentFor{{ .Resource.Kind }}(
	{{ lower .Resource.Kind }} *{{ .Resource.ImportAlias }}.{{ .Resource.Kind }}) (*appsv1.Deployment, error) {
	ls := labelsFor{{ .Resource.Kind }}({{ lower .Resource.Kind }}.Name)
	replicas := {{ lower .Resource.Kind }}.Spec.Size
	
	// Get the Operand image
	image, err := imageFor{{ .Resource.Kind }}()
	if err != nil {
    	return nil, err
	}

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      {{ lower .Resource.Kind }}.Name,
			Namespace: {{ lower .Resource.Kind }}.Namespace,
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
					// TODO(user): Uncomment the following code to configure the nodeAffinity expression
					// according to the platforms which are supported by your solution. It is considered
					// best practice to support multiple architectures. build your manager image using the
					// makefile target docker-buildx. Also, you can use docker manifest inspect <image>
					// to check what are the platforms supported.
					// More info: https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/#node-affinity
					//Affinity: &corev1.Affinity{
					//	NodeAffinity: &corev1.NodeAffinity{
					//		RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
					//			NodeSelectorTerms: []corev1.NodeSelectorTerm{
					//				{
					//					MatchExpressions: []corev1.NodeSelectorRequirement{
					//						{
					//							Key:      "kubernetes.io/arch",
					//							Operator: "In",
					//							Values:   []string{"amd64", "arm64", "ppc64le", "s390x"},
					//						},
					//						{
					//							Key:      "kubernetes.io/os",
					//							Operator: "In",
					//							Values:   []string{"linux"},
					//						},
					//					},
					//				},
					//			},
					//		},
					//	},
					//},
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
	
	// Set the ownerRef for the Deployment
	// More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/owners-dependents/
	if err := ctrl.SetControllerReference({{ lower .Resource.Kind }}, dep, r.Scheme); err != nil {
		return nil, err
	}
	return dep, nil
}

// labelsFor{{ .Resource.Kind }} returns the labels for selecting the resources
// More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/common-labels/
func labelsFor{{ .Resource.Kind }}(name string) map[string]string {
	var imageTag string
	image, err := imageFor{{ .Resource.Kind }}()
	if err == nil {
		imageTag = strings.Split(image, ":")[1]
	}
	return map[string]string{"app.kubernetes.io/name": "{{ .ProjectName }}",
		"app.kubernetes.io/version": imageTag,
		"app.kubernetes.io/managed-by": "{{ .Resource.Kind }}Controller",
	}
}

// imageFor{{ .Resource.Kind }} gets the Operand image which is managed by this controller
// from the {{ upper .Resource.Kind }}_IMAGE environment variable defined in the config/manager/manager.yaml
func imageFor{{ .Resource.Kind }}() (string, error) {
	var imageEnvVar = "{{ upper .Resource.Kind }}_IMAGE"
    image, found := os.LookupEnv(imageEnvVar)
    if !found {
        return "", fmt.Errorf("Unable to find %s environment variable with the image", imageEnvVar)
    }
    return image, nil
}

// SetupWithManager sets up the controller with the Manager.
// Note that the Deployment will be also watched in order to ensure its
// desirable state on the cluster
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
