# Regenerating protos

```
cd pubsub/internal/benchwrapper/proto
protoc --go_out=plugins=grpc:. *.proto
```
