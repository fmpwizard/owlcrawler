// +build test-exec

package main

import (
	"bytes"
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
)

type exampleExecutor struct {
	tasksLaunched int
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
	etcdClient := etcd.NewClient([]string{"127.0.0.1:2379/"})
	ret := etcdClient.SyncCluster()
	if !ret {
		fmt.Println("\n\n\n\n Got problem \n\n\n\n")
	}
	payload := bytes.NewReader(taskInfo.GetData())
	var msgAndID QueueMsg
	dec := gob.NewDecoder(payload)
	err = dec.Decode(&msgAndID)
	if err != nil {
		fmt.Println("decode error:", err)
	}
	queue := mq.New(msgAndID.QueueName)
	resp, err := http.Get(msgAndID.URL)
	if err != nil {
		fmt.Printf("\n\n\n\nError while fetching url: %s, got error: %v\n", msgAndID.URL, err)
		err = queue.ReleaseMessage(msgAndID.ID, 0)
		if err != nil {
			fmt.Printf("Error releasing message id: %s from queue, got: %v\n", msgAndID.ID, err)
		}
		updateStatusDied(driver, taskInfo)
		return
	}
	html, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("\n\n\n\nError while reading html for url: %s, got error: %v\n", msgAndID.URL, err)
		err = queue.ReleaseMessage(msgAndID.ID, 0)
		if err != nil {
			fmt.Printf("Error releasing message id: %s from queue, got: %v\n", msgAndID.ID, err)
		}
		updateStatusDied(driver, taskInfo)
		return
	}
	err = queue.DeleteMessage(msgAndID.ID)
	if err != nil {
		fmt.Printf("Error deleting message id: %s from queue, got: %v\n", msgAndID.ID, err)
	}
	encodedURL := base64.StdEncoding.EncodeToString([]byte(msgAndID.URL))
	_, err = etcdClient.Set(encodedURL, "'url': '"+msgAndID.URL+"', 'html': '"+string(html[:])+"'", 0)
	if err != nil {
		fmt.Printf("Got error adding html to etcd, got: %v\n", err)
	}
	fmt.Printf("\n\n\nhtml url is %s\n\n\n", msgAndID.URL)
	fmt.Printf("\n\n\nhtml encodedURL is %s\n\n\n", encodedURL)
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

type QueueMsg struct {
	URL       string
	ID        string
	QueueName string
}
