package main

import (
	"log"
	"os"
	"os/exec"
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

	log.Printf("Starting to rewrite %s", goroot)
	os.Stderr.Write([]byte("\n"))

	syncDir(filepath.Join(goroot, "src"), filepath.Join(softGoroot, "src"))
	syncDir(filepath.Join(goroot, "pkg", "tool"), filepath.Join(softGoroot, "pkg", "tool"))
	syncDir(filepath.Join(goroot, "pkg", "include"), filepath.Join(softGoroot, "pkg", "include"))
	// go root does not allow external imports, so we have to pretend that "soft" is actually golang package
	syncDir(filepath.Join(gopath, "src", "github.com", "YuriyNasretdinov", "golang-soft-mocks"), filepath.Join(softGoroot, "src", "soft"))

	log.Printf("Starting to rewrite %s", gopath)
	os.Stderr.Write([]byte("\n"))

	syncDir(filepath.Join(gopath, "src", "github.com", "YuriyNasretdinov"), filepath.Join(softGopath, "src", "github.com", "YuriyNasretdinov"))

	os.Setenv("GOPATH", softGopath)
	os.Setenv("GOROOT", softGoroot)

	ex, err := exec.LookPath(os.Args[1])
	if err != nil {
		log.Fatalf("Could not find executable for %s: %s", os.Args[1], err.Error())
	}

	os.Stderr.Write([]byte("\n"))
	log.Printf("Running %s %v", ex, os.Args[1:])

	syscall.Exec(ex, os.Args[1:], os.Environ())
}
