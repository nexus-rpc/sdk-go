# Nexus Go SDK

Go SDK for Nexus.

This project contains multiple packages:

- [nexusapi](./nexusapi) Generated Protocol Buffers / gRPC / gRPC Gateway API definitions.
- [cmd](./cmd) Scripts for building the generated files.
- [nexusclient](./nexusclient) High level SDK client (TODO).

### Prerequisites

- [go](https://go.dev/doc/install)
- [protoc](https://grpc.io/docs/protoc-installation/)

### Build

Build the generated `nexusapi` directory.

```shell
go run ./cmd build
```

### Clean

Clean the generated `nexusapi` directory.

```shell
go run ./cmd clean
```
