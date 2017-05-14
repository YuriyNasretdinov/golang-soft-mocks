#!/bin/sh -e
go install github.com/YuriyNasretdinov/golang-soft-mocks/cmd/soft
$GOPATH/bin/soft go run cmd/example/main.go
$GOPATH/bin/soft go test ./cmd/example/example_test.go

