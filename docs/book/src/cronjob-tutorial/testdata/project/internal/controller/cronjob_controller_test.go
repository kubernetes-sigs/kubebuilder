/*
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
// +kubebuilder:docs-gen:collapse=Apache License

package controller

import (
	"context"
	"math/rand"
	"reflect"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"

	cronjobv1 "tutorial.kubebuilder.io/project/api/v1"
)

// +kubebuilder:docs-gen:collapse=Imports

// Helper function to check condition status with proper timestamp validation
func assertCondition(conditions []metav1.Condition, conditionType string, expectedStatus metav1.ConditionStatus) bool {
	for _, cond := range conditions {
		if cond.Type == conditionType {
			return cond.Status == expectedStatus && !cond.LastTransitionTime.IsZero()
		}
	}
	return false
}

// Manually implement random string generation (compatible with Ginkgo versions below v2.1.0)
func randomString(length int) string {
	rand.Seed(time.Now().UnixNano() + int64(GinkgoParallelProcess()))
	chars := []rune("abcdefghijklmnopqrstuvwxyz0123456789")
	var sb strings.Builder
	for i := 0; i < length; i++ {
		sb.WriteRune(chars[rand.Intn(len(chars))])
	}
	return sb.String()
}

// Helper to create test CronJob
func createTestCronJob(ctx context.Context, name, namespace, schedule string, suspend bool) *cronjobv1.CronJob {
	cj := &cronjobv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: cronjobv1.CronJobSpec{
			Schedule: schedule,
			Suspend:  ptr.To(suspend),
			JobTemplate: batchv1.JobTemplateSpec{
				Spec: batchv1.JobSpec{
					Template: v1.PodTemplateSpec{
						Spec: v1.PodSpec{
							Containers: []v1.Container{{
								Name:  "test-container",
								Image: "busybox",
							}},
							RestartPolicy: v1.RestartPolicyOnFailure,
						},
					},
				},
			},
		},
	}
	Expect(k8sClient.Create(ctx, cj)).To(Succeed())
	return cj
}

// Helper to create owned Job
func createOwnedJob(ctx context.Context, cj *cronjobv1.CronJob, jobName string, active int32) *batchv1.Job {
	gvk := cronjobv1.GroupVersion.WithKind(reflect.TypeOf(cronjobv1.CronJob{}).Name())
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: cj.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(cj, gvk),
			},
		},
		Spec: cj.Spec.JobTemplate.Spec,
	}

	Expect(k8sClient.Create(ctx, job)).To(Succeed())

	// Update job status
	job.Status.Active = active
	Expect(k8sClient.Status().Update(ctx, job)).To(Succeed())
	return job
}

var _ = Describe("CronJob controller", func() {
	const (
		timeout  = time.Second * 15
		interval = time.Millisecond * 500
	)

	// Use unique names for each test to prevent interference
	var (
		ns             = "cronjob-test-" + randomString(5) // Using manually implemented function
		ctx            context.Context
		cronJobName    string
		namespacedName types.NamespacedName
	)

	// Setup test namespace before each test
	BeforeEach(func() {
		ctx = context.Background()
		cronJobName = "test-cj-" + randomString(5) // Using manually implemented function
		namespacedName = types.NamespacedName{Name: cronJobName, Namespace: ns}

		// Create test namespace
		nsObj := &v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: ns}}
		Expect(k8sClient.Create(ctx, nsObj)).To(Succeed())
	})

	// Cleanup after each test
	AfterEach(func() {
		nsObj := &v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: ns}}
		Expect(k8sClient.Delete(ctx, nsObj)).To(Succeed())
	})

	Context("Basic CronJob reconciliation", func() {
		It("Should create and reconcile CronJob successfully", func() {
			By("Creating initial CronJob")
			createTestCronJob(ctx, cronJobName, ns, "*/1 * * * *", false)

			// Verify creation
			createdCj := &cronjobv1.CronJob{}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, namespacedName, createdCj)).To(Succeed())
				g.Expect(createdCj.Spec.Schedule).To(Equal("*/1 * * * *"))
				g.Expect(*createdCj.Spec.Suspend).To(BeFalse())
			}, timeout, interval).Should(Succeed())
		})
	})

	Context("Active Jobs tracking", func() {
		It("Should update Active count when Jobs are created", func() {
			By("Creating base CronJob")
			cj := createTestCronJob(ctx, cronJobName, ns, "*/1 * * * *", false)

			By("Verifying initial state has no active jobs")
			createdCj := &cronjobv1.CronJob{}
			Consistently(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, namespacedName, createdCj)).To(Succeed())
				g.Expect(createdCj.Status.Active).To(BeEmpty())
			}, time.Second*3, interval).Should(Succeed())

			By("Creating owned Job")
			job := createOwnedJob(ctx, cj, "test-job-1", 1)

			By("Verifying Active count updates")
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, namespacedName, createdCj)).To(Succeed())
				g.Expect(createdCj.Status.Active).To(HaveLen(1))
				g.Expect(createdCj.Status.Active[0].Name).To(Equal(job.Name))
			}, timeout, interval).Should(Succeed())

			By("Verifying Available condition is set")
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, namespacedName, createdCj)).To(Succeed())
				g.Expect(assertCondition(createdCj.Status.Conditions, "Available", metav1.ConditionTrue)).To(BeTrue())
			}, timeout, interval).Should(Succeed())
		})
	})
})
