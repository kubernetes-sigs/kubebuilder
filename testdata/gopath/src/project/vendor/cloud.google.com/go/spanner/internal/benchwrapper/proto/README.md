# Regenerating protos

```
cd spanner/internal/benchwrapper/proto
protoc --go_out=plugins=grpc:. *.proto
```