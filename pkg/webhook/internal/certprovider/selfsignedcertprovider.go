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

type selfSignedCertProvider struct {
	cAKey      []byte
	cACert     []byte
	serverKey  []byte
	serverCert []byte

	genConfig *genCertConfig

	secretConfig *secretConfig
}

var _ CertProvider = &selfSignedCertProvider{}

type genCertConfig struct {
	dnsNames     []string
	organization string
}

type secretConfig struct {
	name           string
	namespace      string
	caKeyName      string
	caCertName     string
	serverKeyName  string
	serverCertName string
}

func NewSelfSignedCertProvider() CertProvider { return nil }

func (cp *selfSignedCertProvider) GenerateCA() error { return nil }

func (cp *selfSignedCertProvider) GetCACert() ([]byte, error) { return nil, nil }

func (cp *selfSignedCertProvider) GenerateServerCert(org string, dnsNames []string, days int) error {
	return nil
}

func (cp *selfSignedCertProvider) GetServerCert() ([]byte, []byte, error) { return nil, nil, nil }

func (cp *selfSignedCertProvider) PersistCA(namespace, name, keyName, certName string) error {
	return nil
}

func (cp *selfSignedCertProvider) PersistServerCert(namespace, name, keyName, certName string) error {
	return nil
}

func (cp *selfSignedCertProvider) GetTLSConfig() (*tls.Config, error) { return nil, nil }
