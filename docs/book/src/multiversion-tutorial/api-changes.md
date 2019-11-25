# Changing things up

A fairly common change in a Kubernetes API is to take some data that used
to be unstructured or stored in some special string format, and change it
to structured data.   Our `schedule` field fits the bill quite nicely for
this -- right now, in `v1`, our schedules look like

```yaml
schedule: "*/1 * * * *"
```

That's a pretty textbook example of a special string format (it's also
pretty unreadable unless you're a Unix sysadmin).

Let's make it a bit more structured.  According to the our [CronJob
code][cronjob-sched-code], we support "standard" Cron format.

In Kubernetes, **all versions must be safely round-tripable through each
other**.  This means that if we convert from version 1 to version 2, and
then back to version 1, we must not lose information.  Thus, any change we
make to our API must be compatible with whatever we supported in v1, and
also need to make sure anything we add in v2 is supported in v2.  In some
cases, this means we need to add new fields to v1, but in our case, we
won't have to, since we're not adding new functionality.

Keeping all that in mind, let's convert our example above to be
slightly more structured:

```yaml
schedule:
  minute: */1
```

Now, at least, we've got labels for each of our fields, but we can still
easily support all the different syntax for each field.

We'll need a new API version for this change.  Let's call it v2:

```shell
kubebuilder create api --group batch --version v2 --kind CronJob
```

Now, let's copy over our existing types, and make the change:

{{#literatego ./testdata/project/api/v2/cronjob_types.go}}

## Storage Versions

{{#literatego ./testdata/project/api/v1/cronjob_types.go}}

Now that we've got our types in place, we'll need to set up conversion...

[cronjob-sched-code]: /TODO.md
