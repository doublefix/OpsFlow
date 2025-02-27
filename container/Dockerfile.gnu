FROM --platform=$BUILDPLATFORM golang:1.24.0-bookworm AS build
ARG TARGETOS
ARG TARGETARCH

WORKDIR /workspace
COPY . .
RUN go mod tidy
RUN go mod download
RUN GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -o opsflow ./main.go

FROM ubuntu:24.04
RUN apt-get update && apt-get install -y \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*
RUN useradd -m app
WORKDIR /home/app
COPY --from=build /workspace/opsflow /bin
RUN chown -R app:app /home/app
USER app

ENTRYPOINT ["/bin/opsflow"]

