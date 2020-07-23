FROM golang:1.14.6-buster as builder

LABEL maintainer="zhangguanzhang <zhangguanzhang@qq.com>"

ARG mdbookVersion=v0.3.1

ENV GOBIN=/go/bin/ \
    CGO_ENABLED=0 \
    GO111MODULE="on" \
    GOPROXY="https://goproxy.cn,https://mirrors.aliyun.com/goproxy/,https://goproxy.io,direct"

COPY utils /go/src/utils

RUN os=$(go env GOOS) \
    && arch=$(go env GOARCH) \
    && echo $os $arch  \
    &&  if [ "${arch}" = "amd64" ]; then \
            arch="x86_64"; \
        elif [ "${arch}" = "x86" ]; then \
            arch="i686"; \
        fi \
    && case ${os} in \
        windows) \
            target="pc-windows-msvc" \
            ext="zip" \
            cmd="unzip -d /tmp" \
            ;; \
        darwin) \
            target="apple-darwin" \
            ;; \
        linux) \
            # works for linux, too
            target="unknown-${os}-gnu" \
            ;; \
        *) \
            target="unknown-${os}" \
            ;; \
    esac \
    && curl -L https://github.com/rust-lang-nursery/mdBook/releases/download/${mdbookVersion}/mdBook-${mdbookVersion}-${arch}-${target}.tar.gz \
    | tar -zxf - -C /go/bin/ \
    && go get sigs.k8s.io/controller-tools/cmd/controller-gen@v0.3.0 \
    && cd /go/src/utils/ \
    && go build -o /go/bin/literate-go ./litgo \
    && go build -o /go/bin/marker-docs ./markerdocs

FROM frolvlad/alpine-glibc
COPY --from=builder /go/bin/* /bin/

ENV IN_DOCKER=true

WORKDIR /opt
ENTRYPOINT ["/bin/mdbook"]
CMD ["build"]
