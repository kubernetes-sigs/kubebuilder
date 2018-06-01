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

// Code generated by informer-gen. DO NOT EDIT.

package v1alpha1

import (
	time "time"

	foobar_v1alpha1 "github.com/kubernetes-sigs/kubebuilder/test/webhooks/apis/foobar/v1alpha1"
	versioned "github.com/kubernetes-sigs/kubebuilder/test/webhooks/client/clientset/versioned"
	internalinterfaces "github.com/kubernetes-sigs/kubebuilder/test/webhooks/client/informers/externalversions/internalinterfaces"
	v1alpha1 "github.com/kubernetes-sigs/kubebuilder/test/webhooks/client/listers/foobar/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	cache "k8s.io/client-go/tools/cache"
)

// FooBarInformer provides access to a shared informer and lister for
// FooBars.
type FooBarInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() v1alpha1.FooBarLister
}

type fooBarInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
	namespace        string
}

// NewFooBarInformer constructs a new informer for FooBar type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFooBarInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredFooBarInformer(client, namespace, resyncPeriod, indexers, nil)
}

// NewFilteredFooBarInformer constructs a new informer for FooBar type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredFooBarInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.FoobarV1alpha1().FooBars(namespace).List(options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.FoobarV1alpha1().FooBars(namespace).Watch(options)
			},
		},
		&foobar_v1alpha1.FooBar{},
		resyncPeriod,
		indexers,
	)
}

func (f *fooBarInformer) defaultInformer(client versioned.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredFooBarInformer(client, f.namespace, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *fooBarInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&foobar_v1alpha1.FooBar{}, f.defaultInformer)
}

func (f *fooBarInformer) Lister() v1alpha1.FooBarLister {
	return v1alpha1.NewFooBarLister(f.Informer().GetIndexer())
}
