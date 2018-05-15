package memcached

import (
	"fmt"
	"log"

	"github.com/kubernetes-sigs/kubebuilder/pkg/controller"
	"github.com/kubernetes-sigs/kubebuilder/pkg/controller/types"

	myappsv1alpha1 "github.com/kubernetes-sigs/kubebuilder/samples/memcached-api-server/pkg/apis/myapps/v1alpha1"
	myappsv1alpha1client "github.com/kubernetes-sigs/kubebuilder/samples/memcached-api-server/pkg/client/clientset/versioned/typed/myapps/v1alpha1"
	myappsv1alpha1informer "github.com/kubernetes-sigs/kubebuilder/samples/memcached-api-server/pkg/client/informers/externalversions/myapps/v1alpha1"
	myappsv1alpha1lister "github.com/kubernetes-sigs/kubebuilder/samples/memcached-api-server/pkg/client/listers/myapps/v1alpha1"
	"github.com/kubernetes-sigs/kubebuilder/samples/memcached-api-server/pkg/inject/args"
	"github.com/kubernetes-sigs/kubebuilder/pkg/controller/eventhandlers"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/api/core/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
)

// EDIT THIS FILE
// This files was created by "kubebuilder create resource" for you to edit.
// Controller implementation logic for Memcached resources goes here.

func (bc *MemcachedController) Reconcile(k types.ReconcileKey) error {

	mc, err := bc.memcachedLister.Memcacheds(k.Namespace).Get(k.Name)
	if err != nil {
		if apierrors.IsNotFound(err) {
			log.Printf("custom resource mc doesn't exist in queue for key: %s", k.Name)
			return nil
		}
		return err
	}

	// check if this deployment already exists
	dp, err := bc.KubernetesInformers.Apps().V1().Deployments().Lister().Deployments(mc.Namespace).Get(mc.Name)
	if apierrors.IsNotFound(err) {
		// did not find any deployment corresponding to this mc, so lets create one
		dp, err = bc.KubernetesClientSet.AppsV1().Deployments(mc.Namespace).Create(deploymentForMemcached(mc))
		if err != nil {
			log.Printf("error in creating deployment: %v", err)
		}
	}
	if err != nil {
		log.Printf("error either listing or creating deployment: %v", err)
		return err
	}

	if !metav1.IsControlledBy(dp, mc) {
		log.Printf("deployment: +%v is not controlled by +%v", dp, mc)
		return fmt.Errorf("found orphaned deploymend...dep is not controlled by mc")
	}
	// by now we have dp pointing to the deployment corresponding to mc

	if dp.Spec.Replicas != nil && *dp.Spec.Replicas != mc.Spec.Size {
		dp, err = bc.KubernetesClientSet.AppsV1().Deployments(mc.Namespace).Update(deploymentForMemcached(mc))
		if err != nil {
			return err
		}
	}

	// Update the Memcached status with the pod names
	labelSelector := labels.SelectorFromSet(labelsForMemcached(mc.Name))
	pods, err := bc.KubernetesInformers.Core().V1().Pods().Lister().Pods(mc.Namespace).List(labelSelector)
	if err != nil {
		return fmt.Errorf("failed to list pods: %v", err)
	}
	podNames := getPodNames(pods)

	log.Printf("got the pods names: %v", podNames)

	err = bc.updateMemcachedStatus(mc, podNames)
	if err != nil {
		return fmt.Errorf("failed to update mc with pod-names: %v", err)
	}

	return nil
}

func (bc *MemcachedController) updateMemcachedStatus(mc *myappsv1alpha1.Memcached, nodes []string) error {
	mcCopy := mc.DeepCopy()
	mcCopy.Status.Nodes = nodes

	_, err := bc.memcachedclient.Memcacheds(mc.Namespace).Update(mcCopy)
	return err
}

// +controller:group=myapps,version=v1alpha1,kind=Memcached,resource=memcacheds
// +informers:group=apps,version=v1,kind=Deployment
// +rbac:rbac:groups=apps,resources=Deployment,verbs=get;list;watch;create;update;patch;delete
// +informers:group=apps,version=v1,kind=ReplicaSet
// +rbac:rbac:groups=apps,resources=ReplicaSet,verbs=get;list;watch
// +informers:group=core,version=v1,kind=Pod
// +rbac:rbac:groups="",resources=Pod,verbs=get;list;watch
type MemcachedController struct {
	args.InjectArgs

	// INSERT ADDITIONAL FIELDS HERE
	memcachedLister myappsv1alpha1lister.MemcachedLister
	memcachedclient myappsv1alpha1client.MyappsV1alpha1Interface
}

