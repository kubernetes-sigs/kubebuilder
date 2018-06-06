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
	"encoding/pem"
	"testing"
)

func TestProvisionServingCert(t *testing.T) {
	cn := "mysvc.myns.svc"
	cp := SelfSignedCertProvisioner{CommonName: cn}
	certs, err := cp.ProvisionServingCert()

	// First, create the set of root certificates. For this example we only
	// have one. It's also possible to omit this in order to use the
	// default root set of the current operating system.
	roots := x509.NewCertPool()
	ok := roots.AppendCertsFromPEM(certs.CACert)
	if !ok {
		t.Fatalf("failed to parse root certificate: %s", certs.CACert)
	}

	block, _ := pem.Decode(certs.Cert)
	if block == nil {
		t.Fatalf("failed to parse certificate PEM: %s", certs.Cert)
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatalf("failed to parse certificate: %v", err)
	}

	opts := x509.VerifyOptions{
		DNSName: cn,
		Roots:   roots,
	}

	if _, err := cert.Verify(opts); err != nil {
		t.Fatalf("failed to verify certificate: %v", err)
	}
}
