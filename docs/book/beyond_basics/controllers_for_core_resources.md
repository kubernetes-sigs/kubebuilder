## Controllers for Core Resources

{% method %}

It is possible to create Controllers for Core Resources, or for Resources defined outside your project.

{% sample lang="bash" %}
```bash
kubebuilder create api --group apps --version v1 --kind Deployment
```
{% endmethod %}

{% panel style="warning", title="Failing Scaffold Tests" %}
When creating controllers for core and existing types, it may be necessary to modify
the scaffold tests before they pass.

This is because the empty object may not be valid as required fields are not set by the scaffolding.
{% endpanel %}
