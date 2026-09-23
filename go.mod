module github.com/bazel-contrib/buildtools/v10

go 1.26.0

toolchain go1.27.1

require (
	github.com/golang/protobuf v1.5.4
	github.com/google/go-cmp v0.7.0
	go.starlark.net v0.0.0-20210223155950-e043a3d3c984
	google.golang.org/protobuf v1.36.10
)

require golang.org/x/tools v0.50.0 // indirect

tool golang.org/x/tools/cmd/goyacc
