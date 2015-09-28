// +build extractorExec

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/fmpwizard/owlcrawler/couchdb"
	"github.com/fmpwizard/owlcrawler/parse"
	log "github.com/golang/glog"
	"github.com/nats-io/nats"
	"io/ioutil"
	"os/user"
	"path/filepath"
	"time"
)

const extractQueue = "extract_url"
const fetchQueue = "fetch_url"

var fn = func(url string) bool {
	return couchdb.ShouldURLBeFetched(url)
}
var gnatsdCredentials gnatsdCred

type gnatsdCred struct {
	URL string
}

func extractText(id string) {
	doc, err := getStoredHTMLForDocID(id)
	if err == nil {
		err = saveExtractedData(extractData(doc))
		if err == couchdb.ErrorNoLatestVersion {
			doc, err = getStoredHTMLForDocID(id)
			if err != nil {
				log.Errorf("Failed to get latest version of %s\n", id)
				return
			}
			saveExtractedData(extractData(doc))
		}
	}
	log.V(2).Infof("Finished extracting text for %s\n", id)
}

func extractData(doc couchdb.CouchDoc) couchdb.CouchDoc {
	doc.Text = parse.ExtractText(doc.HTML)
	fetch, storing := parse.ExtractLinks(doc.HTML, doc.URL, fn)
	doc.LinksToQueue = fetch.URL
	doc.Links = storing.URL
	doc.ParsedOn = time.Now().UTC()
	nc, err := nats.Connect(gnatsdCredentials.URL)
	if err != nil {
		log.Fatalf("Could not connect to gnatsd, got: %s\n", err)
	}
	for _, u := range fetch.URL {
		nc.Publish(fetchQueue, []byte(u))
	}
	return doc
}

func saveExtractedData(doc couchdb.CouchDoc) error {
	jsonDocWithExtractedData, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("Error generating json to save docWithText in database, got: %v\n", err)
	}
	ret, err := couchdb.SaveExtractedTextAndLinks(doc.ID, jsonDocWithExtractedData)
	if err == couchdb.ErrorNoLatestVersion {
		return couchdb.ErrorNoLatestVersion
	}
	if err != nil {
		log.Errorf("Error was: %+v\n", err)
		log.Errorf("Doc was: %+v\n", doc)
		return err
	}
	log.V(3).Infof("saveExtractedData gave: %+v\n", ret)
	return nil
}

func getStoredHTMLForDocID(id string) (couchdb.CouchDoc, error) {
	doc, err := couchdb.GetURLData(id)
	if err == couchdb.Error404 {
		return doc, couchdb.Error404
	}
	if err != nil {
		log.Errorf("Error was: %+v\n", err)
		log.Errorf("Doc was: %+v\n", doc)
		return doc, err
	}
	return doc, nil
}

func main() {
	flag.Parse()
	log.V(2).Infoln("Starting Extractor")
	nc, _ := nats.Connect(gnatsdCredentials.URL)
	sub, err := nc.SubscribeSync(extractQueue)
	if err != nil {
		log.Fatalf("Error while subscribing to extract_url, got %s\n", err)
	}
	for {
		if payload, err := sub.NextMsg(30 * time.Second); err == nil {
			if !couchdb.IsItParsed(string(payload.Data[:])) {
				extractText(string(payload.Data[:]))
			}
		}
	}
}

func init() {
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
