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
	"path/filepath"

	"sigs.k8s.io/kubebuilder/v4/pkg/machinery"
)

var _ machinery.Template = &RuntimeManifest{}

// RuntimeManifest scaffolds a file that defines the kustomization scheme for the prometheus folder
type RuntimeManifest struct {
	machinery.TemplateMixin
}

// SetTemplateDefaults implements machinery.Template
func (f *RuntimeManifest) SetTemplateDefaults() error {
	if f.Path == "" {
		f.Path = filepath.Join("grafana", "controller-runtime-metrics.json")
	}

	// Grafana syntax use {{ }} quite often, which is collided with default delimiter for go template parsing.
	// Provide an alternative delimiter here to avoid overlaps.
	f.SetDelim("[[", "]]")
	f.TemplateBody = controllerRuntimeTemplate
	f.IfExistsAction = machinery.OverwriteFile

	return nil
}

//nolint:lll
const controllerRuntimeTemplate = `{
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
        "datasource": {
          "type": "datasource",
          "uid": "grafana"
        },
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
  "panels": [
    {
      "collapsed": false,
      "gridPos": {
        "h": 1,
        "w": 24,
        "x": 0,
        "y": 0
      },
      "id": 9,
      "panels": [],
      "title": "Reconciliation Metrics",
      "type": "row"
    },
    {
      "datasource": "${DS_PROMETHEUS}",
      "fieldConfig": {
        "defaults": {
          "mappings": [],
          "thresholds": {
            "mode": "percentage",
            "steps": [
              {
                "color": "green",
                "value": null
              },
              {
                "color": "orange",
                "value": 70
              },
              {
                "color": "red",
                "value": 85
              }
            ]
          }
        },
        "overrides": []
      },
      "gridPos": {
        "h": 8,
        "w": 3,
        "x": 0,
        "y": 1
      },
      "id": 24,
      "options": {
        "orientation": "auto",
        "reduceOptions": {
          "calcs": ["lastNotNull"],
          "fields": "",
          "values": false
        },
        "showThresholdLabels": false,
        "showThresholdMarkers": true
      },
      "pluginVersion": "9.5.3",
      "targets": [
        {
          "datasource": "${DS_PROMETHEUS}",
          "exemplar": true,
          "expr": "controller_runtime_active_workers{job=\"$job\", namespace=\"$namespace\"}",
          "interval": "",
          "legendFormat": "{{controller}} {{instance}}",
          "refId": "A"
        }
      ],
      "title": "Number of workers in use",
      "type": "gauge"
    },
    {
      "datasource": "${DS_PROMETHEUS}",
      "description": "Total number of reconciliations per controller",
      "fieldConfig": {
        "defaults": {
          "color": {
            "mode": "continuous-GrYlRd"
          },
          "custom": {
            "axisCenteredZero": false,
            "axisColorMode": "text",
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
          "unit": "cpm"
        },
        "overrides": []
      },
      "gridPos": {
        "h": 8,
        "w": 11,
        "x": 3,
        "y": 1
      },
      "id": 7,
      "options": {
        "legend": {
          "calcs": [],
          "displayMode": "table",
          "placement": "bottom",
          "showLegend": true
        },
        "tooltip": {
          "mode": "single",
          "sort": "none"
        }
      },
      "targets": [
        {
          "datasource": "${DS_PROMETHEUS}",
          "editorMode": "code",
          "exemplar": true,
          "expr": "sum(rate(controller_runtime_reconcile_total{job=\"$job\", namespace=\"$namespace\"}[5m])) by (instance, pod)",
          "interval": "",
          "legendFormat": "{{instance}} {{pod}}",
          "range": true,
          "refId": "A"
        }
      ],
      "title": "Total Reconciliation Count Per Controller",
      "type": "timeseries"
    },
    {
      "datasource": "${DS_PROMETHEUS}",
      "description": "Total number of reconciliation errors per controller",
      "fieldConfig": {
        "defaults": {
          "color": {
            "mode": "continuous-GrYlRd"
          },
          "custom": {
            "axisCenteredZero": false,
            "axisColorMode": "text",
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
          "unit": "cpm"
        },
        "overrides": []
      },
      "gridPos": {
        "h": 8,
        "w": 10,
        "x": 14,
        "y": 1
      },
      "id": 6,
      "options": {
        "legend": {
          "calcs": [],
          "displayMode": "table",
          "placement": "bottom",
          "showLegend": true
        },
        "tooltip": {
          "mode": "single",
          "sort": "none"
        }
      },
      "targets": [
        {
          "datasource": "${DS_PROMETHEUS}",
          "editorMode": "code",
          "exemplar": true,
          "expr": "sum(rate(controller_runtime_reconcile_errors_total{job=\"$job\", namespace=\"$namespace\"}[5m])) by (instance, pod)",
          "interval": "",
          "legendFormat": "{{instance}} {{pod}}",
          "range": true,
          "refId": "A"
        }
      ],
      "title": "Reconciliation Error Count Per Controller",
      "type": "timeseries"
    },
    {
      "collapsed": false,
      "gridPos": {
        "h": 1,
        "w": 24,
        "x": 0,
        "y": 9
      },
      "id": 11,
      "panels": [],
      "title": "Work Queue Metrics",
      "type": "row"
    },
    {
      "datasource": "${DS_PROMETHEUS}",
      "fieldConfig": {
        "defaults": {
          "mappings": [],
          "thresholds": {
            "mode": "percentage",
            "steps": [
              {
                "color": "green",
                "value": null
              },
              {
                "color": "orange",
                "value": 70
              },
              {
                "color": "red",
                "value": 85
              }
            ]
          }
        },
        "overrides": []
      },
      "gridPos": {
        "h": 8,
        "w": 3,
        "x": 0,
        "y": 10
      },
      "id": 22,
      "options": {
        "orientation": "auto",
        "reduceOptions": {
          "calcs": ["lastNotNull"],
          "fields": "",
          "values": false
        },
        "showThresholdLabels": false,
        "showThresholdMarkers": true
      },
      "pluginVersion": "9.5.3",
      "targets": [
        {
          "datasource": "${DS_PROMETHEUS}",
          "exemplar": true,
          "expr": "workqueue_depth{job=\"$job\", namespace=\"$namespace\"}",
          "interval": "",
          "legendFormat": "",
          "refId": "A"
        }
      ],
      "title": "WorkQueue Depth",
      "type": "gauge"
    },
    {
      "datasource": "${DS_PROMETHEUS}",
      "description": "How long in seconds an item stays in workqueue before being requested",
      "fieldConfig": {
        "defaults": {
          "color": {
            "mode": "palette-classic"
          },
          "custom": {
            "axisCenteredZero": false,
            "axisColorMode": "text",
            "axisLabel": "",
            "axisPlacement": "auto",
            "barAlignment": 0,
            "drawStyle": "line",
            "fillOpacity": 10,
            "gradientMode": "none",
            "hideFrom": {
              "legend": false,
              "tooltip": false,
              "viz": false
            },
            "lineInterpolation": "linear",
            "lineWidth": 1,
            "pointSize": 5,
            "scaleDistribution": {
              "type": "linear"
            },
            "showPoints": "auto",
            "spanNulls": false,
            "stacking": {
              "group": "A",
              "mode": "normal"
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
          "unit": "s"
        },
        "overrides": []
      },
      "gridPos": {
        "h": 8,
        "w": 11,
        "x": 3,
        "y": 10
      },
      "id": 13,
      "options": {
        "legend": {
          "calcs": [
            "max",
            "mean"
          ],
          "displayMode": "table",
          "placement": "bottom",
          "showLegend": true
        },
        "tooltip": {
          "mode": "single",
          "sort": "none"
        }
      },
      "targets": [
        {
          "datasource": "${DS_PROMETHEUS}",
          "exemplar": true,
          "expr": "histogram_quantile(0.50, sum(rate(workqueue_queue_duration_seconds_bucket{job=\"$job\", namespace=\"$namespace\"}[5m])) by (instance, name, le))",
          "interval": "",
          "legendFormat": "P50 {{name}} {{instance}} ",
          "refId": "A"
        },
        {
          "datasource": "${DS_PROMETHEUS}",
          "exemplar": true,
          "expr": "histogram_quantile(0.90, sum(rate(workqueue_queue_duration_seconds_bucket{job=\"$job\", namespace=\"$namespace\"}[5m])) by (instance, name, le))",
          "hide": false,
          "interval": "",
          "legendFormat": "P90 {{name}} {{instance}} ",
          "refId": "B"
        },
        {
          "datasource": "${DS_PROMETHEUS}",
          "exemplar": true,
          "expr": "histogram_quantile(0.99, sum(rate(workqueue_queue_duration_seconds_bucket{job=\"$job\", namespace=\"$namespace\"}[5m])) by (instance, name, le))",
          "hide": false,
          "interval": "",
          "legendFormat": "P99 {{name}} {{instance}} ",
          "refId": "C"
        }
      ],
      "title": "Seconds For Items Stay In Queue (before being requested) (P50, P90, P99)",
      "type": "timeseries"
    },
    {
      "datasource": "${DS_PROMETHEUS}",
      "fieldConfig": {
        "defaults": {
          "color": {
            "mode": "continuous-GrYlRd"
          },
          "custom": {
            "axisCenteredZero": false,
            "axisColorMode": "text",
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
          "unit": "ops"
        },
        "overrides": []
      },
      "gridPos": {
        "h": 8,
        "w": 10,
        "x": 14,
        "y": 10
      },
      "id": 15,
      "options": {
        "legend": {
          "calcs": [],
          "displayMode": "table",
          "placement": "bottom",
          "showLegend": true
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
          "expr": "sum(rate(workqueue_adds_total{job=\"$job\", namespace=\"$namespace\"}[5m])) by (instance, name)",
          "interval": "",
          "legendFormat": "{{name}} {{instance}}",
          "refId": "A"
        }
      ],
      "title": "Work Queue Add Rate",
      "type": "timeseries"
    },
    {
      "datasource": "${DS_PROMETHEUS}",
      "description": "How many seconds of work has done that is in progress and hasn't been observed by work_duration.\nLarge values indicate stuck threads.\nOne can deduce the number of stuck threads by observing the rate at which this increases.",
      "fieldConfig": {
        "defaults": {
          "mappings": [],
          "thresholds": {
            "mode": "percentage",
            "steps": [
              {
                "color": "green",
                "value": null
              },
              {
                "color": "orange",
                "value": 70
              },
              {
                "color": "red",
                "value": 85
              }
            ]
          },
          "unit": "s"
        },
        "overrides": []
      },
      "gridPos": {
        "h": 9,
        "w": 3,
        "x": 0,
        "y": 18
      },
      "id": 23,
      "options": {
        "orientation": "auto",
        "reduceOptions": {
          "calcs": ["lastNotNull"],
          "fields": "",
          "values": false
        },
        "showThresholdLabels": false,
        "showThresholdMarkers": true
      },
      "pluginVersion": "9.5.3",
      "targets": [
        {
          "datasource": "${DS_PROMETHEUS}",
          "exemplar": true,
          "expr": "rate(workqueue_unfinished_work_seconds{job=\"$job\", namespace=\"$namespace\"}[5m])",
          "interval": "",
          "legendFormat": "",
          "refId": "A"
        }
      ],
      "title": "Unfinished Seconds",
      "type": "gauge"
    },
    {
      "datasource": "${DS_PROMETHEUS}",
      "description": "How long in seconds processing an item from workqueue takes.",
      "fieldConfig": {
        "defaults": {
          "color": {
            "mode": "palette-classic"
          },
          "custom": {
            "axisCenteredZero": false,
            "axisColorMode": "text",
            "axisLabel": "",
            "axisPlacement": "auto",
            "barAlignment": 0,
            "drawStyle": "line",
            "fillOpacity": 10,
            "gradientMode": "none",
            "hideFrom": {
              "legend": false,
              "tooltip": false,
              "viz": false
            },
            "lineInterpolation": "linear",
            "lineWidth": 1,
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
          "unit": "s"
        },
        "overrides": []
      },
      "gridPos": {
        "h": 9,
        "w": 11,
        "x": 3,
        "y": 18
      },
      "id": 19,
      "options": {
        "legend": {
          "calcs": [
            "max",
            "mean"
          ],
          "displayMode": "table",
          "placement": "bottom",
          "showLegend": true
        },
        "tooltip": {
          "mode": "single",
          "sort": "none"
        }
      },
      "targets": [
        {
          "datasource": "${DS_PROMETHEUS}",
          "exemplar": true,
          "expr": "histogram_quantile(0.50, sum(rate(workqueue_work_duration_seconds_bucket{job=\"$job\", namespace=\"$namespace\"}[5m])) by (instance, name, le))",
          "interval": "",
          "legendFormat": "P50 {{name}} {{instance}} ",
          "refId": "A"
        },
        {
          "datasource": "${DS_PROMETHEUS}",
          "exemplar": true,
          "expr": "histogram_quantile(0.90, sum(rate(workqueue_work_duration_seconds_bucket{job=\"$job\", namespace=\"$namespace\"}[5m])) by (instance, name, le))",
          "hide": false,
          "interval": "",
          "legendFormat": "P90 {{name}} {{instance}} ",
          "refId": "B"
        },
        {
          "datasource": "${DS_PROMETHEUS}",
          "exemplar": true,
          "expr": "histogram_quantile(0.99, sum(rate(workqueue_work_duration_seconds_bucket{job=\"$job\", namespace=\"$namespace\"}[5m])) by (instance, name, le))",
          "hide": false,
          "interval": "",
          "legendFormat": "P99 {{name}} {{instance}} ",
          "refId": "C"
        }
      ],
      "title": "Seconds Processing Items From WorkQueue (P50, P90, P99)",
      "type": "timeseries"
    },
    {
      "datasource": "${DS_PROMETHEUS}",
      "description": "Total number of retries handled by workqueue",
      "fieldConfig": {
        "defaults": {
          "color": {
            "mode": "continuous-GrYlRd"
          },
          "custom": {
            "axisCenteredZero": false,
            "axisColorMode": "text",
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
          "unit": "ops"
        },
        "overrides": []
      },
      "gridPos": {
        "h": 9,
        "w": 10,
        "x": 14,
        "y": 18
      },
      "id": 17,
      "options": {
        "legend": {
          "calcs": [],
          "displayMode": "table",
          "placement": "bottom",
          "showLegend": true
        },
        "tooltip": {
          "mode": "single",
          "sort": "none"
        }
      },
      "targets": [
        {
          "datasource": "${DS_PROMETHEUS}",
          "exemplar": true,
          "expr": "sum(rate(workqueue_retries_total{job=\"$job\", namespace=\"$namespace\"}[5m])) by (instance, name)",
          "interval": "",
          "legendFormat": "{{name}} {{instance}} ",
          "refId": "A"
        }
      ],
      "title": "Work Queue Retries Rate",
      "type": "timeseries"
    }
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
          "selected": true,
          "text": [
            "All"
          ],
          "value": [
            "$__all"
          ]
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
  "title": "Controller-Runtime-Metrics",
  "weekStart": ""
}
`
