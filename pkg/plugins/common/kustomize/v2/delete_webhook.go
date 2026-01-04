/*
Copyright 2026 The Kubernetes Authors.

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
	"sigs.k8s.io/kubebuilder/v4/pkg/plugin"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/common/kustomize/v2/scaffolds"
)

var _ plugin.DeleteWebhookSubcommand = &deleteWebhookSubcommand{}

type deleteWebhookSubcommand struct {
	createSubcommand
	storedFS machinery.Filesystem
}

func (p *deleteWebhookSubcommand) Scaffold(fs machinery.Filesystem) error {
	// Store filesystem for PostScaffold use
	p.storedFS = fs
	// Do nothing during Scaffold phase
	// Cleanup happens in PostScaffold after golang plugin updates PROJECT
	return nil
}

// PostScaffold performs kustomize cleanup after golang plugin updates config
func (p *deleteWebhookSubcommand) PostScaffold() error {
	if err := p.configure(); err != nil {
		return err
	}

	scaffolder := scaffolds.NewDeleteWebhookScaffolder(p.config, *p.resource)
	scaffolder.InjectFS(p.storedFS)

	// Now execute the actual cleanup (after golang/v4 updated config)
	if err := scaffolder.Scaffold(); err != nil {
		return fmt.Errorf("failed to clean up kustomize webhook files: %w", err)
	}

	// Also run PostScaffold for final steps
	if ps, ok := scaffolder.(interface{ PostScaffold() error }); ok {
		if err := ps.PostScaffold(); err != nil {
			return fmt.Errorf("failed to run post-scaffold cleanup: %w", err)
		}
	}

	return nil
}
