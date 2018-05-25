# Sample Project

Sample project is a project that created by kubebuilder and moved into `samples` folder. It contains resources and controllers. The main purpose of sample projects is to test and validate the behavior of kubebuilder. Specifically, the new built kubebuilder commands don't break existing projects created by older version of kubebuilder commands.

## Current Sample Projects
- memcached-api-apiserver

## Test Sample Project
Run following command to test a sample project such as memcached-api-apiserver
```
go test -v ./samples/memcached-api-apiserver
```

## Add Sample Project
We can test different aspects of kubebuilder in different sample projects. For example, we can test and validate how kubebuilder handles validation annotations in one sample project and validate how kubebuilder handles rbac rules in a different sample project. Here are steps to add a new project.

- Create a new subdirectory under samples to hold the new sample project and change directory to it
- run `kubebuilder init` to init a project
- run `kubebuilder create resource` to create resources you want to add
- Update the resource or controller files for your test purpose. For example, add validation annotations in resource file.
- Create the expected files under `test`. For example, memcached-api-server has an expected file `test/hack/install.yaml`, which is used to compare with the output of `kubebuilder create config`.
- Write `<project>_test.go` file to test the new sample project similar to memcached_test.go.


