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

// SelfSignedCertProvider implements the CertProvider interface.
// It generates self-signed certificates.
type SelfSignedCertProvider struct {
	// Configuration for certificate generat
	DNSNames     []string
	Organization string
	ValidDays    int // Number of days the certificate will be valid for.

	// Configuration of how to persist the certificates in a k8s secret.
	SecretConfig *SecretConfig
}

var _ CertProvider = &SelfSignedCertProvider{}

type SecretConfig struct {
	Name           string
	Namespace      string
	CAKeyName      string
	CACertName     string
	ServerKeyName  string
	ServerCertName string
}

// GetServerCert generates and persists the CA and server cert if it doesn't exist in the secret.
// If they exist in the secret, GetServerCert gets the secret and returns the server key, server cert and CA cert.
func (cp *SelfSignedCertProvider) GetServerCert() (key []byte, cert []byte, caCert []byte, err error) {
	return nil, nil, nil, nil
}
