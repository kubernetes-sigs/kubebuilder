/*
Copyright 2024 The Kubernetes Authors.

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

package gettingstarted

import (
	"os/exec"
	"path/filepath"

	pluginutil "sigs.k8s.io/kubebuilder/v4/pkg/plugin/util"

	hackutils "sigs.k8s.io/kubebuilder/v4/hack/docs/utils"

	log "github.com/sirupsen/logrus"
	"sigs.k8s.io/kubebuilder/v4/test/e2e/utils"
)

// Sample define the sample which will be scaffolded
type Sample struct {
	ctx *utils.TestContext
}

// NewSample create a new instance of the getting started sample and configure the KB CLI that will be used
func NewSample(binaryPath, samplePath string) Sample {
	log.Infof("Generating the sample context of getting-started...")
	ctx := hackutils.NewSampleContext(binaryPath, samplePath, "GO111MODULE=on")
	return Sample{&ctx}
}

// UpdateTutorial the getting started sample tutorial with the scaffold changes
func (sp *Sample) UpdateTutorial() {
	sp.updateAPI()
	sp.updateSample()
	sp.updateController()
	sp.updateControllerTest()
	sp.updateDefaultKustomize()
}

func (sp *Sample) updateDefaultKustomize() {
	err := pluginutil.UncommentCode(
		filepath.Join(sp.ctx.Dir, "config/default/kustomization.yaml"),
		`#- ../prometheus`, `#`)
	hackutils.CheckError("fixing default/kustomization", err)
}

func (sp *Sample) updateControllerTest() {
	file := "internal/controller/memcached_controller_test.go"
	err := pluginutil.ReplaceInFile(
		filepath.Join(sp.ctx.Dir, file),
		". \"github.com/onsi/gomega\"",
		`. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"`,
	)
	hackutils.CheckError("add imports apis", err)

	err = pluginutil.ReplaceInFile(
		filepath.Join(sp.ctx.Dir, file),
		"// TODO(user): Specify other spec details if needed.",
		`Spec: cachev1alpha1.MemcachedSpec{
						Size: 1,
					},`,
	)
	hackutils.CheckError("add spec apis", err)

	err = pluginutil.ReplaceInFile(
		filepath.Join(sp.ctx.Dir, file),
		`// TODO(user): Add more specific assertions depending on your controller's reconciliation logic.
			// Example: If you expect a certain status condition after reconciliation, verify it here.`,
		`By("Checking if Deployment was successfully created in the reconciliation")
			Eventually(func(g Gomega) {
				found := &appsv1.Deployment{}
				g.Expect(k8sClient.Get(ctx, typeNamespacedName, found)).To(Succeed())
			}).Should(Succeed())

			By("Reconciling the custom resource again")
			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Checking the latest Status Condition added to the Memcached instance")
			Expect(k8sClient.Get(ctx, typeNamespacedName, memcached)).To(Succeed())
			conditions := []metav1.Condition{}
			Expect(memcached.Status.Conditions).To(ContainElement(
				HaveField("Type", Equal(typeAvailableMemcached)), &conditions))
			Expect(conditions).To(HaveLen(1), "Multiple conditions of type %s", typeAvailableMemcached)
			Expect(conditions[0].Status).To(Equal(metav1.ConditionTrue), "condition %s", typeAvailableMemcached)
			Expect(conditions[0].Reason).To(Equal("Reconciling"), "condition %s", typeAvailableMemcached)`,
	)
	hackutils.CheckError("add spec apis", err)
}

func (sp *Sample) updateAPI() {
	var err error
	path := "api/v1alpha1/memcached_types.go"
	err = pluginutil.InsertCode(
		filepath.Join(sp.ctx.Dir, path),
		`limitations under the License.
*/`,
		`
// +kubebuilder:docs-gen:collapse=Apache License

`)
	hackutils.CheckError("collapse license in memcached api", err)

	err = pluginutil.InsertCode(
		filepath.Join(sp.ctx.Dir, path),
		`Any new fields you add must have json tags for the fields to be serialized.
`,
		`