// ProvideController provides a controller that will be run at startup.  Kubebuilder will use codegeneration
// to automatically register this controller in the inject package
func ProvideController(arguments args.InjectArgs) (*controller.GenericController, error) {
	// INSERT INITIALIZATIONS FOR ADDITIONAL FIELDS HERE
	bc := &MemcachedController{
		InjectArgs:      arguments,
		memcachedLister: arguments.ControllerManager.GetInformerProvider(&myappsv1alpha1.Memcached{}).(myappsv1alpha1informer.MemcachedInformer).Lister(),
		memcachedclient: arguments.Clientset.MyappsV1alpha1(),
	}

	// Create a new controller that will call MemcachedController.Reconcile on changes to Memcacheds
	gc := &controller.GenericController{
		Name:             "MemcachedController",
		Reconcile:        bc.Reconcile,
		InformerRegistry: arguments.ControllerManager,
	}
	if err := gc.Watch(&myappsv1alpha1.Memcached{}); err != nil {
		return gc, err
	}
	if err := gc.WatchControllerOf(&v1.Pod{}, eventhandlers.Path{bc.LookupRS, bc.LookupDeployment, bc.LookupMemcached}); err != nil {
		return gc, err
	}

	return gc, nil
}

func (bc *MemcachedController) LookupRS(r types.ReconcileKey) (interface{}, error) {
	log.Printf("looking up ReplicaSet %+v", r)
	return bc.KubernetesInformers.Apps().V1().ReplicaSets().Lister().ReplicaSets(r.Namespace).Get(r.Name)
}

func (bc *MemcachedController) LookupDeployment(r types.ReconcileKey) (interface{}, error) {
	log.Printf("looking up deployment %+v", r)
	return bc.KubernetesInformers.Apps().V1().Deployments().Lister().Deployments(r.Namespace).Get(r.Name)
}

func (bc *MemcachedController) LookupMemcached(r types.ReconcileKey) (interface{}, error) {
	log.Printf("looking up MC: %+v", r)
	return bc.Informers.Myapps().V1alpha1().Memcacheds().Lister().Memcacheds(r.Namespace).Get(r.Name)
}

// deploymentForMemcached returns a memcached Deployment object
func deploymentForMemcached(m *myappsv1alpha1.Memcached) *appsv1.Deployment {
	ls := labelsForMemcached(m.Name)
	replicas := m.Spec.Size

	dep := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.Name,
			Namespace: m.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: ls,
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: ls,
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{{
						Image:   "memcached:1.4.36-alpine",
						Name:    "memcached",
						Command: []string{"memcached", "-m=64", "-o", "modern", "-v"},
						Ports: []v1.ContainerPort{{
							ContainerPort: 11211,
							Name:          "memcached",
						}},
					}},
				},
			},
		},
	}
	addOwnerRefToObject(dep, *metav1.NewControllerRef(m, schema.GroupVersionKind{
		Group:   myappsv1alpha1.SchemeGroupVersion.Group,
		Version: myappsv1alpha1.SchemeGroupVersion.Version,
		Kind:    "Memcached",
	}))
	return dep
}

// labelsForMemcached returns the labels for selecting the resources
// belonging to the given memcached CR name.
func labelsForMemcached(name string) map[string]string {
	return map[string]string{"app": "memcached", "memcached_cr": name}
}

// addOwnerRefToObject appends the desired OwnerReference to the object
func addOwnerRefToObject(obj metav1.Object, ownerRef metav1.OwnerReference) {
	obj.SetOwnerReferences(append(obj.GetOwnerReferences(), ownerRef))
}

// getPodNames returns the pod names of the array of pods passed in
func getPodNames(pods []*v1.Pod) []string {
	var podNames []string
	for _, pod := range pods {
		podNames = append(podNames, pod.Name)
	}
	return podNames
}
