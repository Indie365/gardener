module github.com/gardener/gardener/hack/tools/logcheck

go 1.21

// This is a separate go module to decouple the gardener codebase and production binaries from dependencies that are
// only needed to build the logcheck tool
require (
	golang.org/x/exp v0.0.0-20240222234643-814bf88cf225
	// this has to be kept in sync with the used golangci-lint version
	// use go version -m hack/tools/bin/golangci-lint to detect the dependency versions
	golang.org/x/tools v0.18.0
)

require golang.org/x/mod v0.15.0 // indirect
