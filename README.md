# Soft Mocks for Go!

This is Soft Mocks analogue ( https://github.com/badoo/soft-mocks ) for Golang. The main difference from https://github.com/bouk/monkey is in that there is no "assembler hackery": we just rewrite all GOROOT and GOPATH files and inject a bit of code at the beginning of each function so that it can be mocked. It allows for it to be cross-platform and to work with any build options. This is an example of how it works:

```
$ go get github.com/YuriyNasretdinov/golang-soft-mocks
$ cd $GOPATH/src/github.com/YuriyNasretdinov/golang-soft-mocks
$ cat run-example.sh
#!/bin/sh -e
go install github.com/YuriyNasretdinov/golang-soft-mocks/cmd/soft
$GOPATH/bin/soft go run cmd/example/main.go
$GOPATH/bin/soft go test ./cmd/example/example_test.go

$ ./run-example.sh # rewrites everything in GOPATH and GOROOT to a separate directory
...

File is going to be closed: /dev/null
Hello, world: <nil>!

...

ok  	command-line-arguments	0.006s
```

# Usage ("soft" command)
You install `soft` command by running

```
$ go get github.com/YuriyNasretdinov/golang-soft-mocks/cmd/soft
```

In order to run something under soft mocks (e.g. a test) you prefix your command with `$GOPATH/bin/soft` or just `soft` if you have `$GOPATH/bin` in your PATH:

```
$ $GOPATH/bin/soft go test example_test.go
```

It will rewrite all contents of GOROOT and GOPATH and then run your command with different GOROOT and GOPATH.

# Usage (methods)

There are several functions in "soft" package:

## soft.Mock(src, dst func)
Replace implementation of `src` to `dst`. You can specify both functions and methods.
The following test passes when run using soft.Mock:

## soft.CallOriginal(f func, args ...interface{}) []interface{}
Calls the original function with specified arguments.

## soft.Reset(f func(...) ...)
Resets mock for function f. Function is restored to its original state.

## soft.ResetAll()
Resets all mocks. Designed to be used in either setUp or tearDown of a test suite.

## Example

```

func TestExample(t *testing.T) {
	soft.Mock(os.Open, func(filename string) (*os.File, error) {
		return nil, errors.New("Cannot open files!")
	})

	if _, err := os.Open(os.DevNull); err == nil {
		t.Fatalf("Must be error opening dev null!")
	}

	soft.ResetAll()

	fp, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatalf("Must be no errors opening dev null!")
	}
	defer fp.Close()

	// You cannot use fp.Close because we do not support replacing functions only for a specific receiver.
	closeFunc := (*os.File).Close

	soft.Mock(closeFunc, func(f *os.File) error {
		log.Printf("File is going to be closed: %s", f.Name())
		res, _ := soft.CallOriginal(closeFunc, f)[0].(error)
		return res
	})
}
```

# Limitations
Currently some packages cannot be rewritten because it would otherwise cause cyclic imports (these packages are used by soft mocks themselves):

 * sync/atomic
 * sync
 * reflect
 * soft
 * runtime
 * math
 * unsafe
 * strconv
 * internal
 * errors
 * unicode/utf8

You wouldn't probably need to mock these packages anyway because they mostly contain pure functions.

# Upgrading and uninstalling
Soft Mocks creates cache in $GOPATH/soft directory. So if you want to uninstall it or upgrade to a new version, run

```
$ rm -rf $GOPATH/soft
```
