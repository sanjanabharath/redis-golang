FROM golang:1.21-alpine

WORKDIR /app

COPY . .

RUN go build -o redis-go main.go

EXPOSE 6379

CMD ["./redis-go"]