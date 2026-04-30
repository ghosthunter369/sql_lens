FROM golang:1.22-alpine AS builder

ENV GOPROXY=https://goproxy.cn,https://proxy.golang.org,direct

WORKDIR /app

# Copy all source files first
COPY . .

# Download and verify dependencies
RUN go mod tidy && go mod download

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -o sql-lens-server ./cmd/server

# Runtime stage
FROM alpine:3.20

WORKDIR /app

# Install ca-certificates for HTTPS
RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /app/sql-lens-server /app/sql-lens-server

EXPOSE 8080

CMD ["/app/sql-lens-server"]
