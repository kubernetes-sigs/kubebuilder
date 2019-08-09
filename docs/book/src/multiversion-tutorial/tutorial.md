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

<aside class="note warning">

<h1>Minimum Kubernetes Versions Incoming!</h1>

CRD conversion support was introduced as an alpha feature in Kubernetes
1.13 (which means it's not on by default, and needs to be enabled via
a [feature gate][kube-feature-gates]), and became beta in Kubernetes 1.15
(which means it's on by default).

If you're on Kubernetes 1.13-1.14, make sure to enable the feature gate.
If you're on Kubernetes 1.12 or below, you'll need a new cluster to use
conversion. Check out the [KinD instructions](/reference/kind.md) for
instructions on how to set up a all-in-one cluster.

</aside>

Next, let's figure out what changes we want to make...

[kube-feature-gates]: https://kubernetes.io/docs/reference/command-line-tools-reference/feature-gates/ "Kubernetes Feature Gates"
