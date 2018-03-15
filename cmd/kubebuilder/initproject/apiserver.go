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

package initproject

import (
	"log"
	"os"
	"path/filepath"

	"github.com/kubernetes-sigs/kubebuilder/cmd/kubebuilder/util"
)

func runCreateApiserver(boilerplate string) {
	dir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	path := filepath.Join(dir, "cmd", "apiserver", "main.go")
	util.WriteIfNotFound(path, "apiserver-template", apiserverTemplate,
		apiserverTemplateArguments{
			util.GetDomain(),
			util.GetCopyright(boilerplate),
			util.Repo,
		})
}

type apiserverTemplateArguments struct {
	Domain      string
	BoilerPlate string
	Repo        string
}

var apiserverTemplate = `
{{.BoilerPlate}}

// Note: Ignore this (but don't delete it) if you are using CRDs.  If using
// CRDs this file is necessary to generate docs.

package main

import (
	// Make sure dep tools picks up these dependencies
	_ "k8s.io/apimachinery/pkg/apis/meta/v1"
	_ "github.com/go-openapi/loads"

	"github.com/kubernetes-sigs/kubebuilder/pkg/cmd/server"
	_ "k8s.io/client-go/plugin/pkg/client/auth" // Enable cloud provider auth

	"{{.Repo}}/pkg/apis"
	"{{.Repo}}/pkg/openapi"
)

// Extension (aggregated) apiserver main.
func main() {
	version := "v0"
	server.StartApiServer("/registry/{{ .Domain }}", apis.APIMeta.GetAllApiBuilders(), openapi.GetOpenAPIDefinitions, "Api", version)
}
`
