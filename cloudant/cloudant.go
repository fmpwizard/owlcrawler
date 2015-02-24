package cloudant

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os/user"
	"path/filepath"
	"time"
)

var cloudantCredentials cloudantCred

type cloudantCred struct {
	User     string
	Password string
	URL      string
}

type couchDBFound struct {
	Rows []struct {
		Key   string
		Value int
	}
}

//CouchDoc represents a response fron CouchDB
type CouchDoc struct {
	ID   string    `json:"_id"`
	Rev  string    `json:"_rev"`
	URL  string    `json:"url"`
	HTML string    `json:"html"`
	Date time.Time `json:"date"`
}

func init() {
	if u, err := user.Current(); err == nil {
		path := filepath.Join(u.HomeDir, ".cloudant.json")
		content, err := ioutil.ReadFile(path)
		if err != nil {
			log.Fatalf("Error reading Cloudant credentials, got: %v\n", err)
		}

		err = json.Unmarshal(content, &cloudantCredentials)
		if err != nil {
			log.Fatalf("Invalid Cloudant credentials file, got: %v\n", err)
		}
	}
}

//AddURLData adds the url and data to the database. data is json encoded.
func AddURLData(url string, data []byte) {
	client := &http.Client{}
	document := bytes.NewReader(data)
	req, err := http.NewRequest("POST", cloudantCredentials.URL, document)
	if err != nil {
		log.Printf("Error parsing url, got: %v\n", err)
	}
	req.SetBasicAuth(cloudantCredentials.User, cloudantCredentials.Password)
	req.Header.Set("User-Agent", "OwlCrawler - https://github.com/fmpwizard/owlcrawler")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error sending request to Cloudant, got: %v\n", err)
	}
	defer resp.Body.Close()
	log.Printf("AddURLData respose was %+v\n", resp)
	return
}

//IsURLThere checks if the given url is already stored in the database
func IsURLThere(storedURL string) bool {
	client := &http.Client{}
	url := cloudantCredentials.URL + "/_design/by-url/_view/by-url?key=\"" + url.QueryEscape(storedURL) + "\""
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Printf("Error parsing url, got: %v\n", err)
	}
	req.SetBasicAuth(cloudantCredentials.User, cloudantCredentials.Password)
	req.Header.Set("User-Agent", "OwlCrawler - https://github.com/fmpwizard/owlcrawler")
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error sending request to Cloudant, got: %v\n", err)
	}
	defer resp.Body.Close()

	var result couchDBFound
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error reading body response from IsURLThere %v\n", err)
	}
	err = json.Unmarshal(body, &result)
	if err != nil {
		log.Printf("Error unmarshalling IsURLThere body into a struct, got: %v\n", err)
	}
	return len(result.Rows) > 0
}
