/*
Copyright 2020 The Kubernetes authors.

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
	"context"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	testprojectorgv1 "sigs.k8s.io/kubebuilder/testdata/project-v3-multigroup/apis/v1"
)

// LakersReconciler reconciles a Lakers object
type LakersReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=testproject.org,resources=lakers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=testproject.org,resources=lakers/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=testproject.org,resources=lakers/finalizers,verbs=update

func (r *LakersReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()
	_ = r.Log.WithValues("lakers", req.NamespacedName)

	// your logic here

	return ctrl.Result{}, nil
}

func (r *LakersReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&testprojectorgv1.Lakers{}).
		Complete(r)
}
