
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

{% method %}

## The actual controller

To start out our controller, we'll need some imports.  The scaffolding will
put some of these in place, but we'll need to add the rest.

Most of the dependencies from controller-runtime can be found in the base
`sigs.k8.io/controller-runtime` package, like before.  Since controllers
generally need to fetch and update objects, we'll also need the
`sigs.k8s.io/controller-runtime/pkg/client` package.

Otherwise, we need several packages from the Kubernetes API machinery (e.g.
to refer to `Scheme` like before), and several packages specifically for our
usecases (like `github.com/robfig/cron` to parse cron specifications).

```go

package controllers

import (
	"context"
	"time"
	"fmt"
	"sort"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ref "k8s.io/client-go/tools/reference"
	corev1 "k8s.io/api/core/v1"
	"github.com/robfig/cron"
	apierrs "k8s.io/apimachinery/pkg/api/errors"

	"sigs.k8s.io/kubebuilder/test/project/api/v1"
)

```
{%% endmethod %%}

{% method %}

## Defining the reconciler

Next, we'll need to define the struct that implements our reconciler.  Since we're
working with objects, we'll need a client to access them.  The client provided by the
manager always reads from the cache, and writes to the API server, which is how the core
Kubernetes controllers work as well.

We'll also need a logging handle to log to and a scheme (needed to help set owner references).

Finally, we'll use an interface to fake out our clock, to make it easier to jump around in time
when testing our controller.

```go

type CronJobReconciler struct {
	client.Client
	Log logr.Logger
	scheme *runtime.Scheme
	Clock
}

```
{%% endmethod %%}

{% method %}

## Setup and Initialization

We called `SetupWithManager` from main.go to initialize our controller, so now
we actually have to implement that.  Every controller in has one "root" Kind of
object that it cares about, and can have other "child" objects that it manages,
or associated objects that affect how it functions. Any created objects or
associated objects first have to be mapped back to the set of "root" objects
that they're associated with.

In our case, our "root" object is `CronJob`, and we create and manage `Job`
objects, so we say that the controller is `For()` `CronJob` (which says that we
care directly about `CronJobs`) and `Owns()` some `Job` objects (which says
that we care about `Job` objects owned by `CronJob`s, and that we should map
those back to `CronJob`s by `OwnerReference`).  Each of these methods are
helpers on top of a lower-level framework in controller-runtime for arbitrary
associations, but controllers can often get by using just `For` and `Owns`.

Since we'll need to find all `Job` objects owned by a particular `CronJob`,
we'll want to set up an *index* in our cache.  Indexes simply map some object
to a set of keys (in this case, the owning `CronJob`).  The indexer
automatically namespaces the keys by the indexed object (`Job`), and since
owner references can't cross namespaces, we can just use the `CronJob` name as
our key.

TODO: come up with an excuse to show `Watches`.

```go


var (
	jobOwnerKey = ".metadata.controller"
	apiGVStr = v1.GroupVersion.String()
)

func (r *CronJobReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.Client = mgr.GetClient()
	r.scheme = mgr.GetScheme()
	if r.Clock == nil {
		r.Clock = realClock{}
	}

	if err := mgr.GetFieldIndexer().IndexField(&v1.Job{}, jobOwnerKey, func(rawObj runtime.Object) []string {
		job := rawObj.(*v1.Job)
		owner := metav1.GetControllerOf(job)
		if owner == nil {
			return nil
		}
		if owner.APIVersion != apiGVStr || owner.Kind != "CronJob" {
			return nil
		}

		return []string{owner.Name}
	}); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&v1.CronJob{}).
		Owns(&v1.Job{}).
		Complete(r)
}

```
{%% endmethod %%}

{% method %}

## The actual reconcile function

Most reconcile functions, ours included, follow the same basic pattern:

- fetch the root object
- calculate the state of any associated object
- change the state of the world and/or update the status

Here, we'll update the status first and then change the state of the world, requeing
to recalculate our status afterwards.  This lets us more easily only update the status
if a given CronJob is marked as suspended, and return early if we fail to create a job,
etc.

We'll also set up a context for all of our client calls (we could use this to time out operations,
if we wanted.


