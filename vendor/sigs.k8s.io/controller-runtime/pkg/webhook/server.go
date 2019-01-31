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
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"path"
	"strconv"
	"sync"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/runtime/inject"
	atypes "sigs.k8s.io/controller-runtime/pkg/webhook/admission/types"
)

const (
	certName = "tls.crt"
	keyName  = "tls.key"
)

// ServerOptions are options for configuring an admission webhook server.
type ServerOptions struct {
	// Port is the port number that the server will serve.
	// It will be defaulted to 443 if unspecified.
	Port int32

	// CertDir is the directory that contains the server key and certificate.
	// If using FSCertWriter in Provisioner, the server itself will provision the certificate and
	// store it in this directory.
	// If using SecretCertWriter in Provisioner, the server will provision the certificate in a secret,
	// the user is responsible to mount the secret to the this location for the server to consume.
	CertDir string

	// Client is a client defined in controller-runtime instead of a client-go client.
	// It knows how to talk to a kubernetes cluster.
	// Client will be injected by the manager if not set.
	Client client.Client

	// err will be non-nil if there is an error occur during initialization.
	err error // nolint: structcheck
}

// Server is an admission webhook server that can serve traffic and
// generates related k8s resources for deploying.
type Server struct {
	// Name is the name of server
	Name string

	// ServerOptions contains options for configuring the admission server.
	ServerOptions

	sMux *http.ServeMux
	// registry maps a path to a http.Handler.
	registry map[string]Webhook

	// manager is the manager that this webhook server will be registered.
	manager manager.Manager

	once sync.Once
}

// Webhook defines the basics that a webhook should support.
type Webhook interface {
	// GetPath returns the path that the webhook registered.
	GetPath() string
	// Handler returns a http.Handler for the webhook.
	Handler() http.Handler
	// Validate validates if the webhook itself is valid.
	// If invalid, a non-nil error will be returned.
	Validate() error
}

// NewServer creates a new admission webhook server.
func NewServer(name string, mgr manager.Manager, options ServerOptions) (*Server, error) {
	as := &Server{
		Name:          name,
		sMux:          http.NewServeMux(),
		registry:      map[string]Webhook{},
		ServerOptions: options,
		manager:       mgr,
	}

	return as, nil
}

// setDefault does defaulting for the Server.
func (s *Server) setDefault() {
	if len(s.Name) == 0 {
		s.Name = "default-k8s-webhook-server"
	}
	if s.registry == nil {
		s.registry = map[string]Webhook{}
	}
	if s.sMux == nil {
		s.sMux = http.DefaultServeMux
	}
	if s.Port <= 0 {
		s.Port = 443
	}
	if len(s.CertDir) == 0 {
		s.CertDir = path.Join("k8s-webhook-server", "cert")
	}

	if s.Client == nil {
		cfg, err := config.GetConfig()
		if err != nil {
			s.err = err
			return
		}
		s.Client, err = client.New(cfg, client.Options{})
		if err != nil {
			s.err = err
			return
		}
	}
}

// Register validates and registers webhook(s) in the server
func (s *Server) Register(webhooks ...Webhook) error {
	for i, webhook := range webhooks {
		// validate the webhook before registering it.
		err := webhook.Validate()
		if err != nil {
			return err
		}
		_, found := s.registry[webhook.GetPath()]
		if found {
			return fmt.Errorf("can't register duplicate path: %v", webhook.GetPath())
		}
		s.registry[webhook.GetPath()] = webhooks[i]
		s.sMux.Handle(webhook.GetPath(), webhook.Handler())
	}

	// Lazily add Server to manager.
	// Because the all webhook handlers to be in place, so we can inject the things they need.
	return s.manager.Add(s)
}

// Handle registers a http.Handler for the given pattern.
func (s *Server) Handle(pattern string, handler http.Handler) {
	s.sMux.Handle(pattern, handler)
}

var _ manager.Runnable = &Server{}

// Start runs the server.
// It will install the webhook related resources depend on the server configuration.
func (s *Server) Start(stop <-chan struct{}) error {
	s.once.Do(s.setDefault)
	if s.err != nil {
		return s.err
	}

	// TODO: watch the cert dir. Reload the cert if it changes
	cert, err := tls.LoadX509KeyPair(path.Join(s.CertDir, certName), path.Join(s.CertDir, keyName))
	if err != nil {
		return err
	}

	cfg := &tls.Config{
		Certificates: []tls.Certificate{cert},
	}

	listener, err := tls.Listen("tcp", net.JoinHostPort("", strconv.Itoa(int(s.Port))), cfg)
	if err != nil {
		return err
	}

	srv := &http.Server{
		Handler: s.sMux,
	}

	idleConnsClosed := make(chan struct{})
	go func() {
		<-stop

		// TODO: use a context with reasonable timeout
		if err := srv.Shutdown(context.Background()); err != nil {
			// Error from closing listeners, or context timeout
			log.Error(err, "error shutting down the HTTP server")
		}
		close(idleConnsClosed)
	}()

	err = srv.Serve(listener)
	if err != nil && err != http.ErrServerClosed {
		return err
	}

	<-idleConnsClosed
	return nil
}

var _ inject.Client = &Server{}

// InjectClient injects the client into the server
func (s *Server) InjectClient(c client.Client) error {
	s.Client = c
	for _, wh := range s.registry {
		if _, err := inject.ClientInto(c, wh.Handler()); err != nil {
			return err
		}
	}
	return nil
}

var _ inject.Decoder = &Server{}

// InjectDecoder injects the decoder into the server
func (s *Server) InjectDecoder(d atypes.Decoder) error {
	for _, wh := range s.registry {
		if _, err := inject.DecoderInto(d, wh.Handler()); err != nil {
			return err
		}
	}
	return nil
}
