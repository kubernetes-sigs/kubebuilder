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
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/optional/helm/v2alpha/internal/common"
)

// IsScaffoldedPolicyName reports whether name is the bare scaffolded NetworkPolicy name
// or the policy name prefixed by the detected project prefix.
func IsScaffoldedPolicyName(name, detectedPrefix, policy string) bool {
	return name == policy || (detectedPrefix != "" && name == detectedPrefix+"-"+policy)
}

// TemplatePorts templates port numbers for Services, Deployments, and NetworkPolicies using values.yaml.
func TemplatePorts(yamlContent string, resource *unstructured.Unstructured, detectedPrefix string) string {
	resourceName := resource.GetName()
	resourceKind := resource.GetKind()

	// Use suffix matching to avoid false positives when project name contains "webhook"
	isWebhook := (resourceKind == common.KindService && strings.HasSuffix(resourceName, "-webhook-service")) ||
		(resourceKind == common.KindNetworkPolicy &&
			IsScaffoldedPolicyName(resourceName, detectedPrefix, "allow-webhook-traffic"))

	// Use suffix matching to avoid false positives when project name contains "metrics"
	isMetrics := (resourceKind == common.KindService &&
		(strings.HasSuffix(resourceName, "-controller-manager-metrics-service") ||
			strings.HasSuffix(resourceName, "-metrics-service"))) ||
		(resourceKind == common.KindNetworkPolicy &&
			IsScaffoldedPolicyName(resourceName, detectedPrefix, "allow-metrics-traffic"))

	// For Deployments, detect webhook ports from content
	if resourceKind == common.KindDeployment {
		if strings.Contains(yamlContent, "webhook-server") || strings.Contains(yamlContent, "name: webhook") {
			isWebhook = true
		}
	}

	// Template webhook ports
	if isWebhook {
		if resourceKind == common.KindNetworkPolicy {
			return templateNetworkPolicyIngressPort(yamlContent, "{{ .Values.webhook.port }}")
		}

		// Replace containerPort for webhook-server with template (matches any numeric port).
		// kustomize sorts the keys of a port, so a hostIP or hostPort comes between containerPort
		// and name, and is kept as it is.
		if strings.Contains(yamlContent, "webhook-server") {
			yamlContent = regexp.MustCompile(
				`(?m)(\s*- )?containerPort:\s*\d+((?:\s*\n\s*host(?:IP|Port):[^\n]*)*\s*\n\s*name:\s*webhook-server)`).
				ReplaceAllString(yamlContent, "${1}containerPort: {{ .Values.webhook.port }}${2}")
			yamlContent = makeWebhookContainerPortConditional(yamlContent)
		}

		// Replace targetPort with webhook.port template (matches any numeric port)
		yamlContent = regexp.MustCompile(`(\s*)targetPort:\s*\d+`).
			ReplaceAllString(yamlContent, "${1}targetPort: {{ .Values.webhook.port }}")
	}

	// Template metrics ports
	if isMetrics {
		if resourceKind == common.KindNetworkPolicy {
			return templateNetworkPolicyIngressPort(yamlContent, "{{ .Values.metrics.port }}")
		}

		// Replace port with metrics.port template (matches any numeric port)
		yamlContent = regexp.MustCompile(`(\s*)port:\s*\d+`).
			ReplaceAllString(yamlContent, "${1}port: {{ .Values.metrics.port }}")

		// Replace targetPort with metrics.port template (matches any numeric port)
		yamlContent = regexp.MustCompile(`(\s*)targetPort:\s*\d+`).
			ReplaceAllString(yamlContent, "${1}targetPort: {{ .Values.metrics.port }}")

		// The port name follows metrics.secure so Service and ServiceMonitor agree on the scheme.
		if resource.GetKind() == common.KindService {
			yamlContent = regexp.MustCompile(`(\s*)- name:\s*https(\s+port:)`).
				ReplaceAllString(yamlContent, `${1}- name: {{ if .Values.metrics.secure }}https{{ else }}http{{ end }}${2}`)
		}
	}

	// Template port-related arguments in Deployment
	if resource.GetKind() == common.KindDeployment {
		// Replace --metrics-bind-address with templated port
		// Supports :PORT, HOST:PORT, and IPv6 [::1]:PORT formats
		yamlContent = regexp.MustCompile(`--metrics-bind-address=(\[[^\]]*\]|[^\s:]*):([0-9]+)`).
			ReplaceAllString(yamlContent, "--metrics-bind-address=$1:{{ .Values.metrics.port }}")

		// Replace --webhook-port with templated version. Port -1 is the disabled branch and stays as is.
		yamlContent = regexp.MustCompile(`--webhook-port=[1-9][0-9]*`).
			ReplaceAllString(yamlContent, "--webhook-port={{ .Values.webhook.port }}")

		yamlContent = templateHealthProbePort(yamlContent, resource)
	}

	return yamlContent
}

