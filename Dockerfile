# Use the official Go image as a base
FROM golang:1.24-alpine AS builder

# Set the working directory inside the container
WORKDIR /app

# Copy go.mod and go.sum files
COPY go.mod go.sum ./

# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod tidy

# Copy the rest of the application source code
COPY . .

# IMPORTANT: Removed COPY localhost.crt localhost.key ./ as Render handles TLS

# Build the Go application
# CGO_ENABLED=0 disables CGO, which produces a statically linked executable
# -ldflags="-s -w" removes debug information and symbol table, reducing binary size
RUN go build -o main ./main.go

# --- Start a new stage for a smaller final image ---
FROM alpine:latest

# Set the working directory
WORKDIR /app

# Copy the built executable from the builder stage
COPY --from=builder /app/main .

# IMPORTANT: Removed COPY localhost.crt /app/localhost.key ./ as Render handles TLS

# Expose the port your application listens on.
# Render will typically assign a port via the 'PORT' env var, but 8080 is a common default.
EXPOSE 8080

# Command to run the executable
CMD ["./main"]
