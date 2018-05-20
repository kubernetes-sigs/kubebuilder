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

package controller

import (
	"fmt"
	"os"
	"reflect"
	"runtime"
	"strings"
	"sync"

	"github.com/kubernetes-sigs/kubebuilder/pkg/controller/informers"
	"github.com/kubernetes-sigs/kubebuilder/pkg/inject/run"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
)

type controllers []*GenericController

func (c controllers) runAll(options run.RunArguments) {
	for _, controller := range c {
		go controller.run(options)
	}
}

var (
	// DefaultManager is the ControllerManager used by the package functions
	DefaultManager = &defaultManager
	defaultManager = ControllerManager{}
)

// ControllerManager registers shared informers and controllers
type ControllerManager struct {
	sharedInformersByResource informers.InformerRegistry
	controllers               controllers
	once                      sync.Once
}

func (m *ControllerManager) init() {
	m.controllers = controllers{}
	m.sharedInformersByResource = informers.InformerRegistry{}
}

// AddInformerProvider registers a new shared SharedIndexInformer under the object type.
// SharedIndexInformer will be RunInformersAndControllers by calling RunInformersAndControllers on the ControllerManager.
func (m *ControllerManager) AddInformerProvider(object metav1.Object, informerProvider informers.InformerProvider) error {
	m.once.Do(m.init)
	return m.sharedInformersByResource.Insert(object, informerProvider)
}

// AddInformerProvider registers a new shared SharedIndexInformer under the object type.
// SharedIndexInformer will be RunInformersAndControllers by calling RunInformersAndControllers on the ControllerManager.
func AddInformerProvider(object metav1.Object, informerProvider informers.InformerProvider) error {
	return DefaultManager.AddInformerProvider(object, informerProvider)
}

// GetInformer returns the Informer for an object
func (m *ControllerManager) GetInformer(object metav1.Object) cache.SharedInformer {
	m.once.Do(m.init)
	si := m.sharedInformersByResource.Get(object)
	if si == nil {
		warningMissingInformer(object)
	}
	return si
}

// GetInformerProvider returns the InformerProvider for the object type
func (m *ControllerManager) GetInformerProvider(object metav1.Object) informers.InformerProvider {
	m.once.Do(m.init)
	si := m.sharedInformersByResource.GetInformerProvider(object)
	if si == nil {
		warningMissingInformer(object)
	}
	return si
}

// GetInformerProvider returns the InformerProvider for the object type.
// Use this to get Listers for objects.
func GetInformerProvider(object metav1.Object) informers.InformerProvider {
	return DefaultManager.GetInformerProvider(object)
}

// AddController registers a new controller to be run..
func (m *ControllerManager) AddController(controller *GenericController) {
	m.once.Do(m.init)
	m.controllers = append(m.controllers, controller)
}

// GetController returns a registered controller with the name
func (m *ControllerManager) GetController(name string) *GenericController {
	m.once.Do(m.init)
	for _, c := range m.controllers {
		if c.Name == name {
			return c
		}
	}
	return nil
}

// AddController registers a new controller to be run.
func AddController(controller *GenericController) {
	DefaultManager.AddController(controller)
}

// RunInformersAndControllers starts the registered informers and controllers.
// Sets options.Parallelism to 1 if it is lt 1
// Creates a new channel for options.Stop if it is nil
func (m *ControllerManager) RunInformersAndControllers(options run.RunArguments) {
	m.once.Do(m.init)
	if options.ControllerParallelism < 1 {
		options.ControllerParallelism = 1
	}
	if options.Stop == nil {
		options.Stop = make(<-chan struct{})
	}
	m.sharedInformersByResource.RunAll(options.Stop)
	m.controllers.runAll(options)
}

// RunInformersAndControllers runs all of the informers and controllers
func RunInformersAndControllers(options run.RunArguments) {
	DefaultManager.RunInformersAndControllers(options)
}

// String prints the registered shared informers
func (m *ControllerManager) String() string {
	return fmt.Sprintf("ControllerManager SharedInformers: %v", m.sharedInformersByResource)
}

// warningMissingInformer prints a warning message to stderr that an informer was not registered
func warningMissingInformer(obj interface{}) {
	// Get the type of the object
	t := reflect.TypeOf(reflect.Indirect(reflect.ValueOf(obj)).Interface())

	// Parse the GVK from the object
	path := t.PkgPath()
	groupversion := strings.Split(path, "/")
	group := groupversion[len(groupversion)-2]
	version := groupversion[len(groupversion)-1]
	kind := t.Name()

	// Create a helpful error message
	msg := fmt.Sprintf("\nWARNING: %s\nWARNING: Informer for %s.%s.%s not registered!  "+
		"Must register informer with a // +kubebuilder:informers:group=%s,version=%s,kind=%s annotation on the "+
		"Controller struct and then run `kubebuilder generate`.\n",
		provideControllerLine(), group, version, kind, group, version, kind)
	fmt.Fprint(os.Stderr, msg)
}

// provideControllerLine returns a string with the file and line number where "ProvideController" was called
// in the call stack
func provideControllerLine() string {
	ok := true
	var file string
	var pc uintptr
	var line int
	for i := 0; ok; i++ {
		pc, file, line, ok = runtime.Caller(i)
		if !ok {
			break
		}
		f := runtime.FuncForPC(pc)
		if f == nil {
			ok = false
			break
		}
		parts := strings.Split(f.Name(), ".")
		fn := parts[len(parts)-1]
		if fn == "ProvideController" {
			break
		}
	}
	if ok {
		return fmt.Sprintf("%s:%v", file, line)
	}
	return ""
}
