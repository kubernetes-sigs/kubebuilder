/*
Copyright 2025 The Kubernetes Authors.

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

package e2e

import (
	"path/filepath"

	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
)

var _ machinery.Inserter = &SuiteUpdater{}

// SuiteUpdater updates e2e_suite_test.go to insert setup code when APIs are added
type SuiteUpdater struct {
	machinery.RepositoryMixin
	machinery.ProjectNameMixin
	machinery.ResourceMixin

	// WireController indicates whether to inject controller setup
	WireController bool
}

// GetPath implements file.Builder
func (*SuiteUpdater) GetPath() string {
	return filepath.Join("test", "e2e", "e2e_suite_test.go")
}

// GetIfExistsAction implements file.Builder
func (*SuiteUpdater) GetIfExistsAction() machinery.IfExistsAction {
	return machinery.OverwriteFile
}

// GetMarkers implements file.Inserter
func (f *SuiteUpdater) GetMarkers() []machinery.Marker {
	return []machinery.Marker{
		machinery.NewMarkerFor(f.GetPath(), suiteSetupMarker),
	}
}

// GetCodeFragments implements file.Inserter
func (f *SuiteUpdater) GetCodeFragments() machinery.CodeFragmentsMap {
	if !f.WireController {
		return nil
	}

	fragments := make(machinery.CodeFragmentsMap, 1)
	
	// Add the Docker build and load code when controllers are added
	fragments[machinery.NewMarkerFor(f.GetPath(), suiteSetupMarker)] = []string{dockerSetupCode}
	
	return fragments
}

const dockerSetupCode = `
	// Import needed for Docker build - add "os/exec" to imports if not present
	
	By("building the manager(Operator) image")
	cmd := exec.Command("make", "docker-build", fmt.Sprintf("IMG=%s", projectImage))
	_, err := utils.Run(cmd)
	ExpectWithOffset(1, err).NotTo(HaveOccurred(), "Failed to build the manager(Operator) image")

	// TODO(user): If you want to change the e2e test vendor from Kind, ensure the image is
	// built and available before running the tests. Also, remove the following block.
	By("loading the manager(Operator) image on Kind")
	err = utils.LoadImageToKindClusterWithName(projectImage)
	ExpectWithOffset(1, err).NotTo(HaveOccurred(), "Failed to load the manager(Operator) image into Kind")
`