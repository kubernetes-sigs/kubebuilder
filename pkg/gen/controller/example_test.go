/*
Copyright 2017 The Kubernetes Authors.

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

package controller_test

func Example() {
	// +kubebuilder:controller:group=foo,version=v1beta1,kind=Bar,resource=bars
	// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
	// +kubebuilder:informers:group=apps,version=v1,kind=Deployment
	// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;watch;list
	// +kubebuilder:informers:group=core,version=v1,kind=Pod
	type FooController struct{}
}
