package kustomize_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	kust "sigs.k8s.io/kubebuilder/v4/pkg/plugins/optional/helm/v2alpha/scaffolds/internal/kustomize"
)

var _ = Describe("Generic Services", func() {
	It("should place non-webhook, non-metrics services into the 'services' group", func() {

		// 1. Create a fake Service
		service := &unstructured.Unstructured{}
		service.SetKind("Service")
		service.SetAPIVersion("v1")
		service.SetName("my-alert-service") // does NOT contain 'metrics' or 'webhook'

		// 2. Put it inside ParsedResources
		parsed := &kust.ParsedResources{
			Services: []*unstructured.Unstructured{service},
		}

		// 3. Run the ResourceOrganizer
		organizer := kust.NewResourceOrganizer(parsed)
		groups := organizer.OrganizeByFunction()

		// 4. Expect it inside the "services" group
		Expect(groups).To(HaveKey("services"))
		Expect(groups["services"]).To(HaveLen(1))
		Expect(groups["services"][0].GetName()).To(Equal("my-alert-service"))
	})
})
