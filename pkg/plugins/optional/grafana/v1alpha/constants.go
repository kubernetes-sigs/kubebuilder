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

package v1alpha

//nolint:lll
const metaDataDescription = `This command will add Grafana manifests to the project:
  - A JSON file includes dashboard manifest that can be directly copied to Grafana Web UI.
	('grafana/controller-runtime-metrics.json')

NOTE: This plugin requires:
- Access to Prometheus
- Your project must be using controller-runtime to expose the metrics via the controller metrics and they need to be collected by Prometheus.
- Access to Grafana (https://grafana.com/docs/grafana/latest/setup-grafana/installation/)
Check how to enable the metrics for your project by looking at the doc: https://book.kubebuilder.io/reference/metrics.html
`
