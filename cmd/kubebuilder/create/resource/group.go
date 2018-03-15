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

package resource

import (
	"log"
	"os"
	"path/filepath"

	createutil "github.com/kubernetes-sigs/kubebuilder/cmd/kubebuilder/create/util"
	"github.com/kubernetes-sigs/kubebuilder/cmd/kubebuilder/util"
)

func createGroup(boilerplate string) {
	dir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	a := groupTemplateArgs{
		boilerplate,
		util.Domain,
		createutil.GroupName,
	}

	path := filepath.Join(dir, "pkg", "apis", createutil.GroupName, "doc.go")
	util.WriteIfNotFound(path, "group-template", groupTemplate, a)

	//path = filepath.Join(dir, "pkg", "apis", createutil.GroupName, "install", "doc.go")
	//util.WriteIfNotFound(path, "install-template", installTemplate, a)
}

type groupTemplateArgs struct {
	BoilerPlate string
	Domain      string
	Name        string
}

var groupTemplate = `
{{.BoilerPlate}}


// +k8s:deepcopy-gen=package,register
// +groupName={{.Name}}.{{.Domain}}

// Package api is the internal version of the API.
package {{.Name}}
`

var installTemplate = `
{{.BoilerPlate}}

package install
`
