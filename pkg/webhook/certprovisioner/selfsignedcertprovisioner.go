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

package certprovisioner

import (
	"crypto/x509"
	"fmt"

	"k8s.io/client-go/util/cert"
)

// SelfSignedCertProvisioner implements the CertProvisioner interface.
// It provisions self-signed certificates.
type SelfSignedCertProvisioner struct {
	// Required Common Name
	CommonName string
}

var _ CertProvisioner = &SelfSignedCertProvisioner{}

// ProvisionServingCert creates and returns a CA certificate and certificate and
// key for the server. serverKey and serverCert are used by the server
// to establish trust for clients, CA certificate is used by the
// client to verify the server authentication chain.
// The cert will be valid for 365 days.
func (cp *SelfSignedCertProvisioner) ProvisionServingCert() (serverKey, serverCert, caCert []byte, err error) {
	signingKey, err := cert.NewPrivateKey()
	if err != nil {
		return nil, nil, nil,
			fmt.Errorf("failed to create the CA private key: %v", err)
	}
	signingCert, err := cert.NewSelfSignedCACert(cert.Config{CommonName: "webhook-cert-ca"}, signingKey)
	if err != nil {
		return nil, nil, nil,
			fmt.Errorf("failed to create the CA cert: %v", err)
	}
	key, err := cert.NewPrivateKey()
	if err != nil {
		return nil, nil, nil,
			fmt.Errorf("failed to create the private key: %v", err)
	}
	signedCert, err := cert.NewSignedCert(
		cert.Config{
			CommonName: cp.CommonName,
			Usages:     []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		},
		key, signingCert, signingKey,
	)
	if err != nil {
		return nil, nil, nil,
			fmt.Errorf("failed to create the cert: %v", err)
	}
	return cert.EncodePrivateKeyPEM(key), cert.EncodeCertPEM(signedCert), cert.EncodeCertPEM(signingCert), nil
}
