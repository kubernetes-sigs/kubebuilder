### Deployment

Now we have all our code changes and manifests in place, so let's deploy it to
the cluster and test it out.

Ensure that you have installed [cert-manager](../cronjob-tutorial/cert-manager.md) `0.9.0+` version in your cluster. We have
tested the instructions in this tutorial with [0.9.0-alpha.0](https://github.com/jetstack/cert-manager/releases/tag/v0.9.0-alpha.0) release.

Running `make deploy` will deploy the controller-manager in the cluster.

### Testing

Now that we have deployed the controller-manager with conversion webhook enabled, let's test out the version conversion feature.
We will do the following to perform a simple version conversion test:

 - Create disk object named `disk-sample` using v1 specification
 - Get disk object `disk-sample` using v2 version
 - Get disk object `disk-sample` using v3 version

#### 1. Create v1 disk object

```yaml
{{#include ./testdata/project/config/samples/infra_v1_disk.yaml}}
```

```bash
kubectl apply -f config/samples/infra_v1_disk.yaml
```

#### 2. Get disk object using v2 version

```bash
kubectl get disks.v2.infra.kubebuilder.io/disk-sample -o yaml
```

```yaml
apiVersion: infra.kubebuilder.io/v2 <-- note the v2 version
kind: Disk
metadata:
  name: disk-sample
  selfLink: /apis/infra.kubebuilder.io/v2/namespaces/default/disks/disk-sample
  uid: 0e9be0fd-a284-11e9-bbbe-42010a8001af
spec:
  price: <-- note the structured price object
    amount: 10
    currency: USD
status: {}
```


#### 3. Get disk object using v3 version

```bash
kubectl get disks.v3.infra.kubebuilder.io/disk-sample -o yaml
```
```yaml
apiVersion: infra.kubebuilder.io/v3 <-- note the v3 version
kind: Disk
metadata:
  name: disk-sample
  selfLink: /apis/infra.kubebuilder.io/v3/namespaces/default/disks/disk-sample
  uid: 0e9be0fd-a284-11e9-bbbe-42010a8001af <-- note the same uid as v2
  ....
spec:
  pricePerGB: <-- note the pricePerGB name of the field
    amount: 10
    currency: USD
status: {}
```


### Troubleshooting
TODO(../TODO.md) steps for troubleshoting
