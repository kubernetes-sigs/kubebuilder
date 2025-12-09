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

package multiversion

import (
	log "log/slog"
	"os/exec"
	"path/filepath"

	hackutils "sigs.k8s.io/kubebuilder/v4/hack/docs/utils"
	pluginutil "sigs.k8s.io/kubebuilder/v4/pkg/plugin/util"
	"sigs.k8s.io/kubebuilder/v4/test/e2e/utils"
)

// Sample define the sample which will be scaffolded
type Sample struct {
	ctx *utils.TestContext
}

// NewSample create a new instance of the sample and configure the KB CLI that will be used
func NewSample(binaryPath, samplePath string) Sample {
	log.Info("Generating the sample context of MultiVersion Cronjob...")
	ctx := hackutils.NewSampleContext(binaryPath, samplePath, "GO111MODULE=on")
	return Sample{&ctx}
}

// Prepare the Context for the sample project
func (sp *Sample) Prepare() {
	log.Info("refreshing tools and creating directory for multiversion ...")
	err := sp.ctx.Prepare()
	hackutils.CheckError("creating directory for multiversion project", err)
}

// GenerateSampleProject will generate the sample
func (sp *Sample) GenerateSampleProject() {
	log.Info("Initializing the multiversion cronjob project")

	log.Info("Creating v2 API")
	err := sp.ctx.CreateAPI(
		"--group", "batch",
		"--version", "v2",
		"--kind", "CronJob",
		"--resource=true",
		"--controller=false",
	)
	hackutils.CheckError("Creating the v2 API without controller", err)

	log.Info("Creating conversion webhook for v1")
	err = sp.ctx.CreateWebhook(
		"--group", "batch",
		"--version", "v1",
		"--kind", "CronJob",
		"--conversion",
		"--spoke", "v2",
		"--force",
	)
	hackutils.CheckError("Creating conversion webhook for v1", err)

	log.Info("Workaround to fix the issue with the conversion webhook")
	// FIXME: This is a workaround to fix the issue with the conversion webhook
	// We should be able to inject the code when we create webhooks with different
	// types of webhooks. However, currently, we are not able to do that and we need to
	// force. So, we are copying the code from cronjob tutorial to have the code
	// implemented.
	cmd := exec.Command("cp", "./../../../cronjob-tutorial/testdata/project/internal/webhook/v1/cronjob_webhook.go", "./internal/webhook/v1/cronjob_webhook.go")
	_, err = sp.ctx.Run(cmd)
	hackutils.CheckError("Copying the code from cronjob tutorial", err)

	log.Info("Creating defaulting and validation webhook for v2")
	err = sp.ctx.CreateWebhook(
		"--group", "batch",
		"--version", "v2",
		"--kind", "CronJob",
		"--defaulting",
		"--programmatic-validation",
	)
	hackutils.CheckError("Creating defaulting and validation webhook for v2", err)

	err = pluginutil.InsertCode(
		filepath.Join(sp.ctx.Dir, "internal/webhook/v1/cronjob_webhook.go"),
		`// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as this struct is used only for temporary operations and does not need to be deeply copied.
type CronJobCustomValidator struct {`,
		`// +kubebuilder:docs-gen:collapse=Remaining Webhook Code`)
	hackutils.CheckError("adding marker collapse", err)
}

// UpdateTutorial the muilt-version sample tutorial with the scaffold changes
func (sp *Sample) UpdateTutorial() {
	log.Info("Update tutorial with multiversion code")

	// Update files according to the multiversion
	sp.updateCronjobV1DueForce()
	sp.updateAPIV1()
	sp.updateAPIV2()
	sp.updateWebhookV2()

	path := "internal/webhook/v1/cronjob_webhook_test.go"
	err := pluginutil.InsertCode(filepath.Join(sp.ctx.Dir, path),
		`// TODO (user): Add any additional imports if needed`,
		`
	"k8s.io/utils/ptr"`)
	hackutils.CheckError("add import for webhook tests", err)

	sp.updateConversionFiles()
	sp.updateSampleV2()
	sp.updateMain()
	sp.updateE2EWebhookConversion()
}

