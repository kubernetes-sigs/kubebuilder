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
	"os/exec"
	"path/filepath"

	log "github.com/sirupsen/logrus"
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
	log.Infof("Generating the sample context of MultiVersion Cronjob...")
	ctx := hackutils.NewSampleContext(binaryPath, samplePath, "GO111MODULE=on")
	return Sample{&ctx}
}

// Prepare the Context for the sample project
func (sp *Sample) Prepare() {
	log.Infof("refreshing tools and creating directory for multiversion ...")
	err := sp.ctx.Prepare()
	hackutils.CheckError("creating directory for multiversion project", err)
}

// GenerateSampleProject will generate the sample
func (sp *Sample) GenerateSampleProject() {
	log.Infof("Initializing the multiversion cronjob project")

	log.Infof("Creating v2 API")
	err := sp.ctx.CreateAPI(
		"--group", "batch",
		"--version", "v2",
		"--kind", "CronJob",
		"--resource=true",
		"--controller=false",
	)
	hackutils.CheckError("Creating the v2 API without controller", err)

	log.Infof("Creating conversion webhook for v1")
	err = sp.ctx.CreateWebhook(
		"--group", "batch",
		"--version", "v1",
		"--kind", "CronJob",
		"--conversion",
		"--spoke", "v2",
		"--force",
	)
	hackutils.CheckError("Creating conversion webhook for v1", err)

	log.Infof("Workaround to fix the issue with the conversion webhook")
	// FIXME: This is a workaround to fix the issue with the conversion webhook
	// We should be able to inject the code when we create webhooks with different
	// types of webhooks. However, currently, we are not able to do that and we need to
	// force. So, we are copying the code from cronjob tutorial to have the code
	// implemented.
	cmd := exec.Command("cp", "./../../../cronjob-tutorial/testdata/project/internal/webhook/v1/cronjob_webhook.go", "./internal/webhook/v1/cronjob_webhook.go")
	_, err = sp.ctx.Run(cmd)
	hackutils.CheckError("Copying the code from cronjob tutorial", err)

	log.Infof("Creating defaulting and validation webhook for v2")
	err = sp.ctx.CreateWebhook(
		"--group", "batch",
		"--version", "v2",
		"--kind", "CronJob",
		"--defaulting",
		"--programmatic-validation",
	)
	hackutils.CheckError("Creating defaulting and validation webhook for v2", err)
}

// UpdateTutorial the muilt-version sample tutorial with the scaffold changes
func (sp *Sample) UpdateTutorial() {
	log.Println("Update tutorial with multiversion code")

	// Update files according to the multiversion
	sp.updateCronjobV1DueForce()
	sp.updateAPIV1()
	sp.updateAPIV2()
	sp.updateWebhookV2()
	sp.updateConversionFiles()
	sp.updateSampleV2()
	sp.updateMain()
	sp.updateDefaultKustomize()
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

	BeforeEach(func() {
		obj = &batchv1.CronJob{
			Spec: batchv1.CronJobSpec{
				Schedule:                   "*/5 * * * *",
				ConcurrencyPolicy:          batchv1.AllowConcurrent,
				SuccessfulJobsHistoryLimit: new(int32),
				FailedJobsHistoryLimit:     new(int32),
			},
		}
		*obj.Spec.SuccessfulJobsHistoryLimit = 3
		*obj.Spec.FailedJobsHistoryLimit = 1

		oldObj = &batchv1.CronJob{
			Spec: batchv1.CronJobSpec{
				Schedule:                   "*/5 * * * *",
				ConcurrencyPolicy:          batchv1.AllowConcurrent,
				SuccessfulJobsHistoryLimit: new(int32),
				FailedJobsHistoryLimit:     new(int32),
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
			defaulter.Default(ctx, obj)

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
			defaulter.Default(ctx, obj)

			By("checking that the fields were not overwritten")
			Expect(obj.Spec.ConcurrencyPolicy).To(Equal(batchv1.ForbidConcurrent), "Expected ConcurrencyPolicy to retain its set value")
			Expect(*obj.Spec.Suspend).To(BeTrue(), "Expected Suspend to retain its set value")
			Expect(*obj.Spec.SuccessfulJobsHistoryLimit).To(Equal(int32(5)), "Expected SuccessfulJobsHistoryLimit to retain its set value")
			Expect(*obj.Spec.FailedJobsHistoryLimit).To(Equal(int32(2)), "Expected FailedJobsHistoryLimit to retain its set value")
		})
	})

	Context("When creating or updating CronJob under Validating Webhook", func() {
		It("Should deny creation if the name is too long", func() {
			obj.ObjectMeta.Name = "this-name-is-way-too-long-and-should-fail-validation-because-it-is-way-too-long"
			Expect(validator.ValidateCreate(ctx, obj)).Error().To(
				MatchError(ContainSubstring("must be no more than 52 characters")),
				"Expected name validation to fail for a too-long name")
		})

		It("Should admit creation if the name is valid", func() {
			obj.ObjectMeta.Name = "valid-cronjob-name"
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
			obj.Spec.Schedule = "*/5 * * * *"
			Expect(validator.ValidateCreate(ctx, obj)).To(BeNil(),
				"Expected spec validation to pass for a valid schedule")
		})

		It("Should deny update if both name and spec are invalid", func() {
			oldObj.ObjectMeta.Name = "valid-cronjob-name"
			oldObj.Spec.Schedule = "*/5 * * * *"

			By("simulating an update")
			obj.ObjectMeta.Name = "this-name-is-way-too-long-and-should-fail-validation-because-it-is-way-too-long"
			obj.Spec.Schedule = "invalid-cron-schedule"

			By("validating an update")
			Expect(validator.ValidateUpdate(ctx, oldObj, obj)).Error().To(HaveOccurred(),
				"Expected validation to fail for both name and spec")
		})

		It("Should admit update if both name and spec are valid", func() {
			oldObj.ObjectMeta.Name = "valid-cronjob-name"
			oldObj.Spec.Schedule = "*/5 * * * *"

			By("simulating an update")
			obj.ObjectMeta.Name = "valid-cronjob-name-updated"
			obj.Spec.Schedule = "0 0 * * *"

			By("validating an update")
			Expect(validator.ValidateUpdate(ctx, oldObj, obj)).To(BeNil(),
				"Expected validation to pass for a valid update")
		})
	})
	
	`)
	hackutils.CheckError("fix cronjob v1 tests after each", err)
}

func (sp *Sample) updateDefaultKustomize() {
	// Enable CA for Conversion Webhook
	err := pluginutil.UncommentCode(
		filepath.Join(sp.ctx.Dir, "config/default/kustomization.yaml"),
		caInjectionNamespace, `#`)
	hackutils.CheckError("fixing default/kustomization", err)

	// Enable CA for Conversion Webhook
	err = pluginutil.UncommentCode(
		filepath.Join(sp.ctx.Dir, "config/default/kustomization.yaml"),
		caInjectionCert, `#`)
	hackutils.CheckError("fixing default/kustomization", err)
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
		"// TODO(user): Implement conversion logic from v2 to v1",
		hubV2CovertTo)
	hackutils.CheckError("replace covertTo at hub v2", err)

	err = pluginutil.ReplaceInFile(path,
		"// TODO(user): Implement conversion logic from v1 to v2",
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
		`// +kubebuilder:object:root=true

// CronJobList contains a list of CronJob.`,
		`/*
 */

// +kubebuilder:object:root=true

// CronJobList contains a list of CronJob.`,
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
		boilerplateComment,
		boilerplateReplacement,
	)
	hackutils.CheckError("replacing boilerplate comment with storage version explanation", err)

	err = pluginutil.ReplaceInFile(
		filepath.Join(sp.ctx.Dir, "api/v1/cronjob_types.go"),
		`// +kubebuilder:docs-gen:collapse=Root Object Definitions`,
		`// +kubebuilder:docs-gen:collapse=old stuff`,
	)
	hackutils.CheckError("replacing docs-gen collapse comment", err)
}

