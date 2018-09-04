# Update Kubebuilder

## Update the Kubebuilder install

Install the latest version of kubebuilder from [releases page](https://github.com/kubernetes-sigs/kubebuilder/releases).

## Update Existing Project's Dependencies

{% panel style="warning", title="Kubebuilder 1.0.1 and earlier" %}

Before following the instructions below, make sure to update Kubebuilder
to 1.0.2+, update your dependency file to the latest version by using
`kubebuilder update vendor` (see
[below](#updating-to-new-major-versions)). See the [dependencies
guide](./dependencies.md) for more information on why this is necessary.

{% endpanel %}

You can update dependencies to minor and patch versions using
[dep](https://golang.github.io/dep/), as you would any other dependency in
your project.  See [the dependencies
guide](./dependencies.md#updating-existing-dependencies) for more
information.

### Updating to New Major Versions

{% method %}

Update your project's dependencies to the latest version of the libraries used by kubebuilder.  This
will modify *Gopkg.toml* by rewriting the `[[override]]` elements beneath the
`# DO NOT MODIFY BELOW THIS LINE.` line.  Rules added by the user above this line will be retained.

Gopkg.toml's without the `# DO NOT MODIFY BELOW THIS LINE.` will be ignored.

{% sample lang="bash" %}
```bash
kubebuilder update vendor
```
{% endmethod %}

#### By Hand

You can also update your project by hand.  Simply edit `Gopkg.toml` to
point to a new version of the dependencies listed under the `# DO NOT
MODIFY BELOW THIS LINE.` line, making sure that
`sigs.k8s.io/controller-tools` and `sigs.k8s.io/controller-runtime` always
have the same version listed.  You should then remove the marker line to
indicate that you've updated dependencies by hand, and don't want them
overridden.
