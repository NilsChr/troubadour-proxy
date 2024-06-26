# Start from the official Go image to create a build artifact.
FROM golang:1.22 as builder

# Set the Current Working Directory inside the container.
WORKDIR /app

# Copy go mod and sum files
COPY go.mod  ./

# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed.
RUN go mod download

# Copy the source code into the container.
COPY . .

# Build the Go app as a static binary.
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o troubadour-proxy .

# Start a new stage from scratch to keep the final image clean and small.
FROM alpine:latest

# Set the Current Working Directory inside the container
WORKDIR /root/

# Copy the Pre-built binary file from the previous stage.
COPY --from=builder /app/troubadour-proxy .

# Expose port 8080 to the outside world
EXPOSE 8080

# Command to run the executable
CMD ["./troubadour-proxy"]
