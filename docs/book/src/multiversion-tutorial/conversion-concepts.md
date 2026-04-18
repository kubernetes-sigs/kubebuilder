# Hubs, spokes, and other wheel metaphors

Since there are now two different versions, and users can request either
version, define a way to convert between versions. For
CRDs, this is done using a webhook, similar to the defaulting and
validating webhooks [defined in the base
tutorial](../cronjob-tutorial/webhook-implementation.md).  Like before,
controller-runtime helps wire together the nitty-gritty bits, you
just have to implement the actual conversion.

Before doing that, understand how controller-runtime
thinks about versions.  Namely:

## Complete graphs are insufficiently nautical

A simple approach to defining conversion might be to define conversion
functions to convert between each of the versions.  Then, whenever you need
to convert, you'd look up the appropriate function, and call it to run the
conversion.

This works fine when there are just two versions, but what if there were
4 types? 8 types? That'd be a lot of conversion functions.

Instead, controller-runtime models conversion in terms of a "hub and
spoke" model -- mark one version as the "hub", and all other versions
just define conversion to and from the hub:

<!-- include these inline so we can style an match variables -->
<div class="diagrams">
{{#include ./complete-graph-8.svg}}
<div>becomes</div>
{{#include ./hub-spoke-graph.svg}}
</div>

Then, to convert between two non-hub versions, first convert
to the hub version, and then to the desired version:

<div class="diagrams">
{{#include ./conversion-diagram.svg}}
</div>

This cuts down on the number of conversion functions to
define, and is modeled off of what Kubernetes does internally.

## What does that have to do with webhooks?

When API clients, like kubectl or your controller, request a particular
version of your resource, the Kubernetes API server needs to return
a result that is of that version.  However, that version might not match
the version stored by the API server.

In that case, the API server needs to know how to convert between the
desired version and the stored version.  Since the conversions are not
built in for CRDs, the Kubernetes API server calls out to a webhook to do
the conversion instead.  For Kubebuilder, controller-runtime implements
this webhook, and performs the hub-and-spoke conversions
discussed above.

Now that the model for conversion is clear, actually
implement the conversions.

