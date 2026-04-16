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

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	seacreaturesv1 "sigs.k8s.io/kubebuilder/testdata/project-v4-multigroup/api/sea-creatures/v1"
)

// PrawnReconciler reconciles a Prawn object using Server-Side Apply
type PrawnReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=sea-creatures.testproject.org,resources=prawns,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=sea-creatures.testproject.org,resources=prawns/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=sea-creatures.testproject.org,resources=prawns/finalizers,verbs=update

// Reconcile uses Server-Side Apply to manage resources for the Prawn.
//
// Server-Side Apply (SSA) provides declarative field ownership, allowing multiple actors
// (users, controllers) to safely manage different fields of the same resource. This controller
// only takes ownership of fields it explicitly declares, preserving user customizations to other fields.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.23.3/pkg/reconcile
func (r *PrawnReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	// Fetch the Prawn instance
	var prawn seacreaturesv1.Prawn
	if err := r.Get(ctx, req.NamespacedName, &prawn); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("Prawn resource not found, ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get Prawn")
		return ctrl.Result{}, err
	}

	// TODO(user): your logic here

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *PrawnReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&seacreaturesv1.Prawn{}).
		Named("sea-creatures-prawn").
		Complete(r)
}
