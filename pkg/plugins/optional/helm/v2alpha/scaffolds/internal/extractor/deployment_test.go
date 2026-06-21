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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const (
	keyName         = "name"
	keyReplicas     = "replicas"
	keySecretName   = "secretName"
	keyConfigMap    = "configMap"
	keyImage        = "image"
	keySecret       = "secret"
	keyVolumeMounts = "volumeMounts"
	keyMountPath    = "mountPath"

	valAppsV1            = "apps/v1"
	valDeployment        = "Deployment"
	valTestDeploy        = "test-deployment"
	valManager           = "manager"
	valControllerImage   = "controller:latest"
	valMgrImage          = "mgr:v1"
	valSidecarImage      = "side:latest"
	valCustomConfig      = "custom-config"
	valMountConfigDir    = "/config"
	valAppConfig         = "app-config"
	valAppSecret         = "app-secret"
	valMyConfig          = "my-config"
	valMySecret          = "my-secret"
	valWebhookCert       = "webhook-certs"
	valMetricsCert       = "metrics-certs"
	valWebhookSecretName = "webhook-server-cert"
	valSidecar           = "sidecar"
	valMountConfig       = "/etc/config"
	valMountCerts        = "/certs"
	valMountMetrics      = "/metrics"

	annotationDefaultContainer = "kubectl.kubernetes.io/default-container"
)

type deploymentOpts struct {
	containers  []map[string]any
	volumes     []any
	annotations map[string]string
}

func makeDeployment(opts deploymentOpts) *unstructured.Unstructured {
	podSpec := map[string]any{}
	if opts.containers != nil {
		cs := make([]any, len(opts.containers))
		for i, c := range opts.containers {
			cs[i] = c
		}
		podSpec["containers"] = cs
	}
	if opts.volumes != nil {
		podSpec["volumes"] = opts.volumes
	}
	template := map[string]any{"spec": podSpec}
	if opts.annotations != nil {
		annotations := make(map[string]any, len(opts.annotations))
		for k, v := range opts.annotations {
			annotations[k] = v
		}
		template["metadata"] = map[string]any{"annotations": annotations}
	}
	return &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": valAppsV1,
			"kind":       valDeployment,
			"metadata":   map[string]any{keyName: valTestDeploy},
			"spec": map[string]any{
				"template": template,
			},
		},
	}
}

func podSpec(d *unstructured.Unstructured) map[string]any {
	spec, _, _ := unstructured.NestedFieldNoCopy(d.Object, "spec", "template", "spec")
	m, _ := spec.(map[string]any)
	return m
}

