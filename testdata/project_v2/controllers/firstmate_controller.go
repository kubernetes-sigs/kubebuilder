/*
Copyright 2019 The Kubernetes authors.

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
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	crewv1 "sigs.k8s.io/kubebuilder/testdata/project_v2/api/v1"
)

// FirstMateReconciler reconciles a FirstMate object
type FirstMateReconciler struct {
	client.Client
	Log logr.Logger
}

// +kubebuilder:rbac:groups=crew.testproject.org,resources=firstmates,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=crew.testproject.org,resources=firstmates/status,verbs=get;update;patch

func (r *FirstMateReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()
	_ = r.Log.WithValues("firstmate", req.NamespacedName)

	// your logic here

	return ctrl.Result{}, nil
}

func (r *FirstMateReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&crewv1.FirstMate{}).
		Complete(r)
}
