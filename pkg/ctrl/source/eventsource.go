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

package source

import (
	"github.com/kubernetes-sigs/kubebuilder/pkg/ctrl/event"
	"github.com/kubernetes-sigs/kubebuilder/pkg/ctrl/eventhandler"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/workqueue"
)

// Source is a source of events (e.g. Create, Update, Delete operations on Kubernetes Objects, Webhook callbacks, etc)
// which should be processed by event.EventHandlers to enqueue ReconcileRequests.
//
// * Use KindSource for events originating in the cluster (e.g. Pod Create, Pod Update, Deployment Update).
//
// * Use ChannelSource for events originating outside the cluster (e.g. GitHub Webhook callback, Polling external urls).
type Source interface {
	// Start is an internal function.  It is used by Controllers to start the event Source watching for events and
	// blocks.  Returns an error if there is an error starting the Source.
	Start(Config) error

	// SetEventHandler is an internal function.  It is used by Controllers to set the EventHandler used to handle
	// events from this Source.
	SetEventHandler(eventhandler.EventHandler)

	// SetEventQueue is an internal function.  It used by Controllers to set the EventHandler used to handle
	// events from this Source.
	SetEventQueue(workqueue.RateLimitingInterface)
}

// Config provides shared structures required for starting a Source.
type Config struct{}

var _ Source = ChannelSource(make(chan event.GenericEvent))

// ChannelSource is used to provide a source of events originating outside the cluster
// (e.g. GitHub Webhook callback).  ChannelSource requires the user to wire the external
// source (e.g. http handler) to write GenericEvents to the underlying channel.
type ChannelSource chan event.GenericEvent

// SetEventHandler implements Source and should only be called by the Controller.
func (g ChannelSource) SetEventHandler(handler eventhandler.EventHandler) {}

// SetEventQueue implements Source and should only be called by the Controller.
func (g ChannelSource) SetEventQueue(queue workqueue.RateLimitingInterface) {}

// Start implements Source and should only be called by the Controller.
func (g ChannelSource) Start(Config) error { return nil }

var _ Source = KindSource{}

// KindSource is used to provide a source of events originating inside the cluster from Watches (e.g. Pod Create)
type KindSource v1.GroupVersionKind

// SetEventHandler implements Source and should only be called by the Controller.
func (g KindSource) SetEventHandler(handler eventhandler.EventHandler) {}

// SetEventQueue implements Source and should only be called by the Controller.
func (g KindSource) SetEventQueue(queue workqueue.RateLimitingInterface) {}

// Start implements Source and should only be called by the Controller.
func (g KindSource) Start(Config) error { return nil }
