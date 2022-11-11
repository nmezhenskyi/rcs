# Stage 0: compile proto files

FROM golang:1.19.3-bullseye@sha256:34e901ebac66df44ce97b56a9e1bb407307e54fe13e843d6c59da7826ce4dd2c AS protoc

ENV PATH="${PATH}:$(go env GOPATH)/bin"

WORKDIR /tmp

RUN apt-get update && apt install -y protobuf-compiler

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

FROM golang:1.19.3-alpine3.16@sha256:5dca1a586da5bc601c77a50d489d7fa752fa3fdd2fb22fd3f8f5b4b2f77181d6 AS build

WORKDIR /usr/local/src/rcs

COPY --from=protoc /tmp/genproto ./internal/genproto

COPY go.mod go.sum ./

RUN go mod download

COPY . ./

RUN go build -o /usr/local/bin/rcs ./cmd

EXPOSE 6121 6122 6123

CMD ["/usr/local/bin/rcs"]

#####################################################################