// +kubebuilder:docs-gen:collapse=Imports
`)
	hackutils.CheckError("collapse imports in memcached api", err)

	err = pluginutil.ReplaceInFile(filepath.Join(sp.ctx.Dir, path), oldSpecAPI, newSpecAPI)
	hackutils.CheckError("replace spec api", err)

	err = pluginutil.ReplaceInFile(filepath.Join(sp.ctx.Dir, path), oldStatusAPI, newStatusAPI)
	hackutils.CheckError("replace status api", err)
}

func (sp *Sample) updateSample() {
	file := filepath.Join(sp.ctx.Dir, "config/samples/cache_v1alpha1_memcached.yaml")
	err := pluginutil.ReplaceInFile(file, "# TODO(user): Add fields here", sampleSizeFragment)
	hackutils.CheckError("update sample to add size", err)
}

func (sp *Sample) updateController() {
	pathFile := "internal/controller/memcached_controller.go"
	err := pluginutil.ReplaceInFile(
		filepath.Join(sp.ctx.Dir, pathFile),
		"\"context\"",
		controllerImports,
	)
	hackutils.CheckError("add imports", err)

	err = pluginutil.InsertCode(
		filepath.Join(sp.ctx.Dir, pathFile),
		"cachev1alpha1 \"example.com/memcached/api/v1alpha1\"\n)",
		controllerStatusTypes,
	)
	hackutils.CheckError("add status types", err)

	err = pluginutil.ReplaceInFile(
		filepath.Join(sp.ctx.Dir, pathFile),
		controllerInfoReconcileOld,
		controllerInfoReconcileNew,
	)
	hackutils.CheckError("add status types", err)

	err = pluginutil.InsertCode(
		filepath.Join(sp.ctx.Dir, pathFile),
		"// +kubebuilder:rbac:groups=cache.example.com,resources=memcacheds/finalizers,verbs=update",
		`
// +kubebuilder:rbac:groups=core,resources=events,verbs=create;patch
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch`,
	)
	hackutils.CheckError("add markers", err)

	err = pluginutil.ReplaceInFile(
		filepath.Join(sp.ctx.Dir, pathFile),
		"_ = log.FromContext(ctx)",
		"log := log.FromContext(ctx)",
	)
	hackutils.CheckError("add log var", err)

	err = pluginutil.ReplaceInFile(
		filepath.Join(sp.ctx.Dir, pathFile),
		"// TODO(user): your logic here",
		controllerReconcileImplementation,
	)
	hackutils.CheckError("add reconcile implementation", err)

	err = pluginutil.AppendCodeIfNotExist(
		filepath.Join(sp.ctx.Dir, pathFile),
		controllerDeploymentFunc,
	)
	hackutils.CheckError("add func to create Deployment", err)

	err = pluginutil.InsertCode(
		filepath.Join(sp.ctx.Dir, pathFile),
		"For(&cachev1alpha1.Memcached{}).",
		`
		Owns(&appsv1.Deployment{}).`,
	)
	hackutils.CheckError("add reconcile implementation", err)
}

// Prepare the Context for the sample project
func (sp *Sample) Prepare() {
	log.Infof("Destroying directory for getting-started sample project")
	sp.ctx.Destroy()

	log.Infof("Refreshing tools and creating directory...")
	err := sp.ctx.Prepare()

	hackutils.CheckError("Creating directory for sample project", err)
}

// GenerateSampleProject will generate the sample
func (sp *Sample) GenerateSampleProject() {
	log.Infof("Initializing the getting started project")
	err := sp.ctx.Init(
		"--domain", "example.com",
		"--repo", "example.com/memcached",
		"--license", "apache2",
		"--owner", "The Kubernetes authors",
	)
	hackutils.CheckError("Initializing the getting started project", err)

	log.Infof("Adding a new config type")
	err = sp.ctx.CreateAPI(
		"--group", "cache",
		"--version", "v1alpha1",
		"--kind", "Memcached",
		"--resource", "--controller",
	)
	hackutils.CheckError("Creating the API", err)
}

// CodeGen will call targets to generate code
func (sp *Sample) CodeGen() {
	cmd := exec.Command("go", "mod", "tidy")
	_, err := sp.ctx.Run(cmd)
	hackutils.CheckError("Failed to run go mod tidy all for getting started tutorial", err)

	cmd = exec.Command("make", "all")
	_, err = sp.ctx.Run(cmd)
	hackutils.CheckError("Failed to run make all for getting started tutorial", err)

	cmd = exec.Command("make", "build-installer")
	_, err = sp.ctx.Run(cmd)
	hackutils.CheckError("Failed to run make build-installer for getting started tutorial", err)

	err = sp.ctx.EditHelmPlugin()
	hackutils.CheckError("Failed to enable helm plugin", err)
}

const oldSpecAPI = "// Foo is an example field of Memcached. Edit memcached_types.go to remove/update\n\tFoo string `json:\"foo,omitempty\"`"
const newSpecAPI = `// Size defines the number of Memcached instances
	// The following markers will use OpenAPI v3 schema to validate the value
	// More info: https://book.kubebuilder.io/reference/markers/crd-validation.html
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=3
	// +kubebuilder:validation:ExclusiveMaximum=false
	Size int32 ` + "`json:\"size,omitempty\"`"

