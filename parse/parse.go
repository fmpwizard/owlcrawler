package parse

import (
	"encoding/json"
	"fmt"
	"github.com/fmpwizard/owlcrawler/cloudant"
	"golang.org/x/net/html"
	"log"
	"net/url"
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

type ExtractedLinks struct {
	OriginalURL string
	URL         []string
}

type URLFetchChecker func(url string) bool

func ExtractText(payload []byte) PageStructure {
	var page PageStructure
	var doc cloudant.CouchDoc

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

func ExtractLinks(payload []byte, originalURL string, shouldFetch URLFetchChecker) ExtractedLinks {
	link, err := url.Parse(originalURL)
	if err != nil {
		log.Printf("Error parsing url %s, got: %v\n", originalURL, err)
	}
	var extractedLinks ExtractedLinks
	var doc cloudant.CouchDoc

	err = json.Unmarshal(payload, &doc)
	if err != nil {
		log.Printf("Error reading couch doc while trying to extract data, got: %v\n", err)
	}

	d := html.NewTokenizer(strings.NewReader(doc.HTML))

	for {
		tokenType := d.Next()
		if tokenType == html.ErrorToken {
			return extractedLinks
		}
		token := d.Token()
		switch tokenType {
		case html.StartTagToken:
			if token.DataAtom.String() == "a" {
				for _, attribute := range token.Attr {
					if attribute.Key == "href" {
						if strings.HasPrefix(attribute.Val, "//") {
							url := fmt.Sprintf("%s:%s", link.Scheme, attribute.Val)
							if shouldFetch(url) {
								log.Printf("Sending url: %s:%s\n", url)
								extractedLinks.URL = append(extractedLinks.URL, url)
							}
						} else if strings.HasPrefix(attribute.Val, "/") {
							url := fmt.Sprintf("%s://%s%s", link.Scheme, link.Host, attribute.Val)
							if shouldFetch(url) {
								log.Printf("Sending url: %s\n", url)
								extractedLinks.URL = append(extractedLinks.URL, url)
							}
						} else {
							log.Printf("Not sure what to do with this url: %s\n", attribute.Val)
						}
					}
				}
			}
		}
	}
	return extractedLinks
}

/*func ShouldFetchURL(url string) bool {
	return !cloudant.IsURLThere(url)
}*/
