# Use the official Golang image as the base image
FROM golang:1.17-alpine

# Set the working directory
WORKDIR /app

# Copy the Go modules files
COPY go.mod .
COPY go.sum .

# Download the Go modules
RUN go mod download

# Copy the source code into the container
COPY . .

# Build the application
RUN go build -o websocket-server

# Expose the port that the WebSocket server will run on
EXPOSE 8080

# Set environment variables with default values (replace them with your own values if needed)
ENV PORT 8080
ENV LISTEN_ADDRESS 0.0.0.0

# Run the WebSocket server
CMD ["./websocket-server"]
