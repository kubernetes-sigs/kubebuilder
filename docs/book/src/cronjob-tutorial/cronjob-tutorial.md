# Tutorial: Building CronJob

Too many tutorials start out with some really contrived setup, or some toy
application that gets the basics across, and then stalls out on the more
complicated stuff.  Instead, this tutorial should take you through (almost)
the full gamut of complexity with Kubebuilder, starting off simple and
building up to something pretty full-featured.

Pretend (and sure, this is a teensy bit contrived) that you've
finally gotten tired of the maintenance burden of the non-Kubebuilder
implementation of the CronJob controller in Kubernetes, and you'd like to
rewrite it using Kubebuilder.

The job (no pun intended) of the *CronJob* controller is to run one-off
tasks on the Kubernetes cluster at regular intervals.  It does this by
building on top of the *Job* controller, whose task is to run one-off tasks
once, seeing them to completion.

Instead of trying to tackle rewriting the Job controller as well, use
this as an opportunity to see how to interact with external types.

<aside class="note" role="note">

<p class="note-title">Following Along vs Jumping Ahead</p>

Note that most of this tutorial is generated from literate Go files that
live in the book source directory:
[docs/book/src/cronjob-tutorial/testdata][tutorial-source].  The full,
runnable project lives in [project][tutorial-project-source], while
intermediate files live directly under the [testdata][tutorial-source]
directory.

</aside>

## Scaffolding out our project

As covered in the [quick start](../quick-start.md), scaffold
out a new project.  Make sure you've [installed
Kubebuilder](../quick-start.md#installation), then scaffold out a new
project:

```bash
# create a project directory, and then run the init command.
mkdir project
cd project
# This example uses a domain of tutorial.kubebuilder.io,
# so all API groups is <group>.tutorial.kubebuilder.io.
kubebuilder init --domain tutorial.kubebuilder.io --repo tutorial.kubebuilder.io/project
```

<aside class="note" role="note">

Your project's name defaults to that of your current working directory.
You can pass `--project-name=<dns1123-label-string>` to set a different project name.

</aside>

Now that you have a project in place, take a look at what
Kubebuilder has scaffolded so far...

<aside class="note" role="note">

<p class="note-title">Developing in <code>$GOPATH</code></p>

If you initialize your project within [`GOPATH`][GOPATH-golang-docs], the implicitly called `go mod init` will interpolate the module path for you.
Otherwise `--repo=<module path>` must be set.

Read the [Go modules blogpost][go-modules-blogpost] if unfamiliar with the module system.

</aside>

[tutorial-source]: https://github.com/kubernetes-sigs/kubebuilder/tree/master/docs/book/src/cronjob-tutorial/testdata
[tutorial-project-source]: https://github.com/kubernetes-sigs/kubebuilder/tree/master/docs/book/src/cronjob-tutorial/testdata/project
[GOPATH-golang-docs]: https://golang.org/doc/code.html#GOPATH
[go-modules-blogpost]: https://blog.golang.org/using-go-modules