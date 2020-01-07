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

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	batchv1 "tutorial.kubebuilder.io/project/api/v1"
)

/*
Next, kubebuilder has scaffolded a basic reconciler struct for us.
Pretty much every reconciler needs to log, and needs to be able to fetch
objects, so these are added out of the box.
*/

// CronJobReconciler reconciles a CronJob object
type CronJobReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

/*
Most controllers eventually end up running on the cluster, so they need RBAC
permissions, which we specify using controller-tools [RBAC
markers](/reference/markers/rbac.md).  These are the bare minimum permissions
needed to run.  As we add more functionality, we'll need to revisit these.
*/

// +kubebuilder:rbac:groups=batch.tutorial.kubebuilder.io,resources=cronjobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=batch.tutorial.kubebuilder.io,resources=cronjobs/status,verbs=get;update;patch

/*
`Reconcile` actually performs the reconciling for a single named object.
Our [Request](https://godoc.org/sigs.k8s.io/controller-runtime/pkg/reconcile#Request) just has a name, but we can use the client to fetch
that object from the cache.

We return an empty result and no error, which indicates to controller-runtime that
we've successfully reconciled this object and don't need to try again until there's
some changes.

Most controllers need a logging handle and a context, so we set them up here.

The [context](https://golang.org/pkg/context/) is used to allow cancelation of
requests, and potentially things like tracing.  It's the first argument to all
client methods.  The `Background` context is just a basic context without any
extra data or timing restrictions.

The logging handle lets us log.  controller-runtime uses structured logging through a
library called [logr](https://github.com/go-logr/logr).  As we'll see shortly,
logging works by attaching key-value pairs to a static message.  We can pre-assign
some pairs at the top of our reconcile method to have those attached to all log
lines in this reconciler.
*/
func (r *CronJobReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()
	_ = r.Log.WithValues("cronjob", req.NamespacedName)

	// your logic here

	return ctrl.Result{}, nil
}

/*
Finally, we add this reconciler to the manager, so that it gets started
when the manager is started.

For now, we just note that this reconciler operates on `CronJob`s.  Later,
we'll use this to mark that we care about related objects as well.

*/

func (r *CronJobReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&batchv1.CronJob{}).
		Complete(r)
}
