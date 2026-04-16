/*
Copyright 2026 The Kubernetes authors.

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
	"context"
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	appsv1apply "k8s.io/client-go/applyconfigurations/apps/v1"
	corev1apply "k8s.io/client-go/applyconfigurations/core/v1"
	metav1apply "k8s.io/client-go/applyconfigurations/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	cachev1alpha1 "example.com/memcached-ssa/api/v1alpha1"
)

/*
Status condition types for Memcached.
These follow Kubernetes API conventions for condition types.
*/
const (
	// typeAvailableMemcached represents the Deployment is available
	typeAvailableMemcached = "Available"
	// typeDegradedMemcached represents failures in reconciliation
	typeDegradedMemcached = "Degraded"
)

// MemcachedReconciler reconciles a Memcached object using Server-Side Apply
type MemcachedReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=cache.example.com,resources=memcacheds,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cache.example.com,resources=memcacheds/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=cache.example.com,resources=memcacheds/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=events,verbs=create;patch

// Reconcile uses Server-Side Apply to manage resources for the Memcached.
//
// Server-Side Apply (SSA) provides declarative field ownership, allowing multiple actors
// (users, controllers) to safely manage different fields of the same resource. This controller
// only takes ownership of fields it explicitly declares, preserving user customizations to other fields.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.23.3/pkg/reconcile
func (r *MemcachedReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	/*
		Fetch the Memcached instance.
	*/
	logger := log.FromContext(ctx)
	var memcached cachev1alpha1.Memcached
	if err := r.Get(ctx, req.NamespacedName, &memcached); err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info("Memcached resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Failed to get Memcached")
		return ctrl.Result{}, err
	}

	/*
		Build the desired Deployment state using Server-Side Apply.
		Only the fields we specify will be managed by this controller.
		User customizations to other fields (labels, annotations, resources) are preserved.
	*/
	labels := labelsForMemcached(memcached.Name)

	// Set default values if not specified
	size := int32(1)
	if memcached.Spec.Size != nil {
		size = *memcached.Spec.Size
	}
	containerPort := int32(11211)
	if memcached.Spec.ContainerPort != nil {
		containerPort = *memcached.Spec.ContainerPort
	}

	/*
		Build the Deployment using apply configurations.
		Server-Side Apply will manage only the fields we set here.
	*/
	deployment := appsv1apply.Deployment(memcached.Name, memcached.Namespace).
		WithLabels(labels).
		WithSpec(appsv1apply.DeploymentSpec().
			WithReplicas(size).
			WithSelector(metav1apply.LabelSelector().
				WithMatchLabels(map[string]string{
					"app.kubernetes.io/name":     "memcached",
					"app.kubernetes.io/instance": memcached.Name,
				})).
			WithTemplate(corev1apply.PodTemplateSpec().
				WithLabels(labels).
				WithSpec(corev1apply.PodSpec().
					WithSecurityContext(corev1apply.PodSecurityContext().
						WithRunAsNonRoot(true).
						WithSeccompProfile(corev1apply.SeccompProfile().
							WithType(corev1.SeccompProfileTypeRuntimeDefault))).
					WithContainers(corev1apply.Container().
						WithName("memcached").
						WithImage("memcached:1.6.15-alpine").
						WithCommand("memcached", "-m=64", "-o", "modern", "-v").
						WithPorts(corev1apply.ContainerPort().
							WithName("memcached").
							WithContainerPort(containerPort).
							WithProtocol(corev1.ProtocolTCP))))))

	// Set owner reference using a temporary deployment object
	tempDeploy := &appsv1.Deployment{}
	tempDeploy.Name = memcached.Name
	tempDeploy.Namespace = memcached.Namespace
	if err := ctrl.SetControllerReference(&memcached, tempDeploy, r.Scheme); err != nil {
		return ctrl.Result{}, err
	}

	// Add owner references to apply configuration
	for _, ref := range tempDeploy.OwnerReferences {
		deployment.WithOwnerReferences(metav1apply.OwnerReference().
			WithAPIVersion(ref.APIVersion).
			WithKind(ref.Kind).
			WithName(ref.Name).
			WithUID(ref.UID).
			WithController(true).
			WithBlockOwnerDeletion(true))
	}

	/*
		Apply the Deployment using Server-Side Apply.
		- client.Apply() uses the new Server-Side Apply API
		- ForceOwnership resolves conflicts by taking ownership of fields
		- FieldOwner("memcached-controller") identifies this controller
	*/
	if err := r.Apply(ctx, deployment, client.ForceOwnership,
		client.FieldOwner("memcached-controller")); err != nil {
		logger.Error(err, "Failed to apply Deployment")

		// Update status to Degraded
		meta.SetStatusCondition(&memcached.Status.Conditions, metav1.Condition{
			Type:    typeDegradedMemcached,
			Status:  metav1.ConditionTrue,
			Reason:  "DeploymentFailed",
			Message: fmt.Sprintf("Failed to apply Deployment: %v", err),
		})

		if err := r.Status().Update(ctx, &memcached); err != nil {
			logger.Error(err, "Failed to update Memcached status")
		}

		return ctrl.Result{}, err
	}

	/*
		Update status to Available.
		Note: We use traditional Update() for status, but you could also use SSA here.
	*/
	meta.SetStatusCondition(&memcached.Status.Conditions, metav1.Condition{
		Type:    typeAvailableMemcached,
		Status:  metav1.ConditionTrue,
		Reason:  "Reconciling",
		Message: fmt.Sprintf("Deployment for Memcached (%s) with %d replicas created successfully", memcached.Name, size),
	})

	if err := r.Status().Update(ctx, &memcached); err != nil {
		logger.Error(err, "Failed to update Memcached status")
		return ctrl.Result{}, err
	}

	logger.Info("Successfully reconciled Memcached", "name", memcached.Name)
	return ctrl.Result{RequeueAfter: time.Minute}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *MemcachedReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&cachev1alpha1.Memcached{}).
		Owns(&appsv1.Deployment{}).
		Named("memcached").
		Complete(r)
}

// labelsForMemcached returns the labels for selecting the resources
// belonging to the given memcached CR name.
func labelsForMemcached(name string) map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":       "memcached",
		"app.kubernetes.io/instance":   name,
		"app.kubernetes.io/managed-by": "memcached-controller",
	}
}
