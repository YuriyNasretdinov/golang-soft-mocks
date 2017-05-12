#!/bin/sh -e
go install $GOPATH/src/github.com/YuriyNasretdinov/golang-soft-mocks/cmd/soft
# it is a bit ugly to wrap everything in "sh -c" and $GOPATH, but it works for now
$GOPATH/bin/soft sh -c 'go run $GOPATH/src/github.com/YuriyNasretdinov/golang-soft-mocks/cmd/example/main.go'

