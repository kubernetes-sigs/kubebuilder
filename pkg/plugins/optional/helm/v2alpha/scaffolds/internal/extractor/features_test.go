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

var _ = Describe("FeaturesExtractor", func() {
	var featuresExtractor *FeaturesExtractor

	BeforeEach(func() {
		featuresExtractor = &FeaturesExtractor{}
	})

	deploymentWithManagerArgs := func(args ...string) *unstructured.Unstructured {
		argsList := make([]any, len(args))
		for i, a := range args {
			argsList[i] = a
		}
		return makeDeployment(deploymentOpts{
			containers: []map[string]any{{
				keyName:  valManager,
				keyImage: valControllerImage,
				keyArgs:  argsList,
			}},
		})
	}

	detect := func(deployment *unstructured.Unstructured) FeatureSet {
		return featuresExtractor.DetectFeatures(
			&ResourceSet{Deployment: deployment}, "test-project", "test-system")
	}

	Describe("DetectFeatures port defaults", func() {
		It("should default all ports when there is no deployment", func() {
			features := detect(nil)

			Expect(features.HealthProbePort).To(Equal(8081))
			Expect(features.MetricsPort).To(Equal(8443))
			Expect(features.WebhookPort).To(Equal(9443))
		})
	})

	Describe("DetectFeatures health probe port", func() {
		It("should default to 8081 when the bind-address arg is absent", func() {
			features := detect(deploymentWithManagerArgs("--leader-elect"))

			Expect(features.HealthProbePort).To(Equal(8081))
		})

		It("should detect a custom port from :PORT form", func() {
			features := detect(deploymentWithManagerArgs("--health-probe-bind-address=:9091"))

			Expect(features.HealthProbePort).To(Equal(9091))
		})

		It("should detect a custom port from HOST:PORT form", func() {
			features := detect(deploymentWithManagerArgs("--health-probe-bind-address=localhost:9091"))

			Expect(features.HealthProbePort).To(Equal(9091))
		})

		It("should detect a custom port from IPv6 [::1]:PORT form", func() {
			features := detect(deploymentWithManagerArgs("--health-probe-bind-address=[::1]:9091"))

			Expect(features.HealthProbePort).To(Equal(9091))
		})

		It("should default to 8081 when the port is not numeric", func() {
			features := detect(deploymentWithManagerArgs("--health-probe-bind-address=:invalid"))

			Expect(features.HealthProbePort).To(Equal(8081))
		})

		It("should default to 8081 when the port is out of range", func() {
			features := detect(deploymentWithManagerArgs("--health-probe-bind-address=:99999"))

			Expect(features.HealthProbePort).To(Equal(8081))
		})

		It("should default to 8081 when probes are disabled with bind address 0", func() {
			features := detect(deploymentWithManagerArgs("--health-probe-bind-address=0"))

			Expect(features.HealthProbePort).To(Equal(8081))
		})

		It("should accept the maximum valid port 65535", func() {
			features := detect(deploymentWithManagerArgs("--health-probe-bind-address=:65535"))

			Expect(features.HealthProbePort).To(Equal(65535))
		})

		It("should default to 8081 when the flag and address are separate args", func() {
			features := detect(deploymentWithManagerArgs("--health-probe-bind-address", ":9091"))

			Expect(features.HealthProbePort).To(Equal(8081))
		})

		It("should use the first valid value when the arg is duplicated", func() {
			features := detect(deploymentWithManagerArgs(
				"--health-probe-bind-address=:9091",
				"--health-probe-bind-address=:9092",
			))

			Expect(features.HealthProbePort).To(Equal(9091))
		})

		It("should read the arg from the manager container, not a sidecar", func() {
			deployment := makeDeployment(deploymentOpts{
				containers: []map[string]any{
					{
						keyName:  valSidecar,
						keyImage: valSidecarImage,
						keyArgs:  []any{"--health-probe-bind-address=:7070"},
					},
					{
						keyName:  valManager,
						keyImage: valControllerImage,
						keyArgs:  []any{"--health-probe-bind-address=:9091"},
					},
				},
			})

			features := detect(deployment)

			Expect(features.HealthProbePort).To(Equal(9091))
		})
	})

	Describe("DetectFeatures metrics secure", func() {
		It("should leave the value unset when the arg is absent", func() {
			features := detect(deploymentWithManagerArgs("--leader-elect"))

			Expect(features.MetricsSecure).To(BeNil())
		})

		It("should leave the value unset when there is no deployment", func() {
			features := detect(nil)

			Expect(features.MetricsSecure).To(BeNil())
		})

		It("should detect metrics served over plain HTTP", func() {
			features := detect(deploymentWithManagerArgs("--metrics-secure=false"))

			Expect(features.MetricsSecure).To(HaveValue(BeFalse()))
		})

		It("should detect metrics served over HTTPS", func() {
			features := detect(deploymentWithManagerArgs("--metrics-secure=true"))

			Expect(features.MetricsSecure).To(HaveValue(BeTrue()))
		})

		It("should read a bare flag as true", func() {
			features := detect(deploymentWithManagerArgs("--metrics-secure"))

			Expect(features.MetricsSecure).To(HaveValue(BeTrue()))
		})

		It("should read a separate value as true, matching boolean flag parsing", func() {
			features := detect(deploymentWithManagerArgs("--metrics-secure", "false"))

			Expect(features.MetricsSecure).To(HaveValue(BeTrue()))
		})

		It("should use the first value when the arg is duplicated", func() {
			features := detect(deploymentWithManagerArgs(
				"--metrics-secure=false",
				"--metrics-secure=true",
			))

			Expect(features.MetricsSecure).To(HaveValue(BeFalse()))
		})

		It("should read the arg from the manager container, not a sidecar", func() {
			deployment := makeDeployment(deploymentOpts{
				containers: []map[string]any{
					{
						keyName:  valSidecar,
						keyImage: valSidecarImage,
						keyArgs:  []any{"--metrics-secure=true"},
					},
					{
						keyName:  valManager,
						keyImage: valControllerImage,
						keyArgs:  []any{"--metrics-secure=false"},
					},
				},
			})

			features := detect(deployment)

			Expect(features.MetricsSecure).To(HaveValue(BeFalse()))
		})
	})
})
