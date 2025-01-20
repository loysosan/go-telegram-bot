# Use the official Golang image as the base image
FROM golang:1.23

# Set the working directory inside the container
WORKDIR /app

# Copy go.mod and go.sum files to the container
COPY go.mod go.sum ./

# Download and cache Go modules
RUN go mod download

# Ensure the go.mod and go.sum files are up to date
RUN go mod tidy

# Copy the rest of the application files to the container
COPY . .

# Build the Go application
RUN go build -o main .

# Expose the port that the application will run on
EXPOSE 8080

# Command to run the application
CMD ["./main"]
