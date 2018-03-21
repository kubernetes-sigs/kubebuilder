/*
Copyright 2017 The Kubernetes Authors.

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

package install

import (
	//"bytes"
	//"crypto/rand"
	//"crypto/rsa"
	//"crypto/x509"
	//"crypto/x509/pkix"
	//"encoding/pem"
	"fmt"
	"io/ioutil"
	"log"
	//"math/big"
	//"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	//"time"
)

// Certs contains the certificate information for installing APIs
type Certs struct {
	// ClientKey is the client private key
	ClientKey []byte

	// CACrt is the public CA certificate
	CACrt []byte

	// ClientCrt is the public client certificate
	ClientCrt []byte
}

// CreateCerts creates Certs to be used for registering webhooks and extension apis in a
// Kubernetes apiserver
func CreateCerts(serviceName, serviceNamespace string) *Certs {
	dir := os.TempDir()

	openssl("req", "-x509",
		"-newkey", "rsa:2048",
		"-keyout", filepath.Join(dir, "apiserver_ca.key"),
		"-out", filepath.Join(dir, "apiserver_ca.crt"),
		"-days", "365",
		"-nodes",
		"-subj", fmt.Sprintf("/C=un/ST=st/L=l/O=o/OU=ou/CN=%s-certificate-authority", serviceName),
	)

	// Use <service-Name>.<Namespace>.svc as the domain Name for the certificate
	openssl("req",
		"-out", filepath.Join(dir, "apiserver.csr"),
		"-new",
		"-newkey", "rsa:2048",
		"-nodes",
		"-keyout", filepath.Join(dir, "apiserver.key"),
		"-subj", fmt.Sprintf("/C=un/ST=st/L=l/O=o/OU=ou/CN=%s.%s.svc", serviceName, serviceNamespace),
	)

	openssl("x509", "-req",
		"-days", "365",
		"-in", filepath.Join(dir, "apiserver.csr"),
		"-CA", filepath.Join(dir, "apiserver_ca.crt"),
		"-CAkey", filepath.Join(dir, "apiserver_ca.key"),
		"-CAcreateserial",
		"-out", filepath.Join(dir, "apiserver.crt"),
	)
	cert := &Certs{}
	var err error
	cert.CACrt, err = ioutil.ReadFile(filepath.Join(dir, "apiserver_ca.crt"))
	if err != nil {
		log.Fatalf("read %s failed %v", filepath.Join(dir, "apiserver_ca.crt"), err)
	}
	cert.ClientKey, err = ioutil.ReadFile(filepath.Join(dir, "apiserver.key"))
	if err != nil {
		log.Fatalf("read %s failed %v", filepath.Join(dir, "apiserver.key"), err)
	}
	cert.ClientCrt, err = ioutil.ReadFile(filepath.Join(dir, "apiserver.crt"))
	if err != nil {
		log.Fatalf("read %s failed %v", filepath.Join(dir, "apiserver.crt"), err)
	}
	return cert
}

func openssl(args ...string) {
	c := exec.Command("openssl", args...)
	c.Stderr = os.Stderr
	c.Stdout = os.Stdout
	log.Printf("%s\n", strings.Join(c.Args, " "))
	err := c.Run()
	if err != nil {
		log.Fatalf("command failed %v", err)
	}
}

//type Cert struct {
//	Email string
//	DNS   string
//	Org   string
//	Hosts []string
//}
//
//func (c Cert) Create() ([]byte, []byte) {
//	// yesterday
//	notBefore := time.Now().AddDate(0, 0, -1)
//	// 1 year from yesterday
//	notAfter := notBefore.Add(time.Hour * 24 * 365)
//
//	priv, err := rsa.GenerateKey(rand.Reader, 2048)
//
//	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
//	if err != nil {
//		log.Fatalf("failed to generate serial number: %s", err)
//	}
//
//	// Create the cert template
//	template := x509.Certificate{
//		SerialNumber: serialNumber,
//		Subject: pkix.Name{
//			Organization: []string{c.Org},
//		},
//		NotBefore: notBefore,
//		NotAfter:  notAfter,
//
//		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
//		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
//		BasicConstraintsValid: true,
//		IsCA: true,
//	}
//	template.DNSNames = append(template.DNSNames, c.DNS)
//	template.EmailAddresses = append(template.EmailAddresses, c.Email)
//	template.KeyUsage |= x509.KeyUsageCertSign
//	for _, h := range c.Hosts {
//		if ip := net.ParseIP(h); ip != nil {
//			template.IPAddresses = append(template.IPAddresses, ip)
//		} else {
//			template.DNSNames = append(template.DNSNames, h)
//		}
//	}
//
//	// Create the certificate
//	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
//	if err != nil {
//		log.Fatalf("Failed to create certificate: %s", err)
//	}
//
//	pemBytes := &bytes.Buffer{}
//	err = pem.Encode(pemBytes, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
//	if err != nil {
//		log.Fatalf("failed to encode certificate: %v", err)
//	}
//
//	keyBytes := &bytes.Buffer{}
//	err = pem.Encode(keyBytes, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})
//	if err != nil {
//		log.Fatalf("failed to encode certificate: %v", err)
//	}
//
//	return pemBytes.Bytes(), keyBytes.Bytes()
//}