const oldStatusAPI = `// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file`

const newStatusAPI = `// Represents the observations of a Memcached's current state.
	// Memcached.status.conditions.type are: "Available", "Progressing", and "Degraded"
	// Memcached.status.conditions.status are one of True, False, Unknown.
	// Memcached.status.conditions.reason the value should be a CamelCase string and producers of specific
	// condition types may define expected values and meanings for this field, and whether the values
	// are considered a guaranteed API.
	// Memcached.status.conditions.Message is a human readable message indicating details about the transition.
	// For further information see: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#typical-status-properties

	Conditions []metav1.Condition ` + "`json:\"conditions,omitempty\" patchStrategy:\"merge\" patchMergeKey:\"type\" protobuf:\"bytes,1,rep,name=conditions\"`"

const sampleSizeFragment = `# TODO(user): edit the following value to ensure the number
  # of Pods/Instances your Operand must have on cluster
  size: 1`

const controllerImports = `"context"
	"fmt"
	"time"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
`

const controllerStatusTypes = `
// Definitions to manage status conditions
const (
	// typeAvailableMemcached represents the status of the Deployment reconciliation
	typeAvailableMemcached = "Available"
)`

const controllerInfoReconcileOld = `// TODO(user): Modify the Reconcile function to compare the state specified by
// the Memcached object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.`

const controllerInfoReconcileNew = `// It is essential for the controller's reconciliation loop to be idempotent. By following the Operator
// pattern you will create Controllers which provide a reconcile function
// responsible for synchronizing resources until the desired state is reached on the cluster.
// Breaking this recommendation goes against the design principles of controller-runtime.
// and may lead to unforeseen consequences such as resources becoming stuck and requiring manual intervention.
// For further info:
// - About Operator Pattern: https://kubernetes.io/docs/concepts/extend-kubernetes/operator/
// - About Controllers: https://kubernetes.io/docs/concepts/architecture/controller/`

