# distributed-services-with-go-pants

This repo contains the project detailed in [distributed services with go](https://pragprog.com/titles/tjgo/distributed-services-with-go/) while using pants for protobuf generation, linting and testing.

## Generate symlinks for protobufs

```bash
ln -s $(pwd)/dist/codegen/src/protobuf/api/v1/ $(pwd)/src/go/api_gen/
```
