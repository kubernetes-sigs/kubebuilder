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

package internal

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/golang/glog"

	"k8s.io/api/admission/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type AdmissionFunc func(review *v1beta1.AdmissionRequest) *v1beta1.AdmissionResponse

// HandleEntry
type admissionHandler struct {
	Config *HandlerConfig
	Fn     AdmissionFunc
}

// handle handles an admission request and returns a result
func (ah admissionHandler) handle(request *v1beta1.AdmissionRequest) *v1beta1.AdmissionResponse {
	//
	foundMatchingOperation := false
	for _, op := range ah.Config.Operations {
		if string(request.Operation) == string(op) {
			foundMatchingOperation = true
			break
		}
	}
	if !foundMatchingOperation {
		// If getting an unexpected operation, admit it.
		// This should not happen if the webhook configuration is registered properly.
		glog.Infof("expected operation set: %v but got: %v", ah.Config.Operations, request.Operation)
		return allowResponse()
	}

	foundMatchingGVR := false
	for _, gvr := range ah.Config.GroupVersionResources {
		if request.Resource == gvr {
			foundMatchingGVR = true
			break
		}
	}
	if !foundMatchingGVR {
		// If getting an unexpected GVR, admit it.
		// This should not happen if the webhook configuration is registered properly.
		glog.Infof("expected GroupVersionResource set: %v but got: %v", ah.Config.GroupVersionResources, request.Resource)
		return allowResponse()
	}

	return ah.Fn(request)
}

// AdmissionServer is an admission webhook server that can serve traffic and
// generates related k8s reources for deploying.
type AdmissionServer struct {
	SMux *http.ServeMux

	// A map that maps a path to an admissionHandler.
	Entries map[string]*admissionHandler

	// Umbrella configuration of the admission webhook
	Config *AdmissionWebhookInstallConfig

	KeyPath  string // Path of the server key
	CertPath string // Path of the server certificate

	ConfigMapPath string // Path to mount the configMap
}

// DefaultAdmissionServer returns an AdmissionServer and defaults all possible fields.
func DefaultAdmissionServer(t AdmissionWebhookConfigType, gvrs []metav1.GroupVersionResource) *AdmissionServer {
	return &AdmissionServer{
		Entries: map[string]*admissionHandler{},
		SMux:    http.DefaultServeMux,
		Config: &AdmissionWebhookInstallConfig{
			ServerConfig: []*HandlerConfig{
				{
					WebhookType:           t,
					GroupVersionResources: gvrs,
				},
			},
			Port: 443,
		},
	}
}

// GetDefault does defaulting to the AdmissionServer.
func (s *AdmissionServer) GetDefault() error {
	return nil
}

// HandleFunc registers fn as an admission control webhook callback given the HandlerConfig
func (s *AdmissionServer) HandleFunc(hc *HandlerConfig, fn AdmissionFunc) {
	ah := &admissionHandler{Config: hc, Fn: fn}
	// Register the entry so a Webhook config is created
	s.Entries[hc.Path] = ah
	// Register the handler path
	s.SMux.Handle(hc.Path, httpHandler{ah})
}

// ListenAndServeTLS listens on the TCP network address and starts to serve.
func (s *AdmissionServer) ListenAndServeTLS() error {
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%v", s.Config.Port),
		Handler: s.SMux,
	}
	return srv.ListenAndServeTLS(s.CertPath, s.KeyPath)
}

// WriteConfig writes the configuration of the admission webhook server to the ConfigMapPath.
// So the webhook installer can use the configuration to manage the webhook server.
func (s *AdmissionServer) WriteConfig() error {
	marshaledConfig, err := json.Marshal(s.Config)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(s.ConfigMapPath, marshaledConfig, 0644)
}

// Generate generates k8s resources configuration in yaml format.
// It generates
// - a deployment for the admission webhook server
// - RBAC and serviceAccount for the deployment above
// - a statefulSet for the admission webhook installer
// - RBAC and serviceAccount for the statefulSet above
func (s *AdmissionServer) Generate() (string, error) {
	return "", nil
}