// makeWebhookContainerPortConditional renders the webhook-server container port only when
// webhook.enabled is true, since the manager does not listen on it otherwise. kustomize sorts
// an appProtocol before containerPort and a hostIP or hostPort before name, so the port item
// may start with appProtocol and carry host fields before its name.
func makeWebhookContainerPortConditional(yamlContent string) string {
	if regexp.MustCompile(`\{\{- if \.Values\.webhook\.enabled \}\}\n[ \t]+- containerPort:`).MatchString(yamlContent) {
		return yamlContent
	}
	portPattern := regexp.MustCompile(`(?m)^([ \t]+)- (?:appProtocol: [^\n]*\n[ \t]+)?containerPort: ` +
		`\{\{ \.Values\.webhook\.port \}\}(?:\n[ \t]+host(?:IP|Port): [^\n]*)*` +
		`\n[ \t]+name: webhook-server(?:\n[ \t]+protocol: \w+)?$`)
	return portPattern.ReplaceAllStringFunc(yamlContent, func(match string) string {
		indent, _ := LeadingWhitespace(match)
		return fmt.Sprintf("%s{{- if .Values.webhook.enabled }}\n%s\n%s{{- end }}", indent, match, indent)
	})
}

// templateNetworkPolicyIngressPort rewrites the port only inside the NetworkPolicy's
// ingress rule. The ingress block spans the lines indented deeper than the `ingress:` key;
// it ends at the next sibling key (e.g. `egress:`) or the end of the spec. Ports under an
// egress rule target other services and must be left as-is, and scoping this way also avoids
// rewriting an unrelated port that follows the ingress block.
func templateNetworkPolicyIngressPort(yamlContent, portTemplate string) string {
	portRe := regexp.MustCompile(`(\s*)port:\s*\d+`)
	lines := strings.Split(yamlContent, "\n")
	inIngress := false
	ingressIndent := 0
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		indent := len(line) - len(strings.TrimLeft(line, " "))
		switch {
		case strings.HasPrefix(trimmed, "ingress:"):
			inIngress = true
			ingressIndent = indent
		case inIngress && indent <= ingressIndent && !strings.HasPrefix(trimmed, "-"):
			// A sibling key (e.g. egress:) at or above the ingress indentation ends the block.
			// List items (- ...) at the same indentation are still part of the ingress rule.
			inIngress = false
		case inIngress:
			lines[i] = portRe.ReplaceAllString(line, "${1}port: "+portTemplate)
		}
	}
	return strings.Join(lines, "\n")
}

// defaultHealthProbePort is the port go/v4 scaffolds in cmd/main.go for the health probe server.
const defaultHealthProbePort = 8081

// healthProbePortTemplate is the port of .Values.manager.healthProbeBindAddress, or
// defaultPort when the value is empty, the port the manager listens on without the flag.
// It works for ":PORT", "HOST:PORT" and "[::1]:PORT".
func healthProbePortTemplate(defaultPort int) string {
	return fmt.Sprintf(`{{ .Values.manager.healthProbeBindAddress | default ":%d"`+
		` | toString | splitList ":" | last | int }}`, defaultPort)
}

// healthProbeEnabledCondition is false when .Values.manager.healthProbeBindAddress is "0",
// the address that turns the manager's health probe server off. toString also matches
// `--set manager.healthProbeBindAddress=0`, which Helm reads as the number 0.
const healthProbeEnabledCondition = `{{- if ne (toString .Values.manager.healthProbeBindAddress) "0" }}`

// templateHealthProbePort derives the manager's health probe ports from
// .Values.manager.healthProbeBindAddress, so they follow the address the manager binds to.
// The --health-probe-bind-address arg itself is rendered by templateControllerManagerArgs.
// The "health" containerPort takes the port of the address, and the probes that reach the
// health probe server point at that named port, or at the same port when the container does
// not declare it. The port and the probes are rendered only while the server is on.
// Only the manager container is changed, so a sidecar keeps its own ports and probes.
//
// resource is the Deployment as kustomize rendered it. When its manager does not pass
// --health-probe-bind-address, the manager listens on the default in its cmd/main.go, and
// the manifest's "health" port (or its probes) says which port that is.
func templateHealthProbePort(yamlContent string, resource *unstructured.Unstructured) string {
	start, end := FindManagerContainerRange(yamlContent)
	if start < 0 {
		return yamlContent
	}
	lines := strings.Split(yamlContent, "\n")
	manager := strings.Join(lines[start:end+1], "\n")

	healthProbe := readManagerHealthProbe(resource, GetDefaultContainerName(yamlContent))
	portTemplate := healthProbePortTemplate(healthProbe.defaultPort())

	// containerPort for the port named "health". kustomize sorts the keys of a port, so a
	// hostIP or hostPort comes between containerPort and name, and is kept as it is.
	healthContainerPort := regexp.MustCompile(
		`(?m)(\s*- )?containerPort:\s*\d+((?:\s*\n\s*host(?:IP|Port):[^\n]*)*\s*\n\s*name:\s*health\b)`)
	hasHealthPort := healthContainerPort.MatchString(manager)
	manager = healthContainerPort.
		ReplaceAllString(manager, "${1}containerPort: "+portTemplate+"${2}")

	probePort := portTemplate
	if hasHealthPort {
		probePort = "health"
	}
	manager = templateHealthProbes(manager, healthProbe.serverPort(), probePort)

	if !strings.Contains(manager, healthProbeEnabledCondition) {
		manager = makeHealthContainerPortConditional(manager, portTemplate)
		manager = makeHealthProbesConditional(manager, probePort)
	}

	result := make([]string, 0, len(lines))
	result = append(result, lines[:start]...)
	result = append(result, strings.Split(manager, "\n")...)
	result = append(result, lines[end+1:]...)
	return strings.Join(result, "\n")
}

// managerHealthProbe is what the manager container of the kustomize output says about its
// health probe server.
type managerHealthProbe struct {
	found         bool // the manager container is in the resource
	flagSet       bool // it passes --health-probe-bind-address
	flagPort      int  // the port of that flag, 0 when the flag turns the server off
	containerPort int  // its container port named "health"
	probePort     int  // the numeric httpGet port of its liveness or readiness probe
}

// defaultPort is the port the manager listens on when the chart passes no flag. Without the
// flag in the manifest, that is the default in the project's cmd/main.go, and a manifest
// that works declares it as the "health" port or probes it.
func (h managerHealthProbe) defaultPort() int {
	if h.found && !h.flagSet {
		if h.containerPort > 0 {
			return h.containerPort
		}
		if h.probePort > 0 {
			return h.probePort
		}
	}
	return defaultHealthProbePort
}

// serverPort is the port the health probe server listens on in the manifest, or 0 when it
// is not known.
func (h managerHealthProbe) serverPort() int {
	switch {
	case !h.found:
		return 0
	case h.containerPort > 0:
		return h.containerPort
	case h.flagSet:
		return h.flagPort
	default:
		return h.defaultPort()
	}
}

// readManagerHealthProbe reads the manager container named containerName from resource, or
// its first container when none has that name.
func readManagerHealthProbe(resource *unstructured.Unstructured, containerName string) managerHealthProbe {
	var health managerHealthProbe
	if resource == nil {
		return health
	}
	val, found, err := unstructured.NestedFieldNoCopy(resource.Object, podTemplateContainersPath...)
	if err != nil || !found {
		return health
	}
	containers, ok := val.([]any)
	if !ok || len(containers) == 0 {
		return health
	}
	container, _ := containers[0].(map[string]any)
	for _, c := range containers {
		if named, isMap := c.(map[string]any); isMap && named["name"] == containerName {
			container = named
			break
		}
	}
	if container == nil {
		return health
	}
	health.found = true

	const flagName = "health-probe-bind-address"
	for _, field := range []string{"command", "args"} {
		items, _, _ := unstructured.NestedStringSlice(container, field)
		for i, item := range items {
			flag := strings.TrimLeft(item, "-")
			if flag != flagName && !strings.HasPrefix(flag, flagName+"=") {
				continue
			}
			health.flagSet = true
			address, hasValue := strings.CutPrefix(flag, flagName+"=")
			if !hasValue && i+1 < len(items) {
				address = items[i+1]
			}
			health.flagPort = portOfAddress(address)
		}
	}

	ports, _, _ := unstructured.NestedFieldNoCopy(container, "ports")
	portList, _ := ports.([]any)
	for _, p := range portList {
		if port, isMap := p.(map[string]any); isMap && port["name"] == "health" {
			health.containerPort = intValue(port["containerPort"])
		}
	}
	for _, probe := range []string{"livenessProbe", "readinessProbe"} {
		if port, found, _ := unstructured.NestedFieldNoCopy(container, probe, "httpGet", "port"); found {
			if n := intValue(port); n > 0 && health.probePort == 0 {
				health.probePort = n
			}
		}
	}
	return health
}

