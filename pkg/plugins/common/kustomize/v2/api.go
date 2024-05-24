/*
Copyright 2022 The Kubernetes Authors.

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

package v2

import (
	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugin"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/common/kustomize/v2/scaffolds"
)

var _ plugin.CreateAPISubcommand = &createAPISubcommand{}

type createAPISubcommand struct {
	createSubcommand
}

func (p *createAPISubcommand) Scaffold(fs machinery.Filesystem) error {
	if err := p.configure(); err != nil {
		return err
	}
	scaffolder := scaffolds.NewAPIScaffolder(p.config, *p.resource, p.force)
	scaffolder.InjectFS(fs)
	return scaffolder.Scaffold()
}
