// +build testSched

package main

import (
	"bytes"
	"encoding/gob"
	"flag"
	"fmt"
	"github.com/fmpwizard/owlcrawler/cloudant"
	"github.com/iron-io/iron_go/mq"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	"github.com/gogo/protobuf/proto"
	log "github.com/golang/glog"
	mesos "github.com/mesos/mesos-go/mesosproto"
	util "github.com/mesos/mesos-go/mesosutil"
	sched "github.com/mesos/mesos-go/scheduler"
)

const (
	cpuPerTask = 0.5
	memPerTask = 128
)

var (
	address             = flag.String("address", "127.0.0.1", "Binding address for artifact server")
	artifactPort        = flag.Int("artifactPort", 12345, "Binding port for artifact server")
	master              = flag.String("master", "127.0.0.1:5050", "Master address <ip:port>")
	executorPath        = flag.String("executor", "./test_executor", "Path to test executor")
	mesosAuthPrincipal  = flag.String("mesos_authentication_principal", "", "Mesos authentication principal.")
	mesosAuthSecretFile = flag.String("mesos_authentication_secret_file", "", "Mesos authentication secret file.")
)

//ExampleScheduler Basic scheduler
type ExampleScheduler struct {
	executor      *mesos.ExecutorInfo
	tasksLaunched int
	tasksFinished int
}

func newExampleScheduler(exec *mesos.ExecutorInfo) *ExampleScheduler {
	return &ExampleScheduler{
		executor:      exec,
		tasksLaunched: 0,
		tasksFinished: 0,
	}
}

// Registered implements the Registered handler.
func (sched *ExampleScheduler) Registered(driver sched.SchedulerDriver, frameworkID *mesos.FrameworkID, masterInfo *mesos.MasterInfo) {
	log.Infoln("Framework Registered with Master ", masterInfo)
}

// Reregistered implements the Reregistered handler.
func (sched *ExampleScheduler) Reregistered(driver sched.SchedulerDriver, masterInfo *mesos.MasterInfo) {
	log.Infoln("Framework Re-Registered with Master ", masterInfo)
}

// Disconnected implements the Disconnected handler.
func (sched *ExampleScheduler) Disconnected(sched.SchedulerDriver) {}

//ResourceOffers is where yo udecide if you should use resources or not.
func (sched *ExampleScheduler) ResourceOffers(driver sched.SchedulerDriver, offers []*mesos.Offer) {
	queueName := "urls_to_fetch"
	queue := mq.New(queueName)

	for _, offer := range offers {
		cpuResources := util.FilterResources(offer.Resources, func(res *mesos.Resource) bool {
			return res.GetName() == "cpus"
		})
		cpus := 0.0
		for _, res := range cpuResources {
			cpus += res.GetScalar().GetValue()
		}

		memResources := util.FilterResources(offer.Resources, func(res *mesos.Resource) bool {
			return res.GetName() == "mem"
		})
		mems := 0.0
		for _, res := range memResources {
			mems += res.GetScalar().GetValue()
		}

		log.Infoln("Received Offer <", offer.Id.GetValue(), "> with cpus=", cpus, " mem=", mems)

		remainingCpus := cpus
		remainingMems := mems

		var tasks []*mesos.TaskInfo
		for cpuPerTask <= remainingCpus &&
			memPerTask <= remainingMems {

			msg, err := queue.Get()
			if err != nil {
				break
			} else {

				if cloudant.IsURLThere(msg.Body) { //found an entry, no need to fetch it again
					msg.Delete()
					break
				}
			}

			sched.tasksLaunched++

			taskID := &mesos.TaskID{
				Value: proto.String(strconv.Itoa(sched.tasksLaunched)),
			}
			var msgAndID bytes.Buffer
			enc := gob.NewEncoder(&msgAndID)
			err = enc.Encode(OwlCrawlMsg{
				URL:       msg.Body,
				ID:        msg.Id,
				QueueName: queueName,
			})
			if err != nil {
				log.Fatal("encode error:", err)
			}

			task := &mesos.TaskInfo{
				Name:     proto.String("go-task-" + taskID.GetValue()),
				TaskId:   taskID,
				SlaveId:  offer.SlaveId,
				Executor: sched.executor,
				Resources: []*mesos.Resource{
					util.NewScalarResource("cpus", cpuPerTask),
					util.NewScalarResource("mem", memPerTask),
				},
				Data: msgAndID.Bytes(),
			}

			tasks = append(tasks, task)
			remainingCpus -= cpuPerTask
			remainingMems -= memPerTask

		}
		if len(tasks) > 0 {
			log.Infoln("Launching ", len(tasks), "tasks for offer", offer.Id.GetValue())
		}

		driver.LaunchTasks([]*mesos.OfferID{offer.Id}, tasks, &mesos.Filters{RefuseSeconds: proto.Float64(1)})
	}
}