// portOfAddress returns the port of ":PORT", "HOST:PORT" or "[::1]:PORT", or 0.
func portOfAddress(address string) int {
	i := strings.LastIndex(address, ":")
	if i < 0 {
		return 0
	}
	port, err := strconv.Atoi(address[i+1:])
	if err != nil || port <= 0 {
		return 0
	}
	return port
}

// intValue returns a port number decoded from YAML or JSON, or 0 for a named port.
func intValue(value any) int {
	switch n := value.(type) {
	case int64:
		return int(n)
	case int:
		return n
	case float64:
		return int(n)
	default:
		return 0
	}
}

// templateHealthProbes points the liveness, readiness and startup probes that reach the
// health probe server at probePort. A probe reaches it when it uses serverPort, whatever its
// path, so probes on endpoints renamed in cmd/main.go (LivenessEndpointName,
// ReadinessEndpointName) follow too. When serverPort is not known, the probes on /healthz
// and /readyz are taken.
func templateHealthProbes(manager string, serverPort int, probePort string) string {
	if serverPort <= 0 {
		return regexp.MustCompile(`(path:\s*/(?:healthz|readyz)[ \t]*\n\s*port:\s*)\d+`).
			ReplaceAllString(manager, "${1}"+probePort)
	}
	portLine := regexp.MustCompile(`^([ \t]+port:[ \t]*)` + strconv.Itoa(serverPort) + `[ \t]*$`)
	lines := strings.Split(manager, "\n")
	probeIndent := -1
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		_, indent := LeadingWhitespace(line)
		if probeIndent >= 0 && indent <= probeIndent {
			probeIndent = -1
		}
		if trimmed == "livenessProbe:" || trimmed == "readinessProbe:" || trimmed == "startupProbe:" {
			probeIndent = indent
			continue
		}
		if probeIndent >= 0 {
			lines[i] = portLine.ReplaceAllString(line, "${1}"+probePort)
		}
	}
	return strings.Join(lines, "\n")
}

// makeHealthContainerPortConditional renders the "health" container port only while the
// health probe server is on. kustomize sorts an appProtocol before containerPort, so the
// list item can start with it.
func makeHealthContainerPortConditional(yamlContent, portTemplate string) string {
	portPattern := regexp.MustCompile(`(?m)^([ \t]+)- (?:appProtocol: [^\n]*\n[ \t]+)?containerPort: ` +
		regexp.QuoteMeta(portTemplate) +
		`(?:\n[ \t]+host(?:IP|Port): [^\n]*)*\n[ \t]+name: health(?:\n[ \t]+protocol: \w+)?$`)
	return portPattern.ReplaceAllStringFunc(yamlContent, func(match string) string {
		indent, _ := LeadingWhitespace(match)
		return fmt.Sprintf("%s%s\n%s\n%s{{- end }}", indent, healthProbeEnabledCondition, match, indent)
	})
}

// makeHealthProbesConditional renders the liveness, readiness and startup probes that
// target the health probe server only while that server is on. A probe block spans the
// lines indented deeper than its key.
func makeHealthProbesConditional(yamlContent, probePort string) string {
	lines := strings.Split(yamlContent, "\n")
	result := make([]string, 0, len(lines)+6)
	for i := 0; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if trimmed != "livenessProbe:" && trimmed != "readinessProbe:" && trimmed != "startupProbe:" {
			result = append(result, lines[i])
			continue
		}
		indent, indentLen := LeadingWhitespace(lines[i])
		end := i + 1
		for end < len(lines) {
			if t := strings.TrimSpace(lines[end]); t != "" {
				if _, n := LeadingWhitespace(lines[end]); n <= indentLen {
					break
				}
			}
			end++
		}
		block := lines[i:end]
		if !strings.Contains(strings.Join(block, "\n"), "port: "+probePort) {
			result = append(result, block...)
		} else {
			result = append(result, indent+healthProbeEnabledCondition)
			result = append(result, block...)
			result = append(result, indent+"{{- end }}")
		}
		i = end - 1
	}
	return strings.Join(result, "\n")
}
