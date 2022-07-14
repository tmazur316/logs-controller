# syntax=docker/dockerfile:1

FROM golang:1.18-alpine

WORKDIR /go/src/logs-collector
COPY . /go/src/logs-collector
RUN go mod download
RUN go build -o ./logs-controller
CMD ["./logs-controller"]