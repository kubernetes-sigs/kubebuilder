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

package extractor

import (
	"fmt"
	"strconv"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"sigs.k8s.io/kubebuilder/v4/pkg/plugins/optional/helm/v2alpha/internal/common"
)

// DeploymentExtractor extracts deployment configuration for values.yaml.
type DeploymentExtractor struct{}

// ValuesConfig contains configuration values extracted from the deployment.
type ValuesConfig struct {
	Manager     ManagerConfig
	WebhookPort int
	MetricsPort int
}

// ManagerConfig contains manager deployment configuration.
type ManagerConfig struct {
	Replicas                      *int // Pointer to support scale-to-zero (nil = not set, 0 = explicitly set to 0)
	Image                         ImageConfig
	Resources                     map[string]any
	NodeSelector                  map[string]any
	Tolerations                   []any
	Affinity                      map[string]any
	Args                          []any
	Env                           []any
	SecurityContext               map[string]any
	PodSecurityContext            map[string]any
	ImagePullSecrets              []any
	PriorityClassName             string
	TopologySpreadConstraints     []any
	TerminationGracePeriodSeconds *int
	Strategy                      map[string]any
	ExtraVolumes                  []any
	ExtraVolumeMounts             []any
}

// ImageConfig contains image configuration.
type ImageConfig struct {
	Repository string
	Tag        string
	PullPolicy string
}

// ExtractDeploymentConfig extracts configuration values from the deployment for values.yaml.
func (d *DeploymentExtractor) ExtractDeploymentConfig(deployment *unstructured.Unstructured) (ValuesConfig, error) {
	config := ValuesConfig{
		Manager: ManagerConfig{},
	}

	if deployment == nil {
		return config, nil
	}

	extracted := make(map[string]any)

	extractDeploymentReplicas(deployment, extracted)
	extractDeploymentStrategy(deployment, extracted)

	specMap := extractDeploymentSpec(deployment)
	if specMap != nil {
		extractPodSecurityContext(specMap, extracted)
		extractImagePullSecrets(specMap, extracted)
		extractPodNodeSelector(specMap, extracted)
		extractPodTolerations(specMap, extracted)
		extractPodAffinity(specMap, extracted)
		extractPriorityClassName(specMap, extracted)
		extractTopologySpreadConstraints(specMap, extracted)
		extractTerminationGracePeriodSeconds(specMap, extracted)

		container := findManagerContainer(deployment, specMap)
		if container == nil {
			return config, fmt.Errorf("no manager container found in Deployment %q", deployment.GetName())
		}

		extractContainerEnv(container, extracted)
		extractContainerImage(container, extracted)
		extractContainerArgs(container, extracted)
		extractContainerPorts(container, extracted)
		extractContainerResources(container, extracted)
		extractContainerSecurityContext(container, extracted)

		extractExtraVolumes(specMap, extracted)
		extractExtraVolumeMounts(container, extracted)
	}

	config.Manager = convertToManagerConfig(extracted)

	if webhookPort, ok := extracted["webhookPort"].(int); ok {
		config.WebhookPort = webhookPort
	}
	if metricsPort, ok := extracted["metricsPort"].(int); ok {
		config.MetricsPort = metricsPort
	}

	return config, nil
}

