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
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/kubernetes-sigs/kubebuilder/pkg/webhooks/"
	"github.com/kubernetes-sigs/kubebuilder/test/webhooks/apis/foobar"
	"github.com/kubernetes-sigs/kubebuilder/test/webhooks/apis/foobar/v1alpha1"
	"github.com/mattbaird/jsonpatch"
	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	"k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fakekubeclientset "k8s.io/client-go/kubernetes/fake"
)

func newDefaultOptions() webhook.ControllerOptions {
	return webhook.ControllerOptions{
		ServiceName:      "test-webhook",
		ServiceNamespace: "test-system",
		APIGroupName:     foobar.GroupName,
		APIVersion:       "v1alpha1",
		Port:             443,
		SecretName:       "test-webhook-certs",
		WebhookName:      "webhook.foobar.test",
	}
}

const (
	testNamespace  = "test-namespace"
	testFooBarName = "test-foobar"
)

func newNonRunningTestAdmissionController(t *testing.T, options webhook.ControllerOptions) (
	kubeClient *fakekubeclientset.Clientset,
	ac *webhook.AdmissionController) {
	// Create fake clients
	kubeClient = fakekubeclientset.NewSimpleClientset()

	ac, err := NewAdmissionController(kubeClient, options)
	if err != nil {
		t.Fatalf("Failed to create new admission controller: %s", err)
	}
	return
}

func TestDeleteAllowed(t *testing.T) {
	_, ac := newNonRunningTestAdmissionController(t, newDefaultOptions())

	req := admissionv1beta1.AdmissionRequest{
		Operation: admissionv1beta1.Delete,
	}

	resp := ac.Admit(&req)
	if !resp.Allowed {
		t.Fatalf("unexpected denial of delete")
	}
}

func TestConnectAllowed(t *testing.T) {
	_, ac := newNonRunningTestAdmissionController(t, newDefaultOptions())

	req := admissionv1beta1.AdmissionRequest{
		Operation: admissionv1beta1.Connect,
	}

	resp := ac.Admit(&req)
	if !resp.Allowed {
		t.Fatalf("unexpected denial of connect")
	}
}

func TestUnknownKindFails(t *testing.T) {
	_, ac := newNonRunningTestAdmissionController(t, newDefaultOptions())

	req := admissionv1beta1.AdmissionRequest{
		Operation: admissionv1beta1.Create,
		Kind:      metav1.GroupVersionKind{Kind: "Garbage"},
	}

	expectFailsWith(t, ac.Admit(&req), "unhandled kind")
}

func TestInvalidNewFooBarNameFails(t *testing.T) {
	_, ac := newNonRunningTestAdmissionController(t, newDefaultOptions())
	req := &admissionv1beta1.AdmissionRequest{
		Operation: admissionv1beta1.Create,
		Kind:      metav1.GroupVersionKind{Kind: v1alpha1.Kindz},
	}
	invalidName := "foobar.example"
	config := createFooBar(invalidName)
	marshaled, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("failed to marshal: %s", err)
	}
	req.Object.Raw = marshaled
	expectFailsWith(t, ac.Admit(req), "invalid resource name")

	invalidName = strings.Repeat("a", 64)
	config = createFooBar(invalidName)
	marshaled, err = json.Marshal(config)
	if err != nil {
		t.Fatalf("failed to marshal: %s", err)
	}
	req.Object.Raw = marshaled
	expectFailsWith(t, ac.Admit(req), "invalid resource name")
}

func TestValidNewFooBarObject(t *testing.T) {
	_, ac := newNonRunningTestAdmissionController(t, newDefaultOptions())
	resp := ac.Admit(createValidCreateFooBar())
	expectAllowed(t, resp)
}

func TestValidFooBarNoChanges(t *testing.T) {
	_, ac := newNonRunningTestAdmissionController(t, newDefaultOptions())
	old := createFooBar(testFooBarName)
	new := createFooBar(testFooBarName)
	resp := ac.Admit(createUpdateFooBar(&old, &new))
	expectAllowed(t, resp)
	expectPatches(t, resp.Patch, []jsonpatch.JsonPatchOperation{})
}

func TestValidFooBarChanges(t *testing.T) {
	_, ac := newNonRunningTestAdmissionController(t, newDefaultOptions())
	old := createFooBar(testFooBarName)
	new := createFooBar(testFooBarName)
	new.Spec.Bars = 5
	resp := ac.Admit(createUpdateFooBar(&old, &new))
	expectAllowed(t, resp)
	expectPatches(t, resp.Patch, []jsonpatch.JsonPatchOperation{})
}

func TestValidWebhook(t *testing.T) {
	_, ac := newNonRunningTestAdmissionController(t, newDefaultOptions())
	createDeployment(t, ac)
	ac.Register(ac.Client.AdmissionregistrationV1beta1().MutatingWebhookConfigurations(), []byte{})
	_, err := ac.Client.AdmissionregistrationV1beta1().MutatingWebhookConfigurations().Get(ac.Options.WebhookName, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Failed to create webhook: %s", err)
	}
}

