// +build fetcherExec

package main

import (
	"encoding/base64"
	"encoding/json"
	"flag"
	"github.com/fmpwizard/owlcrawler/couchdb"
	log "github.com/golang/glog"
	"github.com/nats-io/nats"
	"io/ioutil"
	"net/http"
	"os/user"
	"path/filepath"
	"time"
)

const fetchQueue = "fetch_url"
const extractQueue = "extract_url"

type dataStore struct {
	ID        string    `json:"_id"`
	URL       string    `json:"url"`
	HTML      string    `json:"html"`
	FetchedOn time.Time `json:"fetched_on"`
}

var gnatsdCredentials gnatsdCred

type gnatsdCred struct {
	URL string
}

func fetchHTML(url string) {
	log.V(2).Infof("Fetching %s\n", url)

	nc, err := nats.Connect(gnatsdCredentials.URL)
	if err != nil {
		log.Fatalf("Could not connect to gnatsd, got: %s\n", err)
	}

	//Fetch url
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Errorf("Error parsing url: %s, got: %v\n", url, err)
	}
	req.Header.Set("User-Agent", "OwlCrawler - https://github.com/fmpwizard/owlcrawler")
	resp, err := client.Do(req)
	if err != nil {
		log.Errorf("Error while fetching url: %s, got error: %v\n", url, err)
		return
	}

	defer resp.Body.Close()
	htmlData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Errorf("Error while reading html for url: %s, got error: %v\n", url, err)
		return
	}

	data := &dataStore{
		ID:        base64.URLEncoding.EncodeToString([]byte(url)),
		URL:       url,
		HTML:      string(htmlData[:]),
		FetchedOn: time.Now().UTC(),
	}

	pageData, err := json.Marshal(data)
	if err != nil {
		log.Errorf("Error generating json to save in database, got: %v\n", err)
	}

	ret, err := couchdb.AddURLData(url, pageData)
	if err == nil {
		//Send fethed url to parse queue
		err := nc.Publish(extractQueue, []byte(ret.ID))
		if err != nil {
			log.Errorf("Failed to push %s to extract queue\n", url)
		}
	}
	log.V(2).Infof("Finished getting %s", url)
}

func main() {
	flag.Parse()
	log.V(2).Infof("Starting Fetcher.")
	nc, _ := nats.Connect(gnatsdCredentials.URL)
	sub, err := nc.SubscribeSync(fetchQueue)
	if err != nil {
		log.Fatalf("Error while subscribing to fetch_url, got %s\n", err)
	}
	for {
		if payload, err := sub.NextMsg(30 * time.Second); err == nil {
			if couchdb.ShouldURLBeFetched(string(payload.Data[:])) {
				fetchHTML(string(payload.Data[:]))
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
