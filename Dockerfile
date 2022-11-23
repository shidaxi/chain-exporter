FROM golang:1.18-alpine as builder

WORKDIR /app

RUN apk add gcc musl-dev

ADD . .

RUN go build -o chain-exporter main.go

FROM alpine

WORKDIR /app

COPY --from=builder /app/chain-exporter .

EXPOSE 9060

ENTRYPOINT ./chain-exporter
