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
package scaffolds

import (
	"sigs.k8s.io/kubebuilder/v4/pkg/plugin"
	"sigs.k8s.io/kubebuilder/v4/pkg/plugin/external"
)

// DeleteMeta provides help text for the delete subcommand.
var DeleteMeta = plugin.SubcommandMetadata{
	Description: "Remove Prometheus instance files added by this plugin",
	Examples:    "kubebuilder delete --plugins sampleexternalplugin/v1",
}

// DeleteCmd handles the "delete" subcommand.
//
// EXTERNAL PLUGIN FLOW FOR DELETE:
//  1. User runs: kubebuilder delete --plugins sampleexternalplugin/v1
//  2. Kubebuilder checks that the plugin struct has PSupportsDelete:true
//  3. Kubebuilder calls plugin.DeleteFeatures(cfg, fs), which sends Command:"delete"
//  4. Plugin reads PluginRequest and returns a Universe with files set to "" to signal removal
//  5. Kubebuilder removes the files that the Universe maps to empty string
//
// NOTE: To signal that a file should be deleted, set its value to "" in the Universe map.
// Files NOT present in the Universe are left unchanged.
func DeleteCmd(pr *external.PluginRequest) external.PluginResponse {
	pluginResponse := external.PluginResponse{
		APIVersion: "v1alpha1",
		Command:    "delete",
		Universe:   pr.Universe,
	}

	// Mark plugin-owned files for deletion by setting them to empty string.
	// Kubebuilder will remove files whose Universe value is "".
	filesToDelete := []string{
		"config/prometheus/monitor.yaml",
		"config/prometheus/kustomization.yaml",
	}

	for _, f := range filesToDelete {
		pluginResponse.Universe[f] = ""
	}

	return pluginResponse
}
