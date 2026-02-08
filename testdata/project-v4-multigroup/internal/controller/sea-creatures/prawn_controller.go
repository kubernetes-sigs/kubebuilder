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

package seacreatures

import (
	"context"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	seacreaturesv1 "sigs.k8s.io/kubebuilder/testdata/project-v4-multigroup/api/sea-creatures/v1"
	// TODO(user): Uncomment the following imports after running 'make generate'
	// metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	// seacreaturesv1apply "sigs.k8s.io/kubebuilder/testdata/project-v4-multigroup/api/sea-creatures/v1/applyconfiguration"
)

// PrawnReconciler reconciles a Prawn object using Server-Side Apply
type PrawnReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=sea-creatures.testproject.org,resources=prawns,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=sea-creatures.testproject.org,resources=prawns/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=sea-creatures.testproject.org,resources=prawns/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// This controller uses Server-Side Apply to manage resources. Server-Side Apply provides:
// - Declarative field ownership tracking
// - Conflict detection when multiple controllers manage the same resource
// - Safer field management when resources are shared with users
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.23.1/pkg/reconcile
func (r *PrawnReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// Fetch the Prawn instance
	// This is optional - you might build the apply configuration without fetching first
	var prawn seacreaturesv1.Prawn
	if err := r.Get(ctx, req.NamespacedName, &prawn); err != nil {
		if errors.IsNotFound(err) {
			// Resource not found - might have been deleted
			log.Info("Prawn resource not found, ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get Prawn")
		return ctrl.Result{}, err
	}

	// TODO(user): Implement Server-Side Apply logic
	// 1. Run 'make generate' to create apply configuration types
	// 2. Uncomment the import above for seacreaturesv1apply
	// 3. Uncomment and customize the code below
	//
	// Build the desired state using apply configuration
	// Only specify the fields you want this controller to manage
	// User customizations (labels, annotations, other fields) will be preserved
	//
	// prawnApply := seacreaturesv1apply.Prawn(req.Name, req.Namespace)
	// // Add the fields you want to manage, for example:
	// // prawnApply = prawnApply.WithSpec(
	// //     seacreaturesv1apply.PrawnSpec().
	// //         WithYourField("value"))
	//
	// // Apply the desired state using Server-Side Apply
	// // The FieldOwner identifies this controller in the managed fields
	// if err := r.Apply(ctx, prawnApply, client.ForceOwnership, client.FieldOwner("prawn-controller")); err != nil {
	//     log.Error(err, "Failed to apply Prawn")
	//     return ctrl.Result{}, err
	// }
	//
	// log.Info("Successfully applied Prawn")

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *PrawnReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&seacreaturesv1.Prawn{}).
		Named("sea-creatures-prawn").
		Complete(r)
}
