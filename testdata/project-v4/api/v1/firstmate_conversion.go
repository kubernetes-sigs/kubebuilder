/*
Copyright 2023 The Kubernetes authors.

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
package v1

// Hub marks that a given type is the hub type for conversion. -- only the no-op method 'Hub()' is required.
// See https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/conversion#Hub
// or https://book.kubebuilder.io/multiversion-tutorial/conversion.html.
func (FirstMate) Hub() {}
