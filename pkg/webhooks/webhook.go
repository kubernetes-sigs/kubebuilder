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

package webhooks

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/mattbaird/jsonpatch"

	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	"k8s.io/api/extensions/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientadmissionregistrationv1beta1 "k8s.io/client-go/kubernetes/typed/admissionregistration/v1beta1"
)

const (
	secretServerKey  = "server-key.pem"
	secretServerCert = "server-cert.pem"
	secretCACert     = "ca-cert.pem"
)

var deploymentKind = v1beta1.SchemeGroupVersion.WithKind("Deployment")

// Run implements the admission controller run loop.
func (ac *AdmissionController) Run(stop <-chan struct{}) error {
	tlsConfig, caCert, err := BuildCerts(ac.Client, &ac.Options)
	if err != nil {
		glog.Infof("could not configure admission webhook certs: %v", err)
		return err
	}

	server := &http.Server{
		Handler:   ac,
		Addr:      fmt.Sprintf(":%v", ac.Options.Port),
		TLSConfig: tlsConfig,
	}

	glog.Info("Found certificates for webhook...")
	if ac.Options.RegistrationDelay != 0 {
		glog.Infof("Delaying admission webhook registration for %v", ac.Options.RegistrationDelay)
	}

	select {
	case <-time.After(ac.Options.RegistrationDelay):
		cl := ac.Client.AdmissionregistrationV1beta1().MutatingWebhookConfigurations()
		if err := ac.Register(cl, caCert); err != nil {
			glog.Infof("Failed to register webhook: %v", err)
			return err
		}
		defer func() {
			if err := ac.Unregister(cl); err != nil {
				glog.Infof("failed to unregister webhook: %v", err)
			}
		}()
		glog.Info("successfully registered webhook")
	case <-stop:
		return nil
	}

	go func() {
		if err := server.ListenAndServeTLS("", ""); err != nil {
			glog.Infof("ListenAndServeTLS for admission webhook returned error: %v", err)
		}
	}()
	<-stop
	server.Close()
	return nil
}

// Unregister unregisters the external admission webhook
func (ac *AdmissionController) Unregister(client clientadmissionregistrationv1beta1.MutatingWebhookConfigurationInterface) error {
	glog.Info("exiting...")
	return nil
}

// Register registers the external admission webhook for pilot configuration types.
func (ac *AdmissionController) Register(client clientadmissionregistrationv1beta1.MutatingWebhookConfigurationInterface, caCert []byte) error {
	webhook := &admissionregistrationv1beta1.MutatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: ac.Options.WebhookName,
		},
		Webhooks: []admissionregistrationv1beta1.Webhook{
			{
				Name: ac.Options.WebhookName,
				Rules: []admissionregistrationv1beta1.RuleWithOperations{{
					Operations: []admissionregistrationv1beta1.OperationType{
						admissionregistrationv1beta1.Create,
						admissionregistrationv1beta1.Update, // TODO: DELETE and CONNECT are also options.
					},
					Rule: admissionregistrationv1beta1.Rule{
						APIGroups:   []string{ac.Options.APIGroupName},
						APIVersions: []string{ac.Options.APIVersion},
						Resources:   ac.Options.Resources,
					},
				}},
				ClientConfig: admissionregistrationv1beta1.WebhookClientConfig{
					Service: &admissionregistrationv1beta1.ServiceReference{
						Namespace: ac.Options.ServiceNamespace,
						Name:      ac.Options.ServiceName,
					},
					CABundle: caCert,
				},
			},
		},
	}

	// Set the owner to our deployment
	deployment, err := ac.Client.ExtensionsV1beta1().Deployments(ac.Options.ServiceNamespace).Get(ac.Options.ServiceName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to fetch our deployment: %s", err)
	}
	deploymentRef := metav1.NewControllerRef(deployment, deploymentKind)
	webhook.OwnerReferences = append(webhook.OwnerReferences, *deploymentRef)

	// Try to create the webhook and if it already exists validate webhook rules
	_, err = client.Create(webhook)
	if err != nil {
		if !apierrors.IsAlreadyExists(err) {
			return fmt.Errorf("failed to create a webhook: %s", err)
		}
		glog.Infof("webhook already exists")
		configuredWebhook, err := client.Get(ac.Options.WebhookName, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("error retrieving webhook: %s", err)
		}
		if !reflect.DeepEqual(configuredWebhook.Webhooks, webhook.Webhooks) {
			glog.Infof("updating webhook")
			// Set the ResourceVersion as required by update.
			webhook.ObjectMeta.ResourceVersion = configuredWebhook.ObjectMeta.ResourceVersion
			if _, err := client.Update(webhook); err != nil {
				return fmt.Errorf("failed to update webhook: %s", err)
			}
		} else {
			glog.Infof("webhook is already valid")
		}
	} else {
		glog.Infof("created a webhook")
	}
	return nil
}