func (sp *Sample) updateCronjobV1DueForce() {
	// FIXME : This is a workaround to fix the issue with the conversion webhook
	path := "internal/webhook/v1/cronjob_webhook.go"
	err := pluginutil.ReplaceInFile(filepath.Join(sp.ctx.Dir, path),
		"Then, we set up the webhook with the manager.",
		`This setup doubles as setup for our conversion webhooks: as long as our
types implement the
[Hub](https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/conversion?tab=doc#Hub) and
[Convertible](https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/conversion?tab=doc#Convertible)
interfaces, a conversion webhook will be registered.
`)
	hackutils.CheckError("manager fix doc comment", err)

	path = "internal/webhook/v1/cronjob_webhook_test.go"
	err = pluginutil.ReplaceInFile(filepath.Join(sp.ctx.Dir, path),
		`var (
		obj    *batchv1.CronJob
		oldObj *batchv1.CronJob
	)

	BeforeEach(func() {
		obj = &batchv1.CronJob{}
		oldObj = &batchv1.CronJob{}
		Expect(oldObj).NotTo(BeNil(), "Expected oldObj to be initialized")
		Expect(obj).NotTo(BeNil(), "Expected obj to be initialized")
		// TODO (user): Add any setup logic common to all tests
	})`,
		`var (
		obj       *batchv1.CronJob
		oldObj    *batchv1.CronJob
		validator CronJobCustomValidator
		defaulter CronJobCustomDefaulter
	)
	const validCronJobName = "valid-cronjob-name"
	const schedule =  "*/5 * * * *"

	BeforeEach(func() {
		obj = &batchv1.CronJob{
			Spec: batchv1.CronJobSpec{
				Schedule:                   schedule,
				ConcurrencyPolicy:          batchv1.AllowConcurrent,
				SuccessfulJobsHistoryLimit: ptr.To(int32(3)),
				FailedJobsHistoryLimit:     ptr.To(int32(1)),
			},
		}
		*obj.Spec.SuccessfulJobsHistoryLimit = 3
		*obj.Spec.FailedJobsHistoryLimit = 1

		oldObj = &batchv1.CronJob{
			Spec: batchv1.CronJobSpec{
				Schedule:                   schedule,
				ConcurrencyPolicy:          batchv1.AllowConcurrent,
				SuccessfulJobsHistoryLimit: ptr.To(int32(3)),
				FailedJobsHistoryLimit:     ptr.To(int32(1)),
			},
		}
		*oldObj.Spec.SuccessfulJobsHistoryLimit = 3
		*oldObj.Spec.FailedJobsHistoryLimit = 1

		validator = CronJobCustomValidator{}
		defaulter = CronJobCustomDefaulter{
			DefaultConcurrencyPolicy:          batchv1.AllowConcurrent,
			DefaultSuspend:                    false,
			DefaultSuccessfulJobsHistoryLimit: 3,
			DefaultFailedJobsHistoryLimit:     1,
		}

		Expect(obj).NotTo(BeNil(), "Expected obj to be initialized")
		Expect(oldObj).NotTo(BeNil(), "Expected oldObj to be initialized")
	})`)
	hackutils.CheckError("fix cronjob v1 tests", err)

	err = pluginutil.InsertCode(filepath.Join(sp.ctx.Dir, path),
		`AfterEach(func() {
		// TODO (user): Add any teardown logic common to all tests
	})

	`,
		`Context("When creating CronJob under Defaulting Webhook", func() {
		It("Should apply defaults when a required field is empty", func() {
			By("simulating a scenario where defaults should be applied")
			obj.Spec.ConcurrencyPolicy = ""           // This should default to AllowConcurrent
			obj.Spec.Suspend = nil                    // This should default to false
			obj.Spec.SuccessfulJobsHistoryLimit = nil // This should default to 3
			obj.Spec.FailedJobsHistoryLimit = nil     // This should default to 1

			By("calling the Default method to apply defaults")
			_ = defaulter.Default(ctx, obj)

			By("checking that the default values are set")
			Expect(obj.Spec.ConcurrencyPolicy).To(Equal(batchv1.AllowConcurrent), "Expected ConcurrencyPolicy to default to AllowConcurrent")
			Expect(*obj.Spec.Suspend).To(BeFalse(), "Expected Suspend to default to false")
			Expect(*obj.Spec.SuccessfulJobsHistoryLimit).To(Equal(int32(3)), "Expected SuccessfulJobsHistoryLimit to default to 3")
			Expect(*obj.Spec.FailedJobsHistoryLimit).To(Equal(int32(1)), "Expected FailedJobsHistoryLimit to default to 1")
		})

		It("Should not overwrite fields that are already set", func() {
			By("setting fields that would normally get a default")
			obj.Spec.ConcurrencyPolicy = batchv1.ForbidConcurrent
			obj.Spec.Suspend = new(bool)
			*obj.Spec.Suspend = true
			obj.Spec.SuccessfulJobsHistoryLimit = new(int32)
			*obj.Spec.SuccessfulJobsHistoryLimit = 5
			obj.Spec.FailedJobsHistoryLimit = new(int32)
			*obj.Spec.FailedJobsHistoryLimit = 2

			By("calling the Default method to apply defaults")
			_ = defaulter.Default(ctx, obj)

			By("checking that the fields were not overwritten")
			Expect(obj.Spec.ConcurrencyPolicy).To(Equal(batchv1.ForbidConcurrent), "Expected ConcurrencyPolicy to retain its set value")
			Expect(*obj.Spec.Suspend).To(BeTrue(), "Expected Suspend to retain its set value")
			Expect(*obj.Spec.SuccessfulJobsHistoryLimit).To(Equal(int32(5)), "Expected SuccessfulJobsHistoryLimit to retain its set value")
			Expect(*obj.Spec.FailedJobsHistoryLimit).To(Equal(int32(2)), "Expected FailedJobsHistoryLimit to retain its set value")
		})
	})

	Context("When creating or updating CronJob under Validating Webhook", func() {
		It("Should deny creation if the name is too long", func() {
			obj.Name = "this-name-is-way-too-long-and-should-fail-validation-because-it-is-way-too-long"
			Expect(validator.ValidateCreate(ctx, obj)).Error().To(
				MatchError(ContainSubstring("must be no more than 52 characters")),
				"Expected name validation to fail for a too-long name")
		})

		It("Should admit creation if the name is valid", func() {
			obj.Name = validCronJobName
			Expect(validator.ValidateCreate(ctx, obj)).To(BeNil(),
				"Expected name validation to pass for a valid name")
		})

		It("Should deny creation if the schedule is invalid", func() {
			obj.Spec.Schedule = "invalid-cron-schedule"
			Expect(validator.ValidateCreate(ctx, obj)).Error().To(
				MatchError(ContainSubstring("Expected exactly 5 fields, found 1: invalid-cron-schedule")),
				"Expected spec validation to fail for an invalid schedule")
		})

		It("Should admit creation if the schedule is valid", func() {
			obj.Spec.Schedule = schedule
			Expect(validator.ValidateCreate(ctx, obj)).To(BeNil(),
				"Expected spec validation to pass for a valid schedule")
		})

		It("Should deny update if both name and spec are invalid", func() {
			oldObj.Name = validCronJobName
			oldObj.Spec.Schedule = schedule

			By("simulating an update")
			obj.Name = "this-name-is-way-too-long-and-should-fail-validation-because-it-is-way-too-long"
			obj.Spec.Schedule = "invalid-cron-schedule"

			By("validating an update")
			Expect(validator.ValidateUpdate(ctx, oldObj, obj)).Error().To(HaveOccurred(),
				"Expected validation to fail for both name and spec")
		})

		It("Should admit update if both name and spec are valid", func() {
			oldObj.Name = validCronJobName
			oldObj.Spec.Schedule = schedule

			By("simulating an update")
			obj.Name = "valid-cronjob-name-updated"
			obj.Spec.Schedule = "0 0 * * *"

			By("validating an update")
			Expect(validator.ValidateUpdate(ctx, oldObj, obj)).To(BeNil(),
				"Expected validation to pass for a valid update")
		})
	})

	`)
	hackutils.CheckError("fix cronjob v1 tests after each", err)

	err = pluginutil.ReplaceInFile(filepath.Join(sp.ctx.Dir, "internal/webhook/v1/cronjob_webhook.go"),
		"// +kubebuilder:docs-gen:collapse=validateCronJobName() Code Implementation",
		``)
	hackutils.CheckError("removing collapse valida for cronjob tutorial", err)
}