var _ = Describe("DeploymentExtractor", func() {
	Describe("ExtractDeploymentConfig replicas handling", func() {
		var (
			deployment *unstructured.Unstructured
			extractor  *DeploymentExtractor
		)

		BeforeEach(func() {
			deployment = makeDeployment(deploymentOpts{
				containers: []map[string]any{{
					keyName: valManager, keyImage: valControllerImage,
				}},
			})
			extractor = &DeploymentExtractor{}
		})

		It("should return nil when replicas is not set", func() {
			result, err := extractor.ExtractDeploymentConfig(deployment)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Manager.Replicas).To(BeNil())
		})

		It("should preserve scale-to-zero", func() {
			deployment.Object["spec"].(map[string]any)[keyReplicas] = int64(0)
			result, err := extractor.ExtractDeploymentConfig(deployment)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Manager.Replicas).NotTo(BeNil())
			Expect(*result.Manager.Replicas).To(Equal(0))
		})

		It("should extract replicas value", func() {
			deployment.Object["spec"].(map[string]any)[keyReplicas] = int64(3)
			result, err := extractor.ExtractDeploymentConfig(deployment)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Manager.Replicas).NotTo(BeNil())
			Expect(*result.Manager.Replicas).To(Equal(3))
		})
	})

	Describe("findManagerContainer", func() {
		It("should return the container named 'manager' when it is first", func() {
			d := makeDeployment(deploymentOpts{
				containers: []map[string]any{
					{keyName: valManager, keyImage: valMgrImage},
				},
			})
			spec := podSpec(d)
			c := findManagerContainer(d, spec)
			Expect(c).NotTo(BeNil())
			Expect(c[keyName]).To(Equal(valManager))
		})

		It("should return the manager container even when a sidecar comes first", func() {
			d := makeDeployment(deploymentOpts{
				containers: []map[string]any{
					{keyName: valSidecar, keyImage: valSidecarImage},
					{keyName: valManager, keyImage: valMgrImage},
				},
			})
			spec := podSpec(d)
			c := findManagerContainer(d, spec)
			Expect(c).NotTo(BeNil())
			Expect(c[keyName]).To(Equal(valManager))
		})

		It("should respect the default-container annotation", func() {
			d := makeDeployment(deploymentOpts{
				annotations: map[string]string{
					annotationDefaultContainer: "custom-manager",
				},
				containers: []map[string]any{
					{keyName: valSidecar, keyImage: valSidecarImage},
					{keyName: "custom-manager", keyImage: valMgrImage},
				},
			})
			spec := podSpec(d)
			c := findManagerContainer(d, spec)
			Expect(c).NotTo(BeNil())
			Expect(c[keyName]).To(Equal("custom-manager"))
		})

		It("should fall back to the first container when no name matches", func() {
			d := makeDeployment(deploymentOpts{
				containers: []map[string]any{
					{keyName: valSidecar, keyImage: valSidecarImage},
					{keyName: "other", keyImage: "other:v1"},
				},
			})
			spec := podSpec(d)
			c := findManagerContainer(d, spec)
			Expect(c).NotTo(BeNil())
			Expect(c[keyName]).To(Equal(valSidecar))
		})
	})

	Describe("RemoveExtractedVolumes", func() {
		var ext *DeploymentExtractor

		BeforeEach(func() {
			ext = &DeploymentExtractor{}
		})

		It("should remove all custom volumes and leave an empty slice", func() {
			d := makeDeployment(deploymentOpts{
				volumes: []any{
					map[string]any{keyName: valCustomConfig, keyConfigMap: map[string]any{keyName: valMyConfig}},
				},
				containers: []map[string]any{{
					keyName: valManager,
					keyVolumeMounts: []any{
						map[string]any{keyName: valCustomConfig, keyMountPath: valMountConfigDir},
					},
				}},
			})

			ext.RemoveExtractedVolumes(d)

			spec := podSpec(d)
			volumes, _, _ := unstructured.NestedFieldNoCopy(spec, "volumes")
			Expect(volumes.([]any)).To(BeEmpty())
			containers, _, _ := unstructured.NestedFieldNoCopy(spec, "containers")
			manager := containers.([]any)[0].(map[string]any)
			mounts, _, _ := unstructured.NestedFieldNoCopy(manager, keyVolumeMounts)
			Expect(mounts.([]any)).To(BeEmpty())
		})

		It("should remove custom volumes while keeping system volumes", func() {
			d := makeDeployment(deploymentOpts{
				volumes: []any{
					map[string]any{keyName: valWebhookCert, keySecret: map[string]any{keySecretName: valWebhookSecretName}},
					map[string]any{keyName: valCustomConfig, keyConfigMap: map[string]any{keyName: valMyConfig}},
				},
				containers: []map[string]any{{
					keyName:  valManager,
					keyImage: valMgrImage,
					keyVolumeMounts: []any{
						map[string]any{keyName: valWebhookCert, keyMountPath: "/tmp/certs"},
						map[string]any{keyName: valCustomConfig, keyMountPath: valMountConfigDir},
					},
				}},
			})

			ext.RemoveExtractedVolumes(d)

			spec := podSpec(d)
			volumes, _, _ := unstructured.NestedFieldNoCopy(spec, "volumes")
			volumeList, ok := volumes.([]any)
			Expect(ok).To(BeTrue())
			Expect(volumeList).To(HaveLen(1))
			Expect(volumeList[0].(map[string]any)[keyName]).To(Equal(valWebhookCert))

			containers, _, _ := unstructured.NestedFieldNoCopy(spec, "containers")
			managerContainer := containers.([]any)[0].(map[string]any)
			mounts, _, _ := unstructured.NestedFieldNoCopy(managerContainer, keyVolumeMounts)
			mountList, ok := mounts.([]any)
			Expect(ok).To(BeTrue())
			Expect(mountList).To(HaveLen(1))
			Expect(mountList[0].(map[string]any)[keyName]).To(Equal(valWebhookCert))
		})

		It("should keep both metrics-certs and webhook-certs volumes", func() {
			d := makeDeployment(deploymentOpts{
				volumes: []any{
					map[string]any{keyName: valWebhookCert},
					map[string]any{keyName: valMetricsCert},
				},
				containers: []map[string]any{{
					keyName: valManager,
					keyVolumeMounts: []any{
						map[string]any{keyName: valWebhookCert, keyMountPath: valMountCerts},
						map[string]any{keyName: valMetricsCert, keyMountPath: valMountMetrics},
					},
				}},
			})

			ext.RemoveExtractedVolumes(d)

			spec := podSpec(d)
			volumes, _, _ := unstructured.NestedFieldNoCopy(spec, "volumes")
			Expect(volumes.([]any)).To(HaveLen(2))
		})

		It("should only strip volume mounts from the manager container, not sidecars", func() {
			d := makeDeployment(deploymentOpts{
				volumes: []any{
					map[string]any{keyName: valWebhookCert},
					map[string]any{keyName: valCustomConfig},
				},
				containers: []map[string]any{
					{
						keyName: valSidecar,
						keyVolumeMounts: []any{
							map[string]any{keyName: valCustomConfig, keyMountPath: valMountConfigDir},
						},
					},
					{
						keyName: valManager,
						keyVolumeMounts: []any{
							map[string]any{keyName: valWebhookCert, keyMountPath: "/tmp/certs"},
							map[string]any{keyName: valCustomConfig, keyMountPath: valMountConfigDir},
						},
					},
				},
			})

			ext.RemoveExtractedVolumes(d)

			spec := podSpec(d)
			containers, _, _ := unstructured.NestedFieldNoCopy(spec, "containers")
			containerList := containers.([]any)

			sidecar := containerList[0].(map[string]any)
			sidecarMounts, _, _ := unstructured.NestedFieldNoCopy(sidecar, keyVolumeMounts)
			Expect(sidecarMounts.([]any)).To(HaveLen(1), "sidecar mounts must not be touched")

			manager := containerList[1].(map[string]any)
			managerMounts, _, _ := unstructured.NestedFieldNoCopy(manager, keyVolumeMounts)
			Expect(managerMounts.([]any)).To(HaveLen(1))
			Expect(managerMounts.([]any)[0].(map[string]any)[keyName]).To(Equal(valWebhookCert))
		})

		It("should be a no-op on nil deployment", func() {
			Expect(func() { ext.RemoveExtractedVolumes(nil) }).NotTo(Panic())
		})
	})

	Describe("FindManagerDeployment", func() {
		It("should return nil for an empty or nil slice", func() {
			Expect(FindManagerDeployment(nil)).To(BeNil())
			Expect(FindManagerDeployment([]*unstructured.Unstructured{})).To(BeNil())
		})

		It("should return the only deployment with no heuristics applied", func() {
			d := makeDeployment(deploymentOpts{containers: []map[string]any{{keyName: valSidecar}}})
			Expect(FindManagerDeployment([]*unstructured.Unstructured{d})).To(Equal(d))
		})

		Context("with multiple deployments", func() {
			It("should select by label control-plane=controller-manager first", func() {
				other := makeDeployment(deploymentOpts{containers: []map[string]any{{keyName: valManager}}})
				winner := makeDeployment(deploymentOpts{containers: []map[string]any{{keyName: "other"}}})
				winner.SetName("winner")
				winner.SetLabels(map[string]string{"control-plane": "controller-manager"})

				result := FindManagerDeployment([]*unstructured.Unstructured{other, winner})
				Expect(result).To(Equal(winner))
			})

			It("should select by pod-template annotation when no label matches", func() {
				other := makeDeployment(deploymentOpts{containers: []map[string]any{{keyName: valManager}}})
				winner := makeDeployment(deploymentOpts{
					annotations: map[string]string{annotationDefaultContainer: valSidecar},
					containers:  []map[string]any{{keyName: valSidecar}},
				})
				winner.SetName("winner")

				result := FindManagerDeployment([]*unstructured.Unstructured{other, winner})
				Expect(result).To(Equal(winner))
			})

			It("should select by container named manager when no label or annotation matches", func() {
				other := makeDeployment(deploymentOpts{containers: []map[string]any{{keyName: valSidecar}}})
				winner := makeDeployment(deploymentOpts{containers: []map[string]any{{keyName: valManager}}})
				winner.SetName("winner")

				result := FindManagerDeployment([]*unstructured.Unstructured{other, winner})
				Expect(result).To(Equal(winner))
			})

			It("should select by name containing controller-manager when no other signal matches", func() {
				other := makeDeployment(deploymentOpts{containers: []map[string]any{{keyName: valSidecar}}})
				other.SetName("other")
				winner := makeDeployment(deploymentOpts{containers: []map[string]any{{keyName: valSidecar}}})
				winner.SetName("my-controller-manager")

				result := FindManagerDeployment([]*unstructured.Unstructured{other, winner})
				Expect(result).To(Equal(winner))
			})

			It("should return nil when multiple deployments have no identifying signals", func() {
				first := makeDeployment(deploymentOpts{containers: []map[string]any{{keyName: valSidecar}}})
				second := makeDeployment(deploymentOpts{containers: []map[string]any{{keyName: valSidecar}}})

				result := FindManagerDeployment([]*unstructured.Unstructured{first, second})
				Expect(result).To(BeNil())
			})
		})
	})

	Describe("ExtractPortFromArg", func() {
		DescribeTable("valid formats",
			func(arg string, expected int) {
				Expect(ExtractPortFromArg(arg)).To(Equal(expected))
			},
			Entry(":PORT", "--metrics-bind-address=:8443", 8443),
			Entry("HOST:PORT", "--metrics-bind-address=0.0.0.0:8080", 8080),
			Entry("localhost:PORT", "--metrics-bind-address=127.0.0.1:9090", 9090),
			Entry("IPv6", "--webhook-port=[::1]:9443", 9443),
			Entry("min valid port", "--metrics-bind-address=:1", 1),
			Entry("max valid port", "--metrics-bind-address=:65535", 65535),
			Entry("multiple colons (IPv6)", "--metrics-bind-address=::1:8443", 8443),
		)

		DescribeTable("invalid formats",
			func(arg string) {
				Expect(ExtractPortFromArg(arg)).To(Equal(0))
			},
			Entry("missing equals", "--metrics-bind-address:8443"),
			Entry("non-numeric", "--metrics-bind-address=:abc"),
			Entry("too low", "--metrics-bind-address=:0"),
			Entry("too high", "--metrics-bind-address=:99999"),
			Entry("empty port", "--metrics-bind-address=:"),
			Entry("missing value", "--metrics-bind-address="),
		)
	})

	Describe("ExtractDeploymentConfig extraVolumes extraction", func() {
		It("should extract custom volumes without mutating the deployment", func() {
			deployment := makeDeployment(deploymentOpts{
				volumes: []any{
					map[string]any{keyName: valAppConfig, keyConfigMap: map[string]any{keyName: valMyConfig}},
					map[string]any{keyName: valAppSecret, keySecret: map[string]any{keySecretName: valMySecret}},
				},
				containers: []map[string]any{{
					keyName: valManager,
					keyVolumeMounts: []any{
						map[string]any{keyName: valAppConfig, keyMountPath: valMountConfig},
						map[string]any{keyName: valAppSecret, keyMountPath: "/etc/secret"},
					},
				}},
			})

			config, err := (&DeploymentExtractor{}).ExtractDeploymentConfig(deployment)
			Expect(err).NotTo(HaveOccurred())
			Expect(config.Manager.ExtraVolumes).To(HaveLen(2))
			Expect(config.Manager.ExtraVolumeMounts).To(HaveLen(2))
			Expect(podSpec(deployment)["volumes"].([]any)).To(HaveLen(2), "extraction must not mutate the deployment")
		})

		It("should separate system and custom volumes", func() {
			deployment := makeDeployment(deploymentOpts{
				volumes: []any{
					map[string]any{keyName: valWebhookCert, keySecret: map[string]any{keySecretName: valWebhookSecretName}},
					map[string]any{keyName: valMetricsCert, keySecret: map[string]any{keySecretName: "metrics-server-cert"}},
					map[string]any{keyName: valAppConfig, keyConfigMap: map[string]any{keyName: valMyConfig}},
				},
				containers: []map[string]any{{
					keyName: valManager,
					keyVolumeMounts: []any{
						map[string]any{keyName: valWebhookCert, keyMountPath: valMountCerts},
						map[string]any{keyName: valMetricsCert, keyMountPath: valMountMetrics},
						map[string]any{keyName: valAppConfig, keyMountPath: valMountConfig},
					},
				}},
			})

			config, err := (&DeploymentExtractor{}).ExtractDeploymentConfig(deployment)
			Expect(err).NotTo(HaveOccurred())
			Expect(config.Manager.ExtraVolumes).To(HaveLen(1))
			Expect(config.Manager.ExtraVolumeMounts).To(HaveLen(1))
		})

		It("should return nil when only system volumes exist", func() {
			deployment := makeDeployment(deploymentOpts{
				volumes: []any{
					map[string]any{keyName: valWebhookCert},
					map[string]any{keyName: valMetricsCert},
				},
				containers: []map[string]any{{
					keyName: valManager,
					keyVolumeMounts: []any{
						map[string]any{keyName: valWebhookCert, keyMountPath: valMountCerts},
						map[string]any{keyName: valMetricsCert, keyMountPath: valMountMetrics},
					},
				}},
			})

			config, err := (&DeploymentExtractor{}).ExtractDeploymentConfig(deployment)
			Expect(err).NotTo(HaveOccurred())
			Expect(config.Manager.ExtraVolumes).To(BeNil())
			Expect(config.Manager.ExtraVolumeMounts).To(BeNil())
		})
	})

	Describe("Manager container resolution", func() {
		It("should use the default-container annotation to find the manager", func() {
			deployment := makeDeployment(deploymentOpts{
				annotations: map[string]string{
					annotationDefaultContainer: "controller-test",
				},
				containers: []map[string]any{
					{keyName: "controller-test", keyImage: valControllerImage},
				},
			})

			config, err := (&DeploymentExtractor{}).ExtractDeploymentConfig(deployment)
			Expect(err).NotTo(HaveOccurred())
			Expect(config.Manager.Image.Repository).To(Equal("controller"))
			Expect(config.Manager.Image.Tag).To(Equal("latest"))
		})

		It("should find the manager when a sidecar is before it", func() {
			deployment := makeDeployment(deploymentOpts{
				annotations: map[string]string{
					annotationDefaultContainer: valManager,
				},
				containers: []map[string]any{
					{keyName: valSidecar, keyImage: "sidecar:v1"},
					{keyName: valManager, keyImage: valControllerImage},
				},
			})

			config, err := (&DeploymentExtractor{}).ExtractDeploymentConfig(deployment)
			Expect(err).NotTo(HaveOccurred())
			Expect(config.Manager.Image.Repository).To(Equal("controller"))
		})

		It("should fall back to 'manager' name when annotation is missing", func() {
			deployment := makeDeployment(deploymentOpts{
				containers: []map[string]any{
					{keyName: valSidecar, keyImage: "sidecar:v1"},
					{keyName: valManager, keyImage: valControllerImage},
				},
			})

			config, err := (&DeploymentExtractor{}).ExtractDeploymentConfig(deployment)
			Expect(err).NotTo(HaveOccurred())
			Expect(config.Manager.Image.Repository).To(Equal("controller"))
		})

		It("should fall back to containers[0] when no name matches", func() {
			deployment := makeDeployment(deploymentOpts{
				containers: []map[string]any{
					{keyName: "custom-name", keyImage: valControllerImage},
				},
			})

			config, err := (&DeploymentExtractor{}).ExtractDeploymentConfig(deployment)
			Expect(err).NotTo(HaveOccurred())
			Expect(config.Manager.Image.Repository).To(Equal("controller"))
		})

		It("should error when the deployment has no containers", func() {
			deployment := makeDeployment(deploymentOpts{})

			_, err := (&DeploymentExtractor{}).ExtractDeploymentConfig(deployment)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("no manager container found"))
		})

		It("should strip volumes only from the manager container when a sidecar comes first", func() {
			deployment := makeDeployment(deploymentOpts{
				annotations: map[string]string{
					annotationDefaultContainer: valManager,
				},
				volumes: []any{
					map[string]any{keyName: valWebhookCert, keySecret: map[string]any{keySecretName: valWebhookSecretName}},
					map[string]any{keyName: valAppConfig, keyConfigMap: map[string]any{keyName: valMyConfig}},
				},
				containers: []map[string]any{
					{
						keyName: valSidecar,
						keyVolumeMounts: []any{
							map[string]any{keyName: "sidecar-vol", keyMountPath: "/sidecar"},
						},
					},
					{
						keyName: valManager,
						keyVolumeMounts: []any{
							map[string]any{keyName: valWebhookCert, keyMountPath: valMountCerts},
							map[string]any{keyName: valAppConfig, keyMountPath: valMountConfig},
						},
					},
				},
			})

			(&DeploymentExtractor{}).RemoveExtractedVolumes(deployment)

			spec := podSpec(deployment)
			volumes := spec["volumes"].([]any)
			Expect(volumes).To(HaveLen(1))
			Expect(volumes[0].(map[string]any)[keyName]).To(Equal(valWebhookCert))

			sidecar := spec["containers"].([]any)[0].(map[string]any)
			Expect(sidecar[keyVolumeMounts].([]any)).To(HaveLen(1), "sidecar mounts must not be touched")

			manager := spec["containers"].([]any)[1].(map[string]any)
			managerMounts := manager[keyVolumeMounts].([]any)
			Expect(managerMounts).To(HaveLen(1))
			Expect(managerMounts[0].(map[string]any)[keyName]).To(Equal(valWebhookCert))
		})
	})
})
