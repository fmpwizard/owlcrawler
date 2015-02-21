// +build testExec

package main

import (
	"bytes"
	"code.google.com/p/go.net/html"
	"encoding/base64"
	"encoding/gob"
	"flag"
	"fmt"
	"github.com/coreos/go-etcd/etcd"
	"github.com/iron-io/iron_go/mq"
	exec "github.com/mesos/mesos-go/executor"
	mesos "github.com/mesos/mesos-go/mesosproto"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
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
	EtcdHost  string
}

type dataStore struct {
	URL  string
	HTML string
	Date time.Time
}

func (data *dataStore) String() string {
	return "'url': '" + data.URL + "', 'html': '" + data.HTML + "' , 'date' : '" + data.Date.String() + "'"
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
	fmt.Println("\n\n\n\nTotal tasks launched ", exec.tasksLaunched)
	//
	// this is where one would perform the requested task
	//
	payload := bytes.NewReader(taskInfo.GetData())
	var queueMessage OwlCrawlMsg
	dec := gob.NewDecoder(payload)
	err = dec.Decode(&queueMessage)
	if err != nil {
		fmt.Println("decode error:", err)
	}
	queue := mq.New(queueMessage.QueueName)
	resp, err := http.Get(queueMessage.URL)
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
	etcdClient := etcd.NewClient([]string{queueMessage.EtcdHost})
	ret := etcdClient.SyncCluster()
	if !ret {
		fmt.Println("Error: problem sync'ing with etcd server")
	}
	extractLinks(htmlData, queueMessage.URL, queue, etcdClient)

	if err != nil {
		fmt.Printf("\n\n\n\nError while reading html for url: %s, got error: %v\n", queueMessage.URL, err)
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
	encodedURL := base64.StdEncoding.EncodeToString([]byte(queueMessage.URL))
	data := dataStore{
		URL:  queueMessage.URL,
		HTML: string(htmlData[:]),
		Date: time.Now().UTC(),
	}
	_, err = etcdClient.Set(encodedURL, data.String(), 0)
	if err != nil {
		fmt.Printf("Got error adding html to etcd, got: %v\n", err)
	}
	fmt.Printf("==> html encodedURL is %s\n", encodedURL)

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

	driver, err := exec.NewMesosExecutorDriver(newExampleExecutor())

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

func extractLinks(data []byte, originalURL string, q *mq.Queue, etcd *etcd.Client) {
	link, err := url.Parse(originalURL)
	if err != nil {
		fmt.Printf("Error parsing url %s, got: %v\n", originalURL, err)
	}

	d := html.NewTokenizer(bytes.NewReader(data))

	for {
		tokenType := d.Next()
		if tokenType == html.ErrorToken {
			return
		}
		token := d.Token()
		switch tokenType {
		case html.StartTagToken:
			if token.DataAtom.String() == "a" {
				for _, attribute := range token.Attr {
					if attribute.Key == "href" {
						if strings.HasPrefix(attribute.Val, "//") {
							url := fmt.Sprintf("%s:%s", link.Scheme, attribute.Val)
							if sendURLToMQ(url, etcd) {
								fmt.Printf("Sending url: %s:%s\n", url)
								q.PushString(url)
							}
						} else if strings.HasPrefix(attribute.Val, "/") {
							url := fmt.Sprintf("%s://%s%s", link.Scheme, link.Host, attribute.Val)
							if sendURLToMQ(url, etcd) {
								fmt.Printf("Sending url: %s\n", url)
								q.PushString(url)
							}
						} else {
							fmt.Printf("Not sure what to do with this url: %s\n", attribute.Val)
						}
					}
				}
			}
		}
	}
}

func sendURLToMQ(url string, etcd *etcd.Client) bool {
	encodedURL := base64.StdEncoding.EncodeToString([]byte(url))
	_, err := etcd.Get(encodedURL, false, false)
	if err == nil { //found an entry, no need to fetch it again
		return false
	} else {
		return true
	}
}
