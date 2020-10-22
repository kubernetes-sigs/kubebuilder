/*
Copyright 2020 The Kubernetes authors.

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
开始编码之前，先引进基本的依赖。除此之外还需要一些额外的依赖库，在使用到它们时，我们再详细介绍。
*/
package controllers

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/go-logr/logr"
	"github.com/robfig/cron"
	kbatch "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ref "k8s.io/client-go/tools/reference"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	batch "tutorial.kubebuilder.io/project/api/v1"
)

/*
接下来，我们需要一个时钟好让我们在测试中模拟计时。
*/

// CronJob 调谐器对 CronJob 对象进行调谐
type CronJobReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
	Clock
}

/*
我们虚拟了一个时钟以便在测试中方便地来回调节时间，
调用`time.Now`获取真实时间
*/
type realClock struct{}

func (_ realClock) Now() time.Time { return time.Now() }

// clock接口可以获取当前的时间
// 可以帮助我们在测试中模拟计时
type Clock interface {
	Now() time.Time
}

// +kubebuilder:docs-gen:collapse=Clock

/*
注意，我们需要获得RBAC权限——我们需要一些额外权限去
创建和管理job，添加如下一些[字段](/reference/markers/rbac.md)
*/

// +kubebuilder:rbac:groups=batch.tutorial.kubebuilder.io,resources=cronjobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=batch.tutorial.kubebuilder.io,resources=cronjobs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=batch,resources=jobs/status,verbs=get

/*
控制器的核心部分——调谐逻辑
*/
var (
	scheduledTimeAnnotation = "batch.tutorial.kubebuilder.io/scheduled-at"
)

