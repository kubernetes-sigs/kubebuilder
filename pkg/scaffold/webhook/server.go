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
	"path/filepath"

	"sigs.k8s.io/kubebuilder/pkg/scaffold/input"
	"sigs.k8s.io/kubebuilder/pkg/scaffold/resource"
)

var _ input.File = &Server{}

// Server scaffolds how to construct a webhook server and register webhooks.
type Server struct {
	input.Input

	// Resource is a resource in the API group
	Resource *resource.Resource

	Config
}

// GetInput implements input.File
func (a *Server) GetInput() (input.Input, error) {
	if a.Path == "" {
		a.Path = filepath.Join("pkg", "webhook", fmt.Sprintf("%s_server", a.Server), "server.go")
	}
	a.TemplateBody = serverTemplate
	return a.Input, nil
}

var serverTemplate = `{{ .Boilerplate }}

package {{ .Server }}server

import (
	"sigs.k8s.io/controller-runtime/pkg/manager"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission/builder"
)

var (
	log        = logf.Log.WithName("{{ .Server }}_server")
	webhooks []webhook.Webhook
)

// Add adds itself to the manager
// +kubebuilder:webhook:port=9876,cert-dir=/tmp/cert
// +kubebuilder:webhook:service=system:webhook-service,selector=app:webhook-server
// +kubebuilder:webhook:secret=system:webhook-secret
// +kubebuilder:webhook:mutating-webhook-config-name=mutating-webhook-config,validating-webhook-config-name=validating-webhook-config
func Add(mgr manager.Manager) error {
	svr, err := webhook.NewServer("foo-admission-server", mgr, webhook.ServerOptions{
		// TODO(user): change ServerOptions and the annotations starting with +kubebuilder:webhook at the same time.
		Port:    9876,
		CertDir: "/tmp/cert",
	})
	if err != nil {
		return err
	}

	return svr.Register(webhooks...)
}
`
