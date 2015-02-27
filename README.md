# OwlCrawler

It's a distributed web crawler that uses mesos for scheduling workers, written in Go.

## Building.

Build the scheduler

`go build -tags=fetcherSched -o owlcrawler-fetcher-scheduler fetcher/owlcrawler_scheduler.go`

Build the executor

`go build -tags=fetcherExec -o owlcrawler-fetcher-executor fetcher/owlcrawler_executor.go`

## Run

```
./owlcrawler-fetcher-scheduler \
--master=192.168.1.73:5050 \
--executor="owlcrawler-executor" \
--artifactPort=7070 \
--address=192.168.1.73 \
--logtostderr=true

```

`artifactPort` and `address` point to the server that is hosting the executor, in this example, the framework has a built in http handler to serve the file