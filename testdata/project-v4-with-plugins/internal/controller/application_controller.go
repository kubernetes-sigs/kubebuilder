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

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	examplecomv1 "sigs.k8s.io/kubebuilder/testdata/project-v4-with-plugins/api/v1"
)

// ApplicationReconciler reconciles a Application object using Server-Side Apply
type ApplicationReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=example.com.testproject.org,namespace=project-v4-with-plugins-system,resources=applications,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=example.com.testproject.org,namespace=project-v4-with-plugins-system,resources=applications/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=example.com.testproject.org,namespace=project-v4-with-plugins-system,resources=applications/finalizers,verbs=update

// Reconcile uses Server-Side Apply to manage resources for the Application.
//
// Server-Side Apply (SSA) provides declarative field ownership, allowing multiple actors
// (users, controllers) to safely manage different fields of the same resource. This controller
// only takes ownership of fields it explicitly declares, preserving user customizations to other fields.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.23.3/pkg/reconcile
func (r *ApplicationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	// Fetch the Application instance
	var application examplecomv1.Application
	if err := r.Get(ctx, req.NamespacedName, &application); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("Application resource not found, ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get Application")
		return ctrl.Result{}, err
	}

	// TODO(user): your logic here

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ApplicationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&examplecomv1.Application{}).
		Named("application").
		Complete(r)
}
