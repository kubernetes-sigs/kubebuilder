{% panel style="info", title="Under Development" %}
This book is being actively developed.
{% endpanel %}

## Controllers for Core Resources

{% method %}

It it possible to create Controllers for Core resources, or for resources defined outside your project.

{% sample lang="bash" %}
```bash
kubebuilder create controller --group apps --version v1beta2 --kind Deployment --core-type
```
{% endmethod %}

{% panel style="warning", title="Core Only Projects" %}
If your project will *only* contain controllers for types defined outside your project,
you must create the project with the `--controller-only` flag.
{% endpanel %}
