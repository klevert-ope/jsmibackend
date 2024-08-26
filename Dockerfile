# Start from the official Golang image
FROM golang:1.22-alpine AS build

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download

# Copy the source code from the current directory to the Working Directory inside the container
COPY . .

# Build the Go app
RUN go build -o main .

# Start a new stage from scratch
FROM alpine:3.19

# Create a group and user to run the application with non-root privileges
RUN addgroup -S app && adduser -S app -G app

# Set the Working Directory to /app and change the ownership to the app user
WORKDIR /app
RUN chown app:app /app

# Copy the Pre-built binary file from the previous stage
COPY --from=build /app/main /app/main

# Copy the migrations directory from the source code to the Working Directory inside the container
COPY --from=build /app/db/migrations /app/db/migrations

# Change the ownership of the binary file and migrations directory to the app user
RUN chown -R app:app /app/main /app/db/migrations

# Expose port 8000 to the outside world
EXPOSE 8000

# Switch to the app user
USER app

# Command to run the executable
CMD ["./main"]
