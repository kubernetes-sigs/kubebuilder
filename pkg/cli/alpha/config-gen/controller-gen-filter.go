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
	"bytes"
	"fmt"
	"io"
	"os"

	"sigs.k8s.io/controller-tools/pkg/crd"
	"sigs.k8s.io/controller-tools/pkg/genall"
	"sigs.k8s.io/controller-tools/pkg/loader"
	"sigs.k8s.io/controller-tools/pkg/rbac"
	"sigs.k8s.io/controller-tools/pkg/webhook"
	"sigs.k8s.io/kustomize/kyaml/errors"
	"sigs.k8s.io/kustomize/kyaml/kio"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

var _ kio.Filter = &ControllerGenFilter{}

// ControllerGenFilter generates resources from go code using the controller-gen libraries
type ControllerGenFilter struct {
	*KubebuilderConfigGen
}

// Filter implements kio.Filter
func (cgr ControllerGenFilter) Filter(input []*yaml.RNode) ([]*yaml.RNode, error) {
	gens := genall.Generators{}

	// generate CRD definitions
	desclen := 40
	crdGen := genall.Generator(crd.Generator{
		MaxDescLen: &desclen,
	})
	gens = append(gens, &crdGen)

	// generate RBAC definitions
	rbacGen := genall.Generator(rbac.Generator{
		RoleName: cgr.Namespace + "-manager-role",
	})
	gens = append(gens, &rbacGen)

	// generate Webhook definitions
	if cgr.Spec.Webhooks.Enable {
		webhookGen := genall.Generator(webhook.Generator{})
		gens = append(gens, &webhookGen)
	}

	// set the directory
	b := bufferedGenerator{}
	rt, _ := gens.ForRoots(cgr.Spec.CRDs.SourceDirectory) // ignore the spurious error
	rt.OutputRules = genall.OutputRules{Default: &b}

	// run the generators
	if failed := rt.Run(); failed {
		fmt.Fprintln(os.Stderr, "error running controller-gen")
	}

	// Parse the emitted resources
	n, err := (&kio.ByteReader{Reader: &b.Buffer}).Read()
	if err != nil {
		return nil, errors.WrapPrefixf(err, "failed to parse controller-gen output")
	}

	// add inputs after generated resources
	return append(n, input...), nil
}

// bufferedGenerator implements a genall.Generator store the output in a bytes.Buffer
type bufferedGenerator struct {
	bytes.Buffer
}

// Open implements genall.Generator
func (o *bufferedGenerator) Open(_ *loader.Package, _ string) (io.WriteCloser, error) {
	return o, nil
}

// Close implements genall.Generator
func (bufferedGenerator) Close() error {
	return nil
}
