FROM golang:1.21-alpine AS builder

ENV CGO_ENABLED=1
ENV GOOS=linux
ENV GOARCH=amd64

# Set the working directory in the container
WORKDIR /app

# Install gcc to enable CGO
RUN apk add --no-cache gcc musl-dev

# Copy the current directory contents into the container at /app
COPY . .

# Build the Go app
RUN go mod download && go build -o main .

# Start a new stage from scratch
FROM alpine:latest

# Set the working directory in the container
WORKDIR /app

# Copy the pre-built binary from the previous stage
COPY --from=builder /app/main .
COPY --from=builder /app/config/config_prod.yaml ./config/config.yaml
COPY --from=builder /app/database ./database

EXPOSE 7878

# Command to run the executable
CMD ["./main"]
