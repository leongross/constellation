# Build the manager binary
FROM golang:1.20.2@sha256:bd4a3e7eee6d6ea30b2e27d6c1ac3c56809e78e08c7e44ddf91f8c741091f5ad as builder

# Download project root dependencies
WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Download operator dependencies
WORKDIR /workspace/operators/constellation-node-operator
COPY operators/constellation-node-operator/go.mod ./
COPY operators/constellation-node-operator/go.sum ./
RUN go mod download all

WORKDIR /workspace
COPY . .

# Build
WORKDIR /workspace/operators/constellation-node-operator
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o manager main.go

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:nonroot@sha256:ddde70b96543be368208791ad8ddc9b483cbb33a67bce861e73ad519bc0ed616 as release
WORKDIR /
COPY --from=builder /workspace/operators/constellation-node-operator/manager .
USER 65532:65532

ENTRYPOINT ["/manager"]
