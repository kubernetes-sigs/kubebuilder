/*
Copyright 2019 The Kubernetes Authors.

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

package controllers_test

import (
	"context"
	//"strconv"
	//"strings"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	//"k8s.io/apimachinery/pkg/util/sets"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrl "sigs.k8s.io/controller-runtime"

	"sigs.k8s.io/kubebuilder/test/project/controllers"
	api "sigs.k8s.io/kubebuilder/test/project/api/v1"
)

var (
	// schedule is hourly on the hour
	onTheHour     = "0 * * * ?"
	errorSchedule = "obvious error schedule"
)

func justBeforeTheHour() time.Time {
	T1, err := time.Parse(time.RFC3339, "2016-05-19T09:59:00Z")
	if err != nil {
		panic("test setup error")
	}
	return T1
}

func topOfTheHour() time.Time {
	T1, err := time.Parse(time.RFC3339, "2016-05-19T10:00:00Z")
	if err != nil {
		panic("test setup error")
	}
	return T1
}

func justAfterTheHour() time.Time {
	T1, err := time.Parse(time.RFC3339, "2016-05-19T10:01:00Z")
	if err != nil {
		panic("test setup error")
	}
	return T1
}

func weekAfterTheHour() time.Time {
	T1, err := time.Parse(time.RFC3339, "2016-05-26T10:00:00Z")
	if err != nil {
		panic("test setup error")
	}
	return T1
}

func justBeforeThePriorHour() time.Time {
	T1, err := time.Parse(time.RFC3339, "2016-05-19T08:59:00Z")
	if err != nil {
		panic("test setup error")
	}
	return T1
}

func justAfterThePriorHour() time.Time {
	T1, err := time.Parse(time.RFC3339, "2016-05-19T09:01:00Z")
	if err != nil {
		panic("test setup error")
	}
	return T1
}

func startTimeStringToTime(startTime string) time.Time {
	T1, err := time.Parse(time.RFC3339, startTime)
	if err != nil {
		panic("test setup error")
	}
	return T1
}

// returns a cronJob with some fields filled in.
func cronJob() api.CronJob {
	return api.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "mycronjob",
			Namespace:         "snazzycats",
		},
		Spec: api.CronJobSpec{
			Schedule:          "* * * * ?",
			ConcurrencyPolicy: api.AllowConcurrent,
			JobTemplate: api.JobTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      map[string]string{"a": "b"},
					Annotations: map[string]string{"x": "y"},
				},
				Spec: jobSpec(),
			},
		},
	}
}

func jobSpec() api.JobSpec {
	one := int32(1)
	return api.JobSpec{
		Parallelism: &one,
		Completions: &one,
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{
					"foo": "bar",
				},
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{Image: "foo/bar"},
				},
			},
		},
	}
}

func newJob(UID string) api.Job {
	return api.Job{
		ObjectMeta: metav1.ObjectMeta{
			UID:       types.UID(UID),
			Name:      "foobar",
			Namespace: metav1.NamespaceDefault,
		},
		Spec: jobSpec(),
	}
}

var (
	shortDead  int64                          = 10
	mediumDead int64                          = 2 * 60 * 60
	longDead   int64                          = 1000000
	noDead     int64                          = -12345
)

type fakeClock struct {
	now *time.Time
}
func (c *fakeClock) Now() time.Time { return *c.now }

var _ = Describe("CronJob Controller", func() {
	Context("when deciding when to run", func() {
		var cronJobController *controllers.CronJobReconciler
		var mgrStop chan struct{}
		var now time.Time

		BeforeEach(func() {
			mgrStop = make(chan struct{})
			mgr, err := ctrl.NewManager(cfg, ctrl.Options{MapperProvider: mapperProvider})
			Expect(err).NotTo(HaveOccurred())
			api.AddToScheme(mgr.GetScheme())
			cronJobController = &controllers.CronJobReconciler{
				Log: ctrl.Log.WithName("controllers").WithName("cronjob"),
				Clock: &fakeClock{now: &now},
			}
			Expect(cronJobController.SetupWithManager(mgr)).To(Succeed())

			go func() {
				Expect(mgr.Start(mgrStop)).To(Succeed())
			}()
		})
		AfterEach(func() {
			close(mgrStop)
		})
		// Check expectations on deadline parameters
		testCases := map[string]struct {
			// cronJob spec
			concurrencyPolicy api.ConcurrencyPolicy
			suspend           bool
			schedule          string
			deadline          int64

			// cronJob status
			ranPreviously bool
			stillActive   bool

			// environment
			now time.Time

			// expectations
			expectCreate     bool
			expectDelete     bool
			expectActive     int
			expectedWarnings int
		}{
			/*"never ran, not valid schedule, api.AllowConcurrent":      {api.AllowConcurrent, false, errorSchedule, noDead, false, false, justBeforeTheHour(), false, false, 0, 1},
			"never ran, not valid schedule, false":      {api.ForbidConcurrent, false, errorSchedule, noDead, false, false, justBeforeTheHour(), false, false, 0, 1},
			"never ran, not valid schedule, api.ReplaceConcurrent":      {api.ForbidConcurrent, false, errorSchedule, noDead, false, false, justBeforeTheHour(), false, false, 0, 1},
			"never ran, not time, api.AllowConcurrent":                {api.AllowConcurrent, false, onTheHour, noDead, false, false, justBeforeTheHour(), false, false, 0, 0},
			"never ran, not time, false":                {api.ForbidConcurrent, false, onTheHour, noDead, false, false, justBeforeTheHour(), false, false, 0, 0},
			"never ran, not time, api.ReplaceConcurrent":                {api.ReplaceConcurrent, false, onTheHour, noDead, false, false, justBeforeTheHour(), false, false, 0, 0},
			"never ran, is time, api.AllowConcurrent":                 {api.AllowConcurrent, false, onTheHour, noDead, false, false, justAfterTheHour(), true, false, 1, 0},
			"never ran, is time, false":                 {api.ForbidConcurrent, false, onTheHour, noDead, false, false, justAfterTheHour(), true, false, 1, 0},
			"never ran, is time, api.ReplaceConcurrent":                 {api.ReplaceConcurrent, false, onTheHour, noDead, false, false, justAfterTheHour(), true, false, 1, 0},
			"never ran, is time, suspended":         {api.AllowConcurrent, true, onTheHour, noDead, false, false, justAfterTheHour(), false, false, 0, 0},
			"never ran, is time, past deadline":     {api.AllowConcurrent, false, onTheHour, shortDead, false, false, justAfterTheHour(), false, false, 0, 0},
			"never ran, is time, not past deadline": {api.AllowConcurrent, false, onTheHour, longDead, false, false, justAfterTheHour(), true, false, 1, 0},*/

			"prev ran but done, not time, api.AllowConcurrent":   {api.AllowConcurrent, false, onTheHour, noDead, true, false, justBeforeTheHour(), false, false, 0, 0},
			"prev ran but done, not time, api.ReplaceConcurrent": {api.ReplaceConcurrent, false, onTheHour, noDead, true, false, justBeforeTheHour(), false, false, 0, 0},
			"prev ran but done, is time, api.AllowConcurrent":    {api.AllowConcurrent, false, onTheHour, noDead, true, false, justAfterTheHour(), true, false, 1, 0},
			"prev ran but done, is time, false":                  {api.ForbidConcurrent, false, onTheHour, noDead, true, false, justAfterTheHour(), true, false, 1, 0},
			"prev ran but done, is time, api.ReplaceConcurrent":  {api.ReplaceConcurrent, false, onTheHour, noDead, true, false, justAfterTheHour(), true, false, 1, 0},
			"prev ran but done, is time, suspended":              {api.AllowConcurrent, true, onTheHour, noDead, true, false, justAfterTheHour(), false, false, 0, 0},
			"prev ran but done, is time, past deadline":          {api.AllowConcurrent, false, onTheHour, shortDead, true, false, justAfterTheHour(), false, false, 0, 0},
			"prev ran but done, is time, not past deadline":      {api.AllowConcurrent, false, onTheHour, longDead, true, false, justAfterTheHour(), true, false, 1, 0},
			"prev ran but done, not time, false":                 {api.ForbidConcurrent, false, onTheHour, noDead, true, false, justBeforeTheHour(), false, false, 0, 0},

			"still active, not time, api.AllowConcurrent":   {api.AllowConcurrent, false, onTheHour, noDead, true, true, justBeforeTheHour(), false, false, 1, 0},
			"still active, not time, false":                 {api.ForbidConcurrent, false, onTheHour, noDead, true, true, justBeforeTheHour(), false, false, 1, 0},
			"still active, not time, api.ReplaceConcurrent": {api.ReplaceConcurrent, false, onTheHour, noDead, true, true, justBeforeTheHour(), false, false, 1, 0},
			"still active, is time, api.AllowConcurrent":    {api.AllowConcurrent, false, onTheHour, noDead, true, true, justAfterTheHour(), true, false, 2, 0},
			"still active, is time, false":                  {api.ForbidConcurrent, false, onTheHour, noDead, true, true, justAfterTheHour(), false, false, 1, 0},
			"still active, is time, api.ReplaceConcurrent":  {api.ReplaceConcurrent, false, onTheHour, noDead, true, true, justAfterTheHour(), true, true, 1, 0},
			"still active, is time, suspended":              {api.AllowConcurrent, true, onTheHour, noDead, true, true, justAfterTheHour(), false, false, 1, 0},
			"still active, is time, past deadline":          {api.AllowConcurrent, false, onTheHour, shortDead, true, true, justAfterTheHour(), false, false, 1, 0},
			"still active, is time, not past deadline":      {api.AllowConcurrent, false, onTheHour, longDead, true, true, justAfterTheHour(), true, false, 2, 0},

			// Controller should fail to schedule these, as there are too many missed starting times
			// and either no deadline or a too long deadline.
			"prev ran but done, long overdue, not past deadline, api.AllowConcurrent":   {api.AllowConcurrent, false, onTheHour, longDead, true, false, weekAfterTheHour(), false, false, 0, 1},
			"prev ran but done, long overdue, not past deadline, api.ReplaceConcurrent": {api.ReplaceConcurrent, false, onTheHour, longDead, true, false, weekAfterTheHour(), false, false, 0, 1},
			"prev ran but done, long overdue, not past deadline, false":                 {api.ForbidConcurrent, false, onTheHour, longDead, true, false, weekAfterTheHour(), false, false, 0, 1},
			"prev ran but done, long overdue, no deadline, api.AllowConcurrent":         {api.AllowConcurrent, false, onTheHour, noDead, true, false, weekAfterTheHour(), false, false, 0, 1},
			"prev ran but done, long overdue, no deadline, api.ReplaceConcurrent":       {api.ReplaceConcurrent, false, onTheHour, noDead, true, false, weekAfterTheHour(), false, false, 0, 1},
			"prev ran but done, long overdue, no deadline, false":                       {api.ForbidConcurrent, false, onTheHour, noDead, true, false, weekAfterTheHour(), false, false, 0, 1},

			"prev ran but done, long overdue, past medium deadline, api.AllowConcurrent": {api.AllowConcurrent, false, onTheHour, mediumDead, true, false, weekAfterTheHour(), true, false, 1, 0},
			"prev ran but done, long overdue, past short deadline, api.AllowConcurrent":  {api.AllowConcurrent, false, onTheHour, shortDead, true, false, weekAfterTheHour(), true, false, 1, 0},

			"prev ran but done, long overdue, past medium deadline, api.ReplaceConcurrent": {api.ReplaceConcurrent, false, onTheHour, mediumDead, true, false, weekAfterTheHour(), true, false, 1, 0},
			"prev ran but done, long overdue, past short deadline, api.ReplaceConcurrent":  {api.ReplaceConcurrent, false, onTheHour, shortDead, true, false, weekAfterTheHour(), true, false, 1, 0},

			"prev ran but done, long overdue, past medium deadline, false": {api.ForbidConcurrent, false, onTheHour, mediumDead, true, false, weekAfterTheHour(), true, false, 1, 0},
			"prev ran but done, long overdue, past short deadline, false":  {api.ForbidConcurrent, false, onTheHour, shortDead, true, false, weekAfterTheHour(), true, false, 1, 0},
		}
		nsCount := 0
		for name := range testCases {
			// avoid iteration variable issues
			tc := testCases[name]
			currentNS := nsCount
			nsCount++
			It(name, func() {
				ctx := context.Background()
				now = tc.now

				By("creating the namespace")
				nsName := fmt.Sprintf("cronjob-controller-test-%v", currentNS)
				Expect(cl.Create(ctx, &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: nsName,
					},
				})).To(Succeed())
				defer func() {
					By("deleting the namespace")
					Expect(cl.Delete(ctx, &corev1.Namespace{
						ObjectMeta: metav1.ObjectMeta{
							Name: nsName,
						},
					})).To(Succeed())
				}()

				By("creating the cronjob")
				cronJob := cronJob()
				cronJob.Namespace = nsName
				cronJob.Spec.ConcurrencyPolicy = tc.concurrencyPolicy
				cronJob.Spec.Suspend = &tc.suspend
				cronJob.Spec.Schedule = tc.schedule
				if tc.deadline != noDead {
					cronJob.Spec.StartingDeadlineSeconds = &tc.deadline
				}

				var (
					job *api.Job
					err error
				)
				jobs := []api.Job{}
				if tc.ranPreviously {
					cronJob.ObjectMeta.CreationTimestamp = metav1.Time{Time: justBeforeThePriorHour()}
					cronJob.Status.LastScheduleTime = &metav1.Time{Time: justAfterThePriorHour()}
				} else {
					cronJob.ObjectMeta.CreationTimestamp = metav1.Time{Time: justBeforeTheHour()}
					Expect(tc.stillActive).To(BeFalse(), "this test case makes no sense")
				}
				Expect(cl.Create(ctx, &cronJob)).To(Succeed())

				if tc.ranPreviously {
					By("creating the existing job")
					job, err = cronJobController.ConstructJobForCronJob(&cronJob, cronJob.Status.LastScheduleTime.Time)
					Expect(err).NotTo(HaveOccurred())
					jobs = append(jobs, *job)
					Expect(cl.Create(ctx, job)).To(Succeed())
					if tc.stillActive {
						By("setting cronjob active status")
						Eventually(func() error {
							Expect(cl.Get(ctx, types.NamespacedName{Name: cronJob.Name, Namespace: cronJob.Namespace}, &cronJob)).To(Succeed())
							cronJob.Status.Active = []corev1.ObjectReference{{UID: job.UID}}
							return cl.Status().Update(ctx, &cronJob)
						}).Should(Succeed())
					} else {
						job.Status.Conditions = append(job.Status.Conditions, api.JobCondition{
							Type: api.JobComplete,
							Status: corev1.ConditionTrue,
						})
						Expect(cl.Status().Update(ctx, job)).To(Succeed())
					}
				}
				// TODO: still around vs still active?

				expectedCreates := 0
				if tc.expectCreate {
					expectedCreates = 1
				}
				expectedDeletes := 0
				if tc.expectDelete {
					expectedDeletes = 1
				}
				var outJobs api.JobList
				Eventually(func() []api.Job {
					Expect(cl.List(ctx, &outJobs, client.InNamespace(cronJob.Namespace))).To(Succeed())
					return outJobs.Items
				}).Should(HaveLen(len(jobs)+expectedCreates-expectedDeletes))
				// TODO: this doesn't actually measure what we want

				for _, job := range outJobs.Items {
					controllerRef := metav1.GetControllerOf(&job)
					Expect(controllerRef).NotTo(BeNil())
					definitelyTrue := true
					Expect(controllerRef).To(Equal(&metav1.OwnerReference{
						APIVersion: api.GroupVersion.String(),
						Kind: "CronJob",
						Name: cronJob.Name,
						UID: cronJob.UID,
						Controller: &definitelyTrue,
						BlockOwnerDeletion: &definitelyTrue,
					}))
				}

				Eventually(func() []corev1.ObjectReference {
					Expect(cl.Get(ctx, types.NamespacedName{Namespace: cronJob.Namespace, Name: cronJob.Name}, &cronJob)).To(Succeed())
					return cronJob.Status.Active
				}).Should(HaveLen(tc.expectActive))
			})
		}
	})
})

