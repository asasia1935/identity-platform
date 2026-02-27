# 멀티스테이지
# 1단계: build
FROM golang:1.25-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o auth ./cmd/auth
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o gateway ./cmd/gateway

# 2단계: runtime
FROM alpine:latest
WORKDIR /app

COPY --from=builder /app/auth .
COPY --from=builder /app/gateway .

EXPOSE 8080
EXPOSE 8090

CMD ["./gateway"]