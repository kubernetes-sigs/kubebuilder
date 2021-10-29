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

package controllers

import (
	ctrl "sigs.k8s.io/controller-runtime"
)

// +kubebuilder:webhook:failurePolicy="ignore",groups="",resources=pods,verbs=create;update,versions=v1,name=example.m.pod,path=/mutate-pod,mutating=true,sideEffects=None,admissionReviewVersions=v1

// FooReconciler reconciles a Foo object
type FooReconciler struct{}

// +kubebuilder:rbac:groups=example.my.domain,resources=foos,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=example.my.domain,resources=foos/status,verbs=get;update;patch

func (r *FooReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	return ctrl.Result{}, nil
}
