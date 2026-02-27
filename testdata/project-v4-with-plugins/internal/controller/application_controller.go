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

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	examplecomv1 "sigs.k8s.io/kubebuilder/testdata/project-v4-with-plugins/api/v1"
	// TODO(user): Uncomment the following imports after running 'make generate'
	// metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	// examplecomv1apply "sigs.k8s.io/kubebuilder/testdata/project-v4-with-plugins/api/v1/applyconfiguration"
)

// ApplicationReconciler reconciles a Application object using Server-Side Apply
type ApplicationReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=example.com.testproject.org,namespace=project-v4-with-plugins-system,resources=applications,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=example.com.testproject.org,namespace=project-v4-with-plugins-system,resources=applications/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=example.com.testproject.org,namespace=project-v4-with-plugins-system,resources=applications/finalizers,verbs=update

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
func (r *ApplicationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// Fetch the Application instance
	// This is optional - you might build the apply configuration without fetching first
	var application examplecomv1.Application
	if err := r.Get(ctx, req.NamespacedName, &application); err != nil {
		if errors.IsNotFound(err) {
			// Resource not found - might have been deleted
			log.Info("Application resource not found, ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get Application")
		return ctrl.Result{}, err
	}

	// TODO(user): Implement Server-Side Apply logic
	// 1. Run 'make generate' to create apply configuration types
	// 2. Uncomment the import above for examplecomv1apply
	// 3. Uncomment and customize the code below
	//
	// Build the desired state using apply configuration
	// Only specify the fields you want this controller to manage
	// User customizations (labels, annotations, other fields) will be preserved
	//
	// applicationApply := examplecomv1apply.Application(req.Name, req.Namespace)
	// // Add the fields you want to manage, for example:
	// // applicationApply = applicationApply.WithSpec(
	// //     examplecomv1apply.ApplicationSpec().
	// //         WithYourField("value"))
	//
	// // Apply the desired state using Server-Side Apply
	// // The FieldOwner identifies this controller in the managed fields
	// if err := r.Apply(ctx, applicationApply, client.ForceOwnership, client.FieldOwner("application-controller")); err != nil {
	//     log.Error(err, "Failed to apply Application")
	//     return ctrl.Result{}, err
	// }
	//
	// log.Info("Successfully applied Application")

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ApplicationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&examplecomv1.Application{}).
		Named("application").
		Complete(r)
}
