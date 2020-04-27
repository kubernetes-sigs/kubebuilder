# Benchwrapper

A small gRPC wrapper around the pubsub client library. This allows the
benchmarking code to prod at pubsub without speaking Go.

## Running

```bash
cd pubsub/internal/benchwrapper
export PUBSUB_EMULATOR_HOST=localhost:8080
go run *.go --port=8081
```