func (sp *Sample) updateSampleV2() {
	path := filepath.Join(sp.ctx.Dir, "config/samples/batch_v2_cronjob.yaml")
	oldText := `# TODO(user): Add fields here`

	err := pluginutil.ReplaceInFile(
		path,
		oldText,
		sampleV2Code,
	)
	hackutils.CheckError("replacing TODO with sampleV2Code in batch_v2_cronjob.yaml", err)
}

func (sp *Sample) updateConversionFiles() {
	path := filepath.Join(sp.ctx.Dir, "api/v1/cronjob_conversion.go")

	err := pluginutil.InsertCodeIfNotExist(path,
		"limitations under the License.\n*/",
		"\n// +kubebuilder:docs-gen:collapse=Apache License")
	hackutils.CheckError("appending into hub v1 collapse docs", err)

	err = pluginutil.ReplaceInFile(path,
		"// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!",
		hubV1CodeComment)
	hackutils.CheckError("adding comment to hub v1", err)

	path = filepath.Join(sp.ctx.Dir, "api/v2/cronjob_conversion.go")

	err = pluginutil.InsertCodeIfNotExist(path,
		"limitations under the License.\n*/",
		"\n// +kubebuilder:docs-gen:collapse=Apache License")
	hackutils.CheckError("appending into hub v2 collapse docs", err)

	err = pluginutil.InsertCode(path,
		"import (",
		`
	"fmt"
	"strings"

`)
	hackutils.CheckError("adding imports to hub v2", err)

	err = pluginutil.InsertCodeIfNotExist(path,
		"batchv1 \"tutorial.kubebuilder.io/project/api/v1\"\n)",
		`// +kubebuilder:docs-gen:collapse=Imports

/*
Our "spoke" versions need to implement the
[`+"`"+`Convertible`+"`"+`](https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/conversion?tab=doc#Convertible)
interface. Namely, they'll need `+"`"+`ConvertTo()`+"`"+` and `+"`"+`ConvertFrom()`+"`"+`
methods to convert to/from the hub version.
*/
`)
	hackutils.CheckError("appending into hub v2 collapse docs", err)

	err = pluginutil.ReplaceInFile(path,
		"package v2",
		hubV2CodeComment)
	hackutils.CheckError("adding comment to hub v2", err)

	err = pluginutil.ReplaceInFile(path,
		`// TODO(user): Implement conversion logic from v2 to v1
	// Example: Copying Spec fields
	// dst.Spec.Size = src.Spec.Replicas

	// Copy ObjectMeta to preserve name, namespace, labels, etc.
	dst.ObjectMeta = src.ObjectMeta

	return nil
}`,
		hubV2CovertTo)
	hackutils.CheckError("replace covertTo at hub v2", err)

	err = pluginutil.ReplaceInFile(path,
		`// TODO(user): Implement conversion logic from v1 to v2
	// Example: Copying Spec fields
	// dst.Spec.Replicas = src.Spec.Size

	// Copy ObjectMeta to preserve name, namespace, labels, etc.
	dst.ObjectMeta = src.ObjectMeta

	return nil
}
`,
		hubV2ConvertFromCode)
	hackutils.CheckError("replace covert from at hub v2", err)

	err = pluginutil.ReplaceInFile(path,
		"// ConvertFrom converts the Hub version (v1) to this CronJob (v2).",
		`/*
ConvertFrom is expected to modify its receiver to contain the converted object.
Most of the conversion is straightforward copying, except for converting our changed field.
*/

// ConvertFrom converts the Hub version (v1) to this CronJob (v2).`)
	hackutils.CheckError("replace covert from info at hub v2", err)

	err = pluginutil.ReplaceInFile(path,
		"// ConvertTo converts this CronJob (v2) to the Hub version (v1).",
		`/*
ConvertTo is expected to modify its argument to contain the converted object.
Most of the conversion is straightforward copying, except for converting our changed field.
*/

// ConvertTo converts this CronJob (v2) to the Hub version (v1).`)
	hackutils.CheckError("replace covert info at hub v2", err)
}

