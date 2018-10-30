package main

import (
	"bytes"
	"flag"
	"github.com/alecthomas/chroma"
	"github.com/alecthomas/chroma/formatters/html"
	"github.com/alecthomas/chroma/lexers"
	"github.com/alecthomas/chroma/styles"
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

type SearchResult struct {
	Pattern  string
	RootDir  string
	Snippets []Snippet
}

type SourceResult struct {
	Pattern string
	RelPath string
	Source  template.HTML
}

type DirectoryResult struct {
	Pattern string
	RelPath string
	Files   []string
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

func highlight(source string) template.HTML {
	l := chroma.Coalesce(lexers.Get("c"))
	f := html.New(html.WithLineNumbers())
	s := styles.Get("monokai")
	it, err := l.Tokenise(nil, source)
	if err != nil {
		log.Fatalln(err)
	}
	var buf bytes.Buffer
	err = f.Format(&buf, s, it)
	if err != nil {
		log.Fatalln(err)
	}
	return template.HTML(buf.String())
}

func handleSearchResult(w http.ResponseWriter, pattern string) {
	t := template.Must(template.ParseFiles("layout.html", "result.html"))

	snippets, err := grepAllFiles(*rootDir, pattern)
	if err != nil {
		log.Println(err)
		http.Error(w, err.Error(), 500)
		return
	}

	result := &SearchResult{
		Pattern:  pattern,
		RootDir:  *rootDir,
		Snippets: snippets}

	if err := t.ExecuteTemplate(w, "layout", result); err != nil {
		log.Println(err)
		http.Error(w, err.Error(), 500)
	}
}

func handleDirectoryListing(w http.ResponseWriter, relPath, pattern string) {
	t := template.Must(template.ParseFiles("layout.html", "directory.html"))

	files, err := ioutil.ReadDir(filepath.Join(*rootDir, relPath))
	if err != nil {
		log.Println(err)
		http.Error(w, err.Error(), 500)
		return
	}

	fileNames := []string{}
	for _, file := range files {
		fileNames = append(fileNames, file.Name())
	}

	result := &DirectoryResult{
		Pattern: pattern,
		RelPath: relPath,
		Files:   fileNames}

	if err := t.ExecuteTemplate(w, "layout", result); err != nil {
		log.Println(err)
		http.Error(w, err.Error(), 500)
	}
}

func handleSourceListing(w http.ResponseWriter, relPath, pattern string) {
	t := template.Must(template.ParseFiles("layout.html", "source.html"))

	content, err := ioutil.ReadFile(filepath.Join(*rootDir, relPath))
	if err != nil {
		log.Println(err)
		http.Error(w, err.Error(), 500)
		return
	}

	result := &SourceResult{
		Pattern: pattern,
		RelPath: relPath,
		Source:  highlight(string(content))}

	if err := t.ExecuteTemplate(w, "layout", result); err != nil {
		log.Println(err)
		http.Error(w, err.Error(), 500)
	}
}

func httpHandler(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	pattern := r.FormValue("q")

	if path == "/" && pattern != "" {
		handleSearchResult(w, pattern)
		return
	}

	fullPath := filepath.Join(*rootDir, path)
	info, err := os.Stat(fullPath)
	if os.IsNotExist(err) {
		log.Println(err)
		http.NotFound(w, r)
		return
	} else if err != nil {
		log.Println(err)
		http.Error(w, err.Error(), 500)
		return
	}

	relPath, _ := filepath.Rel(*rootDir, fullPath)
	if info.IsDir() {
		handleDirectoryListing(w, relPath, pattern)
	} else {
		handleSourceListing(w, relPath, pattern)
	}
}

func main() {
	flag.Parse()

	if *rootDir == "" {
		log.Fatalln("root dir cannot be empty")
	}

	log.Println("Listening localhost:3000...")

	http.HandleFunc("/", httpHandler)
	if err := http.ListenAndServe(":3000", nil); err != nil {
		log.Fatalln(err)
	}
}
