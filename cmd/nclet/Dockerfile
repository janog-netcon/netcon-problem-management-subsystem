# Build the manager binary
FROM golang:1.19 as builder
ARG TARGETOS
ARG TARGETARCH

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY api/ api/
COPY cmd/nclet cmd/nclet
COPY controllers/nclet controllers/nclet
COPY pkg/ pkg/

# Build
# the GOARCH has not a default value to allow the binary be built according to the host where the command
# was called. For example, if we call make docker-build in a local env which has the Apple Silicon M1 SO
# the docker BUILDPLATFORM arg will be linux/arm64 when for Apple x86 it will be linux/amd64. Therefore,
# by leaving it empty we can ensure that the container and binary shipped on it will have the same platform.
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -a -o nclet ./cmd/nclet

FROM nicolaka/netshoot:v0.7
WORKDIR /

RUN apk add docker-cli

RUN wget https://github.com/srl-labs/containerlab/releases/download/v0.32.0/containerlab_0.32.0_linux_amd64.apk -O /tmp/containerlab.apk && \
    apk add --allow-untrusted /tmp/containerlab.apk && \
    rm -rf /tmp/containerlab.apk

COPY --from=builder /workspace/nclet .

ENTRYPOINT ["/nclet"]
