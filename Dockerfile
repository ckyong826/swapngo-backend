# Stage 1: Build the Go binary
FROM golang:1.25-alpine AS builder

# Install build essentials
RUN apk add --no-cache gcc musl-dev

WORKDIR /app

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the application
# We build the main.go inside /cmd/api/
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/main ./cmd/api/main.go

# Stage 2: Final lightweight image
FROM alpine:latest

# 🌟 FIX: Install ca-certificates AND tzdata
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /root/

# Copy the binary and .env
COPY --from=builder /app/main .
COPY --from=builder /app/.env . 

# 🌟 OPTIONAL: Set the environment variable so the OS defaults to your timezone
ENV TZ=Asia/Kuala_Lumpur

EXPOSE 8080

CMD ["./main"]