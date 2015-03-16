package main

import (
	_ "expvar"
	"flag"
	"fmt"
	"github.com/fmpwizard/owlcrawler/elasticsearch"
	"html/template"
	"log"
	"net/http"
	"os"
	"path"
	"runtime"
	"strings"
)

type message struct {
	ID    string
	Title string
	URL   string
	Text  template.HTML
}

type TemplateInfo struct {
	Results []*message
	Term    string
}

var rootDir string

func init() {
	currentDir, _ := os.Getwd()
	flag.StringVar(&rootDir, "root-dir", currentDir, "specifies the root dir where html and other files will be relative to")
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	flag.Parse()
	http.HandleFunc("/index", search)
	http.HandleFunc("/search", search)
	http.Handle("/bower_components/", http.StripPrefix("/bower_components/", http.FileServer(http.Dir("app/bower_components"))))
	http.Handle("/build/", http.StripPrefix("/build/", http.FileServer(http.Dir("build"))))
	log.Println("Listening on port 7070 ...")
	log.Fatal(http.ListenAndServe(":7070", nil))
}

func search(rw http.ResponseWriter, req *http.Request) {
	term := req.FormValue("term")
	var ret elasticsearch.Result
	err := elasticsearch.Search(term, &ret)
	if err != nil {
		fmt.Printf("Error searching, got %v", err)
	}
	t := template.New("index.html")
	t, err = t.ParseFiles(path.Join(rootDir, "app/index.html"))
	if err != nil {
		fmt.Printf("Error parsing template files: %v", err)
	}
	rw.Header().Add("Content-Type", "text/html; charset=UTF-8")
	var foundSet []*message
	for _, row := range ret.Hits.Hits {
		var txt string
		for _, highlight := range row.Highlight.Text {
			txt = txt + " ... " + highlight
		}
		foundSet = append(foundSet, &message{
			ID:    row.Source.ID,
			URL:   row.Source.URL,
			Text:  sanitizeHTML(txt),
			Title: row.Source.Text.Title,
		})
	}
	err = t.ExecuteTemplate(rw, "index.html", TemplateInfo{
		Results: foundSet,
		Term:    term,
	})
	if err != nil {
		log.Printf("Error executing template, got: %s\n", err)
	}

}

func sanitizeHTML(s string) template.HTML {
	return template.HTML(
		strings.Replace(
			strings.Replace(s, "_-_strong_-_", "<strong>", -1), "_!-_strong_-_", "</strong>", -1))
}
