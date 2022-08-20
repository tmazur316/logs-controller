# syntax=docker/dockerfile:1

FROM golang:1.19-alpine

WORKDIR /go/src/log_controller
COPY . /go/src/log_controller
RUN go mod download
CMD ["go", "run", "./benchmark/benchmark1/benchmark1.go"]
