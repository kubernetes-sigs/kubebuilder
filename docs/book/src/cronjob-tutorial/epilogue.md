# Epilogue

By this point, we've got a pretty full-featured implementation of the
CronJob controller, and have made use of most of the features of
KubeBuilder.

If you want more, head over to the [Multi-Version
Tutorial](/multiversion-tutorial/tutorial.md) to learn how to add new API
versions to a project.

Additionally, you can try the following steps on your own -- we'll have
a tutorial section on them Soon™: 

- writing unit/integration tests (check out [envtest][envtest])
- adding [additional printer columns][printer-columns] `kubectl get`

[envtest]: https://godoc.org/sigs.k8s.io/controller-runtime/pkg/envtest

[printer-columns]: /reference/generating-crd.md#additional-printer-columns
