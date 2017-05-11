# Soft Mocks for Go!

This is proof-of-concept of Soft Mocks ( https://github.com/badoo/soft-mocks ) for Golang.

In order to make this POC work, you need to do the following:

```
$ go get github.com/YuriyNasretdinov/golang-soft-mocks
$ cd $GOPATH/src/github.com/YuriyNasretdinov/golang-soft-mocks
$ sudo chown -R `whoami` /usr/local/go
$ sudo ln -s $PWD /usr/local/go/src/soft
$ go run cmd/soft/main.go # first time function (*os.File).Close cannot be mocked because no files were rewritten yet
panic: Function cannot be mocked, it is not registered
...
$ go run cmd/soft/main.go # second time the rewritten file is used
File is going to be closed: /dev/null
Hello, world: %v! <nil>
$ mv /usr/local/go/src/os/file_unix.go{.bak,} # restore everything
```