```go

var (
	// scheduledTimeAnnotation stores the scheduled time of launch for a particular
	// Job created by a CronJob, so that we can look at it later.
	scheduledTimeAnnotation = "tutorial.kubebuilder.io/scheduled-time"
)

func (r *CronJobReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("cronjob", req.NamespacedName)

	var cronJob v1.CronJob
	if err := r.Get(ctx, req.NamespacedName, &cronJob); err != nil {
		log.Error(err, "unable to fetch CronJob")
		return ctrl.Result{}, ignoreNotFound(err)
	}

	// first, we'll ensure our status is up to date

```
{%% endmethod %%}

{% method %}

All reconcilers should be idempotent and not depend on which action (create, update, delete)
triggered the reconcile.  This is especially important since a create and update event might
be coalesced into a single reconcile call.  Thus, generally, we need to figure out the state
of all the things that we care about each iteration.

In our case, we care about the state of all the Jobs that we've nominally created.  Thankfully,
we've set up an index to help us out.

```go

	// fetch the child jobs
	var childJobs v1.JobList
	if err := r.List(ctx, &childJobs, client.InNamespace(req.Namespace), client.MatchingField(jobOwnerKey, req.Name)); err != nil {
		log.Error(err, "unable to list child Jobs")
		return ctrl.Result{}, err
	}

```
{%% endmethod %%}

{% method %}

Once we have all the jobs we own, we'll split them into active, successfull,
and failed jobs, keeping track of the most recent run so that we can record it
in status.  Remember, status should be able to be reconstituted from the state
of the world, so it's generally not a good idea to read from the status of the
root object.  Instead, you should reconstruct it every run.  That's what we'll
do here.

We can check if a job is "finished" and whether is succeeded or failed using status
conditions.  We'll put that logic in a helper to make our code cleaner.