func convertToManagerConfig(configMap map[string]any) ManagerConfig {
	cfg := ManagerConfig{}

	if replicas, ok := configMap["replicas"].(int); ok {
		r := replicas
		cfg.Replicas = &r
	}
	if image, ok := configMap["image"].(map[string]any); ok {
		cfg.Image = ImageConfig{
			Repository: getStringValue(image, "repository"),
			Tag:        getStringValue(image, "tag"),
			PullPolicy: getStringValue(image, "pullPolicy"),
		}
	}
	if resources, ok := configMap["resources"].(map[string]any); ok {
		cfg.Resources = resources
	}
	if nodeSelector, ok := configMap["podNodeSelector"].(map[string]any); ok {
		cfg.NodeSelector = nodeSelector
	}
	if tolerations, ok := configMap["podTolerations"].([]any); ok {
		cfg.Tolerations = tolerations
	}
	if affinity, ok := configMap["podAffinity"].(map[string]any); ok {
		cfg.Affinity = affinity
	}
	if args, ok := configMap["args"].([]any); ok {
		cfg.Args = args
	}
	if env, ok := configMap["env"].([]any); ok {
		cfg.Env = env
	}
	if securityContext, ok := configMap["securityContext"].(map[string]any); ok {
		cfg.SecurityContext = securityContext
	}
	if podSecurityContext, ok := configMap["podSecurityContext"].(map[string]any); ok {
		cfg.PodSecurityContext = podSecurityContext
	}
	if imagePullSecrets, ok := configMap["imagePullSecrets"].([]any); ok {
		cfg.ImagePullSecrets = imagePullSecrets
	}
	if priorityClassName, ok := configMap["priorityClassName"].(string); ok {
		cfg.PriorityClassName = priorityClassName
	}
	if topologySpreadConstraints, ok := configMap["topologySpreadConstraints"].([]any); ok {
		cfg.TopologySpreadConstraints = topologySpreadConstraints
	}
	if terminationGracePeriodSeconds, ok := configMap["terminationGracePeriodSeconds"].(int); ok {
		gracePeriod := terminationGracePeriodSeconds
		cfg.TerminationGracePeriodSeconds = &gracePeriod
	}
	if strategy, ok := configMap["strategy"].(map[string]any); ok {
		cfg.Strategy = strategy
	}
	if extraVolumes, ok := configMap["extraVolumes"].([]any); ok {
		cfg.ExtraVolumes = extraVolumes
	}
	if extraVolumeMounts, ok := configMap["extraVolumeMounts"].([]any); ok {
		cfg.ExtraVolumeMounts = extraVolumeMounts
	}

	return cfg
}

