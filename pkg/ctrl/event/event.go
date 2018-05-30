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

package event

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// Update is an event where a Kubernetes object was created.
type CreateEvent struct {
	// Meta is the ObjectMeta of the Kubernetes Object that was created
	Meta v1.ObjectMeta
}

// UnmarshalObj parses the object in the event and stores the result in the value pointed to by i.
func (CreateEvent) UnmarshalObj(i runtime.Object) error { return nil }

// Update is an event where a Kubernetes object was updated.
type UpdateEvent struct {
	// MetaOld is the ObjectMeta of the Kubernetes Object that was updated (before the update)
	MetaOld v1.ObjectMeta

	// MetaNew is the ObjectMeta of the Kubernetes Object that was updated (after the update)
	MetaNew v1.ObjectMeta
}

// UnmarshalObjOld parses the old object in the event and stores the result in the value pointed to by i.
func (UpdateEvent) UnmarshalObjOld(runtime.Object) error { return nil }

// UnmarshalObjNew parses the new object in the event and stores the result in the value pointed to by i.
func (UpdateEvent) UnmarshalObjNew(runtime.Object) error { return nil }

// Update is an event where a Kubernetes object was deleted.
type DeleteEvent struct {
	// Meta is the ObjectMeta of the Kubernetes Object that was deleted
	Meta v1.ObjectMeta
}

// UnmarshalObj parses the object in the event and stores the result in the value pointed to by i.
func (DeleteEvent) UnmarshalObj(i runtime.Object) error { return nil }

// GenericEvent is an event where the operation type is unknown (e.g. polling or event originating outside the cluster).
type GenericEvent struct {
	// Meta is the ObjectMeta of a Kubernetes Object this event is for
	Meta v1.ObjectMeta
}

// UnmarshalObj parses the object in the event and stores the result in the value pointed to by i.
func (GenericEvent) UnmarshalObj(i runtime.Object) error { return nil }
