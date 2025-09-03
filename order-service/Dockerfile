FROM golang:1.24-alpine as builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o order-service ./cmd/main.go

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/order-service .
COPY --from=builder /app/web ./web

EXPOSE 8081

CMD ["./order-service"]