func (sp *Sample) updateAPIV1() {
	path := "api/v1/cronjob_types.go"
	err := pluginutil.InsertCode(
		filepath.Join(sp.ctx.Dir, path),
		`// +kubebuilder:subresource:status
`,
		`// +versionName=v1
// +kubebuilder:storageversion`,
	)
	hackutils.CheckError("add version and marker for storage version", err)

	err = pluginutil.ReplaceInFile(
		filepath.Join(sp.ctx.Dir, path),
		cronjobSpecComment,
		"",
	)
	hackutils.CheckError("removing spec explanation", err)

	err = pluginutil.ReplaceInFile(
		filepath.Join(sp.ctx.Dir, path),
		`type CronJob struct {
	/*
	 */`,
		`type CronJob struct {`,
	)
	hackutils.CheckError("removing comment empty from struct", err)

	err = pluginutil.ReplaceInFile(
		filepath.Join(sp.ctx.Dir, path),
		`/*
 Finally, we have the rest of the boilerplate that we've already discussed.
 As previously noted, we don't need to change this, except to mark that
 we want a status subresource, so that we behave like built-in kubernetes types.
*/`,
		``,
	)
	hackutils.CheckError("removing comment from cronjob tutorial", err)

	err = pluginutil.ReplaceInFile(
		filepath.Join(sp.ctx.Dir, path),
		`// +kubebuilder:object:root=true

// CronJobList contains a list of CronJob`,
		`/*
 */

// +kubebuilder:object:root=true

// CronJobList contains a list of CronJob`,
	)
	hackutils.CheckError("add comment empty after struct", err)

	err = pluginutil.ReplaceInFile(
		filepath.Join(sp.ctx.Dir, path),
		concurrencyPolicyComment,
		"",
	)
	hackutils.CheckError("removing concurrency policy explanation", err)

	err = pluginutil.ReplaceInFile(
		filepath.Join(sp.ctx.Dir, path),
		statusDesignComment,
		"",
	)
	hackutils.CheckError("removing status design explanation", err)

	err = pluginutil.ReplaceInFile(
		filepath.Join(sp.ctx.Dir, path),
		`// +kubebuilder:object:root=true
// +kubebuilder:storageversion`,
		boilerplateReplacement,
	)
	hackutils.CheckError("add comment with storage version explanation", err)

	err = pluginutil.ReplaceInFile(
		filepath.Join(sp.ctx.Dir, "api/v1/cronjob_types.go"),
		`// +kubebuilder:docs-gen:collapse=Root Object Definitions`,
		`// +kubebuilder:docs-gen:collapse=Remaining code from cronjob_types.go`,
	)
	hackutils.CheckError("replacing docs-gen collapse comment", err)
}

