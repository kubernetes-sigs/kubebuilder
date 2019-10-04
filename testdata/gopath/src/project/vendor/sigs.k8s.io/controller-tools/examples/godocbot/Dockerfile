# Build and test the manager binary
FROM golang:1.9.3 as builder

# Copy in the go src
WORKDIR /go/src/sigs.k8s.io/controller-tools/examples/godocbot
COPY pkg/    pkg/
COPY cmd/    cmd/
COPY vendor/ vendor/

# Run tests as a sanity check
ENV TEST_ASSET_DIR /usr/local/bin
ENV TEST_ASSET_KUBECTL $TEST_ASSET_DIR/kubectl
ENV TEST_ASSET_KUBE_APISERVER $TEST_ASSET_DIR/kube-apiserver
ENV TEST_ASSET_ETCD $TEST_ASSET_DIR/etcd
ENV TEST_ASSET_URL https://storage.googleapis.com/k8s-c10s-test-binaries
RUN curl ${TEST_ASSET_URL}/etcd-Linux-x86_64 --output $TEST_ASSET_ETCD
RUN curl ${TEST_ASSET_URL}/kube-apiserver-Linux-x86_64 --output $TEST_ASSET_KUBE_APISERVER
RUN curl https://storage.googleapis.com/kubernetes-release/release/v1.9.2/bin/linux/amd64/kubectl --output $TEST_ASSET_KUBECTL
RUN chmod +x $TEST_ASSET_ETCD
RUN chmod +x $TEST_ASSET_KUBE_APISERVER
RUN chmod +x $TEST_ASSET_KUBECTL
RUN go test ./pkg/... ./cmd/...

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o manager sigs.k8s.io/controller-tools/examples/godocbot/cmd/manager

# Copy the controller-manager into a thin image
FROM ubuntu:latest
WORKDIR /
COPY --from=builder /go/src/sigs.k8s.io/controller-tools/examples/godocbot/manager .
ENTRYPOINT ["/manager"]