func (sp *Sample) updateWebhookV2() {
	path := "internal/webhook/v2/cronjob_webhook.go"

	err := pluginutil.InsertCode(
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

	// +kubebuilder:docs-gen:collapse=old stuff`,
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
		`if err = (&controller.CronJobReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "CronJob")
		os.Exit(1)
	}`,
		`
// +kubebuilder:docs-gen:collapse=existing setup
`,
	)
	hackutils.CheckError("insert // +kubebuilder:docs-gen:collapse=existing setup main.go", err)

	err = pluginutil.InsertCode(
		filepath.Join(sp.ctx.Dir, path),
		`// +kubebuilder:scaffold:builder`,
		`

	/*
	 */`,
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

	err = pluginutil.ReplaceInFile(
		filepath.Join(sp.ctx.Dir, path),
		`// +kubebuilder:docs-gen:collapse=old stuff`,
		`// +kubebuilder:docs-gen:collapse=existing setup`,
	)
	hackutils.CheckError("replace +kubebuilder:docs-gen:collapse=old stuff main.go", err)

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
		`// CronJobSpec defines the desired state of CronJob.`,
		`// +kubebuilder:docs-gen:collapse=Imports

/*
We'll leave our spec largely unchanged, except to change the schedule field to a new type.
*/
// CronJobSpec defines the desired state of CronJob.`,
	)
	hackutils.CheckError("replace doc about CronjobSpec v2", err)

	err = pluginutil.ReplaceInFile(
		filepath.Join(sp.ctx.Dir, path),
		"// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster\n\t// Important: Run \"make\" to regenerate code after modifying this file",
		`// The schedule in Cron format, see https://en.wikipedia.org/wiki/Cron.
	Schedule CronSchedule `+"`json:\"schedule\"`"+`

	/*
	 */
`,
	)
	hackutils.CheckError("add new schedule spec type", err)

	err = pluginutil.ReplaceInFile(
		filepath.Join(sp.ctx.Dir, path),
		`// Foo is an example field of CronJob. Edit cronjob_types.go to remove/update
	Foo string `+"`json:\"foo,omitempty\"`",
		cronJobSpecReplace,
	)
	hackutils.CheckError("replace Foo with cronjob spec fields", err)

	err = pluginutil.ReplaceInFile(
		filepath.Join(sp.ctx.Dir, path),
		`// CronJobStatus defines the observed state of CronJob.
type CronJobStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}`,
		cronJobStatusReplace,
	)
	hackutils.CheckError("replace Foo with cronjob spec fields", err)

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
