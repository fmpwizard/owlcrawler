// +build extractorExec

package main

import (
	"github.com/fmpwizard/owlcrawler/couchdb"
	"github.com/fmpwizard/owlcrawler/parse"
	log "github.com/golang/glog"
	"github.com/iron-io/iron_go/mq"
	"time"

	"encoding/json"
	"errors"
	"flag"
	"fmt"
)

//OwlCrawlMsg is used to decode the Data payload from the framework
type OwlCrawlMsg struct {
	URL       string
	ID        string
	QueueName string
}

var fn = func(url string) bool {
	return !couchdb.ShouldURLBeParsed(url)
}

func extractText(queueMessage *OwlCrawlMsg) {
	//Read information about this URL we are about to process
	queue := mq.New(queueMessage.QueueName)
	if queueMessage.URL == "" {
		_ = queue.DeleteMessage(queueMessage.ID)
		return
	}

	doc, err := getStoredHTMLForURL(queueMessage.URL)
	if err != nil {
		queue.DeleteMessage(queueMessage.ID)
	} else {
		err = saveExtractedData(extractData(doc))
		if err == couchdb.ERROR_NO_LATEST_VERSION {
			doc, err = getStoredHTMLForURL(queueMessage.URL)
			if err != nil {
				log.Errorf("Failed to get latest version of %s\n", queueMessage.URL)
				queue.DeleteMessage(queueMessage.ID)
				return
			}
			saveExtractedData(extractData(doc))
		} else if err != nil {
			_ = queue.DeleteMessage(queueMessage.ID)
		}
	}
	// finish task
	log.V(2).Infoln("Task finished")
}

func extractData(doc couchdb.CouchDoc) couchdb.CouchDoc {
	doc.Text = parse.ExtractText(doc.HTML)
	fetch, storing := parse.ExtractLinks(doc.HTML, doc.URL, fn)
	doc.LinksToQueue = fetch.URL
	doc.Links = storing.URL
	doc.ParsedOn = time.Now().UTC()
	urlToFetchQueue := mq.New("urls_to_fetch")
	for _, u := range fetch.URL {
		urlToFetchQueue.PushString(u)
	}
	return doc
}

func saveExtractedData(doc couchdb.CouchDoc) error {
	jsonDocWithExtractedData, err := json.Marshal(doc)
	if err != nil {
		return errors.New(fmt.Sprintf("Error generating json to save docWithText in database, got: %v\n", err))
	}
	ret, err := couchdb.SaveExtractedTextAndLinks(doc.ID, jsonDocWithExtractedData)
	if err == couchdb.ERROR_NO_LATEST_VERSION {
		return couchdb.ERROR_NO_LATEST_VERSION
	}
	if err != nil {
		log.Errorf("Error was: %+v\n", err)
		log.Errorf("Doc was: %+v\n", doc)
		return err
	}
	log.V(3).Infof("saveExtractedData gave: %+v\n", ret)
	return nil
}

func getStoredHTMLForURL(url string) (couchdb.CouchDoc, error) {
	doc, err := couchdb.GetURLData(url)
	if err == couchdb.ERROR_404 {
		return doc, couchdb.ERROR_404
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
	htmlToParseQueueName := "html_to_parse"
	htmlToParseQueue := mq.New(htmlToParseQueueName)
	for {
		if ok, payload := extractTask(htmlToParseQueue); ok == true {
			extractText(payload)
		}
	}
}

func extractTask(queue *mq.Queue) (bool, *OwlCrawlMsg) {
	msgs, err := queue.GetNWithTimeoutAndWait(1, 120, 30)
	if err != nil {
		return false, &OwlCrawlMsg{}
	}

	for _, msg := range msgs {
		if couchdb.IsItParsed(msg.Body) {
			log.Infof("Not going to re parse %s\n", msg.Body)
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
