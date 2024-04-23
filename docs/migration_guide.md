# Migration guide from v0 project to v1 project

This document describes how to migrate a project created by kubebuilder v0 to a project created by kubebuilder v1. Before jumping into the detailed instructions, please take a look at the list of [major differences between kubebuilder v0 and kubebuilder v1](kubebuilder_v0_v1_difference.md).

The recommended way of migrating a v0 project to a v1 project is to create a new v1 project and copy/modify the code from v0 project to it.

## Init a v1 project
Find project's domain name from the old project's pkg/apis/doc.go and use it to initiate a new project with
`kubebuilder init --project-version v1 --domain <domain>`

## Create api
Find the group/version/kind names from the project's pkg/apis. The group and version names are directory names while the kind name can be found from *_types.go. Note that the kind name should be capitalized.

Create api in the new project with
`kubebuilder create api --group <group> --version <version> --kind <kind>`

If there are several resources in the old project, repeat the `kubebuilder create api` command to create all of them.

## Copy types.go
Copy the content of `<type>_types.go` from the old project into the file `<type>_types.go` in the new project.
Note that in the v1 project, there is a section containing `<type>List` and `init` function. Please keep this section.
```
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +genclient:nonNamespaced

// HelloList contains a list of Hello
type HelloList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Hello `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Hello{}, &HelloList{})
}
```

## Copy and modify controller code

### copy and update reconcile function
Note that in v0 and v1 projects, the `Reconcile`
functions have different arguments and return types.

- `Reconcile` function in v0 project: `func (bc *<kind>Controller) Reconcile(k types.ReconcileKey) error`

- `Reconcile` function in v1 project: `func (r *Reconcile<kind>) Reconcile(request reconcile.Request) (reconcile.Result, error)`

Remove the original body of `Reconcile` function inside the v1 project and copy the body of the `Reconcile` function from the v0 project to the v1 project. Then apply following changes:
- add `reconcile.Result{}` as the first value in every `return` statement
- change the call of client functions such as `Get`, `Create`, `Update`. In v0 projects, the call of client functions has the format like `bc.<kind>Lister.<kind>().Get()` or `bc.KubernetesClientSet.<group>.<version>.<Kind>.Get()`. They can be replaced by `r.Client` functions. Here are several examples of updating the client function from v0 project to v1 project:

```
# in v0 project
mc, err := bc.memcachedLister.Memcacheds(k.Namespace).Get(k.Name)
# in v1 project, change to
mc := &myappsv1alpha1.Memcached{}
err := r.Client.Get(context.TODO(), request.NamespacedName, mc)


# in v0 project
dp, err := bc.KubernetesInformers.Apps().V1().Deployments().Lister().Deployments(mc.Namespace).Get(mc.Name)
# in v1 project, change to
dp := &appsv1.Deployment{}
err := r.Client.Get(context.TODO(), request.NamespacedName, dp)


dep := &appsv1.Deployment{...}
# in v0 project
dp, err := bc.KubernetesClientSet.AppsV1().Deployments(mc.Namespace).Create(dep)
# in v1 project, change to
err := r.Client.Create(context.TODO(), dep)


dep := &appsv1.Deployment{...}
# in v0 project
dp, err = bc.KubernetesClientSet.AppsV1().Deployments(mc.Namespace).Update(deploymentForMemcached(mc))
# in v1 project, change to
err := r.Client.Update(context.TODO(), dep)


labelSelector := labels.SelectorFrom{...}
# in v0 project
pods, err := bc.KubernetesInformers.Core().V1().Pods().Lister().Pods(mc.Namespace).List(labelSelector)
# in v1 project, change to
pods := &v1.PodList{}
err = r.Client.List(context.TODO(), &client.ListOptions{LabelSelector: labelSelector}, pods)
```
- add library imports used in the v0 project to v1 project such as log, fmt or k8s libraries. Note that libraries from kubebuilder or from the old project's client package shouldn't be added.


### update add function

In a v0 project controller file, there is a `ProvideController` function creating a controller and adding some watches. In v1 projects, the corresponding function is `add`. For this part, you don't need to copy any code from v0 project to v1 project. You need to add some watchers in v1 project's `add` function based on what `watch` functions are called in v0 project's `ProvideController` function.

Here are several examples:

```
gc := &controller.GenericController{...}
gc.Watch(&myappsv1alpha1.Memcached{})
gc.WatchControllerOf(&v1.Pod{}, eventhandlers.Path{bc.LookupRS, bc.LookupDeployment, bc.LookupMemcached})
```

need to be changed to:

```
c, err := controller.New{...}
c.Watch(&source.Kind{Type: &myappsv1alpha1.Memcached{}}, &handler.EnqueueRequestForObject{})
c.Watch(&source.Kind{Type: &appsv1.Deployment{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &myappsv1alpha1.Memcached{},
	})
```

### copy other functions
If `reconcile` function depends on some other user defined functions, copy those function as well into the v1 project.

## Copy user libraries
If there are some user defined libraries in the old project, make sure to copy them as well into the new project.

## Update dependency

Open the Gopkg.toml file in the old project and find if there is user defined dependency in this block:

```
# Users add deps lines here

[prune]
  go-tests = true
  #unused-packages = true

# Note: Stanzas below are generated by Kubebuilder and may be rewritten when
# upgrading kubebuilder versions.

# DO NOT MODIFY BELOW THIS LINE.
```
Copy those dependencies into the new project's Gopkg.toml file **before** the line
```
# STANZAS BELOW ARE GENERATED AND MAY BE WRITTEN - DO NOT MODIFY BELOW THIS LINE.
```

## Copy other user files
If there are other user created files in the old project, such as any build scripts, README.md files. Copy those files into the new project.

## Confirmation
Run `make` to make sure the new project can build and pass all the tests.
Run `make install` and `make run` to make sure the api and controller work well on cluster.
