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

/*
Package certprovider provides interface and implementation to provision
certificates.

Create a certprovider instance. For example:

	cp := NewSelfSignedCertProvider()

Generate and persist the CA if you want.

	err := cp.GenerateCA()
	if err != nil {
		// handle error
	}
	err = cp.PersistCA(namespace, name, keyName, certName)
	if err != nil {
		// handle error
	}

Get the CA certificate.

	caCert, err := cp.GetCACert()
	if err != nil {
		// handle error
	}

Generate and persist the server certificate.

	err := cp.GenerateServerCert(org, dnsNames, days)
	if err != nil {
		// handle error
	}
	err = cp.PersistServerCert(namespace, name, keyName, certName)
	if err != nil {
		// handle error
	}

Consume the generated cert.
You can consume the certificate in raw byte slice format. This approach is
useful if you want to consume the certificate by mounting the secret as a volume
in a pod.

	key, cert, err := cp.GetServerCert()
	if err != nil {
		// handle error
	}
	// Store key and cert in keyFile and certFile respectively.
	svr := &http.Server{
		// configure your server
	}
	svr.ListenAndServeTLS(certFile, keyFile)

You can also consume the certificate in the tls.Config format.

	tls, err := cp.GetTLSConfig()
	if err != nil {
		// handle error
	}

	svr := &http.Server{
		TLSConfig: tls,
		// other server configuration
	}
	svr.ListenAndServeTLS("", "")

*/
package certprovider
