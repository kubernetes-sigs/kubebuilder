# Build the manager binary
FROM golang:1.10.3 as builder

# Copy in the go src
WORKDIR /go/src/sigs.k8s.io/kubebuilder/test/e2e/e2e-ehon
COPY cmd/    cmd/
COPY vendor/ vendor/
COPY pkg/    pkg/

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o manager sigs.k8s.io/kubebuilder/test/e2e/e2e-ehon/cmd/manager

# Copy the controller-manager into a thin image
FROM ubuntu:latest
WORKDIR /
COPY --from=builder /go/src/sigs.k8s.io/kubebuilder/test/e2e/e2e-ehon/manager .
ENTRYPOINT ["/manager"]
