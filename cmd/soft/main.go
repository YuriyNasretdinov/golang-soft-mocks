package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"soft"
)

func main() {
	filename := filepath.Join(runtime.GOROOT(), "src", "os", "file_unix.go")

	if err := rewriteFile(filename); err != nil {
		log.Printf("Could not rewrite %s: %s", filename, err)
		return
	}

	closeFunc := (*os.File).Close
	soft.Mock(closeFunc, func(f *os.File) error {
		fmt.Printf("File is going to be closed: %s\n", f.Name())
		res, _ := soft.CallOriginal(closeFunc, f)[0].(error)
		return res
	})

	fp, _ := os.Open("/dev/null")
	err := fp.Close()

	fmt.Println("Hello, world: %v!", err)
}
