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


package foo

import (
    "log"

    "github.com/kubernetes-sigs/kubebuilder/pkg/controller"
    "github.com/kubernetes-sigs/kubebuilder/pkg/controller/types"

    samplecontrollerv1alpha1client "samplecontroller/pkg/client/clientset/versioned/typed/samplecontroller/v1alpha1"
    samplecontrollerv1alpha1lister "samplecontroller/pkg/client/listers/samplecontroller/v1alpha1"
    samplecontrollerv1alpha1 "samplecontroller/pkg/apis/samplecontroller/v1alpha1"
    samplecontrollerv1alpha1informer "samplecontroller/pkg/client/informers/externalversions/samplecontroller/v1alpha1"
    "samplecontroller/pkg/inject/args"
)

// EDIT THIS FILE
// This files was created by "kubebuilder create resource" for you to edit.
// Controller implementation logic for Foo resources goes here.

func (bc *FooController) Reconcile(k types.ReconcileKey) error {
    // INSERT YOUR CODE HERE
    log.Printf("Implement the Reconcile function on foo.FooController to reconcile %s\n", k.Name)
    return nil
}

// +controller:group=samplecontroller,version=v1alpha1,kind=Foo,resource=foos
type FooController struct {
    // INSERT ADDITIONAL FIELDS HERE
    fooLister samplecontrollerv1alpha1lister.FooLister
    fooclient samplecontrollerv1alpha1client.SamplecontrollerV1alpha1Interface
}

// ProvideController provides a controller that will be run at startup.  Kubebuilder will use codegeneration
// to automatically register this controller in the inject package
func ProvideController(arguments args.InjectArgs) (*controller.GenericController, error) {
    // INSERT INITIALIZATIONS FOR ADDITIONAL FIELDS HERE
    bc := &FooController{
        fooLister: arguments.ControllerManager.GetInformerProvider(&samplecontrollerv1alpha1.Foo{}).(samplecontrollerv1alpha1informer.FooInformer).Lister(),
        fooclient: arguments.Clientset.SamplecontrollerV1alpha1(),
    }

    // Create a new controller that will call FooController.Reconcile on changes to Foos
    gc := &controller.GenericController{
        Name: "FooController",
        Reconcile: bc.Reconcile,
        InformerRegistry: arguments.ControllerManager,
    }
    if err := gc.Watch(&samplecontrollerv1alpha1.Foo{}); err != nil {
        return gc, err
    }

    // INSERT ADDITIONAL WATCHES HERE BY CALLING gc.Watch.*() FUNCTIONS
    // NOTE: Informers for Kubernetes resources *MUST* be registered in the pkg/inject package so that they are started.
    return gc, nil
}
