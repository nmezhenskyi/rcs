# RCS

[![Go Report Card](https://goreportcard.com/badge/github.com/nmezhenskyi/rcs)](https://goreportcard.com/report/github.com/nmezhenskyi/rcs)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](https://github.com/nmezhenskyi/rcs/blob/main/LICENSE.md)

RCS, which stands for Remote Caching Server, is an in-memory key-value data store written in Go.
It is designed to be used in distributed systems. RCS prioritizes versatility over efficiency.
It provides three distinct APIs: Native, gRPC, and HTTP. This project was created out of curiosity
as a learning exercise. Inspired by Memcached and Redis.

## Features

- Highly customizable in-memory cache (in-progress)
- Three APIs for versatility: Native, gRPC, HTTP
- Optional state snapshots to disk (in-progress)

## API

RCS exposes three distinct APIs: *Native*, *gRPC*, and *HTTP*.

__Why?__ To allow for more flexibility when building a distributed system.
When some performance overhead is not a dealbreaker, it may be easier to communicate
with the cache server over HTTP with JSON payloads. Likewise, if your project already uses gRPC
to communicate between services, it would make sense to use it for RCS as well.
In a situation where you want to keep data communication minimalistic without unneccessary
dependencies (i.e., in a network of Raspberry Pis), the Native API would be a good choice.
Most importantly, you can use all three of these APIs simultaneously.  

In addition, you can disable APIs that you are not using. In the future it will also be possible
to build a binary containing only APIs that you need.

### Native

In progress...

### gRPC

In progress...

### HTTP

In progress...

## Internals

In progress...

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
