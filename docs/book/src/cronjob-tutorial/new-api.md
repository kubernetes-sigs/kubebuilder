# Adding a new API

To scaffold out a new Kind (you were paying attention to the [last
chapter](./gvks.md#kinds-and-resources), right?) and corresponding
controller, use `kubebuilder create api`:

```bash
kubebuilder create api --group batch --version v1 --kind CronJob
```

Press `y` for "Create Resource" and "Create Controller".

The first time you call this command for each group-version, it creates
a directory for the new group-version.

In this case, the command creates the
[`api/v1/`](https://sigs.k8s.io/kubebuilder/docs/book/src/cronjob-tutorial/testdata/project/api/v1)
directory, corresponding to the
`batch.tutorial.kubebuilder.io/v1` (remember our [`--domain`
setting](cronjob-tutorial.md#scaffolding-out-our-project) from the
beginning?).

It has also added a file for the `CronJob` Kind,
`api/v1/cronjob_types.go`.  Each time you call the command with a different
kind, it adds a corresponding new file.

Take a look at what you have been given out of the box, then
move on to filling it out.

{{#literatego ./testdata/emptyapi.go}}

Now that you have seen the basic structure, fill it out!
