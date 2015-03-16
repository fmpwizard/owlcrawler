package elasticsearch

import (
	"errors"
	"fmt"
	"github.com/fmpwizard/owlcrawler/parse"
	log "github.com/golang/glog"
	"io/ioutil"
	"strings"

	"encoding/json"
	"net/http"
)

//ElasticSearchDoc represents a document in Elastic Search
type ElasticSearchDoc struct {
	ID           string              `json:"_id"`
	URL          string              `json:"url"`
	HTML         string              `json:"html"`
	Text         parse.PageStructure `json:"text"`
	Links        []string            `json:"links"`
	LinksToQueue []string            `json:"-"`
}

//DocCreated represents a response to a document just created
type DocCreated struct {
	Index   string `json:"_index"`
	Type    string `json:"_type"`
	ID      string `json:"_id"`
	Version int    `json:"_version"`
	Created bool   `json:"created"`
}

//Result represents a search result
type Result struct {
	Took int64
	Hits struct {
		Total    int64   `json:"total"`
		MaxScore float64 `json:"max_score"`
		Hits     []hits  `json:"hits"`
	}
}

type hits struct {
	Index     string    `json:"_index"`
	Type      string    `json:"_type"`
	ID        string    `json:"_id"`
	Score     float64   `json:"_score"`
	Source    source    `json:"_source"`
	Highlight highlight `json:"highlight"`
}

type source struct {
	Rev   string              `json:"_rev"`
	ID    string              `json:"_id"`
	URL   string              `json:"url"`
	HTML  string              `json:"html"`
	Text  parse.PageStructure `json:"text"`
	Links []string            `json:"links"`
}

type highlight struct {
	Text []string `json:"text"`
}

var ERROR_404 = errors.New("Doc not found.")

const (
	elasticHost = "http://127.0.0.1:9200"
	elasticPath = "/owl-crawler/pages/"
)

//GetURLData gets the data stored in Couch, does a lookup by doc id
func Search(term string, result *Result) error {
	client := &http.Client{}
	//searchURL := elasticHost + elasticPath + "_search?q=" + url.QueryEscape(term)
	searchURL := elasticHost + elasticPath + "_search"
	req, err := http.NewRequest("POST", searchURL, strings.NewReader(highlightJson(term)))
	if err != nil {
		log.Errorf("Error parsing url, got: %v\n", err)
		return err
	}

	req.Header.Set("User-Agent", "OwlCrawler - https://github.com/fmpwizard/owlcrawler")
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		log.Errorf("Error sending request to Cloudant, got: %v\n", err)
		return err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Errorf("Error reading body response from GetURLData %v\n", err)
		return err
	}
	log.V(3).Infof("*********** %+v\n", string(body))
	log.V(3).Infof("*********** %+v\n", highlightJson(term))
	err = json.Unmarshal(body, &result)
	if err != nil {
		log.Errorf("Error unmarshalling Search body:\n%s\n into a struct, got: %v\n", string(body), err)
	}
	if resp.StatusCode == 404 {
		return ERROR_404
	}

	return nil
}

func highlightJson(term string) string {
	return fmt.Sprintf(`
{
    
    "query" : {
        "match" : {
	        "_all" : {
	            "query" : "%s",
	            "type" : "phrase"
	        }
	    }
    },
    "highlight" : {
    	"pre_tags" : ["_-_strong_-_"],
        "post_tags" : ["_!-_strong_-_"],
        "order" : "score",
        "fields" : {
            "text" : {
                "fragment_size" : 150,
                "number_of_fragments" : 3,
                "highlight_query": {
                    "bool": {
                        "must": {
                            "match": {
                                "text": {
                                    "query": "%s"
                                }
                            }
                        },
                        "should": {
                            "match_phrase": {
                                "text": {
                                    "query": "%s",
                                    "phrase_slop": 1,
                                    "boost": 10.0
                                }
                            }
                        },
                        "minimum_should_match": 0
                    }
                }
            }
        }
    }
}`, term, term, term)

}

/*

Remember to restart elastic after installing the river
and check the logs in /var/logs/elasticsearch
to see if indexing is going well

Simple url to test:
http://192.168.1.73:9200/owl-crawler/pages/_search?q=diego


curl -XPUT 'localhost:9200/_river/owl-crawler/_meta' -d '{
    "type" : "couchdb",
    "couchdb" : {
        "host" : "localhost",
        "port" : 5984,
        "db" : "owl-crawler",
        "filter" : null,
        "user" : "couch-user",
        "password" : "couch-password"
    },
    "index" : {
        "index" : "owl-crawler",
        "type" : "pages",
        "bulk_size" : "100",
        "bulk_timeout" : "10ms"
    }
}'

*/
