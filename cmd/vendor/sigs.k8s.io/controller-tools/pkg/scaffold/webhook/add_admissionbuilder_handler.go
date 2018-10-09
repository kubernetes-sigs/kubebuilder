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
	"fmt"
	"path/filepath"
	"strings"

	"sigs.k8s.io/controller-tools/pkg/scaffold/input"
	"sigs.k8s.io/controller-tools/pkg/scaffold/resource"
)

var _ input.File = &AddAdmissionWebhookBuilderHandler{}

// AddAdmissionWebhookBuilderHandler scaffolds adds a new admission webhook builder.
type AddAdmissionWebhookBuilderHandler struct {
	input.Input

	// Resource is a resource in the API group
	Resource *resource.Resource

	Config
}

// GetInput implements input.File
func (a *AddAdmissionWebhookBuilderHandler) GetInput() (input.Input, error) {
	a.Server = strings.ToLower(a.Server)
	if a.Path == "" {
		a.Path = filepath.Join("pkg", "webhook",
			fmt.Sprintf("%s_server", a.Server),
			fmt.Sprintf("add_%s_%s.go", a.Type, strings.ToLower(a.Resource.Kind)))
	}
	a.TemplateBody = addAdmissionWebhookBuilderHandlerTemplate
	return a.Input, nil
}

var addAdmissionWebhookBuilderHandlerTemplate = `{{ .Boilerplate }}

package {{ .Server }}server

import (
	"fmt"

	"{{ .Repo }}/pkg/webhook/{{ .Server }}_server/{{ .Resource.Resource }}/{{ .Type }}"
)

func init() {
	for k, v := range {{ .Type }}.Builders {
		_, found := builderMap[k]
		if found {
			log.V(1).Info(fmt.Sprintf(
				"conflicting webhook builder names in builder map: %v", k))
		}
		builderMap[k] = v
	}
	for k, v := range {{ .Type }}.HandlerMap {
		_, found := HandlerMap[k]
		if found {
			log.V(1).Info(fmt.Sprintf(
				"conflicting webhook builder names in handler map: %v", k))
		}
		_, found = builderMap[k]
		if !found {
			log.V(1).Info(fmt.Sprintf(
				"can't find webhook builder name %q in builder map", k))
			continue
		}
		HandlerMap[k] = v
	}
}
`
