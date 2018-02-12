#!/usr/bin/env sh
set -eu -o pipfail

go test --tags stubpkg ./internal/... > testdata/standard-quiet-format
go test -v --tags stubpkg ./internal/... > testdata/standard-verbose-format
go test --json --tags stubpkg ./internal/... > testdata/go-test-json-output
go test --json --tags 'stubpkg timeoutstub' ./internal/... > testdata/go-test-json-output-timeout


