# Use the official Go image as a base for building (builder stage)
FROM golang:1.24-alpine AS builder

# Set the working directory inside the container for the builder stage
WORKDIR /app

# Copy go.mod and go.sum files to leverage Docker's layer caching
# This step ensures that dependencies are re-downloaded only if go.mod or go.sum changes
COPY go.mod go.sum ./

# Download all Go module dependencies
# `go mod tidy` ensures all necessary modules are present and unused ones are removed
RUN go mod tidy

# Copy the rest of the application source code into the builder stage
# This includes all your Go files and the 'static' directory
COPY . .

# Build the Go application
# CGO_ENABLED=0 disables CGO, producing a statically linked executable which is good for Alpine
# -ldflags="-s -w" removes debug information and symbol table, significantly reducing binary size
RUN go build -o main ./main.go

# --- Start a new stage for the smaller final production image ---
FROM alpine:latest

# Set the working directory in the final image
WORKDIR /app

# Copy the built executable from the builder stage into the final image
COPY --from=builder /app/main .

# NEW: Copy the 'static' directory from the builder stage into the final image
# This is crucial for serving your HTML home page and any other static assets
COPY --from=builder /app/static ./static

# Expose the port your application listens on.
# While Render typically assigns a PORT env var, EXPOSE is good for documentation
# and for running the container in other environments.
EXPOSE 8080

# Command to run the executable when the container starts
# This will execute your compiled Go application
CMD ["./main"]