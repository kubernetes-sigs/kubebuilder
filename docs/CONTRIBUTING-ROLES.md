Contributing Roles
==================

## Direct Code-Related Roles

While anyone (who's signed the [CLA and follows the code of
conduct](../CONTRIBUTING.md)) is welcome to contribute to the Kubebuilder
project, we've got two "formal" roles that carry additional privileges and
responsibilities: *reviewer* and *approver*.

In a nutshell, reviewers and approvers are officially recognized to make
day-to-day and overarching technical decisions within parts of the
project, or the project as a whole.  We follow a similar set of
definitions to the [main Kubernetes project itself][kube-ladder], with
slightly looser requirements.

As much as possible, we want people to help take on responsibility in the
project -- these guidelines are attempts to make it *easier* for this to
happen, *not harder*.  If you've got any questions, just reach out on
Slack to one of the [subproject leads][kb-leads] (called
kubebuilder-admins in the `OWNERS_ALIASES` file).

## Prerequisite: Member

Anyone who wants to become a reviewer or approver must first be a [member
of the Kubernetes project][kube-member].  The aforementioned doc has more
details, but the gist is that you must have made a couple contributions to
some part of the Kubernetes project -- *this includes Kubebuilder and
related repos*.  Then, you need two existing members to sponsor you.

**If you've contributed a few times to Kubebuilder, we'll be happy to
sponsor you, just ping us on Slack :-)**

## Reviewers

Reviewers are recognized as able to provide code reviews for parts of the
codebase, and are entered into the `reviewers` section of one or more
`OWNERS` files.  You'll get auto-assigned reviews for your area of the
codebase, and are generally expected to review for both correctness,
testing, general code organization, etc.  Reviewers may review for design
as well, but approvers have the final say on that.

Things to look for:

- does this code work, and is it written performantly and idomatically?
- is it tested?
- is it organized nicely?  Is it maintainable?
- is it documented?
- does it need to be threadsafe?  Is it?
- Take a glance at the stuff for approvers, if you can.

Reviewers' `/lgtm` marks are generally trusted by approvers to mean that
the code is ready for one last look-over before merging.

### Becoming a Reviewer

The criteria for becoming a reviewer are:

- Give 5-10 reviews on PRs
- Contribute or review 3-5 PRs substantially (i.e. take on the role of the
  defacto "main" reviewer for the PR, contribute a bugfix or feature, etc)

Usually, this will need to occur within a single repository, but if you've
worked on a cross-cutting feature, it's ok to count PRs across
repositories.

Once you meet those criteria, submit yourself as a reviewer in the
`OWNERS` file or files that you feel represent your areas of knowlege via
a PR to the relevant repository.

## Approvers

Approvers provide the final say as to whether a piece of code is merged.
Once approvals (`/approve`) are given for each piece of the affected code
(and a reviewer or approver has added `/lgtm`), the code will merge.

Approvers are responsible for giving the code a final once-over before
merge, and doing an overall design/API review.

Things to look for:

- Does the API exposed to the user make sense, and is it easy to use?
- Is it backwards compatible?
- Will it accommodate new changes in the future?
- Is it extesnible/layerable (see [DESIGN.md](../DESIGN.md))?
- Does it expose a new type from `k8s.io/XYZ`, and, if so, is it worth it?
  Is that piece well-designed?

**For large changes, approvers are responsible for getting reasonable
consensus**.  With the power to approve such changes comes the
responsibility of ensuring that the project as a whole has time to discuss
them.

### Becoming an Approver

All approvers need to start out as reviewers.  The criteria for becoming
an approver are:

- Be a reviewer in the area for a couple months
- Be the "main" reviewer or contributor for 5-10 substantial (bugfixes,
  features, etc) PRs where approvers did not need to leave substantial
  additional comments (i.e. where you were acting as a defacto approver).

Once you've met those criteria, you can submit yourself as an approver
using a PR that edits the revelant `OWNERS` files appropriately.  The
existing approvers will then approve the change with lazy consensus.  If
you feel more comfortable asking before submitting the PR, feel free to
ping one of the [subproject leads][kb-leads] (called kubebuilder-admins in
the `OWNERS_ALIASES` file) on Slack.

## Indirectly Code-Related/Non-Code Roles

We're always looking help with other areas of the project as well, such
as:

### Docs

Docs contributors are always welcome.  Docs folks can also become
reviewers/approvers for the book by following the same process above.

### Triage

Help triaging our issues is also welcome.  Folks doing triage are
responsible for using the following commands to mark PRs and issues with
one or more labels, and should also feel free to help answer questions:

- `/kind {bug|feature|documentation}`: things that are broken/new
  things/things with lots of words, repsectively

- `/triage support`: questions, and things that might be bugs but might
  just be confusion of how to use something

- `/priority {backlog|important-longterm|important-soon|critical-urgent}`:
  how soon we need to deal with the thing (if someone wants
  to/eventually/pretty soon/RIGHT NOW OMG THINGS ARE ON FIRE,
  respectively)

- `/good-first-issue`: this is pretty straightforward to implement, has
  a clear plan, and clear criteria for being complete

- `/help`: this could feasibly still be picked up by someone new-ish, but
  has some wrinkles or nitty-gritty details that might not make it a good
  first issue

See the [Prow reference](https://prow.k8s.io/command-help) for more
details.

[kube-ladder]: https://github.com/kubernetes/community/blob/master/community-membership.md "Kubernetes Community Membership"

[kube-member]: https://github.com/kubernetes/community/blob/master/community-membership.md#member "Kubernetes Project Member"

[kb-leads]: ../OWNERS_ALIASES "Root OWNERS file -- kubebuilder-admins"
