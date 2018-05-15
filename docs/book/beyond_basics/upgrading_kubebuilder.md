{% panel style="info", title="Under Development" %}
This book is being actively developed.
{% endpanel %}

# Update Kubebuilder

## Update the Kubebuilder install

Download the latest version of kubebuilder from [releases page](https://github.com/kubernetes-sigs/kubebuilder/releases)
and install it.

## Update Existing Project's Dependencies

{% method %}

Update your project's dependencies to the latest version of the libraries used by kubebuilder.  This
will modify *Gopkg.toml* by rewriting the `[[override]]` elements beneath the
`# DO NOT MODIFY BELOW THIS LINE.` line.  Rules added by the user above this line will be retained.

{% sample lang="bash" %}
```bash
kubebuilder update vendor
```
{% endmethod %}


