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

package fake

import (
	"context"
	"encoding/json"
	"os"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/testing"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var (
	log = logf.KBLog.WithName("fake-client")
)

type fakeClient struct {
	tracker testing.ObjectTracker
}

var _ client.Client = &fakeClient{}

// NewFakeClient creates a new fake client for testing.
// You can choose to initialize it with a slice of runtime.Object.
func NewFakeClient(initObjs ...runtime.Object) client.Client {
	tracker := testing.NewObjectTracker(scheme.Scheme, scheme.Codecs.UniversalDecoder())
	for _, obj := range initObjs {
		err := tracker.Add(obj)
		if err != nil {
			log.Error(err, "failed to add object", "object", obj)
			os.Exit(1)
			return nil
		}
	}
	return &fakeClient{
		tracker: tracker,
	}
}

func (c *fakeClient) Get(ctx context.Context, key client.ObjectKey, obj runtime.Object) error {
	gvr, err := getGVRFromObject(obj)
	if err != nil {
		return err
	}
	o, err := c.tracker.Get(gvr, key.Namespace, key.Name)
	if err != nil {
		return err
	}
	j, err := json.Marshal(o)
	if err != nil {
		return err
	}
	decoder := scheme.Codecs.UniversalDecoder()
	_, _, err = decoder.Decode(j, nil, obj)
	return err
}

func (c *fakeClient) List(ctx context.Context, opts *client.ListOptions, list runtime.Object) error {
	gvk := opts.Raw.TypeMeta.GroupVersionKind()
	gvr, _ := meta.UnsafeGuessKindToResource(gvk)
	o, err := c.tracker.List(gvr, gvk, opts.Namespace)
	if err != nil {
		return err
	}
	j, err := json.Marshal(o)
	if err != nil {
		return err
	}
	decoder := scheme.Codecs.UniversalDecoder()
	_, _, err = decoder.Decode(j, nil, list)
	return err
}

func (c *fakeClient) Create(ctx context.Context, obj runtime.Object) error {
	gvr, err := getGVRFromObject(obj)
	if err != nil {
		return err
	}
	accessor, err := meta.Accessor(obj)
	if err != nil {
		return err
	}
	return c.tracker.Create(gvr, obj, accessor.GetNamespace())
}

func (c *fakeClient) Delete(ctx context.Context, obj runtime.Object, opts ...client.DeleteOptionFunc) error {
	gvr, err := getGVRFromObject(obj)
	if err != nil {
		return err
	}
	accessor, err := meta.Accessor(obj)
	if err != nil {
		return err
	}
	//TODO: implement propagation
	return c.tracker.Delete(gvr, accessor.GetNamespace(), accessor.GetName())
}

func (c *fakeClient) Update(ctx context.Context, obj runtime.Object) error {
	gvr, err := getGVRFromObject(obj)
	if err != nil {
		return err
	}
	accessor, err := meta.Accessor(obj)
	if err != nil {
		return err
	}
	return c.tracker.Update(gvr, obj, accessor.GetNamespace())
}

func (c *fakeClient) Status() client.StatusWriter {
	return &fakeStatusWriter{client: c}
}

func getGVRFromObject(obj runtime.Object) (schema.GroupVersionResource, error) {
	gvk, err := apiutil.GVKForObject(obj, scheme.Scheme)
	if err != nil {
		return schema.GroupVersionResource{}, err
	}
	gvr, _ := meta.UnsafeGuessKindToResource(gvk)
	return gvr, nil
}

type fakeStatusWriter struct {
	client *fakeClient
}

func (sw *fakeStatusWriter) Update(ctx context.Context, obj runtime.Object) error {
	// TODO(droot): This results in full update of the obj (spec + status). Need
	// a way to update status field only.
	return sw.client.Update(ctx, obj)
}
