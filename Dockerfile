FROM harbor.openpaper.co/base/golang:1.24.4-alpine3.22 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o opsflow cmd/main.go

FROM harbor.openpaper.co/base/alpine:3.22
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /app/opsflow /usr/local/bin/opsflow
ENTRYPOINT ["opsflow"]

# docker buildx build --platform linux/amd64 -t harbor.openpaper.co/chessbod/opsflow:20250610 --push .
