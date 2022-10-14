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
RUN --mount=type=cache,target=/root/.cache/go-build CGO_ENABLED=0 go build -o nclet ./cmd/nclet

FROM nicolaka/netshoot:v0.7
WORKDIR /

RUN apk add docker-cli

RUN wget https://github.com/srl-labs/containerlab/releases/download/v0.32.0/containerlab_0.32.0_linux_amd64.apk -O /tmp/containerlab.apk && \
    apk add --allow-untrusted /tmp/containerlab.apk && \
    rm -rf /tmp/containerlab.apk

COPY --from=builder /workspace/nclet .

ENTRYPOINT ["/nclet"]
