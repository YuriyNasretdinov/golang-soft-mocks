#!/bin/sh -e
go install github.com/YuriyNasretdinov/golang-soft-mocks/cmd/soft
$GOPATH/bin/soft go run cmd/example/main.go