func (sp *Sample) updateWebhookV2() {
	path := "internal/webhook/v2/cronjob_webhook.go"

	err := pluginutil.InsertCodeIfNotExist(
		filepath.Join(sp.ctx.Dir, path),
		"limitations under the License.\n*/",
		"\n// +kubebuilder:docs-gen:collapse=Apache License")
	hackutils.CheckError("adding Apache License collapse marker to webhook v2", err)

	err = pluginutil.InsertCode(
		filepath.Join(sp.ctx.Dir, path),
		`import (
	"context"
	"fmt"`,
		`
	"strings"

	"github.com/robfig/cron"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	validationutils "k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"`,
	)
	hackutils.CheckError("replacing imports in v2", err)

	err = pluginutil.ReplaceInFile(
		filepath.Join(sp.ctx.Dir, path),
		`// TODO(user): Add more fields as needed for defaulting`,
		cronJobFieldsForDefaulting,
	)
	hackutils.CheckError("replacing defaulting fields in v2", err)

	err = pluginutil.ReplaceInFile(
		filepath.Join(sp.ctx.Dir, path),
		`// TODO(user): fill in your defaulting logic.

	return nil`,
		cronJobDefaultingLogic,
	)
	hackutils.CheckError("replacing defaulting logic in v2", err)

	err = pluginutil.ReplaceInFile(
		filepath.Join(sp.ctx.Dir, path),
		`// TODO(user): fill in your validation logic upon object creation.

	return nil, nil`,
		`return nil, validateCronJob(cronjob)`,
	)
	hackutils.CheckError("replacing validation logic for creation in v2", err)

	err = pluginutil.ReplaceInFile(
		filepath.Join(sp.ctx.Dir, path),
		`// TODO(user): fill in your validation logic upon object update.

	return nil, nil`,
		`return nil, validateCronJob(cronjob)`,
	)
	hackutils.CheckError("replacing validation logic for update in v2", err)

	err = pluginutil.ReplaceInFile(
		filepath.Join(sp.ctx.Dir, path),
		originalSetupManager,
		replaceSetupManager,
	)
	hackutils.CheckError("replacing SetupWebhookWithManager in v2", err)

	err = pluginutil.AppendCodeAtTheEnd(
		filepath.Join(sp.ctx.Dir, path),
		cronJobDefaultFunction,
	)
	hackutils.CheckError("adding Default function in v2", err)

	// Add the validateCronJob function
	err = pluginutil.AppendCodeAtTheEnd(
		filepath.Join(sp.ctx.Dir, path),
		cronJobValidationFunction,
	)
	hackutils.CheckError("adding validateCronJob function in v2", err)
}

