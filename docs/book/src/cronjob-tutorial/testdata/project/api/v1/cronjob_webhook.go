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

package v1

import (
	"github.com/robfig/cron"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	validationutils "k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// +kubebuilder:docs-gen:collapse=Go imports

/*
接下来，我们为 webhooks 配置一个日志记录器。
*/

var cronjoblog = logf.Log.WithName("cronjob-resource")

/*
然后，我们将 webhook 和 manager 关联起来。
*/

func (r *CronJob) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

/*
请注意我们用 kubebuilder 标记去生成 webhook 清单。
这个标记负责生成一个 mutating webhook 清单。

每个标记的意义可参考[这里](/reference/markers/webhook.md)。
*/

// +kubebuilder:webhook:path=/mutate-batch-tutorial-kubebuilder-io-v1-cronjob,mutating=true,failurePolicy=fail,groups=batch.tutorial.kubebuilder.io,resources=cronjobs,verbs=create;update,versions=v1,name=mcronjob.kb.io

/*
我们使用 `webhook.Defaulter` 接口给我们的 CRD 设置默认值。
webhook 会自动调用这个默认值。

`Default` 方法期待修改接收者，设置默认值。
*/

var _ webhook.Defaulter = &CronJob{}

// Default 实现了 webhook.Defaulter ，因此将为该类型注册一个webhook。
func (r *CronJob) Default() {
	cronjoblog.Info("default", "name", r.Name)

	if r.Spec.ConcurrencyPolicy == "" {
		r.Spec.ConcurrencyPolicy = AllowConcurrent
	}
	if r.Spec.Suspend == nil {
		r.Spec.Suspend = new(bool)
	}
	if r.Spec.SuccessfulJobsHistoryLimit == nil {
		r.Spec.SuccessfulJobsHistoryLimit = new(int32)
		*r.Spec.SuccessfulJobsHistoryLimit = 3
	}
	if r.Spec.FailedJobsHistoryLimit == nil {
		r.Spec.FailedJobsHistoryLimit = new(int32)
		*r.Spec.FailedJobsHistoryLimit = 1
	}
}

/*
这个标记负责生成一个 validating webhook 清单。
*/

// TODO(user): 如果你要想开启删除验证，请将 verbs 修改为 "verbs=create;update;delete" 。
// +kubebuilder:webhook:verbs=create;update,path=/validate-batch-tutorial-kubebuilder-io-v1-cronjob,mutating=false,failurePolicy=fail,groups=batch.tutorial.kubebuilder.io,resources=cronjobs,versions=v1,name=vcronjob.kb.io

/*
用声明式验证来验证我们的 CRD 。一般来说，声明式验证应该就足够了，但是有时对于复杂的验证需要
更高级的用例。

例如，下面我们将看到，我们使用这个来验证格式良好的 cron 调度，而不需要构造一个很长的正则表达式。

如果实现了 `webhook.Validator` 接口并调用了这个验证，webhook 将会自动被服务。

`ValidateCreate`, `ValidateUpdate` 和 `ValidateDelete` 方法期望在创建、更新和删除时
分别验证其接收者。我们将 ValidateCreate 从 ValidateUpdate 分离开来以允许某些行为，像
使某些字段不可变，以至于仅可以在创建的时候去设置它们。我们也将 ValidateDelete 从 
ValidateUpdate 分离开来以允许在删除的时候的不同验证行为。然而，这里我们只对 `ValidateCreate` 
和 `ValidateUpdate` 用相同的共享验证。在 `ValidateDelete` 不做任何事情，因为我们不需要再
删除的时候做任何验证。
*/

var _ webhook.Validator = &CronJob{}

// ValidateCreate 实现了 webhook.Validator，因此将为该类型注册一个webhook。
func (r *CronJob) ValidateCreate() error {
	cronjoblog.Info("validate create", "name", r.Name)

	return r.validateCronJob()
}

// ValidateUpdate 实现了 webhook.Validator，因此将为该类型注册一个webhook。
func (r *CronJob) ValidateUpdate(old runtime.Object) error {
	cronjoblog.Info("validate update", "name", r.Name)

	return r.validateCronJob()
}

// ValidateDelete 实现了 webhook.Validator，因此将为该类型注册一个webhook。
func (r *CronJob) ValidateDelete() error {
	cronjoblog.Info("validate delete", "name", r.Name)

	// TODO(user): 填写删除对象时你的验证逻辑。
	return nil
}

/*
我们验证 CronJob 的 spec 和 name 。
*/

func (r *CronJob) validateCronJob() error {
	var allErrs field.ErrorList
	if err := r.validateCronJobName(); err != nil {
		allErrs = append(allErrs, err)
	}
	if err := r.validateCronJobSpec(); err != nil {
		allErrs = append(allErrs, err)
	}
	if len(allErrs) == 0 {
		return nil
	}

	return apierrors.NewInvalid(
		schema.GroupKind{Group: "batch.tutorial.kubebuilder.io", Kind: "CronJob"},
		r.Name, allErrs)
}

/*
OpenAPI schema 声明性地验证一些字段。
你可以在 [API](api-design.md) 中发现 kubebuilder 验证标记(前缀是 `// +kubebuilder:validation`)。
你可以通过运行 `controller-gen crd -w` 或者 [这里](/reference/markers/crd-validation.md) 查找到
kubebuilder支持的用于声明验证的所有标记。
*/

func (r *CronJob) validateCronJobSpec() *field.Error {
	// kubernetes API machinery 的字段助手会帮助我们很好地返回结构化的验证错误。
	return validateScheduleFormat(
		r.Spec.Schedule,
		field.NewPath("spec").Child("schedule"))
}

/*
我们将需要验证 [cron](https://en.wikipedia.org/wiki/Cron) 调度是否有良好的格式。
*/

func validateScheduleFormat(schedule string, fldPath *field.Path) *field.Error {
	if _, err := cron.ParseStandard(schedule); err != nil {
		return field.Invalid(fldPath, schedule, err.Error())
	}
	return nil
}

/*
验证 schema 可以声明性地验证字符串字段的长度。

但是 `ObjectMeta.Name` 字段定义在 apimachinery 仓库下的共享的包中，所以
我们不能用验证 schema 声明性地验证它。
*/

func (r *CronJob) validateCronJobName() *field.Error {
	if len(r.ObjectMeta.Name) > validationutils.DNS1035LabelMaxLength-11 {
		// job 的名字长度像所有 Kubernetes 对象一样是是 63 字符(必须适合 DNS 子域)。
		// 在创建 job 的时候，cronjob 的控制器会添加一个 11 字符的后缀(`-$TIMESTAMP`)。
		// job 的名字长度限制在 63 字符。因此 cronjob 的名字的长度一定小于等于 63-11=52 。
		// 如果这里我们没有进行验证，后面当job创建的时候就会失败。
		return field.Invalid(field.NewPath("metadata").Child("name"), r.Name, "must be no more than 52 characters")
	}
	return nil
}

// +kubebuilder:docs-gen:collapse=Validate object name
