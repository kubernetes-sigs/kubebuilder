/*
Copyright 2021 The Kubernetes Authors.

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

package configgen

import (
	"encoding/base64"
	"fmt"

	"github.com/cloudflare/cfssl/cli/genkey"
	"github.com/cloudflare/cfssl/config"
	"github.com/cloudflare/cfssl/csr"
	"github.com/cloudflare/cfssl/helpers"
	"github.com/cloudflare/cfssl/selfsign"
	"sigs.k8s.io/kustomize/kyaml/fn/framework"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

var _ kio.Filter = &CertFilter{}

// CertFilter generates and injects certificates into webhook
type CertFilter struct {
	*KubebuilderConfigGen
}

// Filter implements kio.Filter
// TODO: when v1 CRDs are supported, scaffold conversion webhook versions.
func (c CertFilter) Filter(input []*yaml.RNode) ([]*yaml.RNode, error) {

	if c.Spec.Webhooks.CertificateSource.Type != "dev" {
		return input, nil
	}
	if err := c.generateCert(); err != nil {
		return nil, err
	}

	matches, err := (&framework.Selector{
		Kinds: []string{
			"ValidatingWebhookConfiguration",
			"MutatingWebhookConfiguration",
		},
	}).Filter(input)
	if err != nil {
		return nil, err
	}
	for i := range matches {
		wh := matches[i].Field("webhooks")
		if wh.IsNilOrEmpty() {
			continue
		}
		err := wh.Value.VisitElements(func(node *yaml.RNode) error {
			err := node.PipeE(yaml.LookupCreate(yaml.ScalarNode, "clientConfig", "caBundle"),
				yaml.FieldSetter{StringValue: c.Status.CertCA})
			if err != nil {
				return err
			}
			err = node.PipeE(yaml.LookupCreate(yaml.ScalarNode, "clientConfig", "service", "namespace"),
				yaml.FieldSetter{StringValue: c.Namespace})
			if err != nil {
				return err
			}

			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	matches, err = (&framework.Selector{
		Kinds: []string{"CustomResourceDefinition"},
		ResourceMatcher: func(m *yaml.RNode) bool {
			meta, _ := m.GetMeta()
			return c.Spec.Webhooks.Conversions[meta.Name]
		},
	}).Filter(input)
	if err != nil {
		return nil, err
	}
	for i := range matches {
		err := matches[i].PipeE(yaml.LookupCreate(yaml.ScalarNode, "spec", "conversion", "strategy"),
			yaml.FieldSetter{StringValue: "Webhook"})
		if err != nil {
			return nil, err
		}
		err = matches[i].PipeE(yaml.LookupCreate(
			yaml.ScalarNode, "spec", "conversion", "webhookClientConfig", "caBundle"),
			yaml.FieldSetter{StringValue: c.Status.CertCA})
		if err != nil {
			return nil, err
		}
		err = matches[i].PipeE(yaml.LookupCreate(
			yaml.ScalarNode, "spec", "conversion", "webhookClientConfig", "service", "name"),
			yaml.FieldSetter{StringValue: "webhook-service"})
		if err != nil {
			return nil, err
		}
		err = matches[i].PipeE(yaml.LookupCreate(
			yaml.ScalarNode, "spec", "conversion", "webhookClientConfig", "service", "namespace"),
			yaml.FieldSetter{StringValue: c.Namespace})
		if err != nil {
			return nil, err
		}

		err = matches[i].PipeE(yaml.LookupCreate(
			yaml.ScalarNode, "spec", "conversion", "webhookClientConfig", "service", "path"),
			yaml.FieldSetter{StringValue: "/convert"})
		if err != nil {
			return nil, err
		}
	}

	return input, nil
}

func (c CertFilter) generateCert() error {
	var err error
	var req = csr.New()
	req.Hosts = []string{
		fmt.Sprintf("webhook-service.%s.svc", c.Namespace),
		fmt.Sprintf("webhook-service.%s.svc.cluster.local", c.Namespace),
	}
	req.CN = "kb-dev-controller-manager"

	var key, csrPEM []byte
	g := &csr.Generator{Validator: genkey.Validator}
	csrPEM, key, err = g.ProcessRequest(req)
	if err != nil {
		return err
	}
	priv, err := helpers.ParsePrivateKeyPEM(key)
	if err != nil {
		return err
	}

	profile := config.DefaultConfig()
	profile.Expiry = c.Spec.Webhooks.CertificateSource.DevCertificate.CertDuration
	cert, err := selfsign.Sign(priv, csrPEM, profile)
	if err != nil {
		return err
	}

	c.Status.CertCA = base64.StdEncoding.EncodeToString(cert)
	c.Status.CertKey = base64.StdEncoding.EncodeToString(key)
	return nil
}
