# Build the manager binary
FROM golang:1.19 as builder

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


# Build the access-helper binary
FROM golang:1.19 as access-helper_builder

ARG DEFAULT_USERNAME
ARG DEFAULT_PASSWORD

WORKDIR /workspace
COPY go.mod go.mod
COPY go.sum go.sum
RUN go mod download

COPY api/ api/
COPY cmd/access-helper cmd/access-helper
COPY pkg/ pkg/

# Build
RUN --mount=type=cache,target=/root/.cache/go-build CGO_ENABLED=0 \
    go build -o access-helper -ldflags="-X main.defaultUsername=${DEFAULT_USERNAME} -X main.defaultPassword=${DEFAULT_PASSWORD}" ./cmd/access-helper


# Build the containerlab binary
FROM golang:1.19 as containerlab_builder

WORKDIR /workspace
COPY containerlab/go.mod go.mod
COPY containerlab/go.sum go.sum
RUN go mod download

COPY containerlab/ .

# Build
RUN --mount=type=cache,target=/root/.cache/go-build make build CGO_ENABLED=0


FROM nicolaka/netshoot:v0.7
WORKDIR /

RUN apk add docker-cli

COPY --from=builder /workspace/nclet .
COPY --from=access-helper_builder /workspace/access-helper /bin
COPY --from=containerlab_builder /workspace/bin/containerlab /bin/clab

ENTRYPOINT ["/nclet"]
