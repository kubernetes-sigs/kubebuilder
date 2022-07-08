/*
Copyright 2022 The Kubernetes authors.

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
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	examplecomv1alpha1 "sigs.k8s.io/kubebuilder/testdata/project-v3-with-deploy-image/api/v1alpha1"
)

const memcachedFinalizer = "example.com.testproject.org/finalizer"

// MemcachedReconciler reconciles a Memcached object
type MemcachedReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// The following markers are used to generate the rules permissions on config/rbac using controller-gen
// when the command <make manifests> is executed.
// To know more about markers see: https://book.kubebuilder.io/reference/markers.html

//+kubebuilder:rbac:groups=example.com.testproject.org,resources=memcacheds,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=example.com.testproject.org,resources=memcacheds/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=example.com.testproject.org,resources=memcacheds/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=events,verbs=create;patch
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
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.12.2/pkg/reconcile
func (r *MemcachedReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// Fetch the Memcached instance
	// The purpose is check if the Custom Resource for the Kind Memcached
	// is applied on the cluster if not we return nill to stop the reconciliation
	memcached := &examplecomv1alpha1.Memcached{}
	err := r.Get(ctx, req.NamespacedName, memcached)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			log.Info("memcached resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to get memcached }}")
		return ctrl.Result{}, err
	}

	// Let's add a finalizer. Then, we can define some operations which should
	// occurs before the custom resource to be deleted.
	// More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/finalizers/
	// NOTE: You should not use finalizer to delete the resources that are
	// created in this reconciliation and have the ownerRef set by ctrl.SetControllerReference
	// because these will get deleted via k8s api
	if !controllerutil.ContainsFinalizer(memcached, memcachedFinalizer) {
		log.Info("Adding Finalizer for Memcached")
		controllerutil.AddFinalizer(memcached, memcachedFinalizer)
		err = r.Update(ctx, memcached)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	// Check if the Memcached instance is marked to be deleted, which is
	// indicated by the deletion timestamp being set.
	isMemcachedMarkedToBeDeleted := memcached.GetDeletionTimestamp() != nil
	if isMemcachedMarkedToBeDeleted {
		if controllerutil.ContainsFinalizer(memcached, memcachedFinalizer) {
			// Run finalization logic for memcachedFinalizer. If the
			// finalization logic fails, don't remove the finalizer so
			// that we can retry during the next reconciliation.
			log.Info("Performing Finalizer Operations for Memcached before delete CR")
			r.doFinalizerOperationsForMemcached(memcached)

			// Remove memcachedFinalizer. Once all finalizers have been
			// removed, the object will be deleted.
			if ok := controllerutil.RemoveFinalizer(memcached, memcachedFinalizer); !ok {
				if err != nil {
					log.Error(err, "Failed to remove finalizer for Memcached")
					return ctrl.Result{}, err
				}
			}
			err := r.Update(ctx, memcached)
			if err != nil {
				log.Error(err, "Failed to remove finalizer for Memcached")
			}
		}
		return ctrl.Result{}, nil
	}

	// Check if the deployment already exists, if not create a new one
	found := &appsv1.Deployment{}
	err = r.Get(ctx, types.NamespacedName{Name: memcached.Name, Namespace: memcached.Namespace}, found)
	if err != nil && apierrors.IsNotFound(err) {
		// Define a new deployment
		dep := r.deploymentForMemcached(memcached)
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

	// The API is defining that the Memcached type, have a MemcachedSpec.Size field to set the quantity of Memcached instances (CRs) to be deployed.
	// The following code ensure the deployment size is the same as the spec
	size := memcached.Spec.Size
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

// finalizeMemcached will perform the required operations before delete the CR.
func (r *MemcachedReconciler) doFinalizerOperationsForMemcached(cr *examplecomv1alpha1.Memcached) {
	// TODO(user): Add the cleanup steps that the operator
	// needs to do before the CR can be deleted. Examples
	// of finalizers include performing backups and deleting
	// resources that are not owned by this CR, like a PVC.
	// The following implementation will raise an event
	r.Recorder.Event(cr, "Warning", "Deleting",
		fmt.Sprintf("Custom Resource %s is being deleted from the namespace %s",
			cr.Name,
			cr.Namespace))
}

// deploymentForMemcached returns a Memcached Deployment object
func (r *MemcachedReconciler) deploymentForMemcached(memcached *examplecomv1alpha1.Memcached) *appsv1.Deployment {
	ls := labelsForMemcached(memcached.Name)
	replicas := memcached.Spec.Size

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      memcached.Name,
			Namespace: memcached.Namespace,
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
					Containers: []corev1.Container{{
						Image:           "memcached:1.4.36-alpine",
						Name:            "memcached",
						ImagePullPolicy: corev1.PullIfNotPresent,
						// Ensure restrictive context for the container
						// More info: https://kubernetes.io/docs/concepts/security/pod-security-standards/#restricted
						SecurityContext: &corev1.SecurityContext{
							RunAsNonRoot:             &[]bool{true}[0],
							RunAsUser:                &[]int64{1001}[0],
							AllowPrivilegeEscalation: &[]bool{false}[0],
							Capabilities: &corev1.Capabilities{
								Drop: []corev1.Capability{
									"ALL",
								},
							},
						},
						Ports: []corev1.ContainerPort{{
							ContainerPort: memcached.Spec.ContainerPort,
							Name:          "memcached",
						}},
						Command: []string{"memcached", "-m=64", "-o", "modern", "-v"},
					}},
				},
			},
		},
	}
	// Set Memcached instance as the owner and controller
	// You should use the method ctrl.SetControllerReference for all resources
	// which are created by your controller so that when the Custom Resource be deleted
	// all resources owned by it (child) will also be deleted.
	// To know more about it see: https://kubernetes.io/docs/tasks/administer-cluster/use-cascading-deletion/
	ctrl.SetControllerReference(memcached, dep, r.Scheme)
	return dep
}

// labelsForMemcached returns the labels for selecting the resources
// belonging to the given  Memcached CR name.
func labelsForMemcached(name string) map[string]string {
	return map[string]string{"type": "memcached", "memcached_cr": name}
}

// SetupWithManager sets up the controller with the Manager.
// The following code specifies how the controller is built to watch a CR
// and other resources that are owned and managed by that controller.
// In this way, the reconciliation can be re-trigged when the CR and/or the Deployment
// be created/edit/delete.
func (r *MemcachedReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&examplecomv1alpha1.Memcached{}).
		Owns(&appsv1.Deployment{}).
		Complete(r)
}
