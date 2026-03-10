# Build stage
FROM golang:1.25-alpine AS builder

ARG VERSION=dev
ARG COMMIT=none
ARG DATE=unknown

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build binary
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w \
      -X github.com/varaxlabs/onax/internal/version.Version=${VERSION} \
      -X github.com/varaxlabs/onax/internal/version.Commit=${COMMIT} \
      -X github.com/varaxlabs/onax/internal/version.Date=${DATE}" \
    -o onax ./cmd/operator

# Final stage - Distroless base image
FROM gcr.io/distroless/static:nonroot

WORKDIR /

# Copy binary from builder
COPY --from=builder /app/onax .

USER 65532:65532

EXPOSE 8080 8081

ENTRYPOINT ["/onax"]
