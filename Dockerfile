# Stage 0: compile proto files

FROM golang:1.19.3-bullseye@sha256:34e901ebac66df44ce97b56a9e1bb407307e54fe13e843d6c59da7826ce4dd2c AS protoc

ENV PATH="${PATH}:$(go env GOPATH)/bin"

WORKDIR /tmp

RUN apt-get update && \
    apt-get install -y protobuf-compiler && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.28 && \
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.2

COPY ./api/protobuf ./protobuf

RUN mkdir genproto && \
    protoc --proto_path=protobuf \
	 --go_out=genproto --go_opt=paths=source_relative \
	 --go-grpc_out=genproto --go-grpc_opt=paths=source_relative \
	 $(find ./protobuf -iname "*.proto")

#####################################################################

# Stage 1: build the binary

FROM protoc AS build

WORKDIR /usr/local/src/rcs

COPY --from=protoc /tmp/genproto ./internal/genproto

COPY go.mod go.sum ./

RUN go mod download

COPY . ./

RUN go build -o /usr/local/bin/rcs ./cmd

#####################################################################

# Stage 2: deploy

FROM alpine:3.16.2@sha256:65a2763f593ae85fab3b5406dc9e80f744ec5b449f269b699b5efd37a07ad32e AS deploy

COPY --from=build /usr/local/bin/rcs /usr/local/bin/rcs
COPY rcs.json /var/rcs/rcs.json

RUN mkdir /lib64 && ln -s /lib/libc.musl-x86_64.so.1 /lib64/ld-linux-x86-64.so.2 && \
    addgroup -S rcs && adduser -S rcs -G rcs && \
    chown rcs:rcs /usr/local/bin/rcs && \
    chown rcs:rcs /var/rcs/rcs.json && \
    chmod 740 /usr/local/bin/rcs && \
    chmod 740 /var/rcs/rcs.json

USER rcs

EXPOSE 6121 6122 6123

ENTRYPOINT ["/usr/local/bin/rcs"]

CMD ["-c", "/var/rcs/rcs.json"]