// ServeHTTP implements the external admission webhook for mutating resources.
func (ac *AdmissionController) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	glog.Infof("Webhook ServeHTTP request=%#v", r)

	// verify the content type is accurate
	contentType := r.Header.Get("Content-Type")
	if contentType != "application/json" {
		http.Error(w, "invalid Content-Type, want `application/json`", http.StatusUnsupportedMediaType)
		return
	}

	var review admissionv1beta1.AdmissionReview
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&review); err != nil {
		http.Error(w, fmt.Sprintf("could not decode body: %v", err), http.StatusBadRequest)
		return
	}

	reviewResponse := ac.Admit(review.Request)
	var response admissionv1beta1.AdmissionReview
	if reviewResponse != nil {
		response.Response = reviewResponse
		response.Response.UID = review.Request.UID
	}

	glog.Infof("AdmissionReview for %s: %v/%v response=%v",
		review.Request.Kind, review.Request.Namespace, review.Request.Name, reviewResponse)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, fmt.Sprintf("could encode response: %v", err), http.StatusInternalServerError)
		return
	}
}

func makeErrorStatus(reason string, args ...interface{}) *admissionv1beta1.AdmissionResponse {
	result := apierrors.NewBadRequest(fmt.Sprintf(reason, args...)).Status()
	return &admissionv1beta1.AdmissionResponse{
		Result:  &result,
		Allowed: false,
	}
}

func (ac *AdmissionController) Admit(request *admissionv1beta1.AdmissionRequest) *admissionv1beta1.AdmissionResponse {
	switch request.Operation {
	case admissionv1beta1.Create, admissionv1beta1.Update:
	default:
		glog.Infof("unhandled webhook operation, letting it through %v", request.Operation)
		return &admissionv1beta1.AdmissionResponse{Allowed: true}
	}

	patchBytes, err := ac.Mutate(request.Kind.Kind, request.OldObject.Raw, request.Object.Raw)
	if err != nil {
		return makeErrorStatus("mutation failed: %v", err)
	}
	glog.Infof("Kind: %q PatchBytes: %v", request.Kind, string(patchBytes))

	return &admissionv1beta1.AdmissionResponse{
		Patch:   patchBytes,
		Allowed: true,
		PatchType: func() *admissionv1beta1.PatchType {
			pt := admissionv1beta1.PatchTypeJSONPatch
			return &pt
		}(),
	}
}

func (ac *AdmissionController) Mutate(kind string, oldBytes []byte, newBytes []byte) ([]byte, error) {
	handler, ok := ac.Handlers[kind]
	if !ok {
		glog.Warningf("Unhandled kind %q", kind)
		return nil, fmt.Errorf("unhandled kind: %q", kind)
	}

	oldObj := handler.Factory.DeepCopyObject().(GenericCRD)
	newObj := handler.Factory.DeepCopyObject().(GenericCRD)

	if len(newBytes) != 0 {
		newDecoder := json.NewDecoder(bytes.NewBuffer(newBytes))
		newDecoder.DisallowUnknownFields()
		if err := newDecoder.Decode(&newObj); err != nil {
			return nil, fmt.Errorf("cannot decode incoming new object: %v", err)
		}
	} else {
		// Use nil to denote the absence of a new object (delete)
		newObj = nil
	}

	if len(oldBytes) != 0 {
		oldDecoder := json.NewDecoder(bytes.NewBuffer(oldBytes))
		oldDecoder.DisallowUnknownFields()
		if err := oldDecoder.Decode(&oldObj); err != nil {
			return nil, fmt.Errorf("cannot decode incoming old object: %v", err)
		}
	} else {
		// Use nil to denote the absence of an old object (create)
		oldObj = nil
	}

	var patches []jsonpatch.JsonPatchOperation

	if defaulter := handler.Defaulter; defaulter != nil {
		if err := defaulter(&patches, oldObj, newObj); err != nil {
			glog.Warningf("failed the resource specific defaulter: %s", err)
			// Return the error message as-is to give the defaulter callback
			// discretion over (our portion of) the message that the user sees.
			return nil, err
		}
	}

	if err := handler.Validator(&patches, oldObj, newObj); err != nil {
		glog.Warningf("failed the resource specific validation: %s", err)
		// Return the error message as-is to give the validation callback
		// discretion over (our portion of) the message that the user sees.
		return nil, err
	}

	if err := validateMetadata(newObj); err != nil {
		glog.Warningf("failed to validate : %s", err)
		return nil, fmt.Errorf("failed to validate: %s", err)
	}
	return json.Marshal(patches)
}

func validateMetadata(new GenericCRD) error {
	name := new.GetObjectMeta().GetName()

	if strings.Contains(name, ".") {
		return errors.New("invalid resource name: special character . must not be present")
	}

	if len(name) > 63 {
		return errors.New("invalid resource name: length must be no more than 63 characters")
	}
	return nil
}
