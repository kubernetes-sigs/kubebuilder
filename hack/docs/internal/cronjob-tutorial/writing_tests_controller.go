/*
Copyright 2023 The Kubernetes Authors.

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

package cronjob

const controllerTest = `/*

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

/*
Ideally, we should have one` + " `" + `<kind>_controller_test.go` + "`" + ` for each controller scaffolded and called in the` + " `" + `suite_test.go` + "`" + `.
So, let's write our example test for the CronJob controller (` + "`" + `cronjob_controller_test.go.` + "`" + `)
*/

/*
As usual, we start with the necessary imports. We also define some utility variables.
*/
package controller

import (
	"context"
	"reflect"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	cronjobv1 "tutorial.kubebuilder.io/project/api/v1"
)

// +kubebuilder:docs-gen:collapse=Imports

/*
The first step to writing a simple integration test is to actually create an instance of CronJob you can run tests against.
Note that to create a CronJob, you’ll need to create a stub CronJob struct that contains your CronJob’s specifications.

Note that when we create a stub CronJob, the CronJob also needs stubs of its required downstream objects.
Without the stubbed Job template spec and the Pod template spec below, the Kubernetes API will not be able to
create the CronJob.
*/
var _ = Describe("CronJob controller", func() {

	// Define utility constants for object names and testing timeouts/durations and intervals.
	const (
		CronjobName      = "test-cronjob"
		CronjobNamespace = "default"
		JobName          = "test-job"

		timeout  = time.Second * 10
		duration = time.Second * 10
		interval = time.Millisecond * 250
	)

	Context("When updating CronJob Status", func() {
		It("Should increase CronJob Status.Active count when new Jobs are created", func() {
			By("By creating a new CronJob")
			ctx := context.Background()
			cronJob := &cronjobv1.CronJob{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "batch.tutorial.kubebuilder.io/v1",
					Kind:       "CronJob",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      CronjobName,
					Namespace: CronjobNamespace,
				},
				Spec: cronjobv1.CronJobSpec{
					Schedule: "1 * * * *",
					JobTemplate: batchv1.JobTemplateSpec{
						Spec: batchv1.JobSpec{
							// For simplicity, we only fill out the required fields.
							Template: v1.PodTemplateSpec{
								Spec: v1.PodSpec{
									// For simplicity, we only fill out the required fields.
									Containers: []v1.Container{
										{
											Name:  "test-container",
											Image: "test-image",
										},
									},
									RestartPolicy: v1.RestartPolicyOnFailure,
								},
							},
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, cronJob)).To(Succeed())

			/*
				After creating this CronJob, let's check that the CronJob's Spec fields match what we passed in.
				Note that, because the k8s apiserver may not have finished creating a CronJob after our` + " `" + `Create()` + "`" + ` call from earlier, we will use Gomega’s Eventually() testing function instead of Expect() to give the apiserver an opportunity to finish creating our CronJob.` + `

				` +
	"`" + `Eventually()` + "`" + ` will repeatedly run the function provided as an argument every interval seconds until
				(a) the assertions done by the passed-in ` + "`" + `Gomega` + "`" + ` succeed, or
				(b) the number of attempts * interval period exceed the provided timeout value.

				In the examples below, timeout and interval are Go Duration values of our choosing.
			*/

			cronjobLookupKey := types.NamespacedName{Name: CronjobName, Namespace: CronjobNamespace}
			createdCronjob := &cronjobv1.CronJob{}

			// We'll need to retry getting this newly created CronJob, given that creation may not immediately happen.
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, cronjobLookupKey, createdCronjob)).To(Succeed())
			}, timeout, interval).Should(Succeed())
			// Let's make sure our Schedule string value was properly converted/handled.
			Expect(createdCronjob.Spec.Schedule).To(Equal("1 * * * *"))
			/*
				Now that we've created a CronJob in our test cluster, the next step is to write a test that actually tests our CronJob controller’s behavior.
				Let’s test the CronJob controller’s logic responsible for updating CronJob.Status.Active with actively running jobs.
				We’ll verify that when a CronJob has a single active downstream Job, its CronJob.Status.Active field contains a reference to this Job.

				First, we should get the test CronJob we created earlier, and verify that it currently does not have any active jobs.
				We use Gomega's` + " `" + `Consistently()` + "`" + ` check here to ensure that the active job count remains 0 over a duration of time.
			*/
			By("By checking the CronJob has zero active Jobs")
			Consistently(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, cronjobLookupKey, createdCronjob)).To(Succeed())
				g.Expect(createdCronjob.Status.Active).To(HaveLen(0))
			}, duration, interval).Should(Succeed())
			/*
				Next, we actually create a stubbed Job that will belong to our CronJob, as well as its downstream template specs.
				We set the Job's status's "Active" count to 2 to simulate the Job running two pods, which means the Job is actively running.

				We then take the stubbed Job and set its owner reference to point to our test CronJob.
				This ensures that the test Job belongs to, and is tracked by, our test CronJob.
				Once that’s done, we create our new Job instance.
			*/
			By("By creating a new Job")
			testJob := &batchv1.Job{
				ObjectMeta: metav1.ObjectMeta{
					Name:      JobName,
					Namespace: CronjobNamespace,
				},
				Spec: batchv1.JobSpec{
					Template: v1.PodTemplateSpec{
						Spec: v1.PodSpec{
							// For simplicity, we only fill out the required fields.
							Containers: []v1.Container{
								{
									Name:  "test-container",
									Image: "test-image",
								},
							},
							RestartPolicy: v1.RestartPolicyOnFailure,
						},
					},
				},
			}

			// Note that your CronJob’s GroupVersionKind is required to set up this owner reference.
			kind := reflect.TypeOf(cronjobv1.CronJob{}).Name()
			gvk := cronjobv1.GroupVersion.WithKind(kind)

			controllerRef := metav1.NewControllerRef(createdCronjob, gvk)
			testJob.SetOwnerReferences([]metav1.OwnerReference{*controllerRef})
			Expect(k8sClient.Create(ctx, testJob)).To(Succeed())
			// Note that you can not manage the status values while creating the resource.
			// The status field is managed separately to reflect the current state of the resource.
			// Therefore, it should be updated using a PATCH or PUT operation after the resource has been created.
			// Additionally, it is recommended to use StatusConditions to manage the status. For further information see:
			// https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#spec-and-status
			testJob.Status.Active = 2
			Expect(k8sClient.Status().Update(ctx, testJob)).To(Succeed())
			/*
				Adding this Job to our test CronJob should trigger our controller’s reconciler logic.
				After that, we can write a test that evaluates whether our controller eventually updates our CronJob’s Status field as expected!
			*/
			By("By checking that the CronJob has one active Job")
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, cronjobLookupKey, createdCronjob)).To(Succeed(), "should GET the CronJob")
				g.Expect(createdCronjob.Status.Active).To(HaveLen(1), "should have exactly one active job")
				g.Expect(createdCronjob.Status.Active[0].Name).To(Equal(JobName), "the wrong job is active")
			}, timeout, interval).Should(Succeed(), "should list our active job %s in the active jobs list in status", JobName)
		})
	})

})

/*
	After writing all this code, you can run` + " `" + `go test ./...` + "`" + ` in your` + " `" + `controllers/` + "`" + ` directory again to run your new test!
*/
`