func (sp *Sample) updateMain() {
	path := "cmd/main.go"

	err := pluginutil.ReplaceInFile(
		filepath.Join(sp.ctx.Dir, path),
		`"k8s.io/apimachinery/pkg/runtime"`,
		`kbatchv1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/runtime"`,
	)
	hackutils.CheckError("add import main.go", err)

	err = pluginutil.ReplaceInFile(
		filepath.Join(sp.ctx.Dir, path),
		`utilruntime.Must(batchv1.AddToScheme(scheme))`,
		`utilruntime.Must(kbatchv1.AddToScheme(scheme)) // we've added this ourselves
	utilruntime.Must(batchv1.AddToScheme(scheme))`,
	)
	hackutils.CheckError("add schema main.go", err)

	err = pluginutil.InsertCode(
		filepath.Join(sp.ctx.Dir, path),
		`// +kubebuilder:scaffold:scheme
}`,
		`

// +kubebuilder:docs-gen:collapse=existing setup

/*
 */`,
	)
	hackutils.CheckError("insert doc marker existing setup main.go", err)

	err = pluginutil.ReplaceInFile(
		filepath.Join(sp.ctx.Dir, path),
		`if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	// +kubebuilder:docs-gen:collapse=Remaining code from main.go`,
		`if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}`,
	)
	hackutils.CheckError("remove doc marker old staff from main.go", err)

	err = pluginutil.ReplaceInFile(
		filepath.Join(sp.ctx.Dir, path),
		`/*
The first difference to notice is that kubebuilder has added the new API
group's package (`+"`batchv1`"+`) to our scheme.  This means that we can use those
objects in our controller.

If we would be using any other CRD we would have to add their scheme the same way.
Builtin types such as Job have their scheme added by `+"`clientgoscheme`"+`.
*/`,
		"",
	)
	hackutils.CheckError("remove API group explanation main.go", err)

	err = pluginutil.ReplaceInFile(
		filepath.Join(sp.ctx.Dir, path),
		`/*
The other thing that's changed is that kubebuilder has added a block calling our
CronJob controller's `+"`SetupWithManager`"+` method.
*/`,
		"",
	)
	hackutils.CheckError("remove SetupWithManager explanation main.go", err)

	err = pluginutil.InsertCode(
		filepath.Join(sp.ctx.Dir, path),
		`// +kubebuilder:docs-gen:collapse=Imports`,
		`

/*
 */
`,
	)
	hackutils.CheckError("insert comment after import in the main.go", err)

	err = pluginutil.InsertCode(
		filepath.Join(sp.ctx.Dir, path),
		`if err := (&controller.CronJobReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "CronJob")
		os.Exit(1)
	}`,
		`
 `,
	)
	hackutils.CheckError("insert doc marker existing setup main.go", err)

	err = pluginutil.ReplaceInFile(
		filepath.Join(sp.ctx.Dir, path),
		`/*
		We'll also set up webhooks for our type, which we'll talk about next.
		We just need to add them to the manager.  Since we might want to run
		the webhooks separately, or not run them when testing our controller
		locally, we'll put them behind an environment variable.

		We'll just make sure to set `+"`ENABLE_WEBHOOKS=false`"+` when we run locally.
	*/`,
		`/*
		Our existing call to SetupWebhookWithManager registers our conversion webhooks with the manager, too.
	*/`,
	)
	hackutils.CheckError("replace webhook setup explanation main.go", err)
}

