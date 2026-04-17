# Changing things up

A fairly common change in a Kubernetes API is to take some data that used
to be unstructured or stored in some special string format, and change it
to structured data.   Our `schedule` field fits the bill quite nicely for
this -- right now, in `v1`, our schedules look like

```yaml
schedule: "*/1 * * * *"
```

That is a pretty textbook example of a special string format (it is also
pretty unreadable unless you are a Unix sysadmin).

Make it a bit more structured.  According to the [CronJob
code][cronjob-sched-code], it supports "standard" Cron format.

In Kubernetes, **all versions must be safely round-tripable through each
other**.  This means that if you convert from version 1 to version 2, and
then back to version 1, you must not lose information.  Thus, any change you
make to your API must be compatible with whatever you supported in v1, and
you also need to make sure anything you add in v2 is supported in v1.  In some
cases, this means you need to add new fields to v1, but in this case, you
will not have to, since you are not adding new functionality.

Keeping all that in mind, convert the example above to be
slightly more structured:

```yaml
schedule:
  minute: */1
```

Now, at least, there are labels for each of the fields, but you can still
easily support all the different syntax for each field.

A new API version is needed for this change.  Call it v2:

```shell
kubebuilder create api --group batch --version v2 --kind CronJob
```

Press `y` for "Create Resource" and `n` for "Create Controller".

Now, copy over the existing types, and make the change:

{{#literatego ./testdata/project/api/v2/cronjob_types.go}}

## Storage versions

{{#literatego ./testdata/project/api/v1/cronjob_types.go}}

Now that the types are in place, set up conversion...

[cronjob-sched-code]: ./multiversion-tutorial/testdata/project/api/v2/cronjob_types.go "CronJob Code"
