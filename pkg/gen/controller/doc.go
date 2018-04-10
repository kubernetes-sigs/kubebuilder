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

/*
The controller package describes comment directives that may be applied to controllers
*/
package controller

// Controller annotates a type as being a controller for a specific resource
const Controller = "// +kubebuilder:controller:group=,version=,kind=,resource="

// RBAC annotates a controller struct as needing an RBAC rule to run
const RBAC = "// +kubebuilder:rbac:groups=<group1;group2>,resources=<resource1;resource2>,verbs=<verb1;verb2>"

// Informers indicates that an informer must be started for this controller
const Informers = "// +kubebuilder:informers:group=core,version=v1,kind=Pod"