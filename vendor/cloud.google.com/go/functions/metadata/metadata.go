// Copyright 2018 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package metadata provides methods for creating and accessing context.Context objects
// with Google Cloud Functions metadata.
package metadata // import "cloud.google.com/go/functions/metadata"

import (
	"time"

	"golang.org/x/net/context"
)

type contextKey struct{}

// Metadata holds Google Cloud Functions metadata.
type Metadata struct {
	// EventID is a unique ID for the event. For example: "70172329041928".
	EventID string `json:"eventId"`
	// Timestamp is the date/time this event was created.
	Timestamp time.Time `json:"timestamp"`
	// EventType is the type of the event. For example: "google.pubsub.topic.publish".
	EventType string `json:"eventType"`
	// Resource is the resource that triggered the event.
	Resource Resource `json:"resource"`
}

// Resource holds Google Cloud Functions resource metadata.
// Resource values are dependent on the event type they're from.
type Resource struct {
	// Service is the service that triggered the event.
	Service string `json:"service"`
	// Name is the name associated with the event.
	Name string `json:"name"`
	// Type is the type of event.
	Type string `json:"type"`
}

// NewContext returns a new Context carrying m.
func NewContext(ctx context.Context, m Metadata) context.Context {
	return context.WithValue(ctx, contextKey{}, m)
}

// FromContext extracts the Metadata from the Context, if present.
func FromContext(ctx context.Context) (Metadata, bool) {
	if ctx == nil {
		return Metadata{}, false
	}
	m, ok := ctx.Value(contextKey{}).(Metadata)
	return m, ok
}
