// +build testExec

package main

import (
	"encoding/json"
	"github.com/fmpwizard/owlcrawler/cloudant"

	"bytes"
	"encoding/gob"
	"flag"
	"fmt"
	"github.com/iron-io/iron_go/mq"
	exec "github.com/mesos/mesos-go/executor"
	mesos "github.com/mesos/mesos-go/mesosproto"
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
	fmt.Println("Total tasks launched ", exec.tasksLaunched)
	//
	// this is where one would perform the requested task
	//

	//Read information about this URL we are about to process
	payload := bytes.NewReader(taskInfo.GetData())
	var queueMessage OwlCrawlMsg
	dec := gob.NewDecoder(payload)
	err = dec.Decode(&queueMessage)
	if err != nil {
		fmt.Println("decode error:", err)
	}
	queue := mq.New(queueMessage.QueueName)

	//Fetch url
	client := &http.Client{}
	req, err := http.NewRequest("GET", queueMessage.URL, nil)
	if err != nil {
		fmt.Printf("Error parsing url: %s, got: %v\n", queueMessage.URL, err)
	}
	req.Header.Set("User-Agent", "OwlCrawler - https://github.com/fmpwizard/owlcrawler")
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error while fetching url: %s, got error: %v\n", queueMessage.URL, err)
		err = queue.ReleaseMessage(queueMessage.ID, 0)
		if err != nil {
			fmt.Printf("Error releasing message id: %s from queue, got: %v\n", queueMessage.ID, err)
		}
		updateStatusDied(driver, taskInfo)
		return
	}

	defer resp.Body.Close()
	htmlData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error while reading html for url: %s, got error: %v\n", queueMessage.URL, err)
		err = queue.ReleaseMessage(queueMessage.ID, 0)
		if err != nil {
			fmt.Printf("Error releasing message id: %s from queue, got: %v\n", queueMessage.ID, err)
		}
		updateStatusDied(driver, taskInfo)
		return
	}

	err = queue.DeleteMessage(queueMessage.ID)
	if err != nil {
		fmt.Printf("Error deleting message id: %s from queue, got: %v\n", queueMessage.ID, err)
	}
	data := &dataStore{
		URL:  queueMessage.URL,
		HTML: string(htmlData[:]),
		Date: time.Now().UTC(),
	}

	pageData, err := json.Marshal(data)
	if err != nil {
		fmt.Printf("Error generating json to save in database, got: %v\n", err)
	}

	ret := cloudant.AddURLData(queueMessage.URL, pageData)
	parseHTMLQueue := mq.New("html_to_parse")
	parseHTMLQueue.PushString(ret.ID)

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
	fmt.Println("Shutting down the executor")
}

func (exec *exampleExecutor) Error(driver exec.ExecutorDriver, err string) {
	fmt.Println("Got error message:", err)
}

// -------------------------- func inits () ----------------- //
func init() {
	flag.Parse()
}

func main() {
	fmt.Println("Starting Example Executor (Go)")

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
