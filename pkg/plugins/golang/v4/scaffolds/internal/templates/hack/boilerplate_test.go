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

package hack_test

import (
	"strings"
	"testing"

	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/golang/v4/scaffolds/internal/templates/hack"
)

func TestBoilerplateTemplateContainsYEAR(t *testing.T) {
	bp := &hack.Boilerplate{
		Owner:   "The Kubernetes Authors",
		License: "apache2",
	}
	if err := bp.SetTemplateDefaults(); err != nil {
		t.Fatalf("SetTemplateDefaults() error: %v", err)
	}
	body := bp.GetBody()
	if !strings.Contains(body, "YEAR") {
		t.Errorf("boilerplate template body must contain literal YEAR token; got:\n%s", body)
	}
}
