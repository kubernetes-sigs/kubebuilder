# CBT: Cloud Bigtable Tool

This is the source for `cbt`.

## Build and Run
To build the tool locally, run this command in this directory:

```
go build .
```

That will build the cbt binary in that directory. To run commands with the `cbt` you built (rather
than the official one) invoke it with the directory prefix like this:

```
./cbt help
```

## Documentation

This tool gets documented on the [Go packages website](https://godoc.org/cloud.google.com/go/bigtable/cmd/cbt)
as well as the [cloud site](https://cloud.google.com/bigtable/docs/cbt-reference).

### Go Package Documentation

This command will generate a file with a package description which gets used for the godoc.org. You should update this after any changes to the usage or descriptions of the commands. To
generate the file, run:

```
go generate
```

The output will be in [cbtdoc.go](cbtdoc.go).

You may want to verify this looks good locally. To do that, you will need to generate the doc into
your GOPATH version of the directory, then run `godoc` and you can view the [local version](http://localhost:6060/pkg/cloud.google.com/go/bigtable/cmd/cbt/)

```
go run . -o $(go env GOPATH)/src/cloud.google.com/go/bigtable/cmd/cbt/cbtdoc.go doc
godoc
```

### Cloud Site Documentation

The Cloud documentation uses the `cbt mddoc` command to generate part of the [cbt Reference](https://cloud.google.com/bigtable/docs/cbt-reference) page.
To preview what it will look like upon generation, you can generate it into a file with the command:

```
go run . -o doc.md mddoc
```

This will create a file doc.md. You don't need to check it into this repository, so delete
it once you are happy with the output.

## Configuration

The configuration for the options (`-project`, `-instance`, and `-creds`) is in [cbtconfig.go](../../internal/cbtconfig/cbtconfig.go).
So change that file if you need to modify those.