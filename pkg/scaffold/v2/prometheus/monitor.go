package prometheus

import (
	"path/filepath"
	"sigs.k8s.io/kubebuilder/pkg/scaffold/input"
)

// PrometheusMetricsService scaffolds an issuer CR and a certificate CR
type PrometheusServiceMonitor struct {
	input.Input
}

// GetInput implements input.File
func (p *PrometheusServiceMonitor) GetInput() (input.Input, error) {
	if p.Path == "" {
		p.Path = filepath.Join("config", "prometheus", "monitor.yaml")
	}
	p.TemplateBody = monitorTemplate
	return p.Input, nil
}

var monitorTemplate = `
# Prometheus Monitor Service (Metrics)
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  labels:
    control-plane: controller-manager
  name: controller-manager-metrics-monitor
  namespace: system
spec:
  endpoints:
    - path: /metrics
      port: https
  selector:
    control-plane: controller-manager
`
