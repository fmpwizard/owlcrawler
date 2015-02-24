package parse

import (
	"encoding/json"
	"github.com/fmpwizard/owlcrawler/cloudant"
	"golang.org/x/net/html"
	"log"
	"strings"
)

type PageStructure struct {
	Title string
	H1    []string //I know there should be just one H1 per page, but not eveyone does that
	H2    []string
	H3    []string
	H4    []string
	Text  []string
}

func ExtractText(payload []byte) PageStructure {
	var doc cloudant.CouchDoc
	var page PageStructure
	err := json.Unmarshal(payload, &doc)
	if err != nil {
		log.Printf("Error reading couch doc while trying to extract data, got: %v\n", err)
	}
	nodes, err := html.Parse(strings.NewReader(doc.HTML))
	if err != nil {
		log.Printf("Error parsing html, got: %+v\n", err)
	}
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode {
			if n.Data == "title" {
				page.Title = n.FirstChild.Data
			} else if n.Data == "h1" {
				page.H1 = append(page.H1, n.FirstChild.Data)
			} else if n.Data == "h2" {
				page.H2 = append(page.H2, n.FirstChild.Data)
			} else if n.Data == "h3" {
				page.H3 = append(page.H3, n.FirstChild.Data)
			} else if n.Data == "h4" {
				page.H4 = append(page.H4, n.FirstChild.Data)
			} else if n.FirstChild != nil && strings.TrimSpace(n.FirstChild.Data) != "" {
				page.Text = append(page.Text, n.FirstChild.Data)
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(nodes)
	return page
}
