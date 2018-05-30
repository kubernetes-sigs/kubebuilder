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

package source_test

import (
	"github.com/kubernetes-sigs/kubebuilder/pkg/ctrl"
	"github.com/kubernetes-sigs/kubebuilder/pkg/ctrl/event"
	"github.com/kubernetes-sigs/kubebuilder/pkg/ctrl/eventhandler"
	"github.com/kubernetes-sigs/kubebuilder/pkg/ctrl/source"
)

// This example Watches for Pod Events (e.g. Create / Update / Delete) and enqueues a ReconcileRequest
// with the Name and Namespace of the Pod.
func ExampleKindSource() {
	controller := &ctrl.Controller{Name: "pod-controller"}

	controller.Watch(
		source.KindSource{Group: "core", Version: "v1", Kind: "Pod"},
		eventhandler.EnqueueHandler{},
	)
}

// This example reads GenericEvents from a channel and enqueues a ReconcileRequest containing the Name and Namespace
// provided by the event.
func ExampleChannelSource() {
	controller := &ctrl.Controller{Name: "myresource-controller"}
	events := make(chan event.GenericEvent)

	controller.Watch(
		source.ChannelSource(events),
		eventhandler.EnqueueHandler{},
	)
}
