# RCS

[![Go Report Card](https://goreportcard.com/badge/github.com/nmezhenskyi/rcs)](https://goreportcard.com/report/github.com/nmezhenskyi/rcs)

RCS, which stands for Remote Caching Server, is an in-memory key-value data store written in Go.
It is designed to be used in distributed systems. RCS prioritizes versatility over efficiency.
It provides three distinct APIs: gRPC, HTTP, and Native. This project was created
out of curiosity as a learning exercise. Inspired by Memcached and Redis.

## Features

- Highly customizable in-memory cache (in-progress)
- Three APIs for versatility: gRPC, HTTP, Native
- Optional state snapshots to disk (in-progress)

## API

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

1. Clone the repository `git clone https://github.com/nmezhenskyi/rcs.git`.
2. Generate protobuf & grpc files.
3. Build the source code.

Alternatively if you have Make installed:

1. Clone the repository `git clone https://github.com/nmezhenskyi/rcs.git`.
2. Run `make setup`. This will generate protobuf & grpc files, build the project, and
create the binary in `./bin` directory.

### Run

In progress...

### Containerize

There is a ready-to-use [Dockerfile](https://github.com/nmezhenskyi/rcs/blob/main/Dockerfile) based
on Alpine Linux.

## Contributing

Feel free to create a pull request with new features and/or bug fixes.
Please address a single concern in a PR and provide unit tests and documentation.
When commiting follow [conventional commits](https://www.conventionalcommits.org/en/v1.0.0/).

## License

The project is licensed under MIT License.
