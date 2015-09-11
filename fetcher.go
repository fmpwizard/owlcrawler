// +build fetcherExec

package main

import (
	"encoding/base64"
	"encoding/json"
	"github.com/fmpwizard/owlcrawler/couchdb"
	log "github.com/golang/glog"
	"github.com/iron-io/iron_go/mq"

	"flag"
	"io/ioutil"
	"net/http"
	"time"
)

//OwlCrawlMsg is used to decode the Data payload from the framework
type OwlCrawlMsg struct {
	URL       string
	ID        string
	QueueName string
}

type dataStore struct {
	ID        string    `json:"_id"`
	URL       string    `json:"url"`
	HTML      string    `json:"html"`
	FetchedOn time.Time `json:"fetched_on"`
}

func fetchHTML(taskInfo *OwlCrawlMsg) {
	log.V(2).Infoln("Total tasks launched")

	queue := mq.New(taskInfo.QueueName)

	//Fetch url
	client := &http.Client{}
	req, err := http.NewRequest("GET", taskInfo.URL, nil)
	if err != nil {
		log.Errorf("Error parsing url: %s, got: %v\n", taskInfo.URL, err)
	}
	req.Header.Set("User-Agent", "OwlCrawler - https://github.com/fmpwizard/owlcrawler")
	resp, err := client.Do(req)
	if err != nil {
		log.Errorf("Error while fetching url: %s, got error: %v\n", taskInfo.URL, err)
		err = queue.ReleaseMessage(taskInfo.ID, 0)
		if err != nil {
			log.Errorf("Error releasing message id: %s from queue, got: %v\n", taskInfo.ID, err)
		}
		return
	}

	defer resp.Body.Close()
	htmlData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Errorf("Error while reading html for url: %s, got error: %v\n", taskInfo.URL, err)
		err = queue.ReleaseMessage(taskInfo.ID, 0)
		if err != nil {
			log.Errorf("Error releasing message id: %s from queue, got: %v\n", taskInfo.ID, err)
		}
		return
	}

	err = queue.DeleteMessage(taskInfo.ID)
	if err != nil {
		log.Errorf("Error deleting message id: %s from queue, got: %v\n", taskInfo.ID, err)
	}
	data := &dataStore{
		ID:        base64.URLEncoding.EncodeToString([]byte(taskInfo.URL)),
		URL:       taskInfo.URL,
		HTML:      string(htmlData[:]),
		FetchedOn: time.Now().UTC(),
	}

	pageData, err := json.Marshal(data)
	if err != nil {
		log.Errorf("Error generating json to save in database, got: %v\n", err)
	}

	ret, err := couchdb.AddURLData(taskInfo.URL, pageData)
	if err == nil {
		//Send fethed url to parse queue
		parseHTMLQueue := mq.New("html_to_parse")
		parseHTMLQueue.PushString(ret.ID)
	}

	log.V(2).Infof("Task finished")
}

func main() {
	flag.Parse()
	log.V(2).Infof("Starting Fetcher.")
	urlToFetchQueueName := "urls_to_fetch"
	urlToFetchQueue := mq.New(urlToFetchQueueName)
	for {
		if ok, payload := fetchTask(urlToFetchQueue); ok == true {
			fetchHTML(payload)
		}
	}
}

func fetchTask(queue *mq.Queue) (bool, *OwlCrawlMsg) {
	msgs, err := queue.GetNWithTimeoutAndWait(1, 120, 30)
	if err != nil {
		return false, &OwlCrawlMsg{}
	}

	for _, msg := range msgs {
		if couchdb.ShouldURLBeFetched(msg.Body) { //found an entry, no need to fetch it again
			msg.Delete()
			return false, &OwlCrawlMsg{}
		}

		var msgAndID = &OwlCrawlMsg{
			URL:       msg.Body,
			ID:        msg.Id,
			QueueName: queue.Name,
		}

		return true, msgAndID
	}
	return false, &OwlCrawlMsg{}
}
