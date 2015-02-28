package parse

import (
	"fmt"
	"golang.org/x/net/html"
	"log"
	"net/url"
	"strings"
)

type PageStructure struct {
	Title string   `json:"title,omitempty"`
	H1    []string `json:"h1,omitempty"` //I know there should be just one H1 per page, but not eveyone does that
	H2    []string `json:"h2,omitempty"`
	H3    []string `json:"h3,omitempty"`
	H4    []string `json:"h4,omitempty"`
	Text  []string `json:"text,omitempty"`
}

type ExtractedLinks struct {
	OriginalURL string
	URL         []string
}

type URLFetchChecker func(url string) bool

func ExtractText(payload string) PageStructure {
	var page PageStructure

	nodes, err := html.Parse(strings.NewReader(payload))
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

func ExtractLinks(payload string, originalURL string, shouldFetch URLFetchChecker) ExtractedLinks {
	link, err := url.Parse(originalURL)
	if err != nil {
		log.Fatalf("Error parsing url %s, got: %v\n", originalURL, err)
	}

	d := html.NewTokenizer(strings.NewReader(payload))
	var extractedLinks ExtractedLinks
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
							log.Printf("Ignoring url: %s\n", attribute.Val)
						}
					}
				}
			}
		}
	}
	return extractedLinks
}
