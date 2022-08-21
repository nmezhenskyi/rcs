# syntax=docker/dockerfile:1

FROM golang:1.19-alpine3.15

WORKDIR /usr/local/rcs

RUN apk update && apk add curl bash unzip build-base autoconf automake libtool make g++ git

ENV PROTOBUF_VERSION 3.19.4
ENV PROTOBUF_URL https://github.com/google/protobuf/releases/download/v"$PROTOBUF_VERSION"/protobuf-cpp-"$PROTOBUF_VERSION".zip
RUN curl --silent -L -o protobuf.zip "$PROTOBUF_URL" && \
   unzip protobuf.zip && \
   cd protobuf-"$PROTOBUF_VERSION" && \
   ./configure && \
   make -j$(nproc) && \
   make install && \
   cd .. && rm protobuf.zip

RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.28
RUN go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.2
RUN export PATH="$PATH:$(go env GOPATH)/bin"

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

RUN make genproto
RUN make build

CMD ./bin/rcs
