package cloudant

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/fmpwizard/owlcrawler/parse"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os/user"
	"path/filepath"
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

type couchDBByURL struct {
	TotalRows int64 `json:"total_rows"`
	Offset    int64 `json:"offset"`
	Rows      []struct {
		ID    string
		Key   string
		Value struct {
			HTML string
			Rev  string
		}
	}
}

//CouchDoc represents a response fron CouchDB
type CouchDoc struct {
	ID   string              `json:"_id"`
	Rev  string              `json:"_rev"`
	URL  string              `json:"url"`
	HTML string              `json:"html"`
	Text parse.PageStructure `json:"text"`
}

//CouchDocCreated represents a full document
type CouchDocCreated struct {
	OK  bool   `json:"ok"`
	ID  string `json:"id"`
	Rev string `json:"rev"`
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
func AddURLData(url string, data []byte) CouchDocCreated {
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
	var ret CouchDocCreated
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error parsing result of saving document, got: %v\n", err)
	}
	err = json.Unmarshal(body, &ret)
	if err != nil {
		log.Printf("Error serializing from json to a CouchDocCreated, got: %v\n", err)
	}
	defer resp.Body.Close()
	log.Printf("AddURLData respose was %+v\n", resp)
	return ret
}

func SaveExtractedText(docID string, data []byte) CouchDocCreated {
	client := &http.Client{}
	document := bytes.NewReader(data)
	req, err := http.NewRequest("PUT", cloudantCredentials.URL+"/"+docID, document)
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
	var ret CouchDocCreated
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error parsing result of saving document, got: %v\n", err)
	}
	if resp.StatusCode == 409 {
		log.Fatalln("Error updating extracted text, not latest revision")
	}
	err = json.Unmarshal(body, &ret)
	if err != nil {
		log.Printf("Error serializing from json to a CouchDocCreated, got: %v\n", err)
	}
	defer resp.Body.Close()
	log.Printf("SaveExtractedText respose was %+v\n", resp)
	log.Printf("Body: %+v\n", string(body))
	return ret
}

//GetURLData gets the data stored in Couch, does a lookup by doc id
func GetURLData(id string) (CouchDoc, error) {
	client := &http.Client{}
	url := cloudantCredentials.URL + "/" + id
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Printf("Error parsing url, got: %v\n", err)
		return CouchDoc{}, err
	}
	req.SetBasicAuth(cloudantCredentials.User, cloudantCredentials.Password)
	req.Header.Set("User-Agent", "OwlCrawler - https://github.com/fmpwizard/owlcrawler")
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("\n\nError sending request to Cloudant, got: %v\n", err)
		return CouchDoc{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == 404 {
		return CouchDoc{}, errors.New(fmt.Sprintf("Doc id: %s not found.", id))
	}

	var result CouchDoc
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("\n\nError reading body response from GetURLData %v\n", err)
		return CouchDoc{}, err
	}
	err = json.Unmarshal(body, &result)
	if err != nil {
		log.Printf("\n\nError unmarshalling GetURLData body:\n%s\n into a struct, got: %v\n", string(body), err)
	}
	return result, nil
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

/*Design views:
index name by-url
function(doc) {
    if(doc.url){
        emit(doc.url, 1);
    }
}

reduce _count

index name doc
function(doc) {
    if(doc.url && doc.html){
        var ret = {"html": doc.html, "rev": doc._rev};
        emit(doc.url,  ret);
    }
}


*/
