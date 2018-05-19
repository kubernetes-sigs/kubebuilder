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

package eventhandler

import (
	"github.com/kubernetes-sigs/kubebuilder/pkg/ctrl/event"
	"k8s.io/client-go/util/workqueue"
)

var _ EventHandler = EnqueueHandler{}

// EnqueueHandler enqueues a ReconcileRequest containing the Name and Namespace of the object in the Event.
type EnqueueHandler struct{}

// Create implements EventHandler
func (e EnqueueHandler) Create(q workqueue.RateLimitingInterface, event event.CreateEvent) {}

// Update implements EventHandler
func (e EnqueueHandler) Update(q workqueue.RateLimitingInterface, event event.UpdateEvent) {}

// Delete implements EventHandler
func (e EnqueueHandler) Delete(q workqueue.RateLimitingInterface, event event.DeleteEvent) {}

// Generic implements EventHandler
func (e EnqueueHandler) Generic(workqueue.RateLimitingInterface, event.GenericEvent) {}