func (r *CronJobReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("cronjob", req.NamespacedName)

	/*
		### 1: 根据名称加载定时任务

		通过 client 获取定时任务。所有 client 方法第一个参数都是 context（用于取消定时任务）作为
		第一个参数，把请求对象信息作为最后一个参数。Get 方法例外，它把
		[`NamespacedName`](https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/client?tab=doc#ObjectKey)
		作为中间的第二个参数（大多数方法都没有中间的NamespacedName参数，下文会提到）

		许多client方法的最后一个参数接受变长参数。
	*/
	var cronJob batch.CronJob
	if err := r.Get(ctx, req.NamespacedName, &cronJob); err != nil {
		log.Error(err, "unable to fetch CronJob")
		//忽略掉 not-found 错误，它们不能通过重新排队修复（要等待新的通知）
		//在删除一个不存在的对象时，可能会报这个错误。
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	/*
		### 2: 列出所有有效 job，更新它们的状态

		为确保每个 job 的状态都会被更新到，我们需要列出某个 CronJob 在当前命名空间下的所有 job。
		和 Get 方法类似，我们可以使用 List 方法来列出 CronJob 下所有的 job。注意，我们使用变长参数
		来映射命名空间和任意多个匹配变量（实际上相当于是建立了一个索引）。

	*/
	var childJobs kbatch.JobList
	if err := r.List(ctx, &childJobs, client.InNamespace(req.Namespace), client.MatchingFields{jobOwnerKey: req.Name}); err != nil {
		log.Error(err, "unable to list child Jobs")
		return ctrl.Result{}, err
	}

	/*

		<aside class="note">

		<h1>索引的作用？</h1>


		<p>调谐器会获取 cronjob 下的所有 job 以更新它们状态。随着 cronjob 数量的增加，遍历全部 conjob
		查找会变的相当低效。为了提高查询效率，这些任务会根据控制器名称建立索引。缓存后的 job 对象会
		被添加上一个 jobOwnerKey 字段。这个字段引用其归属控制器和函数作为索引。在下文中，我们将配置
		manager 作为这个字段的索引</p>

		</aside>

		查找到所有的 job 后，将其归类为 active，successful，failed 三种类型，同时持续跟踪其
		最新的执行情况以更新其状态。牢记，status 值应该是从实际的运行状态中实时获取。从 cronjob
		中读取 job 的状态通常不是一个好做法。应该从每次执行状态中获取。我们后续也采用这种方法。

		我们可以检查一个 job 是否已处于 “finished” 状态，使用 status 条件还可以知道它是 succeeded
		或 failed 状态。

	*/
	// 找出所有有效的 job
	var activeJobs []*kbatch.Job
	var successfulJobs []*kbatch.Job
	var failedJobs []*kbatch.Job
	var mostRecentTime *time.Time // 记录其最近一次运行时间以便更新状态

	/*
		当一个 job 被标记为 "succeeded" 或 "failed" 时，我们认为这个任务处于 "finished" 状态。
		Status conditions 允许我们给 job 对象添加额外的状态信息，开发人员或控制器可以通过
		这些校验信息来检查 job 的完成或健康状态。
	*/
	isJobFinished := func(job *kbatch.Job) (bool, kbatch.JobConditionType) {
		for _, c := range job.Status.Conditions {
			if (c.Type == kbatch.JobComplete || c.Type == kbatch.JobFailed) && c.Status == corev1.ConditionTrue {
				return true, c.Type
			}
		}

		return false, ""
	}
	// +kubebuilder:docs-gen:collapse=isJobFinished

	/*
		使用辅助函数来提取创建 job 时注释中排定的执行时间

	*/
	getScheduledTimeForJob := func(job *kbatch.Job) (*time.Time, error) {
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
	// +kubebuilder:docs-gen:collapse=getScheduledTimeForJob

	for i, job := range childJobs.Items {
		_, finishedType := isJobFinished(&job)
		switch finishedType {
		case "": // ongoing
			activeJobs = append(activeJobs, &childJobs.Items[i])
		case kbatch.JobFailed:
			failedJobs = append(failedJobs, &childJobs.Items[i])
		case kbatch.JobComplete:
			successfulJobs = append(successfulJobs, &childJobs.Items[i])
		}

		//将启动时间存放在注释中，当job生效时可以从中读取
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

	if mostRecentTime != nil {
		cronJob.Status.LastScheduleTime = &metav1.Time{Time: *mostRecentTime}
	} else {
		cronJob.Status.LastScheduleTime = nil
	}
	cronJob.Status.Active = nil
	for _, activeJob := range activeJobs {
		jobRef, err := ref.GetReference(r.Scheme, activeJob)
		if err != nil {
			log.Error(err, "unable to make reference to active job", "job", activeJob)
			continue
		}
		cronJob.Status.Active = append(cronJob.Status.Active, *jobRef)
	}

	/*
		此处会记录我们观察到的 job 数量。为便于调试，略微提高日志级别。注意，这里没有使用
		格式化字符串，使用由键值对构成的固定格式信息来输出日志。这样更易于过滤和查询日志

	*/
	log.V(1).Info("job count", "active jobs", len(activeJobs), "successful jobs", len(successfulJobs), "failed jobs", len(failedJobs))

	/*
		使用收集到日期信息来更新 CRD 状态。和之前类似，通过 client 来完成操作。
		针对 status 这一子资源，我们可以使用`Status`部分的`Update`方法。

		status 子资源会忽略掉对 spec 的变更。这与其它更新操作的发生冲突的风险更小，
		而且实现了权限分离。

	*/
	if err := r.Status().Update(ctx, &cronJob); err != nil {
		log.Error(err, "unable to update CronJob status")
		return ctrl.Result{}, err
	}

	/*
		更新状态后，后续要确保状态符合我们在 spec 定下的预期。


		### 3: 根据保留的历史版本数清理过旧的 job


		我们先清理掉一些版本太旧的 job，这样可以不用保留太多无用的 job
	*/

	// 注意: 删除操作采用的“尽力而为”策略
	// 如果个别job删除失败了，不会将其重新排队，直接结束删除操作
	if cronJob.Spec.FailedJobsHistoryLimit != nil {
		sort.Slice(failedJobs, func(i, j int) bool {
			if failedJobs[i].Status.StartTime == nil {
				return failedJobs[j].Status.StartTime != nil
			}
			return failedJobs[i].Status.StartTime.Before(failedJobs[j].Status.StartTime)
		})
		for i, job := range failedJobs {
			if int32(i) >= int32(len(failedJobs))-*cronJob.Spec.FailedJobsHistoryLimit {
				break
			}
			if err := r.Delete(ctx, job, client.PropagationPolicy(metav1.DeletePropagationBackground)); client.IgnoreNotFound(err) != nil {
				log.Error(err, "unable to delete old failed job", "job", job)
			} else {
				log.V(0).Info("deleted old failed job", "job", job)
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
			if int32(i) >= int32(len(successfulJobs))-*cronJob.Spec.SuccessfulJobsHistoryLimit {
				break
			}
			if err := r.Delete(ctx, job, client.PropagationPolicy(metav1.DeletePropagationBackground)); (err) != nil {
				log.Error(err, "unable to delete old successful job", "job", job)
			} else {
				log.V(0).Info("deleted old successful job", "job", job)
			}
		}
	}

	/* ### 4: 检查是否被挂起

	如果当前 cronjob 被挂起，不会再运行其下的任何 job，我们将其停止。这对于某些 job 出现异常
	的排查非常有用。我们无需删除 cronjob 来中止其后续其他 job 的运行。

	*/

	if cronJob.Spec.Suspend != nil && *cronJob.Spec.Suspend {
		log.V(1).Info("cronjob suspended, skipping")
		return ctrl.Result{}, nil
	}

	/*
		### 5: 计算 job 下一次执行时间

		如果 cronjob 没被挂起，则我们需要计算它的下一次执行时间，
		同时检查是否有遗漏的执行没被处理

	*/

	/*
		借助强大的 cron 库，我们可以轻易的计算出定时任务的下一次执行时间。
		我们根据最近一次的执行时间来计算下一次执行时间，如果没有找到最近
		一次执行时间，则根据定时任务的创建时间来计算。

		如果遗漏了很多次执行并且没有为这些执行设置截止时间。我们将其忽略
		以避免异常造成控制器的频繁重启和资源紧张。

		如果遗漏的执行次数并不多，我们返回最近一次的执行时间和下一次将要
		执行的时间，这样我们可以知道该何时去调谐。

	*/
	getNextSchedule := func(cronJob *batch.CronJob, now time.Time) (lastMissed time.Time, next time.Time, err error) {
		sched, err := cron.ParseStandard(cronJob.Spec.Schedule)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("Unparseable schedule %q: %v", cronJob.Spec.Schedule, err)
		}

		// 出于优化的目的，我们可以使用点技巧。从上一次观察到的执行时间开始执行，
		// 这个执行时间可以被在这里被读取。但是意义不大，因为我们刚更新了这个值。

		var earliestTime time.Time
		if cronJob.Status.LastScheduleTime != nil {
			earliestTime = cronJob.Status.LastScheduleTime.Time
		} else {
			earliestTime = cronJob.ObjectMeta.CreationTimestamp.Time
		}
		if cronJob.Spec.StartingDeadlineSeconds != nil {
			// 如果开始执行时间超过了截止时间，不再执行
			schedulingDeadline := now.Add(-time.Second * time.Duration(*cronJob.Spec.StartingDeadlineSeconds))

			if schedulingDeadline.After(earliestTime) {
				earliestTime = schedulingDeadline
			}
		}
		if earliestTime.After(now) {
			return time.Time{}, sched.Next(now), nil
		}

		starts := 0
		for t := sched.Next(earliestTime); !t.After(now); t = sched.Next(t) {
			lastMissed = t
			// 一个 CronJob 可能会遗漏多次执行。举个例子，周五5:00pm技术人员下班后，
			// 控制器在5:01pm发生了异常。然后直到周二早上才有技术人员发现问题并
			// 重启控制器。那么所有的以1小时为周期执行的定时任务，在没有技术人员
			// 进一步的干预下，都会有80多个 job 在恢复正常后一并启动（如果 job 允许
			// 多并发和延迟启动）

			// 如果 CronJob 的某些地方出现异常，控制器或 apiservers (用于设置任务创建时间)
			// 的时钟不正确, 那么就有可能出现错过很多次执行时间的情形（跨度可达数十年）
			// 这将会占满控制器的CPU和内存资源。这种情况下，我们不需要列出错过的全部
			// 执行时间。

			starts++
			if starts > 100 {
				// 获取不到最近一次执行时间，直接返回空切片
				return time.Time{}, time.Time{}, fmt.Errorf("Too many missed start times (> 100). Set or decrease .spec.startingDeadlineSeconds or check clock skew.")
			}
		}
		return lastMissed, sched.Next(now), nil
	}
	// +kubebuilder:docs-gen:collapse=getNextSchedule

	// 计算出定时任务下一次执行时间（或是遗漏的执行时间）
	missedRun, nextRun, err := getNextSchedule(&cronJob, r.Now())
	if err != nil {
		log.Error(err, "unable to figure out CronJob schedule")
		// 重新排队直到有更新修复这次定时任务调度，不必返回错误
		return ctrl.Result{}, nil
	}

	/*
		上述步骤执行完后，将准备好的请求加入队列直到下次执行，
		然后确定这些 job 是否要实际执行

	*/
	scheduledResult := ctrl.Result{RequeueAfter: nextRun.Sub(r.Now())} // 保存以便别处复用
	log = log.WithValues("now", r.Now(), "next run", nextRun)

	/*
		### 6: 如果 job 符合执行时机，并且没有超出截止时间，且不被并发策略阻塞，执行该 job

		如果 job 遗漏了一次执行，且还没超出截止时间，把遗漏的这次执行也不上

	*/
	if missedRun.IsZero() {
		log.V(1).Info("no upcoming scheduled times, sleeping until next")
		return scheduledResult, nil
	}

	// 确保错过的执行没有超过截止时间
	log = log.WithValues("current run", missedRun)
	tooLate := false
	if cronJob.Spec.StartingDeadlineSeconds != nil {
		tooLate = missedRun.Add(time.Duration(*cronJob.Spec.StartingDeadlineSeconds) * time.Second).Before(r.Now())
	}
	if tooLate {
		log.V(1).Info("missed starting deadline for last run, sleeping till next")
		// TODO(directxman12): events
		return scheduledResult, nil
	}

	/*
		如果确认 job 需要实际执行。我们有三种策略执行该 job。要么先等待现有的 job 执行完后，在启动本次 job；
		或是直接覆盖取代现有的job；或是不考虑现有的 job，直接作为新的 job 执行。因为缓存导致的信息有所延迟，
		当更新信息后需要重新排队。

	*/
	// 确定要 job 的执行策略 —— 并发策略可能禁止多个job同时运行
	if cronJob.Spec.ConcurrencyPolicy == batch.ForbidConcurrent && len(activeJobs) > 0 {
		log.V(1).Info("concurrency policy blocks concurrent runs, skipping", "num active", len(activeJobs))
		return scheduledResult, nil
	}

	// 直接覆盖现有 job
	if cronJob.Spec.ConcurrencyPolicy == batch.ReplaceConcurrent {
		for _, activeJob := range activeJobs {
			// we don't care if the job was already deleted
			if err := r.Delete(ctx, activeJob, client.PropagationPolicy(metav1.DeletePropagationBackground)); client.IgnoreNotFound(err) != nil {
				log.Error(err, "unable to delete active job", "job", activeJob)
				return ctrl.Result{}, err
			}
		}
	}

	/*
		确定如何处理现有 job 后，创建符合我们预期的 job
	*/

	/*
		基于 CronJob 模版构建 job，从模板复制 spec 及对象的元信息。

		然后在注解中设置执行时间，这样我们可以在每次的调谐中获取起作为“上一次执行时间”

		最后，还需要设置 owner reference字段。当我们删除 CronJob 时，Kubernetes 垃圾收集
		器会根据这个字段对应的 job。同时，当某个job状态发生变更（创建，删除，完成）时，
		controller-runtime 可以根据这个字段识别出要对那个 CronJob 进行调谐。

	*/
	constructJobForCronJob := func(cronJob *batch.CronJob, scheduledTime time.Time) (*kbatch.Job, error) {
		// job 名称带上执行时间以确保唯一性，避免排定执行时间的 job 创建两次
		name := fmt.Sprintf("%s-%d", cronJob.Name, scheduledTime.Unix())

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
		if err := ctrl.SetControllerReference(cronJob, job, r.Scheme); err != nil {
			return nil, err
		}

		return job, nil
	}
	// +kubebuilder:docs-gen:collapse=constructJobForCronJob

	// 构建 job
	job, err := constructJobForCronJob(&cronJob, missedRun)
	if err != nil {
		log.Error(err, "unable to construct job from template")
		// job 的 spec 没有变更，无需重新排队
		return scheduledResult, nil
	}

	// ...在集群中创建 job
	if err := r.Create(ctx, job); err != nil {
		log.Error(err, "unable to create Job for CronJob", "job", job)
		return ctrl.Result{}, err
	}

	log.V(1).Info("created Job for CronJob run", "job", job)

	/*
		### 7: 当 job 开始运行或到了 job 下一次的执行时间，重新排队

		最终我们返回上述预备的结果。我们还需重新排队当任务还有下一次执行时。
		这被视作最长截止时间——如果期间发生了变更，例如 job 被提前启动或是提前
		结束，或被修改，我们可能会更早进行调谐。

	*/
	// 当有 job 进入运行状态后，重新排队，同时更新状态
	return scheduledResult, nil
}

/*
### 启动 CronJob 控制器

最后，我们还要完善下我们的启动过程。为了让调谐器可以通过 job 的 owner 值快速找到 job。
我们需要一个索引。声明一个索引键，后续我们可以将其用于 client 的虚拟变量名中，从 job
对象中提取索引值。此处的索引会帮我们处理好 namespaces 的映射关系。所以如果 job 有 owner
值，我们快速地获取 owner 值。

另外，我们需要告知 manager，这个控制器拥有哪些 job。当对应的 job 发生变更或被删除时，
自动调用调谐器对 CronJob 进行调谐。

*/
var (
	jobOwnerKey = ".metadata.controller"
	apiGVStr    = batch.GroupVersion.String()
)

func (r *CronJobReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// 此处不是测试，我们需要创建一个真实的时钟
	if r.Clock == nil {
		r.Clock = realClock{}
	}

	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &kbatch.Job{}, jobOwnerKey, func(rawObj runtime.Object) []string {
		//获取 job 对象，提取 owner...
		job := rawObj.(*kbatch.Job)
		owner := metav1.GetControllerOf(job)
		if owner == nil {
			return nil
		}
		// ...确保 owner 是个 CronJob...
		if owner.APIVersion != apiGVStr || owner.Kind != "CronJob" {
			return nil
		}

		// ...是 CronJob，返回
		return []string{owner.Name}
	}); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&batch.CronJob{}).
		Owns(&kbatch.Job{}).
		Complete(r)
}