func TestUpdatingWebhook(t *testing.T) {
	_, ac := newNonRunningTestAdmissionController(t, newDefaultOptions())
	webhook := &admissionregistrationv1beta1.MutatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: ac.Options.WebhookName,
		},
		Webhooks: []admissionregistrationv1beta1.Webhook{
			{
				Name:         ac.Options.WebhookName,
				Rules:        []admissionregistrationv1beta1.RuleWithOperations{{}},
				ClientConfig: admissionregistrationv1beta1.WebhookClientConfig{},
			},
		},
	}

	createDeployment(t, ac)
	createWebhook(ac, webhook)
	ac.Register(ac.Client.AdmissionregistrationV1beta1().MutatingWebhookConfigurations(), []byte{})
	currentWebhook, _ := ac.Client.AdmissionregistrationV1beta1().MutatingWebhookConfigurations().Get(ac.Options.WebhookName, metav1.GetOptions{})
	if reflect.DeepEqual(currentWebhook.Webhooks, webhook.Webhooks) {
		t.Fatalf("Expected webhook to be updated")
	}
}

func createUpdateFooBar(old, new *v1alpha1.FooBar) *admissionv1beta1.AdmissionRequest {
	req := createBaseUpdateFooBar()
	marshaled, err := json.Marshal(old)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal: %v", err))
	}
	req.Object.Raw = marshaled
	marshaledOld, err := json.Marshal(new)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal: %v", err))
	}
	req.OldObject.Raw = marshaledOld
	return req
}

func createValidCreateFooBar() *admissionv1beta1.AdmissionRequest {
	req := &admissionv1beta1.AdmissionRequest{
		Operation: admissionv1beta1.Create,
		Kind:      metav1.GroupVersionKind{Kind: v1alpha1.Kindz},
	}
	config := createFooBar(testFooBarName)
	marshaled, err := json.Marshal(config)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal: %v", err))
	}
	req.Object.Raw = marshaled
	return req
}

func createBaseUpdateFooBar() *admissionv1beta1.AdmissionRequest {
	return &admissionv1beta1.AdmissionRequest{
		Operation: admissionv1beta1.Update,
		Kind:      metav1.GroupVersionKind{Kind: v1alpha1.Kindz},
	}
}

func createWebhook(ac *webhook.AdmissionController, webhook *admissionregistrationv1beta1.MutatingWebhookConfiguration) {
	client := ac.Client.AdmissionregistrationV1beta1().MutatingWebhookConfigurations()
	_, err := client.Create(webhook)
	if err != nil {
		panic(fmt.Sprintf("failed to create test webhook: %s", err))
	}
}

func expectAllowed(t *testing.T, resp *admissionv1beta1.AdmissionResponse) {
	if !resp.Allowed {
		t.Errorf("expected allowed, but failed with %+v", resp.Result)
	}
}

func expectFailsWith(t *testing.T, resp *admissionv1beta1.AdmissionResponse, contains string) {
	if resp.Allowed {
		t.Errorf("expected denial, got allowed")
		return
	}
	if !strings.Contains(resp.Result.Message, contains) {
		t.Errorf("expected failure containing %q got %q", contains, resp.Result.Message)
	}
}

func expectPatches(t *testing.T, a []byte, e []jsonpatch.JsonPatchOperation) {
	var actual []jsonpatch.JsonPatchOperation
	// Keep track of the patches we've found
	foundExpected := make([]bool, len(e))
	foundActual := make([]bool, len(e))

	err := json.Unmarshal(a, &actual)
	if err != nil {
		t.Errorf("failed to unmarshal patches: %s", err)
		return
	}
	if len(actual) != len(e) {
		t.Errorf("unexpected number of patches %d expected %d\n%+v\n%+v", len(actual), len(e), actual, e)
	}
	// Make sure all the expected patches are found
	for i, expectedPatch := range e {
		for j, actualPatch := range actual {
			if actualPatch.Json() == expectedPatch.Json() {
				foundExpected[i] = true
				foundActual[j] = true
			}
		}
	}
	for i, f := range foundExpected {
		if !f {
			t.Errorf("did not find %+v in actual patches: %q", e[i], actual)
		}
	}
	for i, f := range foundActual {
		if !f {
			t.Errorf("Extra patch found %+v in expected patches: %q", a[i], e)
		}
	}
}

func createFooBar(configurationName string) v1alpha1.FooBar {
	return v1alpha1.FooBar{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      configurationName,
		},
		Spec: v1alpha1.FooBarSpec{
			Foos: 0,
			Bars: 0,
		},
	}
}

func createDeployment(t *testing.T, ac *webhook.AdmissionController) {
	deployment := &v1beta1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ac.Options.ServiceName,
			Namespace: ac.Options.ServiceNamespace,
		},
	}
	_, err := ac.Client.ExtensionsV1beta1().Deployments(ac.Options.ServiceNamespace).Create(deployment)
	if err != nil {
		t.Errorf("failed to create deployment: %v", err)
	}
}
