{% panel style="info", title="Under Development" %}
This book is being actively developed.
{% endpanel %}

**Note:** Impatient readers head straight to [Quick Start](quick_start.md).

## Who is this for

#### Users of Kubernetes

Users of Kubernetes will develop a deeper understanding Kubernetes through learning
the fundamental concepts behind how APIs are designed and implemented.  This book
will teach readers how to develop their own Kubernetes APIs and the
principals from which the core Kubernetes APIs are designed.

Including:

- The structure of Kubernetes APIs and Resources
- API versioning semantics
- Self-healing
- Garbage Collection and Finalizers
- Declarative vs Imperative APIs
- Level-Based vs Edge-Base APIs
- Resources vs Subresources

#### Kubernetes API extension developers

API extension developers will learn the principals and concepts behind implementing canonical
Kubernetes APIs, as well as simple tools and libraries for rapid execution.  This
book covers pitfalls and miss-conceptions that extension developers commonly encounter.

Including:

- How to batch multiple events into a single reconciliation call
- How to configure periodic reconciliation
- *Forthcoming*
    - When to use the lister cache vs live lookups
    - Garbage Collection vs Finalizers
    - How to use Declarative vs Webhook Validation
    - How to implement API versioning

## Navigating this book

This section describes how to use the navigation elements of this book

##### Code Navigation

Code samples may be either displayed to the side of the corresponding documentation, or inlined
immediately afterward.  This setting may be toggled using the split-screen icon at the left side
of the top nav.

##### Table of Contents

The table of contents may be hidden using the hamburger icon at the left side of the top nav.

##### OS / Language Navigation

Some chapters have code snippets for multiple OS or Languages.  These chapters will display OS
or Language selections at the right side of the top nav, which may be used to change the
OS or Language of the examples shown.
