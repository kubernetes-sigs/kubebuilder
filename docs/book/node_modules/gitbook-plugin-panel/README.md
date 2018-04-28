# Panel plugin for GitBook

Gitbook is a great tool but specially the cli version is lacking some features. The aim of this plugin is adding one that is really needed: Panels.

## Cool, can I see it working?

The following image shows all the panels you can create:

![terminal themes](https://github.com/davidmogar/gitbook-plugin-panel/blob/resources/images/panels.png?raw=true)

## How can I use this plugin?

You only have to edit your book.json and modify it adding something like this:

```json
"plugins" : [ "panel" ],
```

Now, to define your panels you will have to add a content similar to the next one:

```
{% panel %}
Panel without title.
{% endpanel %}

{% panel title="This is a panel with title" %}
Panel with title and default style.
{% endpanel %}

{% panel style="danger", title="This is a danger panel" %}
Panel with title and danger style.
{% endpanel %}

{% panel style="info", title="This is an info panel" %}
Panel with title and info style.
{% endpanel %}

{% panel style="success", title="This is a success panel" %}
Panel with title and success style.
{% endpanel %}

{% panel style="warning", title="This is a warning panel" %}
Panel with title and warning style.
{% endpanel %}
```

Just choose the panel you want and add it! Awesome right?