const controllerReconcileImplementation = `// Fetch the Memcached instance
	// The purpose is check if the Custom Resource for the Kind Memcached
	// is applied on the cluster if not we return nil to stop the reconciliation
	memcached := &cachev1alpha1.Memcached{}
	err := r.Get(ctx, req.NamespacedName, memcached)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// If the custom resource is not found then it usually means that it was deleted or not created
			// In this way, we will stop the reconciliation
			log.Info("memcached resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to get memcached")
		return ctrl.Result{}, err
	}

	// Let's just set the status as Unknown when no status is available
	if memcached.Status.Conditions == nil || len(memcached.Status.Conditions) == 0 {
		meta.SetStatusCondition(&memcached.Status.Conditions, metav1.Condition{Type: typeAvailableMemcached, Status: metav1.ConditionUnknown, Reason: "Reconciling", Message: "Starting reconciliation"})
		if err = r.Status().Update(ctx, memcached); err != nil {
			log.Error(err, "Failed to update Memcached status")
			return ctrl.Result{}, err
		}

		// Let's re-fetch the memcached Custom Resource after updating the status
		// so that we have the latest state of the resource on the cluster and we will avoid
		// raising the error "the object has been modified, please apply
		// your changes to the latest version and try again" which would re-trigger the reconciliation
		// if we try to update it again in the following operations
		if err := r.Get(ctx, req.NamespacedName, memcached); err != nil {
			log.Error(err, "Failed to re-fetch memcached")
			return ctrl.Result{}, err
		}
	}

	// Check if the deployment already exists, if not create a new one
	found := &appsv1.Deployment{}
	err = r.Get(ctx, types.NamespacedName{Name: memcached.Name, Namespace: memcached.Namespace}, found)
	if err != nil && apierrors.IsNotFound(err) {
		// Define a new deployment
		dep, err := r.deploymentForMemcached(memcached)
		if err != nil {
			log.Error(err, "Failed to define new Deployment resource for Memcached")

			// The following implementation will update the status
			meta.SetStatusCondition(&memcached.Status.Conditions, metav1.Condition{Type: typeAvailableMemcached,
				Status: metav1.ConditionFalse, Reason: "Reconciling",
				Message: fmt.Sprintf("Failed to create Deployment for the custom resource (%s): (%s)", memcached.Name, err)})

			if err := r.Status().Update(ctx, memcached); err != nil {
				log.Error(err, "Failed to update Memcached status")
				return ctrl.Result{}, err
			}

			return ctrl.Result{}, err
		}

		log.Info("Creating a new Deployment",
			"Deployment.Namespace", dep.Namespace, "Deployment.Name", dep.Name)
		if err = r.Create(ctx, dep); err != nil {
			log.Error(err, "Failed to create new Deployment",
				"Deployment.Namespace", dep.Namespace, "Deployment.Name", dep.Name)
			return ctrl.Result{}, err
		}

		// Deployment created successfully
		// We will requeue the reconciliation so that we can ensure the state
		// and move forward for the next operations
		return ctrl.Result{RequeueAfter: time.Minute}, nil
	} else if err != nil {
		log.Error(err, "Failed to get Deployment")
		// Let's return the error for the reconciliation be re-trigged again
		return ctrl.Result{}, err
	}

	// The CRD API defines that the Memcached type have a MemcachedSpec.Size field
	// to set the quantity of Deployment instances to the desired state on the cluster.
	// Therefore, the following code will ensure the Deployment size is the same as defined
	// via the Size spec of the Custom Resource which we are reconciling.
	size := memcached.Spec.Size
	if *found.Spec.Replicas != size {
		found.Spec.Replicas = &size
		if err = r.Update(ctx, found); err != nil {
			log.Error(err, "Failed to update Deployment",
				"Deployment.Namespace", found.Namespace, "Deployment.Name", found.Name)

			// Re-fetch the memcached Custom Resource before updating the status
			// so that we have the latest state of the resource on the cluster and we will avoid
			// raising the error "the object has been modified, please apply
			// your changes to the latest version and try again" which would re-trigger the reconciliation
			if err := r.Get(ctx, req.NamespacedName, memcached); err != nil {
				log.Error(err, "Failed to re-fetch memcached")
				return ctrl.Result{}, err
			}

			// The following implementation will update the status
			meta.SetStatusCondition(&memcached.Status.Conditions, metav1.Condition{Type: typeAvailableMemcached,
				Status: metav1.ConditionFalse, Reason: "Resizing",
				Message: fmt.Sprintf("Failed to update the size for the custom resource (%s): (%s)", memcached.Name, err)})

			if err := r.Status().Update(ctx, memcached); err != nil {
				log.Error(err, "Failed to update Memcached status")
				return ctrl.Result{}, err
			}

			return ctrl.Result{}, err
		}

		// Now, that we update the size we want to requeue the reconciliation
		// so that we can ensure that we have the latest state of the resource before
		// update. Also, it will help ensure the desired state on the cluster
		return ctrl.Result{Requeue: true}, nil
	}

	// The following implementation will update the status
	meta.SetStatusCondition(&memcached.Status.Conditions, metav1.Condition{Type: typeAvailableMemcached,
		Status: metav1.ConditionTrue, Reason: "Reconciling",
		Message: fmt.Sprintf("Deployment for custom resource (%s) with %d replicas created successfully", memcached.Name, size)})

	if err := r.Status().Update(ctx, memcached); err != nil {
		log.Error(err, "Failed to update Memcached status")
		return ctrl.Result{}, err
	}`
const controllerDeploymentFunc = `// deploymentForMemcached returns a Memcached Deployment object
func (r *MemcachedReconciler) deploymentForMemcached(
	memcached *cachev1alpha1.Memcached) (*appsv1.Deployment, error) {
	replicas := memcached.Spec.Size
	image := "memcached:1.6.26-alpine3.19"

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      memcached.Name,
			Namespace: memcached.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app.kubernetes.io/name": "project"},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app.kubernetes.io/name": "project"},
				},
				Spec: corev1.PodSpec{
					SecurityContext: &corev1.PodSecurityContext{
						RunAsNonRoot: ptr.To(true),
						SeccompProfile: &corev1.SeccompProfile{
							Type: corev1.SeccompProfileTypeRuntimeDefault,
						},
					},
					Containers: []corev1.Container{{
						Image:           image,
						Name:            "memcached",
						ImagePullPolicy: corev1.PullIfNotPresent,
						// Ensure restrictive context for the container
						// More info: https://kubernetes.io/docs/concepts/security/pod-security-standards/#restricted
						SecurityContext: &corev1.SecurityContext{
							RunAsNonRoot:             ptr.To(true),
							RunAsUser:                ptr.To(int64(1001)),
							AllowPrivilegeEscalation: ptr.To(false),
							Capabilities: &corev1.Capabilities{
								Drop: []corev1.Capability{
									"ALL",
								},
							},
						},
						Ports: []corev1.ContainerPort{{
							ContainerPort: 11211,
							Name:          "memcached",
						}},
						Command: []string{"memcached", "--memory-limit=64", "-o", "modern", "-v"},
					}},
				},
			},
		},
	}

	// Set the ownerRef for the Deployment
	// More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/owners-dependents/
	if err := ctrl.SetControllerReference(memcached, dep, r.Scheme); err != nil {
		return nil, err
	}
	return dep, nil
}`
