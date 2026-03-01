# Build stage
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Install git for go mod download
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o operator cmd/operator/main.go

# Final stage
FROM alpine:3.19

RUN apk --no-cache add ca-certificates

WORKDIR /

# Copy binary from builder
COPY --from=builder /app/operator .

# Run as non-root user
RUN adduser -D -u 1000 operator
USER 1000

EXPOSE 8080

ENTRYPOINT ["/operator"]
