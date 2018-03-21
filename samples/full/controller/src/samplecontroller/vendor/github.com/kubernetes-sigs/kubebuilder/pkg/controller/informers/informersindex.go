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

package informers

import (
	"reflect"

	"fmt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
)

type InformerProvider interface {
	Informer() cache.SharedIndexInformer
}

type InformerGetter interface {
	GetInformer(object metav1.Object) cache.SharedInformer
}

// InformerRegistry contains a map of
type InformerRegistry map[string]InformerProvider

// Insert adds an SharedInformer to the Map
func (im InformerRegistry) Insert(object metav1.Object, informerprovider InformerProvider) error {
	if _, found := im[reflect.TypeOf(object).String()]; found {
		return fmt.Errorf("Cannot Insert informer for %T, already exists", object)
	}
	im[reflect.TypeOf(object).String()] = informerprovider
	return nil
}

// Get gets an SharedInformer from the Map
func (im InformerRegistry) Get(object metav1.Object) cache.SharedInformer {
	if v, found := im[reflect.TypeOf(object).String()]; found {
		return v.Informer()
	}
	return nil
}

func (im InformerRegistry) GetInformerProvider(object metav1.Object) InformerProvider {
	if v, found := im[reflect.TypeOf(object).String()]; found {
		return v
	}
	return nil
}

// RunAll runs all of the shared informers
func (im InformerRegistry) RunAll(stop <-chan struct{}) {
	for _, i := range im {
		go i.Informer().Run(stop)
	}
}
