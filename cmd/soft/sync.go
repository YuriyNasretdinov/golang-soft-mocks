package main

import (
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
)

func statsEqual(a, b os.FileInfo) bool {
	if a.IsDir() || b.IsDir() {
		return a.IsDir() == b.IsDir()
	}

	if a.Mode().IsRegular() != b.Mode().IsRegular() {
		return false
	}

	if a.Mode().IsRegular() && !a.ModTime().Equal(b.ModTime()) {
		return false
	}

	return true
}

func sync(fi os.FileInfo, dirFrom, dirTo string) {
	name := fi.Name()
	from := filepath.Join(dirFrom, name)
	to := filepath.Join(dirTo, name)

	if fi.IsDir() {
		syncDir(from, to)
		return
	}

	mode := fi.Mode()

	if !mode.IsRegular() {
		if mode&os.ModeSymlink == os.ModeSymlink {
			lnk, err := os.Readlink(from)
			if err != nil {
				log.Printf("Could not read link: %s", err.Error())
				return
			}

			err = os.Symlink(lnk, to)
			if err != nil {
				log.Printf("Could not create symlink: %s", err.Error())
				return
			}
		} else {
			log.Printf("Creation of anything but symlinks is not suppoted yet (%s)", from)
		}
		return
	}

	newContents, err := rewriteFile(from)
	if err != nil {
		log.Printf("Could not rewrite file %s: %s", from, err.Error())
		os.Stderr.Write([]byte("\n"))
		newContents, err = ioutil.ReadFile(from)
		if err != nil {
			log.Printf("Could not read %s: %s", from, err.Error())
			return
		}
	}

	err = ioutil.WriteFile(to, newContents, fi.Mode().Perm())
	if err != nil {
		log.Printf("Could not write %s: %s", to, err.Error())
		return
	}

	err = os.Chtimes(to, fi.ModTime(), fi.ModTime())
	if err != nil {
		log.Printf("Could not chtimes %s: %s", to, err.Error())
		return
	}
}

// TODO: maybe make configurable?
var ignoreDirs = map[string]bool{
	".git":        true,
	".hg":         true,
	".unrealsync": true,
}

func syncDir(dirFrom, dirTo string) {
	os.Stderr.WriteString("\033[A\033[2K" + dirFrom + "\n")

	if ignoreDirs[filepath.Base(dirFrom)] {
		return
	}

	if _, err := os.Lstat(dirTo); err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(dirTo, 0777); err != nil {
				log.Printf("Could not create target dir %s: %s", dirFrom, err.Error())
				return
			}
		}
	}

	fromList, err := ioutil.ReadDir(dirFrom)
	if err != nil {
		log.Printf("Could not read from %s: %s", dirFrom, err.Error())
		return
	}

	toList, err := ioutil.ReadDir(dirTo)
	if err != nil {
		log.Printf("Could not read target dir %s: %s", dirTo, err.Error())
		return
	}

	fromMap := make(map[string]os.FileInfo)
	toMap := make(map[string]os.FileInfo)

	for _, fi := range fromList {
		fromMap[fi.Name()] = fi
	}

	for _, fi := range toList {
		toMap[fi.Name()] = fi
	}

	for name, fi := range fromMap {
		if toFi, ok := toMap[name]; ok {
			if fi.IsDir() && toFi.IsDir() {
				syncDir(filepath.Join(dirFrom, name), filepath.Join(dirTo, name))
				continue
			}

			if statsEqual(fi, toFi) {
				continue
			}

			if err := os.RemoveAll(filepath.Join(dirTo, name)); err != nil {
				log.Printf("Could not remove target: %s", err.Error())
				continue
			}
		}

		sync(fi, dirFrom, dirTo)
	}

	for name := range toMap {
		if _, ok := fromMap[name]; ok {
			continue
		}

		if err := os.RemoveAll(filepath.Join(dirTo, name)); err != nil {
			log.Printf("Could not remove target: %s", err.Error())
			continue
		}
	}
}
