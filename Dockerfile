# syntax=docker/dockerfile:1

FROM golang:1.18-alpine

WORKDIR /usr/local/rcs

RUN apk --update add build-base

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

RUN go mod tidy

# TODO: setup protoc

RUN make genproto
RUN make build
RUN make run
