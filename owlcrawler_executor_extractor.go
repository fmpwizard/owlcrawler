// +build extractorExec

package main

import (
	"encoding/json"
	"github.com/fmpwizard/owlcrawler/cloudant"
	"github.com/fmpwizard/owlcrawler/parse"

	"bytes"
	"encoding/gob"
	"flag"
	"fmt"
	"github.com/iron-io/iron_go/mq"
	exec "github.com/mesos/mesos-go/executor"
	mesos "github.com/mesos/mesos-go/mesosproto"
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
	URL  string    `json:"url"`
	HTML string    `json:"html"`
	Date time.Time `json:"date"`
}

func newExampleExecutor() *exampleExecutor {
	return &exampleExecutor{tasksLaunched: 0}
}

func (exec *exampleExecutor) Registered(driver exec.ExecutorDriver, execInfo *mesos.ExecutorInfo, fwinfo *mesos.FrameworkInfo, slaveInfo *mesos.SlaveInfo) {
	fmt.Println("Registered Executor on slave ", slaveInfo.GetHostname())
}

func (exec *exampleExecutor) Reregistered(driver exec.ExecutorDriver, slaveInfo *mesos.SlaveInfo) {
	fmt.Println("Re-registered Executor on slave ", slaveInfo.GetHostname())
}

func (exec *exampleExecutor) Disconnected(exec.ExecutorDriver) {
	fmt.Println("Executor disconnected.")
}

func (exec *exampleExecutor) LaunchTask(driver exec.ExecutorDriver, taskInfo *mesos.TaskInfo) {
	fmt.Println("Launching task", taskInfo.GetName(), "with command", taskInfo.Command.GetValue())
	runStatus := &mesos.TaskStatus{
		TaskId: taskInfo.GetTaskId(),
		State:  mesos.TaskState_TASK_RUNNING.Enum(),
	}
	_, err := driver.SendStatusUpdate(runStatus)
	if err != nil {
		fmt.Println("Got error", err)
	}

	exec.tasksLaunched++
	go exec.extractText(driver, taskInfo)
}

func (exec *exampleExecutor) extractText(driver exec.ExecutorDriver, taskInfo *mesos.TaskInfo) {

	fmt.Println("Total tasks launched ", exec.tasksLaunched)

	//Read information about this URL we are about to process
	payload := bytes.NewReader(taskInfo.GetData())
	var queueMessage OwlCrawlMsg
	dec := gob.NewDecoder(payload)
	err := dec.Decode(&queueMessage)
	if err != nil {
		fmt.Println("decode error:", err)
	}
	queue := mq.New(queueMessage.QueueName)

	//Fetch stored html and do extraction
	doc, err := cloudant.GetURLData(queueMessage.URL)
	if err != nil {
		err = queue.DeleteMessage(queueMessage.ID)
		if err != nil {
			fmt.Printf("Error deleting message id: %s from queue, got: %v\n", queueMessage.ID, err)
		}
		fmt.Printf("Did not find data for url: %s\n", queueMessage.URL)
		updateStatusDied(driver, taskInfo)
		return
	}
	text := parse.ExtractText(doc.HTML)
	fn := func(url string) bool {
		return !cloudant.IsURLThere(url)
	}
	links := parse.ExtractLinks(doc.HTML, doc.URL, fn)
	doc.Text = text
	doc.Links = links.URL
	urlToFetchQueue := mq.New("urls_to_fetch")

	for _, u := range links.URL {
		urlToFetchQueue.PushString(u)
	}

	jsonDocWithText, err := json.Marshal(doc)
	if err != nil {
		fmt.Printf("Error generating json to save docWithText in database, got: %v\n", err)
	}

	//TODO if rev does not match, retry
	ret := cloudant.SaveExtractedText(doc.ID, jsonDocWithText)
	fmt.Printf("ret is %+v\n", ret)

	err = queue.DeleteMessage(queueMessage.ID)
	if err != nil {
		fmt.Printf("Error deleting message id: %s from queue, got: %v\n", queueMessage.ID, err)
	}

	// finish task
	fmt.Println("Finishing task", taskInfo.GetName())
	finStatus := &mesos.TaskStatus{
		TaskId: taskInfo.GetTaskId(),
		State:  mesos.TaskState_TASK_FINISHED.Enum(),
	}
	_, err = driver.SendStatusUpdate(finStatus)
	if err != nil {
		fmt.Println("Got error", err)
	}
	fmt.Println("Task finished", taskInfo.GetName())
}

func (exec *exampleExecutor) KillTask(exec.ExecutorDriver, *mesos.TaskID) {
	fmt.Println("Kill task")
}

func (exec *exampleExecutor) FrameworkMessage(driver exec.ExecutorDriver, msg string) {
	fmt.Println("Got framework message: ", msg)
}

func (exec *exampleExecutor) Shutdown(exec.ExecutorDriver) {
	fmt.Println("Shutting down the executor ")
}

func (exec *exampleExecutor) Error(driver exec.ExecutorDriver, err string) {
	fmt.Println("Got error message:", err)
}

// -------------------------- func inits () ----------------- //
func init() {
	flag.Parse()
}

func main() {
	fmt.Println("Starting Extractor Executor")

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
	fmt.Println("Executor process has started and running.")
	driver.Join()
}

func updateStatusDied(driver exec.ExecutorDriver, taskInfo *mesos.TaskInfo) {
	runStatus := &mesos.TaskStatus{
		TaskId: taskInfo.GetTaskId(),
		State:  mesos.TaskState_TASK_FAILED.Enum(),
	}
	_, err := driver.SendStatusUpdate(runStatus)
	if err != nil {
		fmt.Printf("Failed to tell mesos that we died, sorry, got: %v", err)
	}

}
