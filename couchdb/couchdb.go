package couchdb

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"github.com/fmpwizard/owlcrawler/parse"
	log "github.com/golang/glog"
	"io/ioutil"
	"net/http"
	"os/user"
	"path/filepath"
	"time"
)

var couchdbCredentials couchdbCred

type couchdbCred struct {
	User     string
	Password string
	URL      string
}

//CouchDoc represents a response fron CouchDB
type CouchDoc struct {
	ID           string              `json:"_id"`
	Rev          string              `json:"_rev"`
	URL          string              `json:"url"`
	HTML         string              `json:"html"`
	Text         parse.PageStructure `json:"text"`
	Links        []string            `json:"links,omitempty"`
	LinksToQueue []string            `json:"-"`
	ParsedOn     time.Time           `json:"parsed_on,omitempty"`
	FetchedOn    time.Time           `json:"fetched_on,omitempty"`
}

//CouchDocCreated represents a full document
type CouchDocCreated struct {
	OK  bool   `json:"ok"`
	ID  string `json:"id"`
	Rev string `json:"rev"`
}

//NewSite is used to add a new url submitted
type NewSite struct {
	URL string `json:"url"`
}

type couchStatsRet struct {
	Rows []struct {
		Key   string `json:"key"`
		Value int    `json:"value"`
	}
}

//StatsIndex Stats related to the search engine index
type StatsIndex struct {
	Parsed, Fetched int
}

//ERROR_NO_LATEST_VERSION error you get when trying to save an old version of a CouchDB document
var ErrorNoLatestVersion = errors.New("Not latest revision.")

//Error404 the error you get when no document was found
var Error404 = errors.New("Doc not found. ")

func init() {
	flag.Parse()
	if u, err := user.Current(); err == nil {
		path := filepath.Join(u.HomeDir, ".couchdb.json")
		content, err := ioutil.ReadFile(path)
		if err != nil {
			log.Fatalf("Error reading Couchdb credentials, got: %v\n", err)
		}

		err = json.Unmarshal(content, &couchdbCredentials)
		if err != nil {
			log.Fatalf("Invalid Couchdb credentials file, got: %v\n", err)
		}
	}
	initDesignDocs()
}

var designSearch = []byte(`
{
   "views": {
       "text": {
           "map": "function(doc) { \nif ( doc.text.text && doc.text.text.length > 0) {  \n   for(var idx in doc.text.text) {\n\tvar words = doc.text.text[idx].split(\" \")\n\tfor(var idx2 in words) {\n\t\tif(\nwords[idx2].length >2\n\n){\n\t\t\temit(words[idx2], null);\n\t\t}\n\n            \n        }\n}\n\n  \n}\n}"
       }
   },
   "language": "javascript"
}
`)
var designReports = []byte(`
{
   "views": {
       "stats": {
           "map": "function(doc) { \nif ( doc.parsed_on) {  \n  emit(\"parsed_on\", doc.parsed_on)\n}\nif (doc.fetched_on) {  \n  emit(\"fetched_on\", doc.parsed_on)\n  }}\n",
           "reduce": "_count"
       }
   },
   "language": "javascript"
}`)

func initDesignDocs() {
	if !isDocPresent("_design/reports", false) {
		saveDesignDoc(designReports, "_design/reports")
	}
	if !isDocPresent("_design/search", false) {
		saveDesignDoc(designSearch, "_design/search")
	}
}

func saveDesignDoc(doc []byte, id string) {
	client := &http.Client{}
	document := bytes.NewReader(doc)
	req, err := http.NewRequest("PUT", couchdbCredentials.URL+"/"+id, document)
	if err != nil {
		log.Errorf("Error parsing url, got: %v\n", err)
	}
	req.SetBasicAuth(couchdbCredentials.User, couchdbCredentials.Password)
	req.Header.Set("User-Agent", "OwlCrawler - https://github.com/fmpwizard/owlcrawler")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		log.Errorf("Error sending request to Couchdb, got: %v\n", err)
	}
	resp.Body.Close()
	if resp.StatusCode == 409 {
		log.Errorf("%s was created by another request.", id)
	}
	log.V(4).Infof("Creating %s view gave status: %v", id, resp.StatusCode)
}

//AddURLData adds the url and data to the database. data is json encoded.
func AddURLData(url string, data []byte) (CouchDocCreated, error) {
	client := &http.Client{}
	document := bytes.NewReader(data)
	req, err := http.NewRequest("PUT", couchdbCredentials.URL+"/"+base64.URLEncoding.EncodeToString([]byte(url)), document)
	if err != nil {
		log.Errorf("Error parsing url, got: %v\n", err)
	}
	req.SetBasicAuth(couchdbCredentials.User, couchdbCredentials.Password)
	req.Header.Set("User-Agent", "OwlCrawler - https://github.com/fmpwizard/owlcrawler")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		log.Errorf("Error sending request to Couchdb, got: %v\n", err)
	}
	if resp.StatusCode == 409 {
		return CouchDocCreated{}, errors.New("Already saved.")
	}
	var ret CouchDocCreated
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Errorf("Error parsing result of saving document, got: %v\n", err)
	}
	err = json.Unmarshal(body, &ret)
	if err != nil {
		log.Errorf("Error serializing from json to a CouchDocCreated, got: %v\n", err)
	}
	resp.Body.Close()
	log.V(3).Infof("AddURLData respose was %+v\n", resp)
	return ret, nil
}