/*type CleanupJobSpec struct {
	StartTime           string
	IsFinished          bool
	IsSuccessful        bool
	ExpectDelete        bool
	IsStillInActiveList bool // only when IsFinished is set
}

func TestCleanupFinishedJobs_DeleteOrNot(t *testing.T) {
	limitThree := int32(3)
	limitTwo := int32(2)
	limitOne := int32(1)
	limitZero := int32(0)

	// Starting times are assumed to be sorted by increasing start time
	// in all the test cases
	testCases := map[string]struct {
		jobSpecs                   []CleanupJobSpec
		now                        time.Time
		successfulJobsHistoryLimit *int32
		failedJobsHistoryLimit     *int32
		expectActive               int
	}{
		"success. job limit reached": {
			[]CleanupJobSpec{
				{"2016-05-19T04:00:00Z", T, T, T, F},
				{"2016-05-19T05:00:00Z", T, T, T, F},
				{"2016-05-19T06:00:00Z", T, T, F, F},
				{"2016-05-19T07:00:00Z", T, T, F, F},
				{"2016-05-19T08:00:00Z", F, F, F, F},
				{"2016-05-19T09:00:00Z", T, F, F, F},
			}, justBeforeTheHour(), &limitTwo, &limitOne, 1},

		"success. jobs not processed by Sync yet": {
			[]CleanupJobSpec{
				{"2016-05-19T04:00:00Z", T, T, T, F},
				{"2016-05-19T05:00:00Z", T, T, T, T},
				{"2016-05-19T06:00:00Z", T, T, F, T},
				{"2016-05-19T07:00:00Z", T, T, F, T},
				{"2016-05-19T08:00:00Z", F, F, F, F},
				{"2016-05-19T09:00:00Z", T, F, F, T},
			}, justBeforeTheHour(), &limitTwo, &limitOne, 4},

		"failed job limit reached": {
			[]CleanupJobSpec{
				{"2016-05-19T04:00:00Z", T, F, T, F},
				{"2016-05-19T05:00:00Z", T, F, T, F},
				{"2016-05-19T06:00:00Z", T, T, F, F},
				{"2016-05-19T07:00:00Z", T, T, F, F},
				{"2016-05-19T08:00:00Z", T, F, F, F},
				{"2016-05-19T09:00:00Z", T, F, F, F},
			}, justBeforeTheHour(), &limitTwo, &limitTwo, 0},

		"success. job limit set to zero": {
			[]CleanupJobSpec{
				{"2016-05-19T04:00:00Z", T, T, T, F},
				{"2016-05-19T05:00:00Z", T, F, T, F},
				{"2016-05-19T06:00:00Z", T, T, T, F},
				{"2016-05-19T07:00:00Z", T, T, T, F},
				{"2016-05-19T08:00:00Z", F, F, F, F},
				{"2016-05-19T09:00:00Z", T, F, F, F},
			}, justBeforeTheHour(), &limitZero, &limitOne, 1},

		"failed job limit set to zero": {
			[]CleanupJobSpec{
				{"2016-05-19T04:00:00Z", T, T, F, F},
				{"2016-05-19T05:00:00Z", T, F, T, F},
				{"2016-05-19T06:00:00Z", T, T, F, F},
				{"2016-05-19T07:00:00Z", T, T, F, F},
				{"2016-05-19T08:00:00Z", F, F, F, F},
				{"2016-05-19T09:00:00Z", T, F, T, F},
			}, justBeforeTheHour(), &limitThree, &limitZero, 1},

		"no limits reached": {
			[]CleanupJobSpec{
				{"2016-05-19T04:00:00Z", T, T, F, F},
				{"2016-05-19T05:00:00Z", T, F, F, F},
				{"2016-05-19T06:00:00Z", T, T, F, F},
				{"2016-05-19T07:00:00Z", T, T, F, F},
				{"2016-05-19T08:00:00Z", T, F, F, F},
				{"2016-05-19T09:00:00Z", T, F, F, F},
			}, justBeforeTheHour(), &limitThree, &limitThree, 0},

		// This test case should trigger the short-circuit
		"limits disabled": {
			[]CleanupJobSpec{
				{"2016-05-19T04:00:00Z", T, T, F, F},
				{"2016-05-19T05:00:00Z", T, F, F, F},
				{"2016-05-19T06:00:00Z", T, T, F, F},
				{"2016-05-19T07:00:00Z", T, T, F, F},
				{"2016-05-19T08:00:00Z", T, F, F, F},
				{"2016-05-19T09:00:00Z", T, F, F, F},
			}, justBeforeTheHour(), nil, nil, 0},

		"success limit disabled": {
			[]CleanupJobSpec{
				{"2016-05-19T04:00:00Z", T, T, F, F},
				{"2016-05-19T05:00:00Z", T, F, F, F},
				{"2016-05-19T06:00:00Z", T, T, F, F},
				{"2016-05-19T07:00:00Z", T, T, F, F},
				{"2016-05-19T08:00:00Z", T, F, F, F},
				{"2016-05-19T09:00:00Z", T, F, F, F},
			}, justBeforeTheHour(), nil, &limitThree, 0},

		"failure limit disabled": {
			[]CleanupJobSpec{
				{"2016-05-19T04:00:00Z", T, T, F, F},
				{"2016-05-19T05:00:00Z", T, F, F, F},
				{"2016-05-19T06:00:00Z", T, T, F, F},
				{"2016-05-19T07:00:00Z", T, T, F, F},
				{"2016-05-19T08:00:00Z", T, F, F, F},
				{"2016-05-19T09:00:00Z", T, F, F, F},
			}, justBeforeTheHour(), &limitThree, nil, 0},

		"no limits reached because still active": {
			[]CleanupJobSpec{
				{"2016-05-19T04:00:00Z", F, F, F, F},
				{"2016-05-19T05:00:00Z", F, F, F, F},
				{"2016-05-19T06:00:00Z", F, F, F, F},
				{"2016-05-19T07:00:00Z", F, F, F, F},
				{"2016-05-19T08:00:00Z", F, F, F, F},
				{"2016-05-19T09:00:00Z", F, F, F, F},
			}, justBeforeTheHour(), &limitZero, &limitZero, 6},
	}

	for name, tc := range testCases {
		cronJob := cronJob()
		suspend := false
		cronJob.Spec.ConcurrencyPolicy = api.ForbidConcurrent
		cronJob.Spec.Suspend = &suspend
		cronJob.Spec.Schedule = onTheHour

		cronJob.Spec.SuccessfulJobsHistoryLimit = tc.successfulJobsHistoryLimit
		cronJob.Spec.FailedJobsHistoryLimit = tc.failedJobsHistoryLimit

		var (
			job *batchv1.Job
			err error
		)

		// Set consistent timestamps for the CronJob
		if len(tc.jobSpecs) != 0 {
			firstTime := startTimeStringToTime(tc.jobSpecs[0].StartTime)
			lastTime := startTimeStringToTime(tc.jobSpecs[len(tc.jobSpecs)-1].StartTime)
			cronJob.ObjectMeta.CreationTimestamp = metav1.Time{Time: firstTime}
			cronJob.Status.LastScheduleTime = &metav1.Time{Time: lastTime}
		} else {
			cronJob.ObjectMeta.CreationTimestamp = metav1.Time{Time: justBeforeTheHour()}
		}

		// Create jobs
		js := []batchv1.Job{}
		jobsToDelete := sets.NewString()
		cronJob.Status.Active = []v1.ObjectReference{}

		for i, spec := range tc.jobSpecs {
			job, err = getJobFromTemplate(&cronJob, startTimeStringToTime(spec.StartTime))
			if err != nil {
				t.Fatalf("%s: unexpected error creating a job from template: %v", name, err)
			}

			job.UID = types.UID(strconv.Itoa(i))
			job.Namespace = ""

			if spec.IsFinished {
				var conditionType batchv1.JobConditionType
				if spec.IsSuccessful {
					conditionType = batchv1.JobComplete
				} else {
					conditionType = batchv1.JobFailed
				}
				condition := batchv1.JobCondition{Type: conditionType, Status: v1.ConditionTrue}
				job.Status.Conditions = append(job.Status.Conditions, condition)

				if spec.IsStillInActiveList {
					cronJob.Status.Active = append(cronJob.Status.Active, v1.ObjectReference{UID: job.UID})
				}
			} else {
				if spec.IsSuccessful || spec.IsStillInActiveList {
					t.Errorf("%s: test setup error: this case makes no sense", name)
				}
				cronJob.Status.Active = append(cronJob.Status.Active, v1.ObjectReference{UID: job.UID})
			}

			js = append(js, *job)
			if spec.ExpectDelete {
				jobsToDelete.Insert(job.Name)
			}
		}

		jc := &fakeJobControl{Job: job}
		sjc := &fakeSJControl{}
		recorder := record.NewFakeRecorder(10)

		cleanupFinishedJobs(&cronJob, js, jc, sjc, recorder)

		// Check we have actually deleted the correct jobs
		if len(jc.DeleteJobName) != len(jobsToDelete) {
			t.Errorf("%s: expected %d job deleted, actually %d", name, len(jobsToDelete), len(jc.DeleteJobName))
		} else {
			jcDeleteJobName := sets.NewString(jc.DeleteJobName...)
			if !jcDeleteJobName.Equal(jobsToDelete) {
				t.Errorf("%s: expected jobs: %v deleted, actually: %v deleted", name, jobsToDelete, jcDeleteJobName)
			}
		}

		// Check for events
		expectedEvents := len(jobsToDelete)
		if name == "failed list pod err" {
			expectedEvents = len(tc.jobSpecs)
		}
		if len(recorder.Events) != expectedEvents {
			t.Errorf("%s: expected %d event, actually %v", name, expectedEvents, len(recorder.Events))
		}

		// Check for jobs still in active list
		numActive := 0
		if len(sjc.Updates) != 0 {
			numActive = len(sjc.Updates[len(sjc.Updates)-1].Status.Active)
		}
		if tc.expectActive != numActive {
			t.Errorf("%s: expected Active size %d, got %d", name, tc.expectActive, numActive)
		}
	}
}

// TODO: simulation where the controller randomly doesn't run, and randomly has errors starting jobs or deleting jobs,
// but over time, all jobs run as expected (assuming Allow and no deadline).

// TestSyncOne_Status tests cronJob.UpdateStatus in syncOne
func TestSyncOne_Status(t *testing.T) {
	finishedJob := newJob("1")
	finishedJob.Status.Conditions = append(finishedJob.Status.Conditions, batchv1.JobCondition{Type: batchv1.JobComplete, Status: v1.ConditionTrue})
	unexpectedJob := newJob("2")
	missingJob := newJob("3")

	testCases := map[string]struct {
		// cronJob spec
		concurrencyPolicy batchV1beta1.ConcurrencyPolicy
		suspend           bool
		schedule          string
		deadline          int64

		// cronJob status
		ranPreviously  bool
		hasFinishedJob bool

		// environment
		now              time.Time
		hasUnexpectedJob bool
		hasMissingJob    bool
		beingDeleted     bool

		// expectations
		expectCreate bool
		expectDelete bool
	}{
		"never ran, not time, api.AllowConcurrent":                {api.AllowConcurrent, false, onTheHour, noDead, false, false, justBeforeTheHour(), false, false, false, false, false},
		"never ran, not time, false":                {api.ForbidConcurrent, false, onTheHour, noDead, false, false, justBeforeTheHour(), false, false, false, false, false},
		"never ran, not time, api.ReplaceConcurrent":                {api.ReplaceConcurrent, false, onTheHour, noDead, false, false, justBeforeTheHour(), false, false, false, false, false},
		"never ran, is time, api.AllowConcurrent":                 {api.AllowConcurrent, false, onTheHour, noDead, false, false, justAfterTheHour(), false, false, false, true, false},
		"never ran, is time, false":                 {api.ForbidConcurrent, false, onTheHour, noDead, false, false, justAfterTheHour(), false, false, false, true, false},
		"never ran, is time, api.ReplaceConcurrent":                 {api.ReplaceConcurrent, false, onTheHour, noDead, false, false, justAfterTheHour(), false, false, false, true, false},
		"never ran, is time, deleting":          {api.AllowConcurrent, false, onTheHour, noDead, false, false, justAfterTheHour(), false, false, true, false, false},
		"never ran, is time, suspended":         {api.AllowConcurrent, true, onTheHour, noDead, false, false, justAfterTheHour(), false, false, false, false, false},
		"never ran, is time, past deadline":     {api.AllowConcurrent, false, onTheHour, shortDead, false, false, justAfterTheHour(), false, false, false, false, false},
		"never ran, is time, not past deadline": {api.AllowConcurrent, false, onTheHour, longDead, false, false, justAfterTheHour(), false, false, false, true, false},

		"prev ran but done, not time, api.AllowConcurrent":                                            {api.AllowConcurrent, false, onTheHour, noDead, true, false, justBeforeTheHour(), false, false, false, false, false},
		"prev ran but done, not time, finished job, api.AllowConcurrent":                              {api.AllowConcurrent, false, onTheHour, noDead, true, true, justBeforeTheHour(), false, false, false, false, false},
		"prev ran but done, not time, unexpected job, api.AllowConcurrent":                            {api.AllowConcurrent, false, onTheHour, noDead, true, false, justBeforeTheHour(), true, false, false, false, false},
		"prev ran but done, not time, missing job, api.AllowConcurrent":                               {api.AllowConcurrent, false, onTheHour, noDead, true, false, justBeforeTheHour(), false, true, false, false, false},
		"prev ran but done, not time, missing job, unexpected job, api.AllowConcurrent":               {api.AllowConcurrent, false, onTheHour, noDead, true, false, justBeforeTheHour(), true, true, false, false, false},
		"prev ran but done, not time, finished job, unexpected job, api.AllowConcurrent":              {api.AllowConcurrent, false, onTheHour, noDead, true, true, justBeforeTheHour(), true, false, false, false, false},
		"prev ran but done, not time, finished job, missing job, api.AllowConcurrent":                 {api.AllowConcurrent, false, onTheHour, noDead, true, true, justBeforeTheHour(), false, true, false, false, false},
		"prev ran but done, not time, finished job, missing job, unexpected job, api.AllowConcurrent": {api.AllowConcurrent, false, onTheHour, noDead, true, true, justBeforeTheHour(), true, true, false, false, false},
		"prev ran but done, not time, finished job, false":                              {api.ForbidConcurrent, false, onTheHour, noDead, true, true, justBeforeTheHour(), false, false, false, false, false},
		"prev ran but done, not time, missing job, false":                               {api.ForbidConcurrent, false, onTheHour, noDead, true, false, justBeforeTheHour(), false, true, false, false, false},
		"prev ran but done, not time, finished job, missing job, false":                 {api.ForbidConcurrent, false, onTheHour, noDead, true, true, justBeforeTheHour(), false, true, false, false, false},
		"prev ran but done, not time, unexpected job, api.ReplaceConcurrent":                            {api.ReplaceConcurrent, false, onTheHour, noDead, true, false, justBeforeTheHour(), true, false, false, false, false},

		"prev ran but done, is time, api.AllowConcurrent":                                               {api.AllowConcurrent, false, onTheHour, noDead, true, false, justAfterTheHour(), false, false, false, true, false},
		"prev ran but done, is time, finished job, api.AllowConcurrent":                                 {api.AllowConcurrent, false, onTheHour, noDead, true, true, justAfterTheHour(), false, false, false, true, false},
		"prev ran but done, is time, unexpected job, api.AllowConcurrent":                               {api.AllowConcurrent, false, onTheHour, noDead, true, false, justAfterTheHour(), true, false, false, true, false},
		"prev ran but done, is time, finished job, unexpected job, api.AllowConcurrent":                 {api.AllowConcurrent, false, onTheHour, noDead, true, true, justAfterTheHour(), true, false, false, true, false},
		"prev ran but done, is time, false":                                               {api.ForbidConcurrent, false, onTheHour, noDead, true, false, justAfterTheHour(), false, false, false, true, false},
		"prev ran but done, is time, finished job, false":                                 {api.ForbidConcurrent, false, onTheHour, noDead, true, true, justAfterTheHour(), false, false, false, true, false},
		"prev ran but done, is time, unexpected job, false":                               {api.ForbidConcurrent, false, onTheHour, noDead, true, false, justAfterTheHour(), true, false, false, true, false},
		"prev ran but done, is time, finished job, unexpected job, false":                 {api.ForbidConcurrent, false, onTheHour, noDead, true, true, justAfterTheHour(), true, false, false, true, false},
		"prev ran but done, is time, api.ReplaceConcurrent":                                               {api.ReplaceConcurrent, false, onTheHour, noDead, true, false, justAfterTheHour(), false, false, false, true, false},
		"prev ran but done, is time, finished job, api.ReplaceConcurrent":                                 {api.ReplaceConcurrent, false, onTheHour, noDead, true, true, justAfterTheHour(), false, false, false, true, false},
		"prev ran but done, is time, unexpected job, api.ReplaceConcurrent":                               {api.ReplaceConcurrent, false, onTheHour, noDead, true, false, justAfterTheHour(), true, false, false, true, false},
		"prev ran but done, is time, finished job, unexpected job, api.ReplaceConcurrent":                 {api.ReplaceConcurrent, false, onTheHour, noDead, true, true, justAfterTheHour(), true, false, false, true, false},
		"prev ran but done, is time, deleting":                                        {api.AllowConcurrent, false, onTheHour, noDead, true, false, justAfterTheHour(), false, false, true, false, false},
		"prev ran but done, is time, suspended":                                       {api.AllowConcurrent, true, onTheHour, noDead, true, false, justAfterTheHour(), false, false, false, false, false},
		"prev ran but done, is time, finished job, suspended":                         {api.AllowConcurrent, true, onTheHour, noDead, true, true, justAfterTheHour(), false, false, false, false, false},
		"prev ran but done, is time, unexpected job, suspended":                       {api.AllowConcurrent, true, onTheHour, noDead, true, false, justAfterTheHour(), true, false, false, false, false},
		"prev ran but done, is time, finished job, unexpected job, suspended":         {api.AllowConcurrent, true, onTheHour, noDead, true, true, justAfterTheHour(), true, false, false, false, false},
		"prev ran but done, is time, past deadline":                                   {api.AllowConcurrent, false, onTheHour, shortDead, true, false, justAfterTheHour(), false, false, false, false, false},
		"prev ran but done, is time, finished job, past deadline":                     {api.AllowConcurrent, false, onTheHour, shortDead, true, true, justAfterTheHour(), false, false, false, false, false},
		"prev ran but done, is time, unexpected job, past deadline":                   {api.AllowConcurrent, false, onTheHour, shortDead, true, false, justAfterTheHour(), true, false, false, false, false},
		"prev ran but done, is time, finished job, unexpected job, past deadline":     {api.AllowConcurrent, false, onTheHour, shortDead, true, true, justAfterTheHour(), true, false, false, false, false},
		"prev ran but done, is time, not past deadline":                               {api.AllowConcurrent, false, onTheHour, longDead, true, false, justAfterTheHour(), false, false, false, true, false},
		"prev ran but done, is time, finished job, not past deadline":                 {api.AllowConcurrent, false, onTheHour, longDead, true, true, justAfterTheHour(), false, false, false, true, false},
		"prev ran but done, is time, unexpected job, not past deadline":               {api.AllowConcurrent, false, onTheHour, longDead, true, false, justAfterTheHour(), true, false, false, true, false},
		"prev ran but done, is time, finished job, unexpected job, not past deadline": {api.AllowConcurrent, false, onTheHour, longDead, true, true, justAfterTheHour(), true, false, false, true, false},
	}

	for name, tc := range testCases {
		// Setup the test
		cronJob := cronJob()
		cronJob.Spec.ConcurrencyPolicy = tc.concurrencyPolicy
		cronJob.Spec.Suspend = &tc.suspend
		cronJob.Spec.Schedule = tc.schedule
		if tc.deadline != noDead {
			cronJob.Spec.StartingDeadlineSeconds = &tc.deadline
		}
		if tc.ranPreviously {
			cronJob.ObjectMeta.CreationTimestamp = metav1.Time{Time: justBeforeThePriorHour()}
			cronJob.Status.LastScheduleTime = &metav1.Time{Time: justAfterThePriorHour()}
		} else {
			if tc.hasFinishedJob || tc.hasUnexpectedJob || tc.hasMissingJob {
				t.Errorf("%s: test setup error: this case makes no sense", name)
			}
			cronJob.ObjectMeta.CreationTimestamp = metav1.Time{Time: justBeforeTheHour()}
		}
		jobs := []batchv1.Job{}
		if tc.hasFinishedJob {
			ref, err := getRef(&finishedJob)
			if err != nil {
				t.Errorf("%s: test setup error: failed to get job's ref: %v.", name, err)
			}
			cronJob.Status.Active = []v1.ObjectReference{*ref}
			jobs = append(jobs, finishedJob)
		}
		if tc.hasUnexpectedJob {
			jobs = append(jobs, unexpectedJob)
		}
		if tc.hasMissingJob {
			ref, err := getRef(&missingJob)
			if err != nil {
				t.Errorf("%s: test setup error: failed to get job's ref: %v.", name, err)
			}
			cronJob.Status.Active = append(cronJob.Status.Active, *ref)
		}
		if tc.beingDeleted {
			timestamp := metav1.NewTime(tc.now)
			cronJob.DeletionTimestamp = &timestamp
		}

		jc := &fakeJobControl{}
		sjc := &fakeSJControl{}
		recorder := record.NewFakeRecorder(10)

		// Run the code
		syncOne(&cronJob, jobs, tc.now, jc, sjc, recorder)

		// Status update happens once when ranging through job list, and another one if create jobs.
		expectUpdates := 1
		// Events happens when there's unexpected / finished jobs, and upon job creation / deletion.
		expectedEvents := 0
		if tc.expectCreate {
			expectUpdates++
			expectedEvents++
		}
		if tc.expectDelete {
			expectedEvents++
		}
		if tc.hasFinishedJob {
			expectedEvents++
		}
		if tc.hasUnexpectedJob {
			expectedEvents++
		}
		if tc.hasMissingJob {
			expectedEvents++
		}

		if len(recorder.Events) != expectedEvents {
			t.Errorf("%s: expected %d event, actually %v: %#v", name, expectedEvents, len(recorder.Events), recorder.Events)
		}

		if expectUpdates != len(sjc.Updates) {
			t.Errorf("%s: expected %d status updates, actually %d", name, expectUpdates, len(sjc.Updates))
		}

		if tc.hasFinishedJob && inActiveList(sjc.Updates[0], finishedJob.UID) {
			t.Errorf("%s: expected finished job removed from active list, actually active list = %#v", name, sjc.Updates[0].Status.Active)
		}

		if tc.hasUnexpectedJob && inActiveList(sjc.Updates[0], unexpectedJob.UID) {
			t.Errorf("%s: expected unexpected job not added to active list, actually active list = %#v", name, sjc.Updates[0].Status.Active)
		}

		if tc.hasMissingJob && inActiveList(sjc.Updates[0], missingJob.UID) {
			t.Errorf("%s: expected missing job to be removed from active list, actually active list = %#v", name, sjc.Updates[0].Status.Active)
		}

		if tc.expectCreate && !sjc.Updates[1].Status.LastScheduleTime.Time.Equal(topOfTheHour()) {
			t.Errorf("%s: expected LastScheduleTime updated to %s, got %s", name, topOfTheHour(), sjc.Updates[1].Status.LastScheduleTime)
		}
	}
}
*/
