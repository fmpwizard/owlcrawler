package cloudant

import (
	"github.com/fmpwizard/owlcrawler/parse"
	log "github.com/golang/glog"

	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"os/user"
	"path/filepath"
)

var cloudantCredentials cloudantCred

type cloudantCred struct {
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
	Links        []string            `json:"links"`
	LinksToQueue []string            `json:"-"`
}

//CouchDocCreated represents a full document
type CouchDocCreated struct {
	OK  bool   `json:"ok"`
	ID  string `json:"id"`
	Rev string `json:"rev"`
}

var ERROR_NO_LATEST_VERSION = errors.New("Not latest revision.")
var ERROR_404 = errors.New("Doc not found.")

type Cloudant struct {
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
func AddURLData(url string, data []byte) (CouchDocCreated, error) {
	client := &http.Client{}
	document := bytes.NewReader(data)
	req, err := http.NewRequest("PUT", cloudantCredentials.URL+"/"+base64.URLEncoding.EncodeToString([]byte(url)), document)
	if err != nil {
		log.Errorf("Error parsing url, got: %v\n", err)
	}
	req.SetBasicAuth(cloudantCredentials.User, cloudantCredentials.Password)
	req.Header.Set("User-Agent", "OwlCrawler - https://github.com/fmpwizard/owlcrawler")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		log.Errorf("Error sending request to Cloudant, got: %v\n", err)
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

func SaveExtractedTextAndLinks(id string, data []byte) (CouchDocCreated, error) {
	client := &http.Client{}
	document := bytes.NewReader(data)
	req, err := http.NewRequest("PUT", cloudantCredentials.URL+"/"+id, document)
	if err != nil {
		log.Errorf("Error parsing url, got: %v\n", err)
	}
	req.SetBasicAuth(cloudantCredentials.User, cloudantCredentials.Password)
	req.Header.Set("User-Agent", "OwlCrawler - https://github.com/fmpwizard/owlcrawler")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		log.Errorf("Error sending request to Cloudant, got: %v\n", err)
	}
	var ret CouchDocCreated
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Errorf("Error parsing result of saving document, got: %v\n", err)
	}
	if resp.StatusCode == 409 {
		return CouchDocCreated{}, ERROR_NO_LATEST_VERSION
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
	docURL := cloudantCredentials.URL + "/" + id
	req, err := http.NewRequest("GET", docURL, nil)
	if err != nil {
		log.Errorf("Error parsing url, got: %v\n", err)
		return CouchDoc{}, err
	}
	req.SetBasicAuth(cloudantCredentials.User, cloudantCredentials.Password)
	req.Header.Set("User-Agent", "OwlCrawler - https://github.com/fmpwizard/owlcrawler")
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		log.Errorf("Error sending request to Cloudant, got: %v\n", err)
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
		return CouchDoc{ID: id}, ERROR_404
	}

	return result, nil
}

//IsURLThere checks if the given url is already stored in the database
func IsURLThere(target string) bool {
	client := &http.Client{}
	url := cloudantCredentials.URL + "/" + base64.URLEncoding.EncodeToString([]byte(target))
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Errorf("Error parsing url, got: %v\n", err)
	}
	req.SetBasicAuth(cloudantCredentials.User, cloudantCredentials.Password)
	req.Header.Set("User-Agent", "OwlCrawler - https://github.com/fmpwizard/owlcrawler")
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		log.Errorf("Error sending request to Cloudant, got: %v\n", err)
	}
	resp.Body.Close()
	log.V(4).Infof(">>>  checking \n%s \n(%s) and got \n%t\n\n", url, target, resp.StatusCode != 404)
	return resp.StatusCode != 404
}

//IsItParsed checks if the given url is already parsed
func IsItParsed(target string) bool {
	client := &http.Client{}
	url := cloudantCredentials.URL + "/" + target
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Errorf("Error parsing url, got: %v\n", err)
	}
	req.SetBasicAuth(cloudantCredentials.User, cloudantCredentials.Password)
	req.Header.Set("User-Agent", "OwlCrawler - https://github.com/fmpwizard/owlcrawler")
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		log.Errorf("Error sending request to Cloudant, got: %v\n", err)
	}
	defer resp.Body.Close()
	var doc CouchDoc
	body, _ := ioutil.ReadAll(resp.Body)
	json.Unmarshal(body, &doc)
	log.V(4).Infof(">>>  checking \n%s and got \n%t\n\n", url, len(doc.Text.Text) > 0)
	return len(doc.Text.Text) > 0
}

//IsItParsed checks if the given url is already parsed
func (store *Cloudant) IsItParsed(target string) bool {
	client := &http.Client{}
	url := cloudantCredentials.URL + "/" + target
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Errorf("Error parsing url, got: %v\n", err)
	}
	req.SetBasicAuth(cloudantCredentials.User, cloudantCredentials.Password)
	req.Header.Set("User-Agent", "OwlCrawler - https://github.com/fmpwizard/owlcrawler")
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		log.Errorf("Error sending request to Cloudant, got: %v\n", err)
	}
	defer resp.Body.Close()
	var doc CouchDoc
	body, _ := ioutil.ReadAll(resp.Body)
	json.Unmarshal(body, &doc)
	log.V(4).Infof(">>>  checking \n%s and got \n%t\n\n", url, len(doc.Text.Text) > 0)
	return len(doc.Text.Text) > 0
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
