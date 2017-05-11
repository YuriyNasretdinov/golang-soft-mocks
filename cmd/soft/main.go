package main

import (
	"log"
	"os"
	"path/filepath"
	"runtime"
	"syscall"
)

var (
	gopath     = os.Getenv("GOPATH")
	goroot     = runtime.GOROOT()
	softDir    = filepath.Join(gopath, "soft")
	softGopath = filepath.Join(softDir, "gopath")
	softGoroot = filepath.Join(softDir, "goroot")
)

func main() {
	if gopath == "" {
		log.Fatal("GOPATH must be set")
	}

	if goroot == "" {
		log.Fatal("GOROOT must be set")
	}

	syncDir(gopath, softGopath)
	syncDir(goroot, softGoroot)

	// go root does not allow external imports, so we have to pretend that "soft" is actually golang package
	syncDir(filepath.Join(gopath, "github.com", "YuriyNasretdinov", "golang-soft-mocks"), filepath.Join(softGoroot, "soft"))

	os.Setenv("GOPATH", softGopath)
	os.Setenv("GOROOT", softGoroot)

	syscall.Exec(os.Args[1], os.Args[2:], os.Environ())
}
