/*

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
// +kubebuilder:docs-gen:collapse=Apache License

/*
Along with the standard imports, we need additional controller-runtime and apimachinery libraries.
The extra imports are necessary for managing the objects that are "Owned" by the controller.
*/
package owned_resource

import (
	"context"

	"github.com/go-logr/logr"
	kapps "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	appsv1 "tutorial.kubebuilder.io/project/api/v1"
)

/*
 */

// SimpleDeploymentReconciler reconciles a SimpleDeployment object
type SimpleDeploymentReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:docs-gen:collapse=Reconciler Declaration

/*
In addition to the `SimpleDeployment` permissions, we will also need permissions to manage `Deployments`.
In order to fully manage the workflow of deployments, our app will need to be able to use all verbs on a deployment as well as "get" it's status.
*/

//+kubebuilder:rbac:groups=apps.tutorial.kubebuilder.io,resources=simpledeployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps.tutorial.kubebuilder.io,resources=simpledeployments/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=apps.tutorial.kubebuilder.io,resources=simpledeployments/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps,resources=deployments/status,verbs=get

/*
`Reconcile` will be in charge of reconciling the state of `SimpleDeployments`.

In this basic example, `SimpleDeployments` are used to create and manage simple `Deployments` that can be configured through the `SimpleDeployment` Spec.
*/
// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *SimpleDeploymentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	/* */
	log := r.Log.WithValues("simpleDeployment", req.NamespacedName)

	var simpleDeployment appsv1.SimpleDeployment
	if err := r.Get(ctx, req.NamespacedName, &simpleDeployment); err != nil {
		log.Error(err, "unable to fetch SimpleDeployment")
		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	// +kubebuilder:docs-gen:collapse=Begin the Reconcile

	/*
		Build the deployment that we want to see exist within the cluster
	*/

	deployment := &kapps.Deployment{}

	// Set the information you care about
	deployment.Spec.Replicas = simpleDeployment.Spec.Replicas

	/*
		Set the controller reference, specifying that this `Deployment` is controlled by the `SimpleDeployment` being reconciled.

		This will allow for the `SimpleDeployment` to be reconciled when changes to the `Deployment` are noticed.
	*/
	if err := controllerutil.SetControllerReference(simpleDeployment, deployment, r.scheme); err != nil {
		return ctrl.Result{}, err
	}

	/*
		Manage your `Deployment`.

		- Create it if it doesn't exist.
		- Update it if it is configured incorrectly.
	*/
	foundDeployment := &kapps.Deployment{}
	err := r.Get(ctx, types.NamespacedName{Name: deployment.Name, Namespace: deployment.Namespace}, foundDeployment)
	if err != nil && errors.IsNotFound(err) {
		log.V(1).Info("Creating Deployment", "deployment", deployment.Name)
		err = r.Create(ctx, deployment)
	} else if err == nil {
		if foundDeployment.Spec.Replicas != deployment.Spec.Replicas {
			foundDeployment.Spec.Replicas = deployment.Spec.Replicas
			log.V(1).Info("Updating Deployment", "deployment", deployment.Name)
			err = r.Update(ctx, foundDeployment)
		}
	}

	return ctrl.Result{}, err
}

/*
Finally, we add this reconciler to the manager, so that it gets started
when the manager is started.

Since we create dependency `Deployments` during the reconcile, we can specify that the controller `Owns` `Deployments`.
This will tell the manager that if a `Deployment`, or its status, is updated, then the `SimpleDeployment` in its ownerRef field should be reconciled.
*/

// SetupWithManager sets up the controller with the Manager.
func (r *SimpleDeploymentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appsv1.SimpleDeployment{}).
		Owns(&kapps.Deployment{}).
		Complete(r)
}
