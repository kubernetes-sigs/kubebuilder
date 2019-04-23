# Deploy the controller-manager in a Kubernetes cluster

Deploying the controller to a Kubernetes cluster involves following steps:
 - Building the docker image
 - Pushing the docker image to the container registry
 - Customizing the deployment manifests
 - Applying the manifests to deploy in the cluster

Kubebuilder generated `Makefile` supports all the above steps.

{% panel style="info", title="Prerequisites" %}
Kubebuilder generated `Makefile` uses [Kustomize](https://github.com/kubernetes-sigs/kustomize) for customizing the manifests
before deploying to the kubernetes cluster. Follow the [instructions](https://github.com/kubernetes-sigs/kustomize/blob/master/INSTALL.md) to install `Kustomize` and
ensure that is available in the PATH. Note that Kubebuilder requires `Kustomize` version `1.0.4` or higher for deploy to work.

```bash
opsys=linux  # or darwin, or windows
curl -s https://api.github.com/repos/kubernetes-sigs/kustomize/releases/latest |\
  grep browser_download |\
  grep $opsys |\
  cut -d '"' -f 4 |\
  xargs curl -O -L
mv kustomize_*_${opsys}_amd64 kustomize
chmod u+x kustomize
```

The yaml configuration for the Manager is automatically created under
`config/manager`.
{% endpanel %}

#### Building the docker image and pushing it to a container registry

`Makefile` has following targets:
- `docker-build` to build the docker image for the controller manager
- `docker-push` to push it to the configured container registry.

Both target support `IMG` variable. If IMG argument is not provided, it is
picked from the environment variable.

```bash
# build the docker image
make docker-build IMG=<image-name>

# build the docker image
make docker-push IMG=<image-name>
```

#### Customizing the controller manager manifests using Kustomize

Kubebuilder scaffolds a basic `kustomization.yaml` under `config/` directory. Current customization:
 - Specifies all controller manager resources to be created under specified `namespace`
 - Adds a prefix (directory name of the project) for controller manager resources
 - Adds a patch `config/default/manager_image_patch.yaml` for override the image.

Kustomize offers primitives for customizing namespace, nameprefix, labels, annotations etc., you can read more about it on [Kustomize](https://github.com/kubernetes-sigs/kustomize) page.

```bash
# examine the manifests before deploying
kustomize build config/default
```
The above command will output the manifests on stdout so that it is easier to pipe it to `kubectl apply`

#### Customizing the controller manager manifests using Kustomize

`deploy` target in `Makefile` generates the base manifests, customizes the base manifests and then applies it to the configured Kubernetes cluster.

```bash
# deploy the controller manager to the cluster
make deploy
```

By now, you should have controller manager resources deployed in cluster. You
can examine controller manager pod by running

```bash
# deploy the controller manager to the cluster
kubectl get pods -n <namespace>
```

