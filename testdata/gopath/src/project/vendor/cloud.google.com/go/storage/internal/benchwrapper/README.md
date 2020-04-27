# Benchwrapper

A small gRPC wrapper around the storage client library. This allows the
benchmarking code to prod at storage without speaking Go.

## Running

```
cd storage/internal/benchwrapper
export STORAGE_EMULATOR_HOST=localhost:8080
go run *.go --port=8081
```
