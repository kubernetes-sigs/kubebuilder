/*
Copyright 2024 The Kubernetes authors.

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
	ctrl "sigs.k8s.io/controller-runtime"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	kbatch "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	batchv1 "tutorial.kubebuilder.io/project/api/v1"
)

var _ = Describe("CronJob Controller", func() {
	const (
		resourceName = "test-resource"
		namespace    = "default"
		timeout      = time.Second * 10
		duration     = time.Second * 10
		interval     = time.Millisecond * 250
	)

	var (
		ctx = context.Background()
	)

	typeNamespacedName := types.NamespacedName{
		Name:      resourceName,
		Namespace: namespace,
	}

	cronJob := &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      resourceName,
			Namespace: namespace,
		},
		Spec: batchv1.CronJobSpec{
			Schedule: "*/1 * * * *",
			JobTemplate: kbatch.JobTemplateSpec{
				Spec: kbatch.JobSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:    "example-container",
									Image:   "example-image",
									Command: []string{"echo", "Hello World"},
								},
							},
							RestartPolicy: corev1.RestartPolicyOnFailure,
						},
					},
				},
			},
		},
	}

	BeforeEach(func() {
		By("Creating the custom resource for the Kind CronJob")
		err := k8sClient.Get(ctx, typeNamespacedName, cronJob)
		if err != nil && errors.IsNotFound(err) {
			cronJob.ResourceVersion = "" // Ensure resourceVersion is not set
			Expect(k8sClient.Create(ctx, cronJob)).To(Succeed())
		}
	})

	AfterEach(func() {
		By("Cleaning up the CronJob resource")
		Expect(k8sClient.Delete(ctx, cronJob)).To(Succeed())
	})

	It("Should reconcile successfully and match the expected CronJob data", func() {
		By("Reconciling the CronJob")
		controllerReconciler := &CronJobReconciler{
			Client: k8sClient,
			Scheme: k8sClient.Scheme(),
			Clock:  realClock{},
		}

		_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
			NamespacedName: typeNamespacedName,
		})
		Expect(err).NotTo(HaveOccurred())

		By("Verifying the CronJob spec schedule")
		createdCronJob := &batchv1.CronJob{}
		Eventually(func() bool {
			err := k8sClient.Get(ctx, typeNamespacedName, createdCronJob)
			return err == nil
		}, timeout, interval).Should(BeTrue())
		Expect(createdCronJob.Spec.Schedule).To(Equal("*/1 * * * *"))

		By("Checking the CronJob has zero active jobs")
		Consistently(func() (int, error) {
			err := k8sClient.Get(ctx, typeNamespacedName, createdCronJob)
			if err != nil {
				return -1, err
			}
			return len(createdCronJob.Status.Active), nil
		}, duration, interval).Should(Equal(0))
	})

	It("Should successfully manage job history according to CronJob policy", func() {
		By("Creating a job for the CronJob")
		scheduledTime := time.Now().Add(-time.Minute)
		controllerReconciler := &CronJobReconciler{
			Client: k8sClient,
			Scheme: k8sClient.Scheme(),
			Clock:  realClock{},
		}
		constructJobForCronJob := func(cronJob *batchv1.CronJob, scheduledTime time.Time) (*kbatch.Job, error) {
			// We want job names for a given nominal start time to have a deterministic name to avoid the same job being created twice
			name := fmt.Sprintf("%s-%d", cronJob.Name, scheduledTime.Unix())

			// actually make the job...
			job := &kbatch.Job{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      make(map[string]string),
					Annotations: make(map[string]string),
					Name:        name,
					Namespace:   cronJob.Namespace,
				},
				Spec: *cronJob.Spec.JobTemplate.Spec.DeepCopy(),
			}

			for k, v := range cronJob.Spec.JobTemplate.Annotations {
				job.Annotations[k] = v
			}
			job.Annotations[scheduledTimeAnnotation] = scheduledTime.Format(time.RFC3339)
			for k, v := range cronJob.Spec.JobTemplate.Labels {
				job.Labels[k] = v
			}

			if err := ctrl.SetControllerReference(cronJob, job, k8sClient.Scheme()); err != nil {
				return nil, err
			}

			return job, nil
		}
		job, err := constructJobForCronJob(cronJob, scheduledTime)
		Expect(err).NotTo(HaveOccurred())

		job.ResourceVersion = "" // Ensure resourceVersion is not set for new job creation
		Expect(k8sClient.Create(ctx, job)).To(Succeed())

		By("Re-running the reconciler to clean up old jobs")
		_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
			NamespacedName: typeNamespacedName,
		})
		Expect(err).NotTo(HaveOccurred())

		By("Verifying that the job history is managed correctly")
		var childJobs kbatch.JobList
		Expect(k8sClient.List(ctx, &childJobs, client.InNamespace(namespace), client.MatchingLabels{"owner-key": resourceName})).To(Succeed())
		if cronJob.Spec.SuccessfulJobsHistoryLimit != nil {
			Expect(len(childJobs.Items)).To(BeNumerically("<=", *cronJob.Spec.SuccessfulJobsHistoryLimit))
		}
	})
})
