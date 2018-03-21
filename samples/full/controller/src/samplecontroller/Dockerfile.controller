# Instructions to install API using the installer
# Build and test the controller-manager
FROM golang:1.9.3 as builder

ENV TEST_ASSET_DIR /usr/local/bin
ENV TEST_ASSET_KUBECTL $TEST_ASSET_DIR/kubectl
ENV TEST_ASSET_KUBE_APISERVER $TEST_ASSET_DIR/kube-apiserver
ENV TEST_ASSET_ETCD $TEST_ASSET_DIR/etcd

# Download test framework binaries
ENV TEST_ASSET_URL https://storage.googleapis.com/k8s-c10s-test-binaries
RUN curl ${TEST_ASSET_URL}/etcd-Linux-x86_64 --output $TEST_ASSET_ETCD
RUN curl ${TEST_ASSET_URL}/kube-apiserver-Linux-x86_64 --output $TEST_ASSET_KUBE_APISERVER
RUN curl https://storage.googleapis.com/kubernetes-release/release/v1.9.2/bin/linux/amd64/kubectl --output $TEST_ASSET_KUBECTL
RUN chmod +x $TEST_ASSET_ETCD
RUN chmod +x $TEST_ASSET_KUBE_APISERVER
RUN chmod +x $TEST_ASSET_KUBECTL

# Copy in the go src
WORKDIR /go/src/samplecontroller
COPY pkg/    pkg/
COPY cmd/    cmd/
COPY vendor/ vendor/

# Build and test the API code
RUN go build -a -o controller-manager ./cmd/controller-manager/main.go
RUN go test ./pkg/... ./cmd/...

# Copy the controller-manager into a thin image
FROM ubuntu:latest  
# RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /go/src/samplecontroller/controller-manager .
CMD ["./controller-manager", "--install-crds=false"]  
