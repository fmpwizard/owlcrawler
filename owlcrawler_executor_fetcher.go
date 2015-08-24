// +build fetcherExec

package main

import (
	"encoding/base64"
	"encoding/json"
	"github.com/fmpwizard/owlcrawler/couchdb"
	log "github.com/golang/glog"
	"github.com/iron-io/iron_go/mq"
	exec "github.com/mesos/mesos-go/executor"
	mesos "github.com/mesos/mesos-go/mesosproto"

	"bytes"
	"encoding/gob"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

type exampleExecutor struct {
	tasksLaunched int
}

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

func newExampleExecutor() *exampleExecutor {
	return &exampleExecutor{tasksLaunched: 0}
}

func (exec *exampleExecutor) Registered(driver exec.ExecutorDriver, execInfo *mesos.ExecutorInfo, fwinfo *mesos.FrameworkInfo, slaveInfo *mesos.SlaveInfo) {
	log.V(3).Infof("Registered Executor on slave ", slaveInfo.GetHostname())
}

func (exec *exampleExecutor) Reregistered(driver exec.ExecutorDriver, slaveInfo *mesos.SlaveInfo) {
	log.V(3).Infof("Re-registered Executor on slave ", slaveInfo.GetHostname())
}

func (exec *exampleExecutor) Disconnected(exec.ExecutorDriver) {
	log.V(3).Infof("Executor disconnected.")
}

func (exec *exampleExecutor) LaunchTask(driver exec.ExecutorDriver, taskInfo *mesos.TaskInfo) {
	log.V(2).Infof("Launching task", taskInfo.GetName())
	runStatus := &mesos.TaskStatus{
		TaskId: taskInfo.GetTaskId(),
		State:  mesos.TaskState_TASK_RUNNING.Enum(),
	}
	_, err := driver.SendStatusUpdate(runStatus)
	if err != nil {
		log.Errorln("Got error", err)
	}

	exec.tasksLaunched++
	go exec.fetchHTML(driver, taskInfo)
}

func (exec *exampleExecutor) fetchHTML(driver exec.ExecutorDriver, taskInfo *mesos.TaskInfo) {
	log.V(2).Infof("Total tasks launched ", exec.tasksLaunched)

	//Read information about this URL we are about to process
	payload := bytes.NewReader(taskInfo.GetData())
	var queueMessage OwlCrawlMsg
	dec := gob.NewDecoder(payload)
	err := dec.Decode(&queueMessage)
	if err != nil {
		log.Errorln("decode error:", err)
	}
	queue := mq.New(queueMessage.QueueName)

	//Fetch url
	client := &http.Client{}
	req, err := http.NewRequest("GET", queueMessage.URL, nil)
	if err != nil {
		log.Errorf("Error parsing url: %s, got: %v\n", queueMessage.URL, err)
	}
	req.Header.Set("User-Agent", "OwlCrawler - https://github.com/fmpwizard/owlcrawler")
	resp, err := client.Do(req)
	if err != nil {
		log.Errorf("Error while fetching url: %s, got error: %v\n", queueMessage.URL, err)
		err = queue.ReleaseMessage(queueMessage.ID, 0)
		if err != nil {
			log.Errorf("Error releasing message id: %s from queue, got: %v\n", queueMessage.ID, err)
		}
		updateStatusDied(driver, taskInfo)
		return
	}

	defer resp.Body.Close()
	htmlData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Errorf("Error while reading html for url: %s, got error: %v\n", queueMessage.URL, err)
		err = queue.ReleaseMessage(queueMessage.ID, 0)
		if err != nil {
			log.Errorf("Error releasing message id: %s from queue, got: %v\n", queueMessage.ID, err)
		}
		updateStatusDied(driver, taskInfo)
		return
	}

	err = queue.DeleteMessage(queueMessage.ID)
	if err != nil {
		log.Errorf("Error deleting message id: %s from queue, got: %v\n", queueMessage.ID, err)
	}
	data := &dataStore{
		ID:        base64.URLEncoding.EncodeToString([]byte(queueMessage.URL)),
		URL:       queueMessage.URL,
		HTML:      string(htmlData[:]),
		FetchedOn: time.Now().UTC(),
	}

	pageData, err := json.Marshal(data)
	if err != nil {
		log.Errorf("Error generating json to save in database, got: %v\n", err)
	}

	ret, err := couchdb.AddURLData(queueMessage.URL, pageData)
	if err == nil {
		//Send fethed url to parse queue
		parseHTMLQueue := mq.New("html_to_parse")
		parseHTMLQueue.PushString(ret.ID)
	}

	// finish task
	finStatus := &mesos.TaskStatus{
		TaskId: taskInfo.GetTaskId(),
		State:  mesos.TaskState_TASK_FINISHED.Enum(),
	}
	_, err = driver.SendStatusUpdate(finStatus)
	if err != nil {
		log.Errorln("Got error", err)
	}
	log.V(2).Infof("Task finished", taskInfo.GetName())
}

func (exec *exampleExecutor) KillTask(exec.ExecutorDriver, *mesos.TaskID) {
	log.V(3).Infof("Kill task")
}

func (exec *exampleExecutor) FrameworkMessage(driver exec.ExecutorDriver, msg string) {
	log.V(3).Infof("Got framework message: ", msg)
}

func (exec *exampleExecutor) Shutdown(exec.ExecutorDriver) {
	log.V(3).Infof("Shutting down the executor")
}

func (exec *exampleExecutor) Error(driver exec.ExecutorDriver, err string) {
	log.Errorln("Got error message:", err)
}

// -------------------------- func inits () ----------------- //
func init() {
	flag.Parse()
}

func main() {
	log.V(2).Infof("Starting Fetcher Executor")

	dconfig := exec.DriverConfig{
		Executor: newExampleExecutor(),
	}
	driver, err := exec.NewMesosExecutorDriver(dconfig)

	if err != nil {
		fmt.Println("Unable to create a ExecutorDriver ", err.Error())
	}

	_, err = driver.Start()
	if err != nil {
		fmt.Println("Got error:", err)
		return
	}
	driver.Join()
}

func updateStatusDied(driver exec.ExecutorDriver, taskInfo *mesos.TaskInfo) {
	runStatus := &mesos.TaskStatus{
		TaskId: taskInfo.GetTaskId(),
		State:  mesos.TaskState_TASK_FAILED.Enum(),
	}
	_, err := driver.SendStatusUpdate(runStatus)
	if err != nil {
		log.Errorf("Failed to tell mesos that we died, sorry, got: %v", err)
	}

}