//SaveExtractedTextAndLinks updates the document with extraced information
//like text and links
func SaveExtractedTextAndLinks(id string, data []byte) (CouchDocCreated, error) {
	client := &http.Client{}
	document := bytes.NewReader(data)
	req, err := http.NewRequest("PUT", couchdbCredentials.URL+"/"+id, document)
	if err != nil {
		log.Errorf("Error parsing url, got: %v\n", err)
	}
	req.SetBasicAuth(couchdbCredentials.User, couchdbCredentials.Password)
	req.Header.Set("User-Agent", "OwlCrawler - https://github.com/fmpwizard/owlcrawler")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		log.Errorf("Error sending request to Couchdb, got: %v\n", err)
	}
	var ret CouchDocCreated
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Errorf("Error parsing result of saving document, got: %v\n", err)
	}
	if resp.StatusCode == 409 {
		return CouchDocCreated{}, ErrorNoLatestVersion
	}
	err = json.Unmarshal(body, &ret)
	if err != nil {
		log.Errorf("Error serializing from json to a CouchDocCreated, got: %v\n", err)
	}
	defer resp.Body.Close()
	log.V(3).Infof("SaveExtractedText respose was %+v\n", resp)
	log.V(3).Infof("Body: %+v\n", string(body))
	return ret, nil
}

//GetURLData gets the data stored in Couch, does a lookup by doc id
func GetURLData(id string) (CouchDoc, error) {
	client := &http.Client{}
	docURL := couchdbCredentials.URL + "/" + id
	req, err := http.NewRequest("GET", docURL, nil)
	if err != nil {
		log.Errorf("Error parsing url, got: %v\n", err)
		return CouchDoc{}, err
	}
	req.SetBasicAuth(couchdbCredentials.User, couchdbCredentials.Password)
	req.Header.Set("User-Agent", "OwlCrawler - https://github.com/fmpwizard/owlcrawler")
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		log.Errorf("Error sending request to Couchdb, got: %v\n", err)
		return CouchDoc{}, err
	}
	defer resp.Body.Close()

	var result CouchDoc
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Errorf("Error reading body response from GetURLData %v\n", err)
		return CouchDoc{}, err
	}
	//log.V(3).Infof("*********** %+v\n", string(body))
	err = json.Unmarshal(body, &result)
	if err != nil {
		log.Errorf("Error unmarshalling GetURLData body:\n%s\n into a struct, got: %v\n", string(body), err)
	}
	if resp.StatusCode == 404 {
		return CouchDoc{ID: id}, Error404
	}

	return result, nil
}

//ShouldURLBeFetched checks if the given url is already stored in the database
func ShouldURLBeFetched(target string) bool {
	return !isDocPresent(target, true)
}

func isDocPresent(target string, encode bool) bool {
	client := &http.Client{}
	url := ""
	if encode {
		url = couchdbCredentials.URL + "/" + base64.URLEncoding.EncodeToString([]byte(target))
	} else {
		url = couchdbCredentials.URL + "/" + target
	}
	req, err := http.NewRequest("HEAD", url, nil)
	if err != nil {
		log.Errorf("Error parsing url, got: %v\n", err)
	}
	req.SetBasicAuth(couchdbCredentials.User, couchdbCredentials.Password)
	req.Header.Set("User-Agent", "OwlCrawler - https://github.com/fmpwizard/owlcrawler")
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		log.Errorf("Error sending request to Couchdb, got: %v\n", err)
	}
	resp.Body.Close()
	log.V(4).Infof("Checking %s (%s) and got %d\n", url, target, resp.StatusCode)
	return resp.StatusCode != 404
}

//IsItParsed checks if the given url is already parsed
func IsItParsed(target string) bool {
	url := couchdbCredentials.URL + "/" + target
	var doc CouchDoc
	json.Unmarshal(fetchData(url), &doc)
	log.V(4).Infof(">>>  checking \n%s and got \n%t\n\n", url, len(doc.Text.Text) > 0)
	return len(doc.Text.Text) > 0
}

//IndexStats returns stats related to the index, cnt of parsed/fetched/etc
func IndexStats() *StatsIndex {
	path := "/_design/reports/_view/stats?group=true&group_level=1"
	var stat couchStatsRet
	json.Unmarshal(fetchData(path), &stat)
	ret := &StatsIndex{}
	for _, value := range stat.Rows {
		if value.Key == "fetched_on" {
			ret.Fetched = value.Value
		}
		if value.Key == "parsed_on" {
			ret.Parsed = value.Value
		}
	}
	return ret
}

func fetchData(path string) []byte {

	client := &http.Client{}
	url := couchdbCredentials.URL + path
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Errorf("Error parsing parsedCnt design view, got: %v\n", err)
	}
	req.SetBasicAuth(couchdbCredentials.User, couchdbCredentials.Password)
	req.Header.Set("User-Agent", "OwlCrawler - https://github.com/fmpwizard/owlcrawler")
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		log.Errorf("Error sending request to Couchdb parsedCnt view, got: %v\n", err)
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	return body
}
