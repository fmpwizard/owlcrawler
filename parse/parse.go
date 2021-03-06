package parse

import (
	log "github.com/golang/glog"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"

	"fmt"
	"net/url"
	"strings"
)

//PageStructure holds the parsed/extracted data from a page
type PageStructure struct {
	Title string   `json:"title,omitempty"`
	H1    []string `json:"h1,omitempty"`
	H2    []string `json:"h2,omitempty"`
	H3    []string `json:"h3,omitempty"`
	H4    []string `json:"h4,omitempty"`
	Text  []string `json:"text,omitempty"`
}

//ExtractedLinks holds the current url we parsed and the links extracted from it
type ExtractedLinks struct {
	OriginalURL string
	URL         []string
}

//URLFetchChecker is a function that tells us if we should fetch a link or not
type URLFetchChecker func(url string) bool

//ExtractText extracts text from a page
func ExtractText(payload string) PageStructure {
	var page PageStructure

	d := html.NewTokenizer(strings.NewReader(payload))
	var tok atom.Atom
Loop:
	for {
		tokenType := d.Next()
		if tokenType == html.ErrorToken {
			break Loop
		}
		token := d.Token()
		switch tokenType {
		case html.StartTagToken:
			if token.DataAtom == atom.Title {
				tok = atom.Title
			} else if token.DataAtom == atom.H1 {
				tok = atom.H1
			} else if token.DataAtom == atom.H2 {
				tok = atom.H2
			} else if token.DataAtom == atom.H3 {
				tok = atom.H3
			} else if token.DataAtom == atom.H4 {
				tok = atom.H4
			} else if token.DataAtom == atom.Script {
				tok = atom.Script
			} else {
				tok = 0
			}
		case html.EndTagToken:
			tok = 0
		case html.TextToken:
			if txt := strings.TrimSpace(token.Data); len(txt) > 0 && tok == atom.Title {
				page.Title = txt
			} else if txt := strings.TrimSpace(token.Data); len(txt) > 0 && tok == atom.H1 {
				page.H1 = append(page.H1, txt)
			} else if txt := strings.TrimSpace(token.Data); len(txt) > 0 && tok == atom.H2 {
				page.H2 = append(page.H2, txt)
			} else if txt := strings.TrimSpace(token.Data); len(txt) > 0 && tok == atom.H3 {
				page.H3 = append(page.H3, txt)
			} else if txt := strings.TrimSpace(token.Data); len(txt) > 0 && tok == atom.H4 {
				page.H4 = append(page.H4, txt)
			} else if txt := strings.TrimSpace(token.Data); len(txt) > 0 && tok == atom.H4 {
				page.H4 = append(page.H4, txt)
			} else if txt := strings.TrimSpace(token.Data); len(txt) > 0 && tok == atom.Script {
				continue
			} else if txt := strings.TrimSpace(token.Data); len(txt) > 0 {
				page.Text = append(page.Text, txt)
			}
		}
	}
	return page
}

//ExtractLinks gets links from a page
func ExtractLinks(payload string, originalURL string, shouldFetch URLFetchChecker) (toFetch ExtractedLinks, toStore ExtractedLinks) {
	link, err := url.Parse(originalURL)
	if err != nil {
		log.Errorf("Error parsing url %s, got: %v\n", originalURL, err)
	}

	d := html.NewTokenizer(strings.NewReader(payload))
Loop:
	for {
		tokenType := d.Next()
		if tokenType == html.ErrorToken {
			break Loop
		}
		token := d.Token()
		switch tokenType {
		case html.StartTagToken:
			if token.DataAtom.String() == "a" {
				for _, attribute := range token.Attr {
					if attribute.Key == "href" {
						if strings.HasPrefix(attribute.Val, "//") {
							url := fmt.Sprintf("%s:%s", link.Scheme, attribute.Val)
							toStore.URL = append(toStore.URL, url)
							if shouldFetch(url) {
								log.V(3).Infof("Sending url: %s\n", url)
								toFetch.URL = append(toFetch.URL, url)
							}
						} else if strings.HasPrefix(attribute.Val, "/") {
							url := fmt.Sprintf("%s://%s%s", link.Scheme, link.Host, attribute.Val)
							toStore.URL = append(toStore.URL, url)
							if shouldFetch(url) {
								log.V(3).Infof("Sending url: %s\n", url)
								toFetch.URL = append(toFetch.URL, url)
							}
						} else {
							toStore.URL = append(toStore.URL, attribute.Val)
							log.V(3).Infof("Simply storing url: %s\n", attribute.Val)
						}
					}
				}
			}
		}
	}
	return toFetch, toStore
}
