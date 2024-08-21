# Stage 1: Build the Go binary
FROM golang:1.20-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o symctl

# Stage 2: Create a minimal image with the Go binary
FROM alpine:latest

WORKDIR /app

# Create the /app/bin directory
RUN mkdir -p /app/bin

# Copy the binary and any necessary plugins or dependencies
COPY --from=builder /app/symctl .

# Expose the port
EXPOSE 8080

CMD ["./symctl"]
