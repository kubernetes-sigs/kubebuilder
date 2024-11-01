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
	"os"
	"os/exec"
	"path/filepath"

	log "github.com/sirupsen/logrus"
	hackutils "sigs.k8s.io/kubebuilder/v4/hack/docs/utils"
	pluginutil "sigs.k8s.io/kubebuilder/v4/pkg/plugin/util"
	"sigs.k8s.io/kubebuilder/v4/test/e2e/utils"
)

type Sample struct {
	ctx *utils.TestContext
}

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

	log.Infof("Creating defaulting and validation webhook for v2")
	err = sp.ctx.CreateWebhook(
		"--group", "batch",
		"--version", "v2",
		"--kind", "CronJob",
		"--defaulting",
		"--programmatic-validation",
		"--conversion",
	)
	hackutils.CheckError("Creating defaulting and validation webhook for v2", err)
}

func (sp *Sample) UpdateTutorial() {
	log.Println("Update tutorial with multiversion code")

	// Update files according to the multiversion
	sp.updateApiV1()
	sp.updateApiV2()
	sp.updateWebhookV1()
	sp.updateWebhookV2()
	sp.createHubFiles()
	sp.updateSampleV2()
	sp.updateMain()
	sp.updateDefaultKustomize()
}

func (sp *Sample) updateDefaultKustomize() {
	// Enable CA for Conversion Webhook
	err := pluginutil.UncommentCode(
		filepath.Join(sp.ctx.Dir, "config/default/kustomization.yaml"),
		caConversionCRDDefaultKustomize, `#`)
	hackutils.CheckError("fixing default/kustomization", err)
}

func (sp *Sample) updateWebhookV1() {
	err := pluginutil.ReplaceInFile(
		filepath.Join(sp.ctx.Dir, "internal/webhook/v1/cronjob_webhook.go"),
		"Then, we set up the webhook with the manager.",
		`This setup doubles as setup for our conversion webhooks: as long as our
types implement the
[Hub](https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/conversion?tab=doc#Hub) and
[Convertible](https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/conversion?tab=doc#Convertible)
interfaces, a conversion webhook will be registered.`,
	)
	hackutils.CheckError("replace webhook setup text", err)
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

func (sp *Sample) createHubFiles() {
	path := filepath.Join(sp.ctx.Dir, "api/v1/cronjob_conversion.go")

	_, err := os.Create(path)
	hackutils.CheckError("creating conversion file v1", err)

	err = pluginutil.AppendCodeAtTheEnd(path, "")
	hackutils.CheckError("creating empty conversion file v1", err)

	err = pluginutil.AppendCodeAtTheEnd(path, hubV1Code)
	hackutils.CheckError("appending hubV1Code to cronjob_conversion.go", err)

	path = filepath.Join(sp.ctx.Dir, "api/v2/cronjob_conversion.go")

	_, err = os.Create(path)
	hackutils.CheckError("creating conversion file v2", err)

	err = pluginutil.AppendCodeAtTheEnd(path, "")
	hackutils.CheckError("creating empty conversion file v2", err)

	err = pluginutil.AppendCodeAtTheEnd(path, hubV2Code)
	hackutils.CheckError("appending hubV2Code to cronjob_conversion.go", err)

}

func (sp *Sample) updateApiV1() {
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

func (sp *Sample) updateApiV2() {
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

func (sp *Sample) CodeGen() {
	cmd := exec.Command("make", "all")
	_, err := sp.ctx.Run(cmd)
	hackutils.CheckError("Failed to run make all for cronjob tutorial", err)

	cmd = exec.Command("make", "build-installer")
	_, err = sp.ctx.Run(cmd)
	hackutils.CheckError("Failed to run make build-installer for  multiversion cronjob tutorial", err)
}
