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
All additional libraries, necessary for Watching, have the comment `Required For Watching` appended.
*/
package external_indexed_field

import (
	"context"

	"github.com/go-logr/logr"
	kapps "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields" // Required for Watching
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types" // Required for Watching
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder" // Required for Watching
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler" // Required for Watching
	"sigs.k8s.io/controller-runtime/pkg/predicate" // Required for Watching
	"sigs.k8s.io/controller-runtime/pkg/reconcile" // Required for Watching
	"sigs.k8s.io/controller-runtime/pkg/source" // Required for Watching

	appsv1 "tutorial.kubebuilder.io/project/api/v1"
)

/*
Determine the path of the field in the ConfigDeployment CRD that we wish to use as the "object reference".
This will be used in both the indexing and watching.
*/
const (
	configMapField = ".spec.configMap"
)

/*
*/

// ConfigDeploymentReconciler reconciles a ConfigDeployment object
type ConfigDeploymentReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}
// +kubebuilder:docs-gen:collapse=Reconciler Declaration

/*
There are two additional resources that the controller needs to have access to, other than ConfigDeployments.
- It needs to be able to fully manage Deployments, as well as check their status.
- It also needs to be able to get, list and watch ConfigMaps.
All 3 of these are important, and you will see usages of each below.
*/

//+kubebuilder:rbac:groups=apps.tutorial.kubebuilder.io,resources=configdeployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps.tutorial.kubebuilder.io,resources=configdeployments/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=apps.tutorial.kubebuilder.io,resources=configdeployments/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps,resources=deployments/status,verbs=get
//+kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch

/*
`Reconcile` will be in charge of reconciling the state of ConfigDeployments.
ConfigDeployments are used to manage Deployments whose pods are updated whenever the configMap that they use is updated.

For that reason we need to add an annotation to the PodTemplate within the Deployment we create.
This annotation will keep track of the latest version of the data within the referenced ConfigMap.
Therefore when the version of the configMap is changed, the PodTemplate in the Deployment will change.
This will cause a rolling upgrade of all Pods managed by the Deployment.

Skip down to the `SetupWithManager` function to see how we ensure that `Reconcile` is called when the referenced `ConfigMaps` are updated.
*/
// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *ConfigDeploymentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	/* */
	log := r.Log.WithValues("configDeployment", req.NamespacedName)

	var configDeployment appsv1.ConfigDeployment
	if err := r.Get(ctx, req.NamespacedName, &configDeployment); err != nil {
		log.Error(err, "unable to fetch ConfigDeployment")
		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	// +kubebuilder:docs-gen:collapse=Begin the Reconcile

	// your logic here

	var configMapVersion string
	if configDeployment.Spec.ConfigMap != "" {
		configMapName := configDeployment.Spec.ConfigMap
		foundConfigMap := &corev1.ConfigMap{}
		err := r.Get(ctx, types.NamespacedName{Name: configMapName, Namespace: configDeployment.Namespace}, foundConfigMap)
		if err != nil {
			// If a configMap name is provided, then it must exist
			// You will likely want to create an Event for the user to understand why their reconcile is failing.
			return ctrl.Result{}, err
		}

		// Hash the data in some way, or just use the version of the Object
		configMapVersion = foundConfigMap.ResourceVersion
	}

	// Logic here to add the configMapVersion as an annotation on your Deployment Pods.

	return ctrl.Result{}, nil
}

/*
Finally, we add this reconciler to the manager, so that it gets started
when the manager is started.

Since we create dependency Deployments during the reconcile, we can specify that the controller `Owns` Deployments.

However the ConfigMaps that we want to watch are not owned by the ConfigDeployment object.
Therefore we must specify a custom way of watching those objects.
This watch logic is complex, so we have split it into a separate method.
*/

// SetupWithManager sets up the controller with the Manager.
func (r *ConfigDeploymentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	/*
		The `configMap` field must be indexed by the manager, so that we will be able to lookup `ConfigDeployments` by a referenced `ConfigMap` name.
		This will allow for quickly answer the question:
		- If ConfigMap _x_ is updated, which ConfigDeployments are affected?
	*/

	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &appsv1.ConfigDeployment{}, configMapField, func(rawObj client.Object) []string {
		// Extract the ConfigMap name from the ConfigDeployment Spec, if one is provided
		configDeployment := rawObj.(*appsv1.ConfigDeployment)
		if configDeployment.Spec.ConfigMap == "" {
			return nil
		}
		return []string{configDeployment.Spec.ConfigMap}
	}); err != nil {
		return err
	}

	/*
	As explained in the CronJob tutorial, the controller will first register the Type that it manages, as well as the types of subresources that it controls.
	Since we also want to watch ConfigMaps that are not controlled or managed by the controller, we will need to use the `Watches()` functionality as well.

	The `Watches()` function is a controller-runtime API that takes:
	- A Kind (i.e. `ConfigMap`)
	- A mapping function that converts a `ConfigMap` object to a list of reconcile requests for `ConfigDeployments`.
	We have separated this out into a separate function.
	- A list of options for watching the `ConfigMaps`
	  - In our case, we only want the watch to be triggered when the ResourceVersion of the ConfigMap is changed.
	 */

	return ctrl.NewControllerManagedBy(mgr).
		For(&appsv1.ConfigDeployment{}).
		Owns(&kapps.Deployment{}).
		Watches(
			&source.Kind{Type: &corev1.ConfigMap{}},
			handler.EnqueueRequestsFromMapFunc(r.findObjectsForConfigMap),
			builder.WithPredicates(predicate.ResourceVersionChangedPredicate{}),
		).
		Complete(r)
}

/*
	Because we have already created an index on the `configMap` reference field, this mapping function is quite straight forward.
	We first need to list out all `ConfigDeployments` that use `ConfigMap` given in the mapping function.
	This is done by merely submitting a List request using our indexed field as the field selector.

	When the list of `ConfigDeployments` that reference the `ConfigMap` is found,
	we just need to loop through the list and create a reconcile request for each one.
	If an error occurs fetching the list, or no `ConfigDeployments` are found, then no reconcile requests will be returned.
*/
func (r *ConfigDeploymentReconciler) findObjectsForConfigMap(ctx context.Context, configMap client.Object) []reconcile.Request {
	attachedConfigDeployments := &appsv1.ConfigDeploymentList{}
	listOps := &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(configMapField, configMap.GetName()),
		Namespace:     configMap.GetNamespace(),
	}
	err := r.List(ctx, attachedConfigDeployments, listOps)
	if err != nil {
		return []reconcile.Request{}
	}

	requests := make([]reconcile.Request, len(attachedConfigDeployments.Items))
	for i, item := range attachedConfigDeployments.Items {
		requests[i] = reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      item.GetName(),
				Namespace: item.GetNamespace(),
			},
		}
	}
	return requests
}
