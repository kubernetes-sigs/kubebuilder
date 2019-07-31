# Hubs, spokes, and other wheel metaphors

Since we now have two different versions, and users can request either
version, we'll have to define a way to convert between our version. For
CRDs, this is done using a webhook, similar to the defaulting and
validating webhooks we [defined in the base
tutorial](/cronjob-tutorial/webhook-implementation.md).  Like before,
controller-runtime will help us wire together the nitty-gritty bits, we
just have to implement the actual conversion.

Before we do that, though, we'll need to understand how controller-runtime
thinks about versions.  Namely:

## Complete graphs are insufficiently nautical

A simple approach to defining conversion might be to define conversion
functions to convert between each of our versions.  Then, whenever we need
to convert, we'd look up the appropriate function, and call it to run the
conversion.

This works fine when we just have two versions, but what if we had
4 types? 8 types? That'd be a lot of conversion functions.

Instead, controller-runtime models conversion in terms of a "hub and
spoke" model -- we mark one version as the "hub", and all other versions
just define conversion to and from the hub:

<!-- include these inline so we can style an match variables -->
<div class="diagrams">
{{#include ./complete-graph-8.svg}}
<div>becomes</div>
{{#include ./hub-spoke-graph.svg}}
</div>

Then, if we have to convert between two non-hub versions, we first convert
to the hub version, and then to our desired version:

<div class="diagrams">
{{#include ./conversion-diagram.svg}}
</div>

This cuts down on the number of conversion functions that we have to
define, and is modeled off of what Kubernetes does internally.

## What does that have to do with Webhooks?

When API clients, like kubectl or your controller, request a particular
version of your resource, the Kubernetes API server needs to return
a result that's of that version.  However, that version might not match
the version stored by the API server.

In that case, the API server needs to know how to convert between the
desired version and the stored version.  Since the conversions aren't
built in for CRDs, the Kubernetes API server calls out to a webhook to do
the conversion instead.  For KubeBuilder, this webhook is implemented by
controller-runtime, and performs the hub-and-spoke conversions that we
discussed above.

Now that we have the model for conversion down pat, we can actually
implement our conversions.