func (sp *Sample) updateAPIV2() {
	path := "api/v2/cronjob_types.go"
	err := pluginutil.InsertCode(
		filepath.Join(sp.ctx.Dir, path),
		`// +kubebuilder:subresource:status
`,
		`// +versionName=v2`,
	)
	hackutils.CheckError("add marker version for v2", err)

	err = pluginutil.InsertCode(
		filepath.Join(sp.ctx.Dir, path),
		"limitations under the License.\n*/", // This is the anchor point where we want to insert the code
		`
// +kubebuilder:docs-gen:collapse=Apache License

/*
Since we're in a v2 package, controller-gen will assume this is for the v2
version automatically.  We could override that with the [`+"`+versionName`"+`
marker](/reference/markers/crd.md).
*/`)
	hackutils.CheckError("insert doc marker for license", err)

	err = pluginutil.ReplaceInFile(
		filepath.Join(sp.ctx.Dir, path),
		importV2,
		importReplacement,
	)
	hackutils.CheckError("replace imports v2", err)

	err = pluginutil.ReplaceInFile(
		filepath.Join(sp.ctx.Dir, path),
		`// CronJobSpec defines the desired state of CronJob`,
		`// +kubebuilder:docs-gen:collapse=Imports

/*
We'll leave our spec largely unchanged, except to change the schedule field to a new type.
*/
// CronJobSpec defines the desired state of CronJob`,
	)
	hackutils.CheckError("replace doc about CronjobSpec v2", err)

	err = pluginutil.ReplaceInFile(
		filepath.Join(sp.ctx.Dir, path),
		`// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	// The following markers will use OpenAPI v3 schema to validate the value
	// More info: https://book.kubebuilder.io/reference/markers/crd-validation.html`,
		`// schedule in Cron format, see https://en.wikipedia.org/wiki/Cron.
	// +required
	Schedule CronSchedule `+"`json:\"schedule\"`"+`

	/*
	 */
`,
	)
	hackutils.CheckError("add new schedule spec type", err)

	err = pluginutil.ReplaceInFile(
		filepath.Join(sp.ctx.Dir, path),
		`// foo is an example field of CronJob. Edit cronjob_types.go to remove/update
	// +optional
	Foo *string `+"`json:\"foo,omitempty\"`",
		cronjobSpecMore,
	)
	hackutils.CheckError("replace Foo with cronjob spec fields", err)

	err = pluginutil.ReplaceInFile(
		filepath.Join(sp.ctx.Dir, path),
		`)

}`,
		`)
`,
	)
	hackutils.CheckError("replace Foo with cronjob spec fields", err)

	err = pluginutil.InsertCode(
		filepath.Join(sp.ctx.Dir, path),
		`type CronJobStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file`,
		`
	// active defines a list of pointers to currently running jobs.
	// +optional
	// +listType=atomic
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:MaxItems=10
	Active []corev1.ObjectReference `+"`json:\"active,omitempty\"`"+`

	// lastScheduleTime defines the information when was the last time the job was successfully scheduled.
	// +optional
	LastScheduleTime *metav1.Time `+"`json:\"lastScheduleTime,omitempty\"`"+`
`)
	hackutils.CheckError("insert status for cronjob v2", err)

	err = pluginutil.AppendCodeAtTheEnd(
		filepath.Join(sp.ctx.Dir, path), `
	// +kubebuilder:docs-gen:collapse=Other Types`)
	hackutils.CheckError("append marker at the end of the docs", err)
}

// CodeGen will call targets to generate code
func (sp *Sample) CodeGen() {
	cmd := exec.Command("make", "all")
	_, err := sp.ctx.Run(cmd)
	hackutils.CheckError("Failed to run make all for multiversion tutorial", err)

	cmd = exec.Command("make", "build-installer")
	_, err = sp.ctx.Run(cmd)
	hackutils.CheckError("Failed to run make build-installer for  multiversion tutorial", err)

	err = sp.ctx.EditHelmPlugin()
	hackutils.CheckError("Failed to enable helm plugin", err)
}

