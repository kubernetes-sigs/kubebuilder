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

package webhook

import (
	"fmt"

	"github.com/golang/glog"
	"github.com/kubernetes-sigs/kubebuilder/pkg/webhooks/"
	"github.com/kubernetes-sigs/kubebuilder/test/webhooks/apis/foobar/v1alpha1"
	"github.com/mattbaird/jsonpatch"
	"k8s.io/client-go/kubernetes"
)

// NewAdmissionController creates a new instance of the admission webhook controller.
func NewAdmissionController(client kubernetes.Interface, options webhook.ControllerOptions) (*webhook.AdmissionController, error) {
	options.Resources = []string{"foobars"}

	return &webhook.AdmissionController{
		Client:  client,
		Options: options,
		Handlers: map[string]webhook.GenericCRDHandler{
			"FooBar": {
				Factory:   &v1alpha1.FooBar{},
				Defaulter: SetFooBarDefaults,
				Validator: ValidateFooBar,
			},
		},
	}, nil
}

func SetFooBarDefaults(patches *[]jsonpatch.JsonPatchOperation, old webhook.GenericCRD, new webhook.GenericCRD) error {
	_, newFB, err := unmarshal(old, new, "SetFooBarDefaults")
	if err != nil {
		return err
	}

	// Allowe foos to be [0, 10]
	if newFB.Spec.Foos < 0 {
		*patches = append(*patches, jsonpatch.JsonPatchOperation{
			Operation: "add",
			Path:      "/spec/foos",
			Value:     0,
		})
	} else if newFB.Spec.Foos > 10 {
		*patches = append(*patches, jsonpatch.JsonPatchOperation{
			Operation: "add",
			Path:      "/spec/foos",
			Value:     10,
		})
	}

	// Allow bars to be [0, 42]
	if newFB.Spec.Bars < 0 {
		*patches = append(*patches, jsonpatch.JsonPatchOperation{
			Operation: "add",
			Path:      "/spec/bars",
			Value:     0,
		})
	} else if newFB.Spec.Bars > 42 {
		*patches = append(*patches, jsonpatch.JsonPatchOperation{
			Operation: "add",
			Path:      "/spec/bars",
			Value:     42,
		})
	}

	return nil
}

// ValidateFooBar is FooBar resource specific validation and mutation handler
func ValidateFooBar(patches *[]jsonpatch.JsonPatchOperation, old webhook.GenericCRD, new webhook.GenericCRD) error {
	// TODO: need a validate function for FooBar as an example.
	//o, n, err := unmarshal(old, new, "ValidateFooBar")
	//if err != nil {
	//	return err
	//}

	return nil
}

func unmarshal(old webhook.GenericCRD, new webhook.GenericCRD, fnName string) (*v1alpha1.FooBar, *v1alpha1.FooBar, error) {
	var oldFB *v1alpha1.FooBar
	if old != nil {
		var ok bool
		oldFB, ok = old.(*v1alpha1.FooBar)
		if !ok {
			return nil, nil, fmt.Errorf("failed to convert old into FooBar: %+v", old)
		}
	}
	glog.Infof("%s: OLD Revision is\n%+v", fnName, oldFB)

	newFB, ok := new.(*v1alpha1.FooBar)
	if !ok {
		return nil, nil, fmt.Errorf("failed to convert new into FooBar: %+v", new)
	}
	glog.Infof("%s: NEW FooBard is\n%+v", fnName, newFB)

	return oldFB, newFB, nil
}
