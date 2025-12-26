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
As usual, we start with the necessary imports.
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
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	cronjobv1 "tutorial.kubebuilder.io/project/api/v1"
)

// +kubebuilder:docs-gen:collapse=Imports

/*
The first step to writing a simple integration test is to actually create an instance of CronJob you can run tests against.
Note that to create a CronJob, you'll need to create a stub CronJob struct that contains your CronJob's specifications.

Note that when we create a stub CronJob, the CronJob also needs stubs of its required downstream objects.
Without the stubbed Job template spec and the Pod template spec below, the Kubernetes API will not be able to
create the CronJob.
*/
var _ = Describe("CronJob controller", func() {
	Context("CronJob controller test", func() {

		const CronjobName = "test-cronjob"

		ctx := context.Background()

		namespace := &v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name:      CronjobName,
				Namespace: CronjobName,
			},
		}

		typeNamespacedName := types.NamespacedName{
			Name:      CronjobName,
			Namespace: CronjobName,
		}
		cronJob := &cronjobv1.CronJob{}

		SetDefaultEventuallyTimeout(2 * time.Minute)
		SetDefaultEventuallyPollingInterval(time.Second)

		BeforeEach(func() {
			By("Creating the Namespace to perform the tests")
			err := k8sClient.Get(ctx, types.NamespacedName{Name: CronjobName}, &v1.Namespace{})
			if err != nil && errors.IsNotFound(err) {
				err = k8sClient.Create(ctx, namespace)
				Expect(err).NotTo(HaveOccurred())
			}

			By("creating the custom resource for the Kind CronJob")
			cronJob = &cronjobv1.CronJob{}
			err = k8sClient.Get(ctx, typeNamespacedName, cronJob)
			if err != nil && errors.IsNotFound(err) {
				/*
					Let's mock our custom resource the same way we would apply it from
					the manifest under config/samples
				*/
				cronJob = &cronjobv1.CronJob{
					ObjectMeta: metav1.ObjectMeta{
						Name:      CronjobName,
						Namespace: namespace.Name,
					},
					Spec: cronjobv1.CronJobSpec{
						Schedule: "1 * * * *",
						JobTemplate: batchv1.JobTemplateSpec{
							Spec: batchv1.JobSpec{
								Template: v1.PodTemplateSpec{
									Spec: v1.PodSpec{
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

				err = k8sClient.Create(ctx, cronJob)
				Expect(err).NotTo(HaveOccurred())
			}
		})

		/*
			After each test, we clean up the resources created above.
		*/

		AfterEach(func() {
			By("removing the custom resource for the Kind CronJob")
			found := &cronjobv1.CronJob{}
			err := k8sClient.Get(ctx, typeNamespacedName, found)
			Expect(err).NotTo(HaveOccurred())

			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Delete(context.TODO(), found)).To(Succeed())
			}).Should(Succeed())

			// TODO(user): Attention if you improve this code by adding other context test you MUST
			// be aware of the current delete namespace limitations.
			// More info: https://book.kubebuilder.io/reference/envtest.html#testing-considerations
			By("Deleting the Namespace to perform the tests")
			_ = k8sClient.Delete(ctx, namespace)
		})

		/*
			Now we can start implementing the test that validates the controller’s reconciliation behavior.
		*/

		It("should successfully reconcile a custom resource for CronJob", func() {
			By("Checking if the custom resource was successfully created")
			Eventually(func(g Gomega) {
				found := &cronjobv1.CronJob{}
				g.Expect(k8sClient.Get(ctx, typeNamespacedName, found)).To(Succeed())
			}).Should(Succeed())

			/*
				After creating this CronJob, let's verify that the controller properly initializes the status conditions.
				The controller runs in the background (started in suite_test.go), so it will automatically
				detect our CronJob and set initial conditions.
			*/
			By("Checking that status conditions are initialized")
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, typeNamespacedName, cronJob)).To(Succeed())
				g.Expect(cronJob.Status.Conditions).NotTo(BeEmpty())
			}).Should(Succeed())

			/*
				Now let's verify the CronJob has no active jobs initially.
				We use Gomega's` + " `" + `Consistently()` + "`" + ` check here to ensure the status remains stable,
				confirming the controller isn't creating jobs prematurely.
			*/
			By("Checking that the CronJob has zero active Jobs")
			Consistently(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, typeNamespacedName, cronJob)).To(Succeed())
				g.Expect(cronJob.Status.Active).To(BeEmpty())
			}).WithTimeout(time.Second * 10).WithPolling(time.Millisecond * 250).Should(Succeed())

			/*
				Next, we actually create a stubbed Job that will belong to our CronJob.
				We set the Job's status Active count to 2 to simulate the Job running two pods,
				which means the Job is actively running.

				We then set the Job's owner reference to point to our test CronJob.
				This ensures that the test Job belongs to, and is tracked by, our test CronJob.
			*/
			By("Creating a new Job owned by the CronJob")
			testJob := &batchv1.Job{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-job",
					Namespace: namespace.Name,
				},
				Spec: batchv1.JobSpec{
					Template: v1.PodTemplateSpec{
						Spec: v1.PodSpec{
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
			kind := reflect.TypeFor[cronjobv1.CronJob]().Name()
			gvk := cronjobv1.GroupVersion.WithKind(kind)

			controllerRef := metav1.NewControllerRef(cronJob, gvk)
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
				Adding this Job to our test CronJob should trigger our controller's reconciler logic.
				After that, we can verify whether our controller eventually updates our CronJob's Status field as expected!
			*/
			By("Checking that the CronJob has one active Job in status")
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, typeNamespacedName, cronJob)).To(Succeed())
				g.Expect(cronJob.Status.Active).To(HaveLen(1), "should have exactly one active job")
				g.Expect(cronJob.Status.Active[0].Name).To(Equal("test-job"), "the active job name should match")
			}).Should(Succeed())

			/*
				Finally, let's verify that the controller properly set status conditions.
				Status conditions are a key part of Kubernetes API conventions and allow users and other
				controllers to understand the resource state.
				
				When there are active jobs, the Available condition should be True with reason JobsActive.
			*/
			By("Checking the latest Status Condition added to the CronJob instance")
			Expect(k8sClient.Get(ctx, typeNamespacedName, cronJob)).To(Succeed())
			var conditions []metav1.Condition
			Expect(cronJob.Status.Conditions).To(ContainElement(
				HaveField("Type", Equal("Available")), &conditions))
			Expect(conditions).To(HaveLen(1), "should have one Available condition")
			Expect(conditions[0].Status).To(Equal(metav1.ConditionTrue), "Available should be True")
			Expect(conditions[0].Reason).To(Equal("JobsActive"), "reason should be JobsActive")
		})
	})
})

/*
	After writing all this code, you can run ` + "`" + `make test` + "`" + ` or ` + "`" + `go test ./...` + "`" + ` in your ` + "`" + `controllers/` + "`" + ` directory again to run your new test!
*/
`
