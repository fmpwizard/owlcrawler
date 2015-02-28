# OwlCrawler

It's a distributed web crawler that uses mesos for scheduling workers, written in Go.

## Building.

Build the scheduler

`go build -tags=fetcherSched -o owlcrawler-scheduler owlcrawler_scheduler.go`

Build the two executors

```
go build -tags=fetcherExec -o owlcrawler-executor-fetcher owlcrawler_executor_fetcher.go && \
go build -tags=extractorExec -o owlcrawler-executor-extractor owlcrawler_executor_extractor.go 
```

## Run

```
./owlcrawler-scheduler \
--master=127.0.0.1:5050 \
--artifactPort=7070 \
--address=127.0.0.1 \
--logtostderr=true

```

`artifactPort` and `address` point to the server that is hosting the executor, in this example, the framework has a built in http handler to serve the file