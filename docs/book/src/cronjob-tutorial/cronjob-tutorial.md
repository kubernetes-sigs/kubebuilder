# Tutorial: Building CronJob

Too many tutorials start out with some really contrived setup, or some toy
application that gets the basics across, and then stalls out on the more
complicated stuff.  Instead, this tutorial should take you through (almost)
the full gamut of complexity with Kubebuilder, starting off simple and
building up to something pretty full-featured.

Let's pretend (and sure, this is a teensy bit contrived) that we've
finally gotten tired of the maintenance burden of the non-Kubebuilder
implementation of the CronJob controller in Kubernetes, and we'd like to
rewrite it using KubeBuilder.

The job (no pun intended) of the *CronJob* controller is to run one-off
tasks on the Kubernetes cluster at regular intervals.  It does this by
building on top of the *Job* controller, whose task is to run one-off tasks
once, seeing them to completion.

Instead of trying to tackle rewriting the Job controller as well, we'll
use this as an opportunity to see how to interact with external types.

<aside class="note">

<h1>Following Along vs Jumping Ahead</h1>

Note that most of this tutorial is generated from literate Go files that
live in the book source directory:
[docs/book/src/cronjob-tutorial/testdata][tutorial-source].  The full,
runnable project lives in [project][tutorial-project-source], while
intermediate files live directly under the [testdata][tutorial-source]
directory.

[tutorial-source]: https://github.com/kubernetes-sigs/kubebuilder/tree/master/docs/book/src/cronjob-tutorial/testdata

[tutorial-project-source]: https://github.com/kubernetes-sigs/kubebuilder/tree/master/docs/book/src/cronjob-tutorial/testdata/project

</aside>

## Scaffolding Out Our Project

As covered in the [quick start](../quick-start.md), we'll need to scaffold
out a new project.  Make sure you've [installed
Kubebuilder](../quick-start.md#installation), then scaffold out a new
project:

```bash
# we'll use a domain of tutorial.kubebuilder.io,
# so all API groups will be <group>.tutorial.kubebuilder.io.
kubebuilder init --domain tutorial.kubebuilder.io
```

Now that we've got a project in place, let's take a look at what
Kubebuilder has scaffolded for us so far...
