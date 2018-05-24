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

package certprovider

import (
	"crypto/tls"
)

// CertProvider is an interface to provide CA and server certificate.
type CertProvider interface {
	// GenerateCA generates a pair of CA key and certificate for signing the server certificate.
	GenerateCA() error
	// PersistCA persists the CA into a k8s secret named with name in namespace ns.
	// The keyName and certName are the keys in the k8s secret.
	PersistCA(ns, name, keyName, certName string) error
	// GetCACert returns the CA certificate.
	GetCACert() (cert []byte, err error)

	// GenerateServerCert generates a pair of server key and certificate.
	GenerateServerCert(org string, dnsNames []string, days int) error
	// PersistServerCert persists the server key and certificate into a k8s
	// secret named with name in namespace ns. The keyName and certName are the
	// keys in the k8s secret.
	PersistServerCert(ns, name, keyName, certName string) error
	// GetServerCert returns the server key and certificate.
	GetServerCert() (key []byte, cert []byte, err error)

	// GetTLSConfig returns a tls config that can be used to config the http.Server.
	GetTLSConfig() (*tls.Config, error)
}
