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

// SelfSignedCertProvisioner implements the CertProvisioner interface.
// It provisions self-signed certificates.
type SelfSignedCertProvisioner struct {
	// Required DNS names for your certificate
	DNSNames []string
	// Organization name
	Organization string
	// Number of days the certificate will be valid for.
	ValidDays int
}

var _ CertProvisioner = &SelfSignedCertProvisioner{}

// ProvisionServingCert generates a CA and a serving cert. It returns the key, serving cert, CA cert and a potential error.
func (cp *SelfSignedCertProvisioner) ProvisionServingCert() (key []byte, cert []byte, caCert []byte, err error) {
	// TODO: implement this
	return nil, nil, nil, nil
}
