{% panel style="info", title="Under Development" %}
This book is being actively developed.
{% endpanel %}

# Defining Boilerplate License Headers

{% method %}

It is possible to add boilerplate license headers to all generated code by
modifying `hack/boilerplate.go.txt`.

If you don't create `boilerplate.go.txt` an empty version will be created for you by
`kubebuilder init`.  Modifying this file will only impact files created afterward.

{% sample lang="bash" %}

```bash
mkdir hack
echo "// MY LICENSE" > hack/boilerplate.go.txt
kubebuilder init --domain k8s.io
```

{% endmethod %}

