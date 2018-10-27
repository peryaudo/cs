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

	// If the path is /src, return the file content.
	if strings.HasPrefix(path, "/src") {
		fullPath := filepath.Join(*rootDir, path[4:])
		http.ServeFile(w, r, fullPath)
		return
	}

	if path != "/" {
		http.NotFound(w, r)
		return
	}

	t := template.Must(template.ParseFiles("index.html", "result.html"))

	// If the path is root and query string is empty, return the index page.
	pattern := r.FormValue("q")
	if pattern == "" {
		if err := t.ExecuteTemplate(w, "index.html", *rootDir); err != nil {
			log.Fatalln(err)
		}
		return
	}

	// Otherwise, return the search result.
	result, err := grepAllFiles(*rootDir, pattern)
	if err != nil {
		fmt.Fprintln(w, err)
		return
	}

	if err := t.ExecuteTemplate(w, "result.html", result); err != nil {
		log.Fatalln(err)
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
