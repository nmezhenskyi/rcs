# RCS

[![Go Report Card](https://goreportcard.com/badge/github.com/nmezhenskyi/rcs)](https://goreportcard.com/report/github.com/nmezhenskyi/rcs)
![Build Workflow](https://github.com/nmezhenskyi/rcs/actions/workflows/go.yml/badge.svg)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](https://github.com/nmezhenskyi/rcs/blob/main/LICENSE.md)

RCS, which stands for Remote Caching Server, is an in-memory key-value data store written in Go.
It is designed to be used in distributed systems. RCS prioritizes versatility over efficiency.
It provides three distinct APIs: Native, gRPC, and HTTP. This project was created out of curiosity
as a learning exercise. Inspired by Memcached and Redis.

## Features

- Highly customizable in-memory cache (in-progress)
- Exposes three APIs for versatility: Native, gRPC, HTTP
- Supports SSL connections
- Optional state snapshots to disk (in-progress)
- Optional structured / unstructured logging
- Packaged as a single binary file

## API

RCS exposes three distinct APIs: *Native*, *gRPC*, and *HTTP*.

__Why?__ To allow for more flexibility when building a distributed system.
When some performance overhead is not a dealbreaker, it may be easier to communicate
with the cache server over HTTP with JSON payloads. Likewise, if your project already uses gRPC
to communicate between services, it would make sense to use it for RCS as well.
In a situation where you want to keep data communication minimalistic without unnecessary
dependencies (i.e., in a network of Raspberry Pis), the Native API would be a good choice.
More importantly, you can use all three of these APIs simultaneously.  

In addition, you can disable APIs that you are not using. It is also possible to completely remove HTTP
and/or gRPC APIs together with their related dependencies from the binary. To do this you need to use
build tags `rmgrpc` and/or `rmhttp`.

### Native

Native API uses a custom application layer protocol (RCSP) built on top of TCP/IP. The complete
specification can be found [here](https://github.com/nmezhenskyi/rcs/blob/main/api/native/rcs.md).
The API supports and encourages long-living connections over one-off requests. There are no client
libraries for the RCSP yet, so you would have to implement one according to the specification.
The API supports SSL connections.

### gRPC

gRPC API uses `rcs.proto` file, which can be found
[here](https://github.com/nmezhenskyi/rcs/blob/main/api/protobuf/rcs.proto),
to generate the service and proto messages. You should use this file to generate client bindings with
`protoc`. The API supports SSL connections.

### HTTP

HTTP API exposes HTTP end-points and communicates using JSON payloads. The OpenAPI specification can be
found [here](https://github.com/nmezhenskyi/rcs/blob/main/api/openapi/rcs.yaml).

## Internals

Internally, RCS uses a hash table with strings as keys and stores values in binary representation.
In future releases RCS will support multiple storage & eviction strategies, as well as cache serialization
to disk storage.

## Getting Started

### Build from source

Prerequisites:

- Go 1.18 compiler and tools
- Protocol buffer compiler v3
- Go plugins for protoc
- GNU Make (optional)

Steps:

1. Clone the repository:
   ```sh
   git clone https://github.com/nmezhenskyi/rcs.git
   ```
2. Generate protobuf and grpc files: 
   ```sh
   mkdir -p internal/genproto
   
   protoc --proto_path=api/protobuf \
   --go_out=internal/genproto --go_opt=paths=source_relative \
   --go-grpc_out=internal/genproto --go-grpc_opt=paths=source_relative \
   rcs.proto
   ```
3. Build the source code:
   ```sh
   go build -o <destination> ./cmd
   ```

Alternatively if you have Make installed:

1. Clone the repository: `git clone https://github.com/nmezhenskyi/rcs.git`.
2. Run `make setup` command. This will generate protobuf and grpc files, build the project,
and create the binary in the `./bin` directory.

### Run

To run RCS you would need to first create the configuration file `rcs.json`.

### Containerize

There is a ready-to-use [Dockerfile](https://github.com/nmezhenskyi/rcs/blob/main/Dockerfile) based
on Alpine Linux.

## Contributing

Feel free to create a pull request with new features and/or bug fixes.
Please address a single concern in a PR and provide unit tests and documentation.
When commiting follow [conventional commits](https://www.conventionalcommits.org/en/v1.0.0/).

## License

The project is licensed under MIT License.
