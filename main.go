package main

import (
	"flag"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

var rootDir = flag.String("root_dir", "", "root directory of the source tree")

func grepInPath(w io.Writer, path string, pattern string) error {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}

	for i, line := range strings.Split(string(b), "\n") {
		if !strings.Contains(line, pattern) {
			continue
		}
		fmt.Fprintf(w, "%s:%d: %s\n", path, i, line)
	}
	return nil
}

func httpHandler(w http.ResponseWriter, r *http.Request) {
	pattern := r.FormValue("q")
	if pattern == "" {
		t := template.Must(template.ParseFiles("index.html"))
		if err := t.ExecuteTemplate(w, "index.html", *rootDir); err != nil {
			log.Fatalln(err)
		}
		return
	}

	err := filepath.Walk(*rootDir, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		if strings.Contains(path, ".git") {
			return nil
		}
		return grepInPath(w, path, pattern)
	})

	if err != nil {
		fmt.Fprintln(w, err)
	}
}

func main() {
	flag.Parse()

	if *rootDir == "" {
		log.Fatalln("root dir cannot be empty")
	}

	log.Println("Listening localhost:3000...")

	http.HandleFunc("/", httpHandler)
	http.ListenAndServe(":3000", nil)
}
