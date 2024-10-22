/*
Copyright 2023 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package multiversion

const cronjobSpecComment = `/*
 First, let's take a look at our spec.  As we discussed before, spec holds
 *desired state*, so any "inputs" to our controller go here.

 Fundamentally a CronJob needs the following pieces:

 - A schedule (the *cron* in CronJob)
 - A template for the Job to run (the
 *job* in CronJob)

 We'll also want a few extras, which will make our users' lives easier:

 - A deadline for starting jobs (if we miss this deadline, we'll just wait till
   the next scheduled time)
 - What to do if multiple jobs would run at once (do we wait? stop the old one? run both?)
 - A way to pause the running of a CronJob, in case something's wrong with it
 - Limits on old job history

 Remember, since we never read our own status, we need to have some other way to
 keep track of whether a job has run.  We can use at least one old job to do
 this.

 We'll use several markers (` + "`// +comment`" + `) to specify additional metadata.  These
 will be used by [controller-tools](https://github.com/kubernetes-sigs/controller-tools) when generating our CRD manifest.
 As we'll see in a bit, controller-tools will also use GoDoc to form descriptions for
 the fields.
*/`

const concurrencyPolicyComment = `/*
 We define a custom type to hold our concurrency policy.  It's actually
 just a string under the hood, but the type gives extra documentation,
 and allows us to attach validation on the type instead of the field,
 making the validation more easily reusable.
*/`

const statusDesignComment = `/*
 Next, let's design our status, which holds observed state.  It contains any information
 we want users or other controllers to be able to easily obtain.

 We'll keep a list of actively running jobs, as well as the last time that we successfully
 ran our job.  Notice that we use ` + "`metav1.Time`" + ` instead of ` + "`time.Time`" + ` to get the stable
 serialization, as mentioned above.
*/`

const boilerplateComment = `/*
 Finally, we have the rest of the boilerplate that we've already discussed.
 As previously noted, we don't need to change this, except to mark that
 we want a status subresource, so that we behave like built-in kubernetes types.
*/`

const boilerplateReplacement = `// +kubebuilder:docs-gen:collapse=old stuff

/*
 Since we'll have more than one version, we'll need to mark a storage version.
 This is the version that the Kubernetes API server uses to store our data.
 We'll chose the v1 version for our project.

 We'll use the [` + "`+kubebuilder:storageversion`" + `](/reference/markers/crd.md) to do this.

 Note that multiple versions may exist in storage if they were written before
 the storage version changes -- changing the storage version only affects how
 objects are created/updated after the change.
*/`