//StatusUpdate is called to get the latest status of the task
func (sched *ExampleScheduler) StatusUpdate(driver sched.SchedulerDriver, status *mesos.TaskStatus) {
	log.Infoln("Status update: task", status.TaskId.GetValue(), " is in state ", status.State.Enum().String())
	if status.GetState() == mesos.TaskState_TASK_FINISHED {
		sched.tasksFinished++
	}

	if status.GetState() == mesos.TaskState_TASK_LOST ||
		status.GetState() == mesos.TaskState_TASK_KILLED ||
		status.GetState() == mesos.TaskState_TASK_FAILED {
		log.Infoln(
			"Aborting because task", status.TaskId.GetValue(),
			"is in unexpected state", status.State.String(),
			"with message", status.GetMessage(),
		)
		driver.Abort()
	}
}

// OfferRescinded is invoked when an offer is no longer valid (e.g., the slave was
// lost or another framework used resources in the offer). If for
// whatever reason an offer is never rescinded (e.g., dropped
// message, failing over framework, etc.), a framwork that attempts
// to launch tasks using an invalid offer will receive TASK_LOST
// status updates for those tasks (see Scheduler::resourceOffers).
func (sched *ExampleScheduler) OfferRescinded(sched.SchedulerDriver, *mesos.OfferID) {}

// FrameworkMessage is invoked when an executor sends a message. These messages are best
// effort; do not expect a framework message to be retransmitted in
// any reliable fashion.
func (sched *ExampleScheduler) FrameworkMessage(sched.SchedulerDriver, *mesos.ExecutorID, *mesos.SlaveID, string) {
}

//SlaveLost is invoked when a slave has been determined unreachable (e.g.,
// machine failure, network partition). Most frameworks will need to
// reschedule any tasks launched on this slave on a new slave.
func (sched *ExampleScheduler) SlaveLost(sched.SchedulerDriver, *mesos.SlaveID) {}

//ExecutorLost is invoked when an executor has exited/terminated. Note that any
// tasks running will have TASK_LOST status updates automagically
// generated.
func (sched *ExampleScheduler) ExecutorLost(sched.SchedulerDriver, *mesos.ExecutorID, *mesos.SlaveID, int) {
}

//Error is invoked when there is an unrecoverable error in the scheduler or
// scheduler driver. The driver will be aborted BEFORE invoking this
// callback.
func (sched *ExampleScheduler) Error(driver sched.SchedulerDriver, err string) {
	log.Infoln("Scheduler received error:", err)
}

// ----------------------- func init() ------------------------- //

func init() {
	flag.Parse()
	log.Infoln("Initializing the Example Scheduler...")
}

// returns (downloadURI, basename(path))
func serveExecutorArtifact(path string) (*string, string) {
	serveFile := func(pattern string, filename string) {
		http.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
			http.ServeFile(w, r, filename)
		})
	}

	// Create base path (http://foobar:5000/<base>)
	pathSplit := strings.Split(path, "/")
	var base string
	if len(pathSplit) > 0 {
		base = pathSplit[len(pathSplit)-1]
	} else {
		base = path
	}
	serveFile("/"+base, path)

	hostURI := fmt.Sprintf("http://%s:%d/%s", *address, *artifactPort, base)
	log.V(2).Infof("Hosting artifact '%s' at '%s'", path, hostURI)

	return &hostURI, base
}

func prepareExecutorInfo() *mesos.ExecutorInfo {
	executorUris := []*mesos.CommandInfo_URI{}
	uri, executorCmd := serveExecutorArtifact(*executorPath)
	executorUris = append(executorUris, &mesos.CommandInfo_URI{Value: uri, Executable: proto.Bool(true)})

	executorCommand := fmt.Sprintf("./%s", executorCmd)

	go http.ListenAndServe(fmt.Sprintf("%s:%d", *address, *artifactPort), nil)
	log.V(2).Info("Serving executor artifacts...")

	// Create mesos scheduler driver.
	return &mesos.ExecutorInfo{
		ExecutorId: util.NewExecutorID("default"),
		Name:       proto.String("Test Executor (Go)"),
		Source:     proto.String("go_test"),
		Command: &mesos.CommandInfo{
			Value: proto.String(executorCommand),
			Uris:  executorUris,
		},
	}
}

// ----------------------- func main() ------------------------- //

func main() {

	// build command executor
	exec := prepareExecutorInfo()

	// the framework
	fwinfo := &mesos.FrameworkInfo{
		User: proto.String(""), // Mesos-go will fill in user.
		Name: proto.String("Test Framework (Go)"),
	}

	cred := (*mesos.Credential)(nil)
	if *mesosAuthPrincipal != "" {
		fwinfo.Principal = proto.String(*mesosAuthPrincipal)
		secret, err := ioutil.ReadFile(*mesosAuthSecretFile)
		if err != nil {
			log.Fatal(err)
		}
		cred = &mesos.Credential{
			Principal: proto.String(*mesosAuthPrincipal),
			Secret:    secret,
		}
	}

	driver, err := sched.NewMesosSchedulerDriver(
		newExampleScheduler(exec),
		fwinfo,
		*master,
		cred,
	)

	if err != nil {
		log.Errorln("Unable to create a SchedulerDriver ", err.Error())
	}

	if stat, err := driver.Run(); err != nil {
		log.Infof("Framework stopped with status %s and error: %s\n", stat.String(), err.Error())
	}

}

//OwlCrawlMsg is used to pass info to the executor
type OwlCrawlMsg struct {
	URL       string
	ID        string
	QueueName string
}
