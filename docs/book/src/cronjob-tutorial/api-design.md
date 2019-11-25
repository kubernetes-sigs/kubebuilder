# Designing an API

In Kubernetes, we have a few rules for how we design APIs. Namely, all
serialized fields *must* be `camelCase`, so we use JSON struct tags to
specify this.  We can also use the `omitempty` struct tag to mark that
a field should be omitted from serialization when empty.

Fields may use most of the primitive types.  Numbers are the exception:
for API compatibility purposes, we accept three forms of numbers: `int32`
and `int64` for integers, and `resource.Quantity` for decimals.

<details><summary>Hold up, what's a Quantity?</summary>

Quantities are a special notation for decimal numbers that have an
explicitly fixed representation that makes them more portable across
machines.  You've probably noticed them when specifying resources requests
and limits on pods in Kubernetes.

They conceptually work similar to floating point numbers: they have
a significand, base, and exponent. Their serializable and human readable format
uses whole numbers and suffixes to specify values much the way we describe
computer storage.

For instance, the value `2m` means `0.002` in decimal notation.  `2Ki`
means `2048` in decimal, while `2K` means `2000` in decimal.  If we want
to specify fractions, we switch to a suffix that lets us use a whole
number: `2.5` is `2500m`.

There are two supported bases: 10 and 2 (called decimal and binary,
respectively).  Decimal base is indicated with "normal" SI suffixes (e.g.
`M` and `K`), while Binary base is specified in "mebi" notation (e.g. `Mi`
and `Ki`).  Think [megabytes vs
mebibytes](https://en.wikipedia.org/wiki/Binary_prefix).

</details>

There's one other special type that we use: `metav1.Time`.  This functions
identically to `time.Time`, except that it has a fixed, portable
serialization format.

With that out of the way, let's take a look at what our CronJob object
looks like!

{{#literatego ./testdata/project/api/v1/cronjob_types.go}}

Now that we have an API, we'll need to write a controller to actually
implement the functionality.
