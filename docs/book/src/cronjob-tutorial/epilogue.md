# Epilogue

By this point, we've got a pretty full-featured implementation of the
CronJob controller, made use of most of the features of
Kubebuilder, and written tests for the controller using envtest.

If you want more, head over to the [Multi-Version
Tutorial](/multiversion-tutorial/tutorial.md) to learn how to add new API
versions to a project.

Additionally, you can try the following steps on your own -- we'll have
a tutorial section on them Soon™:

- adding [additional printer columns][printer-columns] `kubectl get`

[printer-columns]: /reference/generating-crd.md#additional-printer-columns
