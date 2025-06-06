FROM golang:1.21-alpine

WORKDIR /app

# Copy files
COPY . .

# Download dependencies and build
RUN go mod download
RUN go build -o main .

# Expose port
EXPOSE 8080

# Run the application
CMD ["./main"]