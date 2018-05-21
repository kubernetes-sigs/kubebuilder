{% panel style="info", title="Under Development" %}
This book is being actively developed.
{% endpanel %}

## Controllers for Core Resources

{% panel style="warning", title="Core Only Projects" %}
If your project will *only* contain controllers for types defined outside your project,
you must create the project with the `--controller-only` flag.
{% endpanel %}

{% method %}

It is possible to create Controllers for Core resources, or for resources defined outside your project.

{% sample lang="bash" %}
```bash
kubebuilder create controller --group apps --version v1beta2 --kind Deployment --core-type
```
{% endmethod %}

{% panel style="warning", title="Scaffold Tests May Not Pass" %}
When creating controllers for core and existing types, it may be necessary to modify
the scaffold tests before they pass.

This is because the empty objet may not be valid and have required fields not
set in the scaffold test.
{% endpanel %}
