package main

import (
	"encoding/json"
	_ "expvar"
	"flag"
	"fmt"
	"github.com/fmpwizard/owlcrawler/couchdb"
	"github.com/fmpwizard/owlcrawler/elasticsearch"
	log "github.com/golang/glog"
	"github.com/nats-io/nats"
	"html/template"
	"io/ioutil"
	"net/http"
	"os"
	"os/user"
	"path"
	"path/filepath"
	"runtime"
	"strings"
)

type message struct {
	ID    string
	Title string
	URL   string
	Text  template.HTML
}

//TemplateInfo holds search results for the html template
type TemplateInfo struct {
	Results []*message
	Term    string
}

var gnatsdCredentials gnatsdCred

type gnatsdCred struct {
	URL string
}

//IndexStats includes information from our search engine index
type IndexStats struct {
	FetchedPages int
	ParsedPages  int
}

var rootDir string

func init() {
	currentDir, _ := os.Getwd()
	flag.StringVar(
		&rootDir,
		"root-dir",
		currentDir,
		"specifies the root dir where html and other files will be relative to",
	)
	if u, err := user.Current(); err == nil {
		path := filepath.Join(u.HomeDir, ".gnatsd.json")
		content, err := ioutil.ReadFile(path)
		if err != nil {
			log.Fatalf("Error reading gnatds user file, got: %v\n", err)
		}

		err = json.Unmarshal(content, &gnatsdCredentials)
		if err != nil {
			log.Fatalf("Invalid gnatsd credentials file, got: %v\n", err)
		}
	}
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	flag.Parse()
	http.HandleFunc("/", search)
	http.HandleFunc("/index", search)
	http.HandleFunc("/add-site", addSiteToIndex)
	http.HandleFunc("/index-status", indexStatus)
	http.Handle("/bower_components/", http.StripPrefix("/bower_components/", http.FileServer(http.Dir("bower_components"))))
	http.Handle("/styles/", http.StripPrefix("/styles/", http.FileServer(http.Dir(".tmp/styles"))))
	http.Handle("/scripts/", http.StripPrefix("/scripts/", http.FileServer(http.Dir("app/scripts"))))
	log.Infoln("Listening on port 7070 ...")
	log.Fatal(http.ListenAndServe(":7070", nil))
}

func htmlTemplate(name, target string) *template.Template {
	t := template.New(name)
	t, err := t.ParseFiles(path.Join(rootDir, target))
	if err != nil {
		fmt.Printf("Error parsing template files: %v", err)
		return nil
	}
	return t
}

func search(rw http.ResponseWriter, req *http.Request) {
	term := req.FormValue("term")
	var ret elasticsearch.Result
	err := elasticsearch.Search(term, &ret)
	if err != nil {
		fmt.Printf("Error searching, got %v", err)
	}
	t := htmlTemplate("index.html", "app/index.html")
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
		log.Infof("Error executing template, got: %s\n", err)
	}

}

func sanitizeHTML(s string) template.HTML {
	return template.HTML(
		strings.Replace(
			strings.Replace(s, "_-_strong_-_", "<strong>", -1), "_!-_strong_-_", "</strong>", -1))
}

func addSiteToIndex(rw http.ResponseWriter, req *http.Request) {
	url := req.FormValue("url")
	t := htmlTemplate("add-site.html", "app/add-site.html")
	rw.Header().Add("Content-Type", "text/html; charset=UTF-8")

	nc, err := nats.Connect(gnatsdCredentials.URL)
	if err != nil {
		log.Errorf("Could not connect to gnatsd, got: %s\n", err)
		err = t.ExecuteTemplate(rw, "add-site.html", err.Error())
		if err != nil {
			log.Errorf("Error executing template, got: %s\n", err)
		}
		return
	}

	if url != "" {
		pushError := nc.Publish("fetch_url", []byte(url))
		if pushError != nil {
			log.Errorf("Error searching, got %v", err)
			err = t.ExecuteTemplate(rw, "add-site.html", pushError.Error())
			if err != nil {
				log.Errorf("Error executing template, got: %s\n", err)
			}
			return
		}
		err = t.ExecuteTemplate(rw, "add-site.html", "Site submitted")
		if err != nil {
			log.Errorf("Error executing template, got: %s\n", err)
		}
	} else {
		err = t.ExecuteTemplate(rw, "add-site.html", "")
		if err != nil {
			log.Errorf("Error executing template, got: %s\n", err)
		}
	}
}

func indexStatus(rw http.ResponseWriter, req *http.Request) {
	t := htmlTemplate("index-status.html", "app/index-status.html")
	rw.Header().Add("Content-Type", "text/html; charset=UTF-8")
	stats := couchdb.IndexStats()
	info := &IndexStats{
		FetchedPages: stats.Fetched,
		ParsedPages:  stats.Parsed,
	}

	err := t.ExecuteTemplate(rw, "index-status.html", info)
	if err != nil {
		log.Errorf("Error executing template, got: %s\n", err)
	}
}
