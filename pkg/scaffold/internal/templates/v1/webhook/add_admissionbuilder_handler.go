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
	"path/filepath"
	"strings"

	"sigs.k8s.io/kubebuilder/pkg/model/file"
)

var _ file.Template = &AddAdmissionWebhookBuilderHandler{}

// AddAdmissionWebhookBuilderHandler scaffolds adds a new admission webhook builder.
type AddAdmissionWebhookBuilderHandler struct {
	file.TemplateMixin
	file.RepositoryMixin
	file.BoilerplateMixin
	file.ResourceMixin

	Config
}

// SetTemplateDefaults implements input.Template
func (f *AddAdmissionWebhookBuilderHandler) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = filepath.Join("pkg", "webhook", f.Server+"_server", "add_"+f.Type+"_%[kind].go")
	}
	f.Path = f.Resource.Replacer().Replace(f.Path)

	f.TemplateBody = addAdmissionWebhookBuilderHandlerTemplate

	f.Server = strings.ToLower(f.Server)

	return nil
}

const addAdmissionWebhookBuilderHandlerTemplate = `{{ .Boilerplate }}

package {{ .Server }}server

import (
	"fmt"
	"{{ .Repo }}/pkg/webhook/{{ .Server }}_server/{{ lower .Resource.Kind }}/{{ .Type }}"
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
