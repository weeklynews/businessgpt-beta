FROM golang:1.21-alpine

WORKDIR /app

# Copy go.mod first
COPY go.mod ./

# Download dependencies (this will generate go.sum)
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the application
RUN go build -o main .

# Expose port
EXPOSE 8080

# Run the application
CMD ["./main"]