```go


	// find the active list of jobs
	var activeJobs []*v1.Job
	var successfulJobs []*v1.Job
	var failedJobs []*v1.Job
	var mostRecentTime *time.Time  // find the last run so we can update the status

	for i, job := range childJobs.Items {
		_, finishedType := isJobFinished(&job)
		switch finishedType {
		case "": // ongoing
			activeJobs = append(activeJobs, &childJobs.Items[i])
		case v1.JobFailed:
			failedJobs = append(failedJobs, &childJobs.Items[i])
		case v1.JobComplete:
			successfulJobs = append(successfulJobs, &childJobs.Items[i])
		}

```
<a name="return-from-func-isJobFinished"></a>
[func isJobFinished](#jump-to-func-isJobFinished)
{%% endmethod %%}

{% method %}

We've stored the launch time in an annotation, so we'll reconsitute that from
the active jobs themselves.


```go

		scheduledTimeForJob, err := getScheduledTimeForJob(&job)
		if err != nil {
			log.Error(err, "unable to parse schedule time for child job", "job", &job)
			continue
		}
		if scheduledTimeForJob != nil {
			if mostRecentTime == nil {
				mostRecentTime = scheduledTimeForJob
			} else if mostRecentTime.Before(*scheduledTimeForJob) {
				mostRecentTime = scheduledTimeForJob
			}
		}
	}
	log.V(1).Info("job count", "active jobs", len(activeJobs), "successful jobs", len(successfulJobs), "failed jobs", len(failedJobs))
	// TODO(directxman12): log on orphaned job/completed job?

	// record the status
	if mostRecentTime != nil {
		cronJob.Status.LastScheduleTime = &metav1.Time{Time: *mostRecentTime}
	} else {
		cronJob.Status.LastScheduleTime = nil
	}
	cronJob.Status.Active = nil
	for _, activeJob := range activeJobs {
		jobRef, err := ref.GetReference(r.scheme, activeJob)
		if err != nil {
			log.Error(err, "unable to make reference to active job", "job", activeJob)
			continue
		}
		cronJob.Status.Active = append(cronJob.Status.Active, *jobRef)
	}
```
<a name="return-from-func-getScheduledTimeForJob"></a>
[func getScheduledTimeForJob](#jump-to-func-getScheduledTimeForJob)
{%% endmethod %%}

{% method %}

In order to actually update the Status object, we'll have to use the status client.
When the status subresource is enabled on CRDs, Kubernetes ignores status when
submitting updates to the main endpoint, and ignores spec when submitting updates
to the status subresource.

```go

	if err := r.Status().Update(ctx, &cronJob); err != nil {
		log.Error(err, "unable to update CronJob status")
		return ctrl.Result{}, err
	}

```
{%% endmethod %%}

{% method %}

Once we've updated our status, we can move on to ensuring that the status of
the world matches what we want in our spec.

First, we'll try to clean up old jobs, so that we don't leave too many lying
around.

```go


	// then, we'll clean up jobs according to our history limits
	// NB: deleting these is "best effort" -- if we fail on a particular one,
	// we won't requeue just to finish the deleting.
	if cronJob.Spec.FailedJobsHistoryLimit != nil {
		sort.Slice(failedJobs, func(i, j int) bool {
			if failedJobs[i].Status.StartTime == nil {
				return failedJobs[j].Status.StartTime != nil
			}
			return failedJobs[i].Status.StartTime.Before(failedJobs[j].Status.StartTime)
		})
		for i, job := range failedJobs {
			if err := r.Delete(ctx, job); err != nil {
				log.Error(err, "unable to delete old failed job", "job", job)
			}
			if int32(i) >= *cronJob.Spec.FailedJobsHistoryLimit {
				break
			}
		}
	}

	if cronJob.Spec.SuccessfulJobsHistoryLimit != nil {
		sort.Slice(successfulJobs, func(i, j int) bool {
			if successfulJobs[i].Status.StartTime == nil {
				return successfulJobs[j].Status.StartTime != nil
			}
			return successfulJobs[i].Status.StartTime.Before(successfulJobs[j].Status.StartTime)
		})
		for i, job := range successfulJobs {
			if err := r.Delete(ctx, job); err != nil {
				log.Error(err, "unable to delete old successful job", "job", job)
			}
			if int32(i) >= *cronJob.Spec.SuccessfulJobsHistoryLimit {
				break
			}
		}
	}

```
{%% endmethod %%}

{% method %}

Then, we'll check and see if any jobs should be running.  If the CronJob is suspended,
we shouldn't do anything.

```go


	// get on with the actual reconciling

	// check to see if we're supposed to do anything
	// (user might've wanted a pause to putz with stuff)
	if cronJob.Spec.Suspend != nil && *cronJob.Spec.Suspend {
		log.V(1).Info("cronjob suspended, skipping")
		return ctrl.Result{}, nil
	}

```
{%% endmethod %%}

{% method %}

Otherwise, we'll need to calculate any runs that are past due (in which case we
might need to run a job), as well as the next run, so that we know how long we
need to wait until checking again.


```go


	// figure out the next times that we need to create
	// jobs at (or anything we missed).
	missedRuns, nextRun, err := getNextSchedule(&cronJob, r.Now())
	if err != nil {
		log.Error(err, "unable to figure out CronJob schedule")
		// we don't really care about requeuing until we get an update that fixes the schedule,
		// so don't return an error
		return ctrl.Result{}, nil
	}

```
<a name="return-from-func-getNextSchedule"></a>
[func getNextSchedule](#jump-to-func-getNextSchedule)
{%% endmethod %%}

{% method %}

We'll prep our eventual request to requeue until the next job, and then figure
out if we actually need to run.  If we've missed a run, and we're still within
the deadline to start it, we'll need to run a job.

```go


	scheduledResult := ctrl.Result{RequeueAfter: nextRun.Sub(r.Now())} // save this so we can re-use it elsewhere
	log = log.WithValues("now", r.Now(), "next run", nextRun)

	if len(missedRuns) == 0 {
		log.V(1).Info("no upcoming scheduled times, sleeping until next")
		return scheduledResult, nil
	}

	if len(missedRuns) > 1 {
		log.V(1).Info("multiple missed scheduled runs, on running most recent")
	}


	// make sure we're not too late to start the run
	lastMissedRun := missedRuns[len(missedRuns)-1]
	log = log.WithValues("current run", lastMissedRun)
	tooLate := false
	if cronJob.Spec.StartingDeadlineSeconds != nil {
		tooLate = lastMissedRun.Add(time.Duration(*cronJob.Spec.StartingDeadlineSeconds)*time.Second).Before(r.Now())
	}
	if tooLate {
		log.V(1).Info("missed starting deadline for last run, sleeping till next")
		// TODO(directxman12): events
		return scheduledResult, nil
	}

```
{%% endmethod %%}

{% method %}

If we actually have to run a job, we'll need to either wait till existing ones finish,
replace the existing ones, or just add new ones.  If our information is out of date due
to cache delay, we'll get a requeue when we get up-to-date information.

```go

	// figure out how to run this job -- concurrency policy might forbid us from running
	// multiple at the same time...
	if cronJob.Spec.ConcurrencyPolicy == v1.ForbidConcurrent && len(activeJobs) > 0 {
		log.V(1).Info("concurrency policy blocks concurrent runs, skipping", "num active", len(activeJobs))
		return scheduledResult, nil
	}

	// ...or instruct us to replace existing ones...
	if cronJob.Spec.ConcurrencyPolicy == v1.ReplaceConcurrent {
		for _, activeJob := range activeJobs {
			// we don't care if the job was already deleted
			if err := r.Delete(ctx, activeJob); ignoreNotFound(err) != nil {
				log.Error(err, "unable to delete active job", "job", activeJob)
				return ctrl.Result{}, err
			}
		}
	}

```
{%% endmethod %%}

{% method %}

Once we've figured out what to do with existing jobs, we'll actually create our desired job
object.


```go

	// actually make the job
	job, err := r.ConstructJobForCronJob(&cronJob, lastMissedRun)
	if err != nil {
		log.Error(err, "unable to construct job from template")
		// don't bother requeuing until we get a change to the spec
		return scheduledResult, nil
	}

```
<a name="return-from-func--r--CronJobReconciler--ConstructJobForCronJob"></a>
[func (r *CronJobReconciler) ConstructJobForCronJob](#jump-to-func--r--CronJobReconciler--ConstructJobForCronJob)
{%% endmethod %%}

{% method %}

Finally, we'll create our Job, and ask to requeue in time to check for our next run.

```go

	if err := r.Create(ctx, job); err != nil {
		log.Error(err, "unable to create Job for CronJob", "job", job)
		return ctrl.Result{}, err
	}

	log.V(1).Info("created Job for CronJob run", "job", job)

	// we'll requeue once we see the running job, and update our status
	return scheduledResult, nil
}

```
{%% endmethod %%}

{% method %}

We generally want to ignore (not requeue) on NotFound errors,
since we'll get a reconcile request once the object becomes found,
and requeuing in the mean time won't help.

```go

func ignoreNotFound(err error) error {
	if apierrs.IsNotFound(err) {
		return err
	}
	return nil
}

```
{%% endmethod %%}

{% method %}

We'll mock out the clock to make it easier to jump around in time while testing
The "real" clock just calls time.Now.

```go

type realClock struct{}
func (_ realClock) Now() time.Time { return time.Now() }

// clock knows how to get the current time.
// It can be used to fake out timing for testing.
type Clock interface {
	Now() time.Time
}

// TODO(directxman12): have a "now to launch the controller" section.

```
{%% endmethod %%}

{% method %}

We need to construct a job based on our CronJob's template.  We'll copy over the spec
from the template and copy some basic object meta.

Then, we'll set the "scheduled time" annotation so that we can reconstitute our
`LastScheduleTime` field each reconcile.

Finally, we'll need to set an owner reference.  This allows the Kubernetes garbage collector
to clean up jobs when we delete the CronJob, and allows controller-runtime to figure out
which cronjob needs to be reconciled when a given job changes (is added, deleted, completes, etc).


```go

func (r *CronJobReconciler) ConstructJobForCronJob(cronJob *v1.CronJob, scheduledTime time.Time) (*v1.Job, error) {
	// We want job names for a given nominal start time to have a deterministic name to avoid the same job being created twice
	name := fmt.Sprintf("%s-%d", cronJob.Name, scheduledTime.Unix())

	job := &v1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Labels:          make(map[string]string),
			Annotations:     make(map[string]string),
			Name:            name,
			Namespace:       cronJob.Namespace,
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
	if err := ctrl.SetControllerReference(cronJob, job, r.scheme); err != nil {
		return nil, err
	}

	return job, nil
}

```
<a name="jump-to-func--r--CronJobReconciler--ConstructJobForCronJob"></a>
[return](#return-from-func (r *CronJobReconciler) ConstructJobForCronJob)
%!(EXTRA string=func--r--CronJobReconciler--ConstructJobForCronJob){%% endmethod %%}

{% method %}

We'll use a helper to extract the scheduled time from the annotation that
we added during job creation.


```go

func getScheduledTimeForJob(job *v1.Job) (*time.Time, error) {
	timeRaw := job.Annotations[scheduledTimeAnnotation]
	if len(timeRaw) == 0 {
		return nil, nil
	}

	timeParsed, err := time.Parse(time.RFC3339, timeRaw)
	if err != nil {
		return nil, err
	}
	return &timeParsed, nil
}

```
<a name="jump-to-func-getScheduledTimeForJob"></a>
[return](#return-from-func getScheduledTimeForJob)
%!(EXTRA string=func-getScheduledTimeForJob){%% endmethod %%}

{% method %}

We'll calculate the next scheduled time using our helpful cron library.
We'll start calculating appropriate times from our last run, or the creation
of the CronJob if we can't find a last run.

If there are too many missed runs and we don't have any deadlines set, we'll
bail so that we don't cause issues on controller restarts or wedges.

Otherwise, we'll just return the missed runs (of which we'll just use the latest),
and the next run, so that we can know the latest time to reconcile again.


```go


// getNextSchedule fetches any recently missed runs of this cronjob that we might
// want to create a job for, as well as the next run we're expecting to make
// (so that we can wait till then).
func getNextSchedule(cronJob *v1.CronJob, now time.Time) ([]time.Time, time.Time, error) {
	starts := []time.Time{}
	sched, err := cron.ParseStandard(cronJob.Spec.Schedule)
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("Unparseable schedule %q: %v", cronJob.Spec.Schedule, err)
	}

	var earliestTime time.Time
	if cronJob.Status.LastScheduleTime != nil {
		earliestTime = cronJob.Status.LastScheduleTime.Time
	} else {
		// If none found, then this is either a recently created scheduledJob,
		// or the active/completed info was somehow lost (contract for status
		// in kubernetes says it may need to be recreated), or that we have
		// started a job, but have not noticed it yet (distributed systems can
		// have arbitrary delays).  In any case, use the creation time of the
		// CronJob as last known start time.
		earliestTime = cronJob.ObjectMeta.CreationTimestamp.Time
	}
	if cronJob.Spec.StartingDeadlineSeconds != nil {
		// Controller is not going to schedule anything below this point
		schedulingDeadline := now.Add(-time.Second * time.Duration(*cronJob.Spec.StartingDeadlineSeconds))

		if schedulingDeadline.After(earliestTime) {
			earliestTime = schedulingDeadline
		}
	}
	if earliestTime.After(now) {
		return nil, sched.Next(now), nil
	}

	for t := sched.Next(earliestTime); !t.After(now); t = sched.Next(t) {
		starts = append(starts, t)
		// An object might miss several starts. For example, if
		// controller gets wedged on friday at 5:01pm when everyone has
		// gone home, and someone comes in on tuesday AM and discovers
		// the problem and restarts the controller, then all the hourly
		// jobs, more than 80 of them for one hourly scheduledJob, should
		// all start running with no further intervention (if the scheduledJob
		// allows concurrency and late starts).
		//
		// However, if there is a bug somewhere, or incorrect clock
		// on controller's server or apiservers (for setting creationTimestamp)
		// then there could be so many missed start times (it could be off
		// by decades or more), that it would eat up all the CPU and memory
		// of this controller. In that case, we want to not try to list
		// all the missed start times.
		//
		// I've somewhat arbitrarily picked 100, as more than 80,
		// but less than "lots".
		if len(starts) > 100 {
			// We can't get the most recent times so just return an empty slice
			return nil, time.Time{}, fmt.Errorf("Too many missed start time (> 100). Set or decrease .spec.startingDeadlineSeconds or check clock skew.")
		}
	}
	return starts, sched.Next(now), nil
}

```
<a name="jump-to-func-getNextSchedule"></a>
[return](#return-from-func getNextSchedule)
%!(EXTRA string=func-getNextSchedule){%% endmethod %%}

{% method %}

We consider a job "finished" if it has a "succeeded" or "failed" condition marked as true.
Status conditions allow us to add extensible status information to our objects that other
humans and controllers can examine to check things like completion and health.


```go

// isJobFinished checks if the given job is either complete (successful) or failed.
func isJobFinished(job *v1.Job) (bool, v1.JobConditionType) {
    for _, c := range job.Status.Conditions {
        if (c.Type == v1.JobComplete || c.Type == v1.JobFailed) && c.Status == corev1.ConditionTrue {
            return true, c.Type
        }
    }

	return false, ""
}

```
<a name="jump-to-func-isJobFinished"></a>
[return](#return-from-func isJobFinished)
%!(EXTRA string=func-isJobFinished){%% endmethod %%}

