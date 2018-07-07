# Defining Boilerplate License Headers

{% method %}

It is possible to add boilerplate license headers to all generated code by
defining `hack/boilerplate.go.txt` before initializing a project.

If you don't create `boilerplate.go.txt` an apache2 boilerplate header before
running init an apache2 header will be created for you by default.

{% sample lang="bash" %}
```bash
mkdir hack
echo "// MY LICENSE" > hack/boilerplate.go.txt
kubebuilder init --domain k8s.io
```
{% endmethod %}

