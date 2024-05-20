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

package templates

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strings"
	"text/template"

	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
)

type CustomMetricsConfig struct {
	CustomMetrics []CustomMetricItem `json:"customMetrics"`
}

type CustomMetricItem struct {
	Metric string `json:"metric"`
	Type   string `json:"type"`
	Expr   string `json:"expr,omitempty"`
	Unit   string `json:"unit,omitempty"`
}

var _ machinery.Template = &CustomMetricsDashManifest{}

// Kustomization scaffolds a file that defines the kustomization scheme for the prometheus folder
type CustomMetricsDashManifest struct {
	machinery.TemplateMixin

	Items []CustomMetricItem
}

// SetTemplateDefaults implements file.Template
func (f *CustomMetricsDashManifest) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = filepath.Join("grafana", "custom-metrics", "custom-metrics-dashboard.json")
	}

	defaultTemplate, err := f.createTemplate()
	if err != nil {
		return err
	}

	f.TemplateBody = defaultTemplate

	f.IfExistsAction = machinery.OverwriteFile

	return nil
}

var fns = template.FuncMap{
	"plus1": func(x int) int {
		return x + 1
	},
	"hasSuffix": strings.HasSuffix,
}

func (f *CustomMetricsDashManifest) createTemplate() (string, error) {
	t := template.Must(template.New("customMetricsDashTemplate").Funcs(fns).Parse(customMetricsDashTemplate))

	outputTmpl := &bytes.Buffer{}
	if err := t.Execute(outputTmpl, f.Items); err != nil {
		return "", fmt.Errorf("error when generating manifest from config: %w", err)
	}

	return outputTmpl.String(), nil
}

// nolint: lll
const customMetricsDashTemplate = `{
  "__inputs": [
    {
      "name": "DS_PROMETHEUS",
      "label": "Prometheus",
      "description": "",
      "type": "datasource",
      "pluginId": "prometheus",
      "pluginName": "Prometheus"
    }
  ],
  "__requires": [
    {
      "type": "datasource",
      "id": "prometheus",
      "name": "Prometheus",
      "version": "1.0.0"
    }
  ],
  "annotations": {
    "list": [
      {
        "builtIn": 1,
        "datasource": "-- Grafana --",
        "enable": true,
        "hide": true,
        "iconColor": "rgba(0, 211, 255, 1)",
        "name": "Annotations & Alerts",
        "target": {
          "limit": 100,
          "matchAny": false,
          "tags": [],
          "type": "dashboard"
        },
        "type": "dashboard"
      }
    ]
  },
  "editable": true,
  "fiscalYearStartMonth": 0,
  "graphTooltip": 0,
  "links": [],
  "liveNow": false,
  "panels": [{{ $n := len . }}{{ range $i, $e := . }}
    {
      "datasource": "${DS_PROMETHEUS}",
      "fieldConfig": {
        "defaults": {
          "color": {
            "mode": "continuous-GrYlRd"
          },
          "custom": {
            "axisLabel": "",
            "axisPlacement": "auto",
            "barAlignment": 0,
            "drawStyle": "line",
            "fillOpacity": 20,
            "gradientMode": "scheme",
            "hideFrom": {
              "legend": false,
              "tooltip": false,
              "viz": false
            },
            "lineInterpolation": "smooth",
            "lineWidth": 3,
            "pointSize": 5,
            "scaleDistribution": {
              "type": "linear"
            },
            "showPoints": "auto",
            "spanNulls": false,
            "stacking": {
              "group": "A",
              "mode": "none"
            },
            "thresholdsStyle": {
              "mode": "off"
            }
          },
          "mappings": [],
          "thresholds": {
            "mode": "absolute",
            "steps": [
              {
                "color": "green",
                "value": null
              },
              {
                "color": "red",
                "value": 80
              }
            ]
          },
          "unit": "{{ .Unit }}"
        },
        "overrides": []
      },
      "gridPos": {
        "h": 7,
        "w": 24
      },
      "interval": "1m",
      "links": [],
      "options": {
        "legend": {
          "calcs": [],
          "displayMode": "list",
          "placement": "bottom"
        },
        "tooltip": {
          "mode": "single",
          "sort": "none"
        }
      },
      "pluginVersion": "8.4.3",
      "targets": [
        {
          "datasource": "${DS_PROMETHEUS}",
          "exemplar": true,
          "expr": "{{ .Expr }}",
          "format": "time_series",
          "interval": "",
          "intervalFactor": 2,
          "refId": "A",
          "step": 10
        }
      ],
      "title": "{{ .Metric }} ({{ .Type }})",
{{- if hasSuffix .Metric "_info" }}
      "transformations": [
        {
          "id": "labelsToFields",
          "options": {
            "mode": "rows"
          }
        }
      ],
      "type": "table"
{{- else }}
      "type": "timeseries"
{{- end }}
    }{{ if ne (plus1 $i) $n }},
		{{end}}{{end}}
  ],
  "refresh": "",
  "style": "dark",
  "tags": [],
  "templating": {
    "list": [
      {
        "datasource": "${DS_PROMETHEUS}",
        "definition": "label_values(controller_runtime_reconcile_total{namespace=~\"$namespace\"}, job)",
        "hide": 0,
        "includeAll": false,
        "multi": false,
        "name": "job",
        "options": [],
        "query": {
          "query": "label_values(controller_runtime_reconcile_total{namespace=~\"$namespace\"}, job)",
          "refId": "StandardVariableQuery"
        },
        "refresh": 2,
        "regex": "",
        "skipUrlSync": false,
        "sort": 0,
        "type": "query"
      },
      {
        "current": {
          "selected": false,
          "text": "observability",
          "value": "observability"
        },
        "datasource": "${DS_PROMETHEUS}",
        "definition": "label_values(controller_runtime_reconcile_total, namespace)",
        "hide": 0,
        "includeAll": false,
        "multi": false,
        "name": "namespace",
        "options": [],
        "query": {
          "query": "label_values(controller_runtime_reconcile_total, namespace)",
          "refId": "StandardVariableQuery"
        },
        "refresh": 1,
        "regex": "",
        "skipUrlSync": false,
        "sort": 0,
        "type": "query"
      },
      {
        "current": {
          "selected": false,
          "text": "All",
          "value": "$__all"
        },
        "datasource": "${DS_PROMETHEUS}",
        "definition": "label_values(controller_runtime_reconcile_total{namespace=~\"$namespace\", job=~\"$job\"}, pod)",
        "hide": 2,
        "includeAll": true,
        "label": "pod",
        "multi": true,
        "name": "pod",
        "options": [],
        "query": {
          "query": "label_values(controller_runtime_reconcile_total{namespace=~\"$namespace\", job=~\"$job\"}, pod)",
          "refId": "StandardVariableQuery"
        },
        "refresh": 2,
        "regex": "",
        "skipUrlSync": false,
        "sort": 0,
        "type": "query"
      }
    ]
  },
  "time": {
    "from": "now-15m",
    "to": "now"
  },
  "timepicker": {},
  "timezone": "",
  "title": "Custom-Metrics",
  "weekStart": ""
}
`
