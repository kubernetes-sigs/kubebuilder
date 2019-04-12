# Build the manager binary
FROM golang:1.10.3 as builder

# Copy in the go src
WORKDIR /go/src/sigs.k8s.io/kubebuilder/test/project_v2
COPY vendor/ vendor/
COPY main.go main.go
COPY api/ api/
COPY controllers/ controllers/

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o manager sigs.k8s.io/kubebuilder/test/project_v2/

# Copy the controller-manager into a thin image
FROM ubuntu:latest
WORKDIR /
COPY --from=builder /go/src/sigs.k8s.io/kubebuilder/test/project_v2/manager .
ENTRYPOINT ["/manager"]
