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
	"fmt"

	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
	"sigs.k8s.io/kubebuilder/v4/pkg/model/resource"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugin"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/common/kustomize/v2/scaffolds"
)

var _ plugin.CreateWebhookSubcommand = &createWebhookSubcommand{}

type createWebhookSubcommand struct {
	createSubcommand
}

func (p *createWebhookSubcommand) InjectResource(res *resource.Resource) error {
	// Multi-GVK webhook: populate Webhooks from the Webhooks field.
	// Mark as External to skip CRD-related scaffolding (no GVK/plural).
	if res.Webhooks != nil && res.Webhooks.IsMultiGVK() {
		wh := res.Webhooks.Copy()
		p.resource = &resource.Resource{
			External: true,
			Webhooks: &wh,
		}
		return nil
	}

	// GVK-based webhook: store a pointer to the shared resource so that
	// changes made by other plugins during their InjectResource phase (e.g.,
	// the Go plugin looking up an irregular Plural from the PROJECT file)
	// are visible at Scaffold time. A deep copy is deferred to Scaffold
	// after all InjectResource hooks have completed.
	p.resource = res
	return nil
}

func (p *createWebhookSubcommand) Scaffold(fs machinery.Filesystem) error {
	if p.resource == nil {
		// No resource to scaffold webhooks for; this should not happen
		// but guard against it.
		return nil
	}

	scaffolder := scaffolds.NewWebhookScaffolder(p.config, *p.resource, p.force)
	scaffolder.InjectFS(fs)
	if err := scaffolder.Scaffold(); err != nil {
		return fmt.Errorf("failed to scaffold webhook subcommand: %w", err)
	}

	return nil
}