func getStringValue(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func extractDeploymentSpec(deployment *unstructured.Unstructured) map[string]any {
	spec, found, err := unstructured.NestedFieldNoCopy(deployment.Object, "spec", "template", "spec")
	if !found || err != nil {
		return nil
	}

	specMap, ok := spec.(map[string]any)
	if !ok {
		return nil
	}

	return specMap
}

func extractImagePullSecrets(specMap map[string]any, config map[string]any) {
	imagePullSecrets, found, err := unstructured.NestedFieldNoCopy(specMap, "imagePullSecrets")
	if !found || err != nil {
		return
	}

	imagePullSecretsList, ok := imagePullSecrets.([]any)
	if !ok || len(imagePullSecretsList) == 0 {
		return
	}

	config["imagePullSecrets"] = imagePullSecretsList
}

func extractPodSecurityContext(specMap map[string]any, config map[string]any) {
	podSecurityContext, found, err := unstructured.NestedFieldNoCopy(specMap, "securityContext")
	if !found || err != nil {
		return
	}

	podSecMap, ok := podSecurityContext.(map[string]any)
	if !ok || len(podSecMap) == 0 {
		return
	}

	config["podSecurityContext"] = podSecurityContext
}

func extractPodNodeSelector(specMap map[string]any, config map[string]any) {
	raw, found, err := unstructured.NestedFieldNoCopy(specMap, "nodeSelector")
	if !found || err != nil {
		return
	}

	result, ok := raw.(map[string]any)
	if !ok || len(result) == 0 {
		return
	}

	config["podNodeSelector"] = result
}

func extractPodTolerations(specMap map[string]any, config map[string]any) {
	raw, found, err := unstructured.NestedFieldNoCopy(specMap, "tolerations")
	if !found || err != nil {
		return
	}

	result, ok := raw.([]any)
	if !ok || len(result) == 0 {
		return
	}

	config["podTolerations"] = result
}

func extractPodAffinity(specMap map[string]any, config map[string]any) {
	raw, found, err := unstructured.NestedFieldNoCopy(specMap, "affinity")
	if !found || err != nil {
		return
	}

	result, ok := raw.(map[string]any)
	if !ok || len(result) == 0 {
		return
	}

	config["podAffinity"] = result
}

// managerContainerName reads the default-container annotation; falls back to "manager".
func managerContainerName(deployment *unstructured.Unstructured) string {
	if deployment == nil {
		return common.DefaultManagerContainerName
	}
	annotations, found, err := unstructured.NestedStringMap(
		deployment.Object, "spec", "template", "metadata", "annotations")
	if found && err == nil {
		if name, ok := annotations[common.DefaultContainerAnnotation]; ok && name != "" {
			return name
		}
	}
	return common.DefaultManagerContainerName
}

// findManagerContainer returns the named manager container; falls back to the first container.
func findManagerContainer(deployment *unstructured.Unstructured, specMap map[string]any) map[string]any {
	containers, found, err := unstructured.NestedFieldNoCopy(specMap, "containers")
	if !found || err != nil {
		return nil
	}

	containersList, ok := containers.([]any)
	if !ok || len(containersList) == 0 {
		return nil
	}

	targetName := managerContainerName(deployment)
	for _, c := range containersList {
		container, containerOK := c.(map[string]any)
		if !containerOK {
			continue
		}
		if name, nameOK := container["name"].(string); nameOK && name == targetName {
			return container
		}
	}

	firstContainer, firstOK := containersList[0].(map[string]any)
	if !firstOK {
		return nil
	}
	return firstContainer
}

// RemoveExtractedVolumes removes custom volumes and mounts from the deployment manifest.
// Must be called after ExtractDeploymentConfig: extraction captures them in values.yaml first,
// then this removes them from the manifest so they are injected only via Helm values at install time.
func (d *DeploymentExtractor) RemoveExtractedVolumes(deployment *unstructured.Unstructured) {
	if deployment == nil {
		return
	}
	specMap := extractDeploymentSpec(deployment)
	if specMap == nil {
		return
	}

	if volumes, found, err := unstructured.NestedFieldNoCopy(specMap, "volumes"); found && err == nil {
		if volumesList, ok := volumes.([]any); ok {
			specMap["volumes"] = retainSystemEntries(volumesList)
		}
	}

	container := findManagerContainer(deployment, specMap)
	if container == nil {
		return
	}
	if mounts, found, err := unstructured.NestedFieldNoCopy(container, "volumeMounts"); found && err == nil {
		if mountsList, ok := mounts.([]any); ok {
			container["volumeMounts"] = retainSystemEntries(mountsList)
		}
	}
}

// retainSystemEntries keeps only webhook-certs and metrics-certs entries from a volume or volumeMount slice.
func retainSystemEntries(entries []any) []any {
	result := make([]any, 0, len(entries))
	for _, entry := range entries {
		entryMap, ok := entry.(map[string]any)
		if !ok {
			continue
		}
		name, _ := entryMap["name"].(string)
		if _, isSystem := defaultWebhookMetricsVolumeNames[name]; isSystem {
			result = append(result, entry)
		}
	}
	return result
}

func extractContainerEnv(container map[string]any, config map[string]any) {
	env, found, err := unstructured.NestedFieldNoCopy(container, "env")
	if !found || err != nil {
		return
	}

	envList, ok := env.([]any)
	if !ok || len(envList) == 0 {
		return
	}

	config["env"] = envList
}

func extractContainerImage(container map[string]any, config map[string]any) {
	imageValue, found, err := unstructured.NestedString(container, "image")
	if !found || err != nil || imageValue == "" {
		return
	}

	repository := imageValue
	tag := ""

	// Digest-pinned images stay whole in repository; tag-based images are split.
	if !strings.Contains(imageValue, "@") {
		tag = "latest"
		lastColon := strings.LastIndex(imageValue, ":")
		lastSlash := strings.LastIndex(imageValue, "/")
		if lastColon != -1 && lastColon > lastSlash {
			repository = imageValue[:lastColon]
			if lastColon+1 < len(imageValue) {
				tag = imageValue[lastColon+1:]
			}
		}
	}

	pullPolicy, _, err := unstructured.NestedString(container, "imagePullPolicy")
	if err != nil || pullPolicy == "" {
		pullPolicy = "IfNotPresent"
	}

	config["image"] = map[string]any{
		"repository": repository,
		"tag":        tag,
		"pullPolicy": pullPolicy,
	}
}

func extractContainerArgs(container map[string]any, config map[string]any) {
	args, found, err := unstructured.NestedFieldNoCopy(container, "args")
	if !found || err != nil {
		return
	}

	argsList, ok := args.([]any)
	if !ok || len(argsList) == 0 {
		return
	}

	filteredArgs := make([]any, 0, len(argsList))
	for _, rawArg := range argsList {
		strArg, ok := rawArg.(string)
		if !ok {
			filteredArgs = append(filteredArgs, rawArg)
			continue
		}

		// Extract port from metrics-bind-address. The arg itself is filtered out.
		if strings.Contains(strArg, "--metrics-bind-address") {
			if port := ExtractPortFromArg(strArg); port > 0 {
				if _, exists := config["metricsPort"]; !exists {
					config["metricsPort"] = port
				}
			}
			continue
		}
		if strings.Contains(strArg, "--health-probe-bind-address") {
			continue
		}
		if strings.Contains(strArg, "--webhook-cert-path") ||
			strings.Contains(strArg, "--metrics-cert-path") {
			continue
		}
		filteredArgs = append(filteredArgs, strArg)
	}

	if len(filteredArgs) > 0 {
		config["args"] = filteredArgs
	}
}

// ExtractPortFromArg extracts the port number from bind-address arguments like "--metrics-bind-address=:8443".
func ExtractPortFromArg(arg string) int {
	parts := strings.Split(arg, "=")
	if len(parts) != 2 {
		return 0
	}

	portPart := parts[1]
	if idx := strings.LastIndex(portPart, ":"); idx != -1 {
		portPart = portPart[idx+1:]
	}

	port, err := strconv.Atoi(portPart)
	if err != nil || port <= 0 || port > 65535 {
		return 0
	}
	return port
}

func extractContainerPorts(container map[string]any, config map[string]any) {
	portsField, found, err := unstructured.NestedFieldNoCopy(container, "ports")
	if !found || err != nil {
		return
	}

	ports, ok := portsField.([]any)
	if !ok {
		return
	}

	for _, p := range ports {
		portMap, ok := p.(map[string]any)
		if !ok {
			continue
		}

		name, _ := portMap["name"].(string)
		containerPort, ok := toInt(portMap["containerPort"])
		if !ok {
			continue
		}

		if isWebhookPortName(name) {
			if _, exists := config["webhookPort"]; !exists {
				config["webhookPort"] = containerPort
			}
		}
	}
}

func extractContainerResources(container map[string]any, config map[string]any) {
	resources, found, err := unstructured.NestedFieldNoCopy(container, "resources")
	if !found || err != nil {
		return
	}

	resourcesMap, ok := resources.(map[string]any)
	if !ok || len(resourcesMap) == 0 {
		return
	}

	config["resources"] = resources
}

func extractContainerSecurityContext(container map[string]any, config map[string]any) {
	securityContext, found, err := unstructured.NestedFieldNoCopy(container, "securityContext")
	if !found || err != nil {
		return
	}

	secMap, ok := securityContext.(map[string]any)
	if !ok || len(secMap) == 0 {
		return
	}

	config["securityContext"] = securityContext
}

// defaultWebhookMetricsVolumeNames are excluded from extraVolumes (managed by chart).
var defaultWebhookMetricsVolumeNames = map[string]struct{}{
	"webhook-certs": {},
	"metrics-certs": {},
}

// extractExtraEntries filters a slice of named entries (volumes or volumeMounts),
// dropping any whose "name" field is in defaultWebhookMetricsVolumeNames, and stores
// the result under configKey in config.
func extractExtraEntries(source map[string]any, fieldKey, configKey string, config map[string]any) {
	raw, found, err := unstructured.NestedFieldNoCopy(source, fieldKey)
	if !found || err != nil {
		return
	}
	entries, ok := raw.([]any)
	if !ok || len(entries) == 0 {
		return
	}
	extra := make([]any, 0, len(entries))
	for _, e := range entries {
		entry, ok := e.(map[string]any)
		if !ok {
			continue
		}
		name, _ := entry["name"].(string)
		if _, isDefault := defaultWebhookMetricsVolumeNames[name]; isDefault {
			continue
		}
		extra = append(extra, e)
	}
	if len(extra) > 0 {
		config[configKey] = extra
	}
}

// extractExtraVolumes extracts volumes, excluding webhook-certs and metrics-certs.
func extractExtraVolumes(specMap map[string]any, config map[string]any) {
	extractExtraEntries(specMap, "volumes", "extraVolumes", config)
}

// extractExtraVolumeMounts extracts volume mounts, excluding webhook-certs and metrics-certs.
func extractExtraVolumeMounts(container map[string]any, config map[string]any) {
	extractExtraEntries(container, "volumeMounts", "extraVolumeMounts", config)
}

// extractDeploymentReplicas extracts the replicas count from the deployment spec.
func extractDeploymentReplicas(deployment *unstructured.Unstructured, config map[string]any) {
	replicas, found, err := unstructured.NestedInt64(deployment.Object, "spec", "replicas")
	if !found || err != nil {
		return
	}

	config["replicas"] = int(replicas)
}

// extractDeploymentStrategy extracts the deployment strategy.
func extractDeploymentStrategy(deployment *unstructured.Unstructured, config map[string]any) {
	strategy, found, err := unstructured.NestedFieldNoCopy(deployment.Object, "spec", "strategy")
	if !found || err != nil {
		return
	}

	strategyMap, ok := strategy.(map[string]any)
	if !ok || len(strategyMap) == 0 {
		return
	}

	config["strategy"] = strategy
}

// extractPriorityClassName extracts the priorityClassName from the pod spec.
func extractPriorityClassName(specMap map[string]any, config map[string]any) {
	priorityClassName, found, err := unstructured.NestedString(specMap, "priorityClassName")
	if !found || err != nil || priorityClassName == "" {
		return
	}

	config["priorityClassName"] = priorityClassName
}

// extractTopologySpreadConstraints extracts the topologySpreadConstraints from the pod spec.
func extractTopologySpreadConstraints(specMap map[string]any, config map[string]any) {
	topologySpreadConstraints, found, err := unstructured.NestedFieldNoCopy(specMap, "topologySpreadConstraints")
	if !found || err != nil {
		return
	}

	tscList, ok := topologySpreadConstraints.([]any)
	if !ok || len(tscList) == 0 {
		return
	}

	config["topologySpreadConstraints"] = topologySpreadConstraints
}

// extractTerminationGracePeriodSeconds extracts the terminationGracePeriodSeconds from the pod spec.
func extractTerminationGracePeriodSeconds(specMap map[string]any, config map[string]any) {
	val, found := specMap["terminationGracePeriodSeconds"]
	if !found {
		return
	}

	if gracePeriod, ok := toInt(val); ok {
		config["terminationGracePeriodSeconds"] = gracePeriod
	}
}

// isWebhookPortName reports whether name identifies a webhook port.
func isWebhookPortName(name string) bool {
	name = strings.ToLower(name)
	return name == "webhook-server" || name == "webhook"
}

// FindManagerDeployment returns the controller-manager Deployment from the slice.
// With exactly one deployment, it is returned directly — no heuristics needed.
// With multiple deployments, the following signals are tried in order:
//  1. deployment label control-plane=controller-manager
//  2. pod-template annotation kubectl.kubernetes.io/default-container is non-empty
//  3. a container whose name is exactly "manager"
//  4. deployment name contains "controller-manager"
//
// Returns nil when the slice is empty or when multiple deployments are present
// but none matches any signal. Callers must treat nil as a hard error.
func FindManagerDeployment(deployments []*unstructured.Unstructured) *unstructured.Unstructured {
	if len(deployments) == 0 {
		return nil
	}
	if len(deployments) == 1 {
		return deployments[0]
	}

	for _, d := range deployments {
		if d.GetLabels()["control-plane"] == "controller-manager" {
			return d
		}
	}
	for _, d := range deployments {
		annotations, _, _ := unstructured.NestedStringMap(d.Object, "spec", "template", "metadata", "annotations")
		if annotations[common.DefaultContainerAnnotation] != "" {
			return d
		}
	}
	for _, d := range deployments {
		specMap := extractDeploymentSpec(d)
		if specMap != nil && hasContainerNamed(specMap, common.DefaultManagerContainerName) {
			return d
		}
	}
	for _, d := range deployments {
		if strings.Contains(d.GetName(), "controller-manager") {
			return d
		}
	}
	return nil
}

func hasContainerNamed(specMap map[string]any, name string) bool {
	containers, _, _ := unstructured.NestedFieldNoCopy(specMap, "containers")
	containersList, ok := containers.([]any)
	if !ok {
		return false
	}
	for _, c := range containersList {
		container, ok := c.(map[string]any)
		if !ok {
			continue
		}
		if container["name"] == name {
			return true
		}
	}
	return false
}

// toInt converts numeric types (int, int32, int64, float64) to int.
func toInt(v any) (int, bool) {
	switch n := v.(type) {
	case int:
		return n, true
	case int32:
		return int(n), true
	case int64:
		return int(n), true
	case float64:
		return int(n), true
	default:
		return 0, false
	}
}
