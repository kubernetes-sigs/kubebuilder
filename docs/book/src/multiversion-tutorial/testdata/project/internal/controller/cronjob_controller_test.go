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

package controller

import (
	"context"
	"fmt"
	"reflect"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"

	cronjobv1 "tutorial.kubebuilder.io/project/api/v1"
)

var _ = Describe("CronJob controller", func() {
	Context("CronJob controller test", func() {

		const NamespaceName = "test-cronjob"

		ctx := context.Background()

		namespace := &v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name:      NamespaceName,
				Namespace: NamespaceName,
			},
		}

		SetDefaultEventuallyTimeout(2 * time.Minute)
		SetDefaultEventuallyPollingInterval(time.Second)

		BeforeEach(func() {
			By("Creating the Namespace to perform the tests")
			err := k8sClient.Get(ctx, types.NamespacedName{Name: NamespaceName}, &v1.Namespace{})
			if err != nil && errors.IsNotFound(err) {
				err = k8sClient.Create(ctx, namespace)
				Expect(err).NotTo(HaveOccurred())
			}
		})

		AfterEach(func() {
			// Note: We don't delete the namespace here to avoid issues with parallel test execution.
			// The namespace will be cleaned up when the test suite finishes.
		})

		It("should initialize status conditions on first reconciliation", func() {
			cronJobName := fmt.Sprintf("test-cronjob-%d", GinkgoRandomSeed())
			typeNamespacedName := types.NamespacedName{
				Name:      cronJobName,
				Namespace: NamespaceName,
			}

			cronJob := &cronjobv1.CronJob{
				ObjectMeta: metav1.ObjectMeta{
					Name:      cronJobName,
					Namespace: NamespaceName,
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

			Expect(k8sClient.Create(ctx, cronJob)).To(Succeed())

			By("Checking that status conditions are initialized")
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, typeNamespacedName, cronJob)).To(Succeed())
				g.Expect(cronJob.Status.Conditions).NotTo(BeEmpty())
			}).Should(Succeed())

			By("Cleaning up the CronJob")
			Expect(k8sClient.Delete(ctx, cronJob)).To(Succeed())
		})

		It("should set AllJobsCompleted condition when no active jobs exist", func() {
			cronJobName := fmt.Sprintf("test-cronjob-%d", GinkgoRandomSeed())
			typeNamespacedName := types.NamespacedName{
				Name:      cronJobName,
				Namespace: NamespaceName,
			}

			cronJob := &cronjobv1.CronJob{
				ObjectMeta: metav1.ObjectMeta{
					Name:      cronJobName,
					Namespace: NamespaceName,
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

			Expect(k8sClient.Create(ctx, cronJob)).To(Succeed())

			By("Checking that the CronJob has zero active Jobs")
			Consistently(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, typeNamespacedName, cronJob)).To(Succeed())
				g.Expect(cronJob.Status.Active).To(BeEmpty())
			}).WithTimeout(time.Second * 5).WithPolling(time.Millisecond * 250).Should(Succeed())

			By("Checking AllJobsCompleted condition")
			Expect(k8sClient.Get(ctx, typeNamespacedName, cronJob)).To(Succeed())
			var availableConditions []metav1.Condition
			Expect(cronJob.Status.Conditions).To(ContainElement(
				HaveField("Type", Equal("Available")), &availableConditions))
			if len(availableConditions) > 0 {
				Expect(availableConditions[0].Status).To(Equal(metav1.ConditionTrue))
				Expect(availableConditions[0].Reason).To(Equal("AllJobsCompleted"))
			}

			var progressingConditions []metav1.Condition
			Expect(cronJob.Status.Conditions).To(ContainElement(
				HaveField("Type", Equal("Progressing")), &progressingConditions))
			if len(progressingConditions) > 0 {
				Expect(progressingConditions[0].Status).To(Equal(metav1.ConditionFalse))
				Expect(progressingConditions[0].Reason).To(Equal("NoJobsActive"))
			}

			By("Cleaning up the CronJob")
			Expect(k8sClient.Delete(ctx, cronJob)).To(Succeed())
		})

		It("should track active jobs and set JobsActive condition", func() {
			cronJobName := fmt.Sprintf("test-cronjob-%d", GinkgoRandomSeed())
			typeNamespacedName := types.NamespacedName{
				Name:      cronJobName,
				Namespace: NamespaceName,
			}

			cronJob := &cronjobv1.CronJob{
				ObjectMeta: metav1.ObjectMeta{
					Name:      cronJobName,
					Namespace: NamespaceName,
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

			Expect(k8sClient.Create(ctx, cronJob)).To(Succeed())

			By("Creating an active Job owned by the CronJob")
			testJob := &batchv1.Job{
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("test-job-%d", GinkgoRandomSeed()),
					Namespace: NamespaceName,
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

			kind := reflect.TypeFor[cronjobv1.CronJob]().Name()
			gvk := cronjobv1.GroupVersion.WithKind(kind)
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, typeNamespacedName, cronJob)).To(Succeed())
			}).Should(Succeed())

			controllerRef := metav1.NewControllerRef(cronJob, gvk)
			testJob.SetOwnerReferences([]metav1.OwnerReference{*controllerRef})
			Expect(k8sClient.Create(ctx, testJob)).To(Succeed())

			testJob.Status.Active = 2
			Expect(k8sClient.Status().Update(ctx, testJob)).To(Succeed())

			By("Checking that the CronJob has one active Job in status")
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, typeNamespacedName, cronJob)).To(Succeed())
				g.Expect(cronJob.Status.Active).To(HaveLen(1))
				g.Expect(cronJob.Status.Active[0].Name).To(Equal(testJob.Name))
			}).Should(Succeed())

			By("Checking JobsActive conditions")
			Expect(k8sClient.Get(ctx, typeNamespacedName, cronJob)).To(Succeed())
			var availableConditions []metav1.Condition
			Expect(cronJob.Status.Conditions).To(ContainElement(
				HaveField("Type", Equal("Available")), &availableConditions))
			Expect(availableConditions).To(HaveLen(1))
			Expect(availableConditions[0].Status).To(Equal(metav1.ConditionTrue))
			Expect(availableConditions[0].Reason).To(Equal("JobsActive"))

			var progressingConditions []metav1.Condition
			Expect(cronJob.Status.Conditions).To(ContainElement(
				HaveField("Type", Equal("Progressing")), &progressingConditions))
			Expect(progressingConditions).To(HaveLen(1))
			Expect(progressingConditions[0].Status).To(Equal(metav1.ConditionTrue))
			Expect(progressingConditions[0].Reason).To(Equal("JobsActive"))

			By("Cleaning up")
			Expect(k8sClient.Delete(ctx, testJob)).To(Succeed())
			Expect(k8sClient.Delete(ctx, cronJob)).To(Succeed())
		})

		It("should set Degraded condition when jobs fail", func() {
			cronJobName := fmt.Sprintf("test-cronjob-%d", GinkgoRandomSeed())
			typeNamespacedName := types.NamespacedName{
				Name:      cronJobName,
				Namespace: NamespaceName,
			}

			cronJob := &cronjobv1.CronJob{
				ObjectMeta: metav1.ObjectMeta{
					Name:      cronJobName,
					Namespace: NamespaceName,
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

			Expect(k8sClient.Create(ctx, cronJob)).To(Succeed())

			By("Creating a failed Job owned by the CronJob")
			failedJob := &batchv1.Job{
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("test-job-failed-%d", GinkgoRandomSeed()),
					Namespace: NamespaceName,
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

			kind := reflect.TypeFor[cronjobv1.CronJob]().Name()
			gvk := cronjobv1.GroupVersion.WithKind(kind)
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, typeNamespacedName, cronJob)).To(Succeed())
			}).Should(Succeed())

			controllerRef := metav1.NewControllerRef(cronJob, gvk)
			failedJob.SetOwnerReferences([]metav1.OwnerReference{*controllerRef})
			Expect(k8sClient.Create(ctx, failedJob)).To(Succeed())

			now := metav1.Now()
			failedJob.Status.StartTime = &now
			failedJob.Status.Conditions = append(failedJob.Status.Conditions,
				batchv1.JobCondition{
					Type:   batchv1.JobFailureTarget,
					Status: v1.ConditionTrue,
				},
				batchv1.JobCondition{
					Type:   batchv1.JobFailed,
					Status: v1.ConditionTrue,
				})
			Expect(k8sClient.Status().Update(ctx, failedJob)).To(Succeed())

			By("Checking that Degraded=True when jobs fail")
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, typeNamespacedName, cronJob)).To(Succeed())
				var degradedConditions []metav1.Condition
				g.Expect(cronJob.Status.Conditions).To(ContainElement(
					HaveField("Type", Equal("Degraded")), &degradedConditions))
				if len(degradedConditions) > 0 {
					g.Expect(degradedConditions[0].Status).To(Equal(metav1.ConditionTrue))
					g.Expect(degradedConditions[0].Reason).To(Equal("JobsFailed"))
				}
			}).Should(Succeed())

			By("Checking that Available=False when jobs fail")
			Expect(k8sClient.Get(ctx, typeNamespacedName, cronJob)).To(Succeed())
			var availableConditions []metav1.Condition
			Expect(cronJob.Status.Conditions).To(ContainElement(
				HaveField("Type", Equal("Available")), &availableConditions))
			if len(availableConditions) > 0 {
				Expect(availableConditions[0].Status).To(Equal(metav1.ConditionFalse))
				Expect(availableConditions[0].Reason).To(Equal("JobsFailed"))
			}

			By("Cleaning up")
			Expect(k8sClient.Delete(ctx, failedJob)).To(Succeed())
			Expect(k8sClient.Delete(ctx, cronJob)).To(Succeed())
		})

		It("should set Available=False when CronJob is suspended", func() {
			cronJobName := fmt.Sprintf("test-cronjob-%d", GinkgoRandomSeed())
			typeNamespacedName := types.NamespacedName{
				Name:      cronJobName,
				Namespace: NamespaceName,
			}

			cronJob := &cronjobv1.CronJob{
				ObjectMeta: metav1.ObjectMeta{
					Name:      cronJobName,
					Namespace: NamespaceName,
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

			Expect(k8sClient.Create(ctx, cronJob)).To(Succeed())

			By("Updating the CronJob to suspend it")
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, typeNamespacedName, cronJob)).To(Succeed())
				cronJob.Spec.Suspend = ptr.To(true)
				g.Expect(k8sClient.Update(ctx, cronJob)).To(Succeed())
			}).Should(Succeed())

			By("Checking that Available=False when suspended")
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, typeNamespacedName, cronJob)).To(Succeed())
				var availableConditions []metav1.Condition
				g.Expect(cronJob.Status.Conditions).To(ContainElement(
					HaveField("Type", Equal("Available")), &availableConditions))
				if len(availableConditions) > 0 {
					g.Expect(availableConditions[0].Status).To(Equal(metav1.ConditionFalse))
					g.Expect(availableConditions[0].Reason).To(Equal("Suspended"))
				}
			}).Should(Succeed())

			By("Cleaning up the CronJob")
			Expect(k8sClient.Delete(ctx, cronJob)).To(Succeed())
		})
	})
})
