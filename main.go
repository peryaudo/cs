package main

import (
	"flag"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

var rootDir = flag.String("root_dir", "", "root directory of the source tree")

type Snippet struct {
	RelPath string
	LineNum int
	Line    string
}

func grepFile(fileName string, pattern string) ([]Snippet, error) {
	b, err := ioutil.ReadFile(fileName)
	if err != nil {
		return nil, err
	}

	result := []Snippet{}
	for i, line := range strings.Split(string(b), "\n") {
		if !strings.Contains(line, pattern) {
			continue
		}
		snippet := Snippet{
			RelPath: "",
			LineNum: i + 1,
			Line:    line}
		result = append(result, snippet)
	}
	return result, nil
}

func grepAllFiles(rootDir string, pattern string) ([]Snippet, error) {
	results := []Snippet{}
	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		if strings.Contains(path, ".git") {
			return nil
		}

		relPath, err := filepath.Rel(rootDir, path)
		if err != nil {
			return err
		}

		result, err := grepFile(path, pattern)
		if err != nil {
			return err
		}

		for _, snippet := range result {
			snippet.RelPath = relPath
			results = append(results, snippet)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return results, nil
}

func httpHandler(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	if path != "/" {
		fullPath := filepath.Join(*rootDir, path)
		http.ServeFile(w, r, fullPath)
		return
	}

	pattern := r.FormValue("q")
	if pattern == "" {
		t := template.Must(template.ParseFiles("index.html"))
		if err := t.ExecuteTemplate(w, "index.html", *rootDir); err != nil {
			log.Fatalln(err)
		}
		return
	}

	result, err := grepAllFiles(*rootDir, pattern)
	if err != nil {
		fmt.Fprintln(w, err)
		return
	}

	for _, snippet := range result {
		fmt.Fprintf(w, "%s:%d: %s\n", snippet.RelPath, snippet.LineNum, snippet.Line)
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
