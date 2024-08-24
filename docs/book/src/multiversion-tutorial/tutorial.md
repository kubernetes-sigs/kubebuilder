# Tutorial: Multi-Version API

Most projects start out with an alpha API that changes release to release.
However, eventually, most projects will need to move to a more stable API.
Once your API is stable though, you can't make breaking changes to it.
That's where API versions come into play.

Let's make some changes to the `CronJob` API spec and make sure all the
different versions are supported by our CronJob project.

If you haven't already, make sure you've gone through the base [CronJob
Tutorial](/cronjob-tutorial/cronjob-tutorial.md).

<aside class="note">

<h1>Following Along vs Jumping Ahead</h1>

Note that most of this tutorial is generated from literate Go files that
form a runnable project, and live in the book source directory:
[docs/book/src/multiversion-tutorial/testdata/project][tutorial-source].

[tutorial-source]: https://github.com/kubernetes-sigs/kubebuilder/tree/master/docs/book/src/multiversion-tutorial/testdata/project

</aside>

Next, let's figure out what changes we want to make...
