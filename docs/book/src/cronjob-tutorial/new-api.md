# Adding a new API

To scaffold out a new Kind (you were paying attention to the [last
chapter](./gvks.md#kinds-and-resources), right?) and corresponding
controller, we can use `kubebuilder create api`:

```bash
kubebuilder create api --group batch --version v1 --kind CronJob
```

Press `y` for "Create Resource" and "Create Controller".

The first time we call this command for each group-version, it will create
a directory for the new group-version.

<aside class="note">

<h1>Supporting older cluster versions</h1>

The default CustomResourceDefinition manifests created alongside your Go API types
use API version `v1`. If your project intends to support Kubernetes cluster versions older
than v1.16, you must set `--crd-version v1beta1` and remove `preserveUnknownFields=false`
from the `CRD_OPTIONS` Makefile variable.
See the [CustomResourceDefinition generation reference][crd-reference] for details.

[crd-reference]: /reference/generating-crd.md#supporting-older-cluster-versions

</aside>

In this case, the
[`api/v1/`](https://sigs.k8s.io/kubebuilder/docs/book/src/cronjob-tutorial/testdata/project/api/v1)
directory is created, corresponding to the
`batch.tutorial.kubebuilder.io/v1` (remember our [`--domain`
setting](cronjob-tutorial.md#scaffolding-out-our-project) from the
beginning?).

It has also added a file for our `CronJob` Kind,
`api/v1/cronjob_types.go`.  Each time we call the command with a different
kind, it'll add a corresponding new file.

Let's take a look at what we've been given out of the box, then we can
move on to filling it out.

{{#literatego ./testdata/emptyapi.go}}

Now that we've seen the basic structure, let's fill it out!
