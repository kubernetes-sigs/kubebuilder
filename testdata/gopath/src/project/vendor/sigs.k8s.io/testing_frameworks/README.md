# Frameworks

Test frameworks for testing kubernetes

This was created as a result of [kubernetes/community#1524](https://github.com/kubernetes/community/pull/1524)

## What lives here?

- The [integration test framework](integration/)

## What is allowed to live here?

Any test framework for testing any part of kubernetes is welcome so long as we
can avoid vendor loops.

Right now, the only things vendored into this repo are
[ginkgo](https://github.com/onsi/ginkgo) and
[gomega](https://github.com/onsi/gomega). We would like to keep vendored
libraries to a minimum in order to make it as easy as possible to import these
frameworks into other kubernetes repos. Code in this repo should certainly
never import `k8s.io/kubernetes`.
