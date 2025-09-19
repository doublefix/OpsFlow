# Build the opsflow binary
FROM harbor.openpaper.co/base/golang:1.24 AS builder
ARG TARGETOS
ARG TARGETARCH

WORKDIR /workspace

# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
RUN go mod download

# Copy the go source
COPY . .

# Build
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -a -o opsflow cmd/main.go

# Use distroless as minimal base image to package the opsflow binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM harbor.openpaper.co/base/distroless-static:nonroot
WORKDIR /
COPY --from=builder /workspace/opsflow .
USER 65532:65532

ENTRYPOINT ["opsflow"]

# docker buildx build --platform linux/amd64 -t harbor.openpaper.co/chessbod/opsflow:20250702 --push .