const webhookConversionE2ETest = `
		It("should successfully convert between v1 and v2 versions", func() {
			By("waiting for the webhook service to be ready")
			Eventually(func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "endpoints", "-n", namespace, 
					"-l", "control-plane=controller-manager", 
					"-o", "jsonpath={.items[0].subsets[0].addresses[0].ip}")
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred(), "Failed to get webhook service endpoints")
				g.Expect(strings.TrimSpace(output)).NotTo(BeEmpty(), "Webhook endpoint should have an IP")
			}, time.Minute, time.Second).Should(Succeed())

			By("creating a v1 CronJob with a specific schedule")
			cmd := exec.Command("kubectl", "apply", "-f", "config/samples/batch_v1_cronjob.yaml", "-n", namespace)
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to create v1 CronJob")

		By("waiting for the v1 CronJob to be created")
		Eventually(func(g Gomega) {
			cmd := exec.Command("kubectl", "get", "cronjob.batch.tutorial.kubebuilder.io", "cronjob-sample", "-n", namespace)
			output, err := utils.Run(cmd)
			if err != nil {
				// Log controller logs on failure for debugging
				logCmd := exec.Command("kubectl", "logs", "-l", "control-plane=controller-manager", "-n", namespace, "--tail=50")
				logs, _ := utils.Run(logCmd)
				_, _ = fmt.Fprintf(GinkgoWriter, "Controller logs when CronJob not found:\n%s\n", logs)
			}
			g.Expect(err).NotTo(HaveOccurred(), "v1 CronJob should exist, output: "+output)
		}, time.Minute, time.Second).Should(Succeed())

		By("fetching the v1 CronJob and verifying the schedule format")
			cmd = exec.Command("kubectl", "get", "cronjob.v1.batch.tutorial.kubebuilder.io", "cronjob-sample",
				"-n", namespace, "-o", "jsonpath={.spec.schedule}")
			v1Schedule, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to get v1 CronJob schedule")
			Expect(strings.TrimSpace(v1Schedule)).To(Equal("*/1 * * * *"),
				"v1 schedule should be in cron format")

			By("fetching the same CronJob as v2 and verifying the converted schedule")
			Eventually(func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "cronjob.v2.batch.tutorial.kubebuilder.io", "cronjob-sample",
					"-n", namespace, "-o", "jsonpath={.spec.schedule.minute}")
				v2Minute, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred(), "Failed to get v2 CronJob schedule")
				g.Expect(strings.TrimSpace(v2Minute)).To(Equal("*/1"),
					"v2 schedule.minute should be converted from v1 schedule")
			}, time.Minute, time.Second).Should(Succeed())

			By("creating a v2 CronJob with structured schedule fields")
			cmd = exec.Command("kubectl", "apply", "-f", "config/samples/batch_v2_cronjob.yaml", "-n", namespace)
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to create v2 CronJob")

			By("verifying the v2 CronJob has the correct structured schedule")
			Eventually(func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "cronjob.v2.batch.tutorial.kubebuilder.io", "cronjob-sample",
					"-n", namespace, "-o", "jsonpath={.spec.schedule.minute}")
				v2Minute, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred(), "Failed to get v2 CronJob schedule")
				g.Expect(strings.TrimSpace(v2Minute)).To(Equal("*/1"),
					"v2 CronJob should have minute field set")
			}, time.Minute, time.Second).Should(Succeed())

			By("fetching the v2 CronJob as v1 and verifying schedule conversion")
			Eventually(func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "cronjob.v1.batch.tutorial.kubebuilder.io", "cronjob-sample",
					"-n", namespace, "-o", "jsonpath={.spec.schedule}")
				v1Schedule, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred(), "Failed to get converted v1 schedule")
				// When v2 only has minute field set, it converts to "*/1 * * * *"
				g.Expect(strings.TrimSpace(v1Schedule)).To(Equal("*/1 * * * *"),
					"v1 schedule should be converted from v2 structured schedule")
			}, time.Minute, time.Second).Should(Succeed())
		})`

func (sp *Sample) updateE2EWebhookConversion() {
	cronjobE2ETest := filepath.Join(sp.ctx.Dir, "test", "e2e", "e2e_test.go")

	// Add strings import if not already present
	err := pluginutil.InsertCodeIfNotExist(cronjobE2ETest,
		`	"os/exec"
	"path/filepath"
	"time"`,
		`
	"strings"`)
	hackutils.CheckError("adding strings import for e2e test", err)

	// Add CronJob cleanup to the AfterEach block
	err = pluginutil.InsertCode(cronjobE2ETest,
		`	// After each test, check for failures and collect logs, events,
	// and pod descriptions for debugging.
	AfterEach(func() {`,
		`
		By("Cleaning up test CronJob resources")
		cmd := exec.Command("kubectl", "delete", "-f", "config/samples/batch_v1_cronjob.yaml", "-n", namespace, "--ignore-not-found=true")
		_, _ = utils.Run(cmd)
		cmd = exec.Command("kubectl", "delete", "-f", "config/samples/batch_v2_cronjob.yaml", "-n", namespace, "--ignore-not-found=true")
		_, _ = utils.Run(cmd)
`)
	hackutils.CheckError("adding CronJob cleanup to AfterEach", err)

	// Add webhook conversion test after the existing TODO comment
	err = pluginutil.InsertCode(cronjobE2ETest,
		`		// TODO: Customize the e2e test suite with scenarios specific to your project.
		// Consider applying sample/CR(s) and check their status and/or verifying
		// the reconciliation by using the metrics, i.e.:
		// metricsOutput, err := getMetricsOutput()
		// Expect(err).NotTo(HaveOccurred(), "Failed to retrieve logs from curl pod")
		// Expect(metricsOutput).To(ContainSubstring(
		//    fmt.Sprintf(`+"`"+`controller_runtime_reconcile_total{controller="%s",result="success"} 1`+"`"+`,
		//    strings.ToLower(<Kind>),
		// ))`,
		webhookConversionE2ETest)
	hackutils.CheckError("adding webhook conversion e2e test", err)
}
