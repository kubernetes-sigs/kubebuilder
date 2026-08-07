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

package appliers

import (
	"strings"
	"testing"
)

func TestTemplateWebhookConfiguration(t *testing.T) {
	content := `apiVersion: admissionregistration.k8s.io/v1
kind: ValidatingWebhookConfiguration
webhooks:
- name: validation.example.com
  rules:
  - apiGroups:
    - example.com
`

	result := TemplateWebhookConfiguration(content)
	for _, want := range []string{
		".Values.webhook.namespaceSelector",
		".Values.webhook.objectSelector",
		".Values.webhook.matchConditions",
		".Values.webhook.timeoutSeconds",
	} {
		if !strings.Contains(result, want) {
			t.Errorf("expected generated template to contain %q", want)
		}
	}

	configured := `apiVersion: admissionregistration.k8s.io/v1
kind: ValidatingWebhookConfiguration
webhooks:
- name: validation.example.com
  namespaceSelector:
    matchLabels:
      webhook: enabled
  rules:
  - apiGroups:
    - example.com
`
	configuredResult := TemplateWebhookConfiguration(configured)
	if strings.Count(configuredResult, "  namespaceSelector:") != 1 {
		t.Fatalf("expected an existing namespaceSelector to be preserved without duplication:\n%s", configuredResult)
	}
}
