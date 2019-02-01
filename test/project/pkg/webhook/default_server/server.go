/*
Copyright 2019 The Kubernetes authors.

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

package defaultserver

import (
	"sigs.k8s.io/controller-runtime/pkg/manager"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

var (
	log      = logf.Log.WithName("default_server")
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
