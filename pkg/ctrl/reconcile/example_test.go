/*
Copyright 2018 The Kubernetes Authors.

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

package reconcile_test

import (
	"fmt"

	"github.com/kubernetes-sigs/kubebuilder/pkg/ctrl/reconcile"
)

// This example implements a simple no-op Reconcile function that prints the object to be Reconciled.
func ExampleReconcileFunc() {
	r := reconcile.ReconcileFunc(func(o reconcile.ReconcileRequest) (reconcile.ReconcileResult, error) {
		// Create your business logic to create, update, delete objects here.
		fmt.Printf("Name: %s, Namespace: %s", o.Name, o.Namespace)
		return reconcile.ReconcileResult{}, nil
	})

	r.Reconcile(reconcile.ReconcileRequest{Namespace: "default", Name: "test"})

	// Output: Name: test, Namespace: default
}

// This example declares a simple type that implements Reconcile.
func ExampleReconcile() {
	type MyReconcileImplementation struct {
		reconcile.ReconcileFunc
	}
}
