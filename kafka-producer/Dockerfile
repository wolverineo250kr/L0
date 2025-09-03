FROM golang:1.24-alpine

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o kafka-producer ./main.go

EXPOSE 9000

CMD ["./kafka-producer"]
