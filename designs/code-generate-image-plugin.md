| Authors       | Creation Date | Status      | Extra |
|---------------|---------------|-------------|---|
| @camilamacedo86 | 2021-02-14 | Implemented | [deploy-image-plugin-v1-alpha](../docs/book/src/plugins/available/deploy-image-plugin-v1-alpha.md) |

# New Plugin (`deploy-image.go.kubebuilder.io/v1beta1`) to generate code

## Summary

This proposal defines a new plugin which allow users get the scaffold with the
 required code to have a project that will deploy and manage an image on the cluster following the guidelines and what have been considered as good practices.

## Motivation

The biggest part of the Kubebuilder users looking for to create a project that will at the end only deploy an image. In this way, one of the  mainly motivations of this proposal is to abstract the complexities to achieve this goal and still giving the possibility of users improve and customize their projects according to their requirements.

**Note:** This plugin will address requests that has been raised for a while and for many users in the community. Check [here](https://github.com/operator-framework/operator-sdk/pull/2158), for example, a request done in the past for the SDK project which is integrated with Kubebuidler to address the same need.

### Goals

- Add a new plugin to generate the code required to deploy and manage an image on the cluster
- Promote the best practices as give example of common implementations
- Make the process to develop  operators projects easier and more agil.
- Give flexibility to the users and allow them to change the code according to their needs
- Provide examples of code implementations and of the most common features usage and reduce the learning curve

### Non-Goals

The idea of this proposal is provide a facility for the users. This plugin can be improved
in the future, however, this proposal just covers the basic requirements. In this way, is a non-goal
allow extra configurations such as; scaffold the project using webhooks and the controller covered by tests.

## Proposal

Add the new plugin code generate which will scaffold code implementation to deploy the image informed which would like such as; `kubebuilder create api --group=crew --version=v1 --image=myexample:0.0.1 --kind=App --plugins=deploy-image.go.kubebuilder.io/v1beta1` which will:

- Add a code implementation which will do the Custom Resource reconciliation and create a Deployment resource for the `--image`;

- Add an EnvVar on the manager manifest (`config/manager/manager.yaml`) which will store the image informed and shows its possibility to users:

```yaml
    ..
    spec:
      containers:
        - name: manager
          env:
            - name: {{ resource}}-IMAGE
              value: {{image:tag}}
          image: controller:latest
      ...
```

- Add a check into reconcile to ensure that the replicas of the deployment on cluster are equals the size defined in the CR:

```go
	// Ensure the deployment size is the same as the spec
	size := {{ resource }}.Spec.Size
	if *found.Spec.Replicas != size {
		found.Spec.Replicas = &size
		err = r.Update(ctx, found)
		if err != nil {
			log.Error(err, "Failed to update Deployment", "Deployment.Namespace", found.Namespace, "Deployment.Name", found.Name)
			return ctrl.Result{}, err
		}
		// Spec updated - return and requeue
		return ctrl.Result{Requeue: true}, nil
	}
```

- Add the watch feature for the Deployment managed by the controller:

```go
func (r *{{ resource }}Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&cachev1alpha1.{{ resource }}{}).
		Owns(&appsv1.Deployment{}).
		Complete(r)
}
```

- Add the RBAC permissions required for the scenario such as:

```go
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
```

- A status [conditions][conditions] to allow users check that if the deployment occurred successfully or its errors

- Add a [marker][markers] in the spec definition to demonstrate how to use OpenAPI schemas validation such as `+kubebuilder:validation:Minimum=1`

- Add the specs on the `_types.go` to generate the CRD/CR sample with default values for `ImagePullPolicy` (`Always`), `ContainerPort` (`80`) and the `Replicas Size` (`3`)

- Add a finalizer implementation with TODO for the CR managed by the controller such as described in the SDK doc [Handle Cleanup on Deletion](https://sdk.operatorframework.io/docs/building-operators/golang/advanced-topics/#handle-cleanup-on-deletion)

### User Stories

- I am as user, would like to use a command to scaffold my common need which is deploy an image of my application, so that I do not need to know exactly how to implement it

- I am as user, would like to have a good example code base which uses the common features, so that I can easily learn its concepts and have a good start point to address my needs.

- I am as maintainer, would like to have a good example to address the common questions, so that I can easily describe how to implement the projects and/or use the common features.

### Implementation Details/Notes/Constraints

**Example of the controller template**

```go
// +kubebuilder:rbac:groups=cache.example.com,resources={{ resource.plural }},verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=cache.example.com,resources={{ resource.plural }}/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=cache.example.com,resources={{ resource.plural }}/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete

func (r *{{ resource }}.Reconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("{{ resource }}", req.NamespacedName)

	// Fetch the {{ resource }} instance
	{{ resource }} := &{{ apiimportalias }}.{{ resource }}{}
	err := r.Get(ctx, req.NamespacedName, {{ resource }})
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			log.Info("{{ resource }} resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to get {{ resource }}")
		return ctrl.Result{}, err
	}

	// Check if the deployment already exists, if not create a new one
	found := &appsv1.Deployment{}
	err = r.Get(ctx, types.NamespacedName{Name: {{ resource }}.Name, Namespace: {{ resource }}.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		// Define a new deployment
		dep := r.deploymentFor{{ resource }}({{ resource }})
		log.Info("Creating a new Deployment", "Deployment.Namespace", dep.Namespace, "Deployment.Name", dep.Name)
		err = r.Create(ctx, dep)
		if err != nil {
			log.Error(err, "Failed to create new Deployment", "Deployment.Namespace", dep.Namespace, "Deployment.Name", dep.Name)
			return ctrl.Result{}, err
		}
		// Deployment created successfully - return and requeue
		return ctrl.Result{Requeue: true}, nil
	} else if err != nil {
		log.Error(err, "Failed to get Deployment")
		return ctrl.Result{}, err
	}

	// Ensure the deployment size is the same as the spec
	size := {{ resource }}.Spec.Size
	if *found.Spec.Replicas != size {
		found.Spec.Replicas = &size
		err = r.Update(ctx, found)
		if err != nil {
			log.Error(err, "Failed to update Deployment", "Deployment.Namespace", found.Namespace, "Deployment.Name", found.Name)
			return ctrl.Result{}, err
		}
		// Spec updated - return and requeue
		return ctrl.Result{Requeue: true}, nil
	}

    // TODO: add here code implementation to update/manage the status

	return ctrl.Result{}, nil
}

// deploymentFor{{ resource }} returns a {{ resource }} Deployment object
func (r *{{ resource }}Reconciler) deploymentFor{{ resource }}(m *{{ apiimportalias }}.{{ resource }}) *appsv1.Deployment {
	ls := labelsFor{{ resource }}(m.Name)
	replicas := m.Spec.Size

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.Name,
			Namespace: m.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: ls,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: ls,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Image:   imageFor{{ resource }}(m.Name),
						Name:    {{ resource }},
                        ImagePullPolicy: {{ resource }}.Spec.ContainerImagePullPolicy,
						Command: []string{"{{ resource }}"},
						Ports: []corev1.ContainerPort{{
							ContainerPort: {{ resource }}.Spec.ContainerPort,
							Name:          "{{ resource }}",
						}},
					}},
				},
			},
		},
	}
	// Set {{ resource }} instance as the owner and controller
	ctrl.SetControllerReference(m, dep, r.Scheme)
	return dep
}

// labelsFor{{ resource }} returns the labels for selecting the resources
// belonging to the given {{ resource }} CR name.
func labelsFor{{ resource }}(name string) map[string]string {
	return map[string]string{"type": "{{ resource }}", "{{ resource }}_cr": name}
}

// imageFor{{ resource }} returns the image for the resources
// belonging to the given {{ resource }} CR name.
func imageFor{{ resource }}(name string) string {
	// TODO: this method will return the value of the envvar create to store the image:tag informed
}

func (r *{{ resource }}Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&cachev1alpha1.{{ resource }}{}).
		Owns(&appsv1.Deployment{}).
		Complete(r)
}

```

**Example of the spec for the <kind>_types.go template**

```go
// {{ resource }}Spec defines the desired state of {{ resource }}
type {{ resource }}Spec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

    // +kubebuilder:validation:Minimum=1
	// Size defines the number of {{ resource }} instances
	Size int32 `json:"size,omitempty"`

    // ImagePullPolicy defines the policy to pull the container images
	ImagePullPolicy string `json:"image-pull-policy,omitempty"`

    // ContainerPort specifies the port which will be used by the image container
	ContainerPort int `json:"container-port,omitempty"`

}
```

## Design Details

### Test Plan

To ensure this implementation a new project example should be generated in the [testdata](../testdata/) directory of the project. See the [test/testdata/generate.sh](../test/testadata/generate.sh). Also, we should use this scaffold in the [integration tests](../test/e2e/) to ensure that the data scaffolded with works on the cluster as expected.

### Graduation Criteria

- The new plugin will only be support `project-version=3`
- The attribute image with the value informed should be added to the resources model in the PROJECT file to let the tool know that the Resource get done with the common basic code implementation:

```yaml
plugins:
    deploy-image.go.kubebuilder.io/v1beta1:
        resources:
          - domain: example.io
            group: crew
            kind: Captain
            version: v1
            image: "<some-registry>/<project-name>:<tag>
```

For further information check the definition agreement register in the comment https://github.com/kubernetes-sigs/kubebuilder/issues/1941#issuecomment-778649947.

## Open Questions

1. Should we allow to scaffold the code for an API that is already created for the project?
No, at least in the first moment to keep the simplicity.

2. Should we support StatefulSet and Deployments?
The idea is we start it by using a Deployment. However, we can improve the feature in follow-ups to support more default types of scaffolds which could be like `kubebuilder create api --group=crew --version=v1 --image=myexample:0.0.1 --kind=App --plugins=deploy-image.go.kubebuilder.io/v1beta1 --type=[deployment|statefulset|webhook]`

3. Could this feature be useful to other languages or is it just valid to Go based operators?

This plugin would is reponsable to scaffold content and files for Go-based operators. In a future, if other language-based operators starts to be supported (either officially or by the community) this plugin could be used as reference to create an equivalent one for their languages. Therefore, it should probably not to be a `subdomain` of `go.kubebuilder.io.`

For its integration with SDK, it might be valid for the Ansible-based operators where a new `playbook/role` could be generated as well. However, for example, for the Helm plugin it might to be useless. E.g `deploy-image.ansible.sdk.operatorframework.io/v1beta1`

4. Should we consider create a separate repo for plugins?

In the long term yes. However, see that currently, Kubebuilder has not too many plugins yet. And then, and the preliminary support for plugins did not indeed release. For more info see the [Extensible CLI and Scaffolding Plugins][plugins-phase1-design-doc].

In this way, at this moment, it shows to be a little Premature Optimization. Note that the issue [#2016](https://github.com/kubernetes-sigs/kubebuilder/issues/1378) will check the possibility of the plugins be as separate binaries that can be discovered by the Kubebuilder CLI binary via user-specified plugin file paths. Then, the discussion over the best approach to dealing with many plugins and if they should or not leave in the Kubebuilder repository would be better addressed after that.

5. Is Kubebuilder prepared to receive this implementation already?

The [Extensible CLI and Scaffolding Plugins - Phase 1.5](extensible-cli-and-scaffolding-plugins-phase-1-5.md) and the issue #1941 requires to be implemented before this proposal. Also, to have a better idea over the proposed solutions made so for the Plugin Ecosystem see the meta issue [#2016](https://github.com/kubernetes-sigs/kubebuilder/issues/2016)

[markers]: ../docs/book/src/reference/markers.md
[conditions]: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#typical-status-properties
[plugins-phase1-design-doc]: https://github.com/kubernetes-sigs/kubebuilder/blob/master/designs/extensible-cli-and-scaffolding-plugins-phase-1.md