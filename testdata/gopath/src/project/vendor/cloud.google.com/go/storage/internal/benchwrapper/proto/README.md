# Regenerating protos

```
cd storage/internal/benchwrapper/proto
protoc --go_out=plugins=grpc:. *.proto
```