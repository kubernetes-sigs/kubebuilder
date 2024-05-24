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
First, we start out with some standard imports.
As before, we need the core controller-runtime library, as well as
the client package, and the package for our API types.
*/
package controllers

import (
	"context"

	"k8s.io/kubernetes/pkg/apis/batch"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	batchv1 "tutorial.kubebuilder.io/project/api/v1"
)

// +kubebuilder:docs-gen:collapse=Imports

/*
By default, kubebuilder will include the RBAC rules necessary to update finalizers for CronJobs.
*/

// +kubebuilder:rbac:groups=batch.tutorial.kubebuilder.io,resources=cronjobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=batch.tutorial.kubebuilder.io,resources=cronjobs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=batch.tutorial.kubebuilder.io,resources=cronjobs/finalizers,verbs=update

/*
The code snippet below shows skeleton code for implementing a finalizer.
*/

func (r *CronJobReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("cronjob", req.NamespacedName)

	cronJob := &batchv1.CronJob{}
	if err := r.Get(ctx, req.NamespacedName, cronJob); err != nil {
		log.Error(err, "unable to fetch CronJob")
		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// name of our custom finalizer
	myFinalizerName := "batch.tutorial.kubebuilder.io/finalizer"

	// examine DeletionTimestamp to determine if object is under deletion
	if cronJob.ObjectMeta.DeletionTimestamp.IsZero() {
		// The object is not being deleted, so if it does not have our finalizer,
		// then lets add the finalizer and update the object. This is equivalent
		// to registering our finalizer.
		if !controllerutil.ContainsFinalizer(cronJob, myFinalizerName) {
			controllerutil.AddFinalizer(cronJob, myFinalizerName)
			if err := r.Update(ctx, cronJob); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		// The object is being deleted
		if controllerutil.ContainsFinalizer(cronJob, myFinalizerName) {
			// our finalizer is present, so lets handle any external dependency
			if err := r.deleteExternalResources(cronJob); err != nil {
				// if fail to delete the external dependency here, return with error
				// so that it can be retried.
				return ctrl.Result{}, err
			}

			// remove our finalizer from the list and update it.
			controllerutil.RemoveFinalizer(cronJob, myFinalizerName)
			if err := r.Update(ctx, cronJob); err != nil {
				return ctrl.Result{}, err
			}
		}

		// Stop reconciliation as the item is being deleted
		return ctrl.Result{}, nil
	}

	// Your reconcile logic

	return ctrl.Result{}, nil
}

func (r *Reconciler) deleteExternalResources(cronJob *batch.CronJob) error {
	//
	// delete any external resources associated with the cronJob
	//
	// Ensure that delete implementation is idempotent and safe to invoke
	// multiple times for same object.
}
