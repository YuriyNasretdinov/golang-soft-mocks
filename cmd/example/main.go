package main

import (
	"fmt"
	"os"
	"soft"
)

func main() {
	closeFunc := (*os.File).Close
	soft.Mock(closeFunc, func(f *os.File) error {
		fmt.Printf("File is going to be closed: %s\n", f.Name())
		res, _ := soft.CallOriginal(closeFunc, f)[0].(error)
		return res
	})
	fp, _ := os.Open("/dev/null")
	fmt.Printf("Hello, world: %v!\n", fp.Close())
}
