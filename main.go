package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"
	"strings"
	"io/ioutil"
)

var rootDir = flag.String("root_dir", "", "root directory of the source tree")
var pattern = flag.String("pattern", "", "pattern to be found")

func grepInPath(path string) error {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}

	for i, line := range strings.Split(string(b), "\n") {
		if !strings.Contains(line, *pattern) {
			continue
		}
		log.Printf("%s:%d: %s\n", path, i, line)
	}
	return nil
}

func main() {
	flag.Parse()

	if *rootDir == ""  || *pattern == "" {
		log.Fatalln("root dir and pattern cannot be empty")
	}

	err := filepath.Walk(*rootDir, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		if strings.Contains(path, ".git") {
			return nil
		}
		return grepInPath(path)
	})

	if err != nil {
		log.Fatalln("filepath.Walk:", err)
	}
}
