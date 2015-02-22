# OwlCrawler

It's a distributed web crawler that uses mesos for scheduling workers, written in Go.

## Building.

Build the framework

`go build -tags=testSched -o owlcrawler-framework owlcrawler_framework.go`

Build the executor

`go build -tags=testExec -o owlcrawler-executor owlcrawler_executor.go`

## Run

```
./owlcrawler-framework \
--master=192.168.1.73:5050 \
--executor="owlcrawler-executor" \
--artifactPort=7070 \
--address=192.168.1.73 \
--logtostderr=true

```

`artifactPort` and `address` point to the server that is hosting the executor, in this example, the framework has a built in http handler to serve the file