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

package controllergen

import (
	"github.com/kubernetes-sigs/kubebuilder/cmd/internal/codegen"
	"k8s.io/gengo/generator"
)

type Generator struct{}

// GenerateInject returns a Generator for the controller package e.g. pkg/controller
func (g *Generator) GenerateInject(controllers []codegen.Controller, apis *codegen.APIs, filename string) generator.Generator {
	return &injectGenerator{
		generator.DefaultGen{OptionalName: filename},
		controllers,
		apis,
	}
}
