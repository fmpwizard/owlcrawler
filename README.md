# OwlCrawler

It's a distributed web crawler that uses mesos for scheduling workers, written in Go.

## Dependencies

* CouchDB 1.x (tested on 1.6.1)
* IronMQ account

## Building.

Build the scheduler

`go build -tags=fetcherSched -o owlcrawler-scheduler owlcrawler_scheduler.go`

Build the two executors

```
go build -tags=fetcherExec -o owlcrawler-executor-fetcher owlcrawler_executor_fetcher.go && \
go build -tags=extractorExec -o owlcrawler-executor-extractor owlcrawler_executor_extractor.go 
```

### Setup

1. Setup couchdb with at least one admin user, you can follow the instructions [here](http://stackoverflow.com/a/6418670/309896)
2. create a file `.couchdb.json` and place it in your `$HOME` directory


Sample `.couchdb.json`

```
{
	"user": "user-here",
	"password": "super-secret-password",
	"url": "http://localhost:5984/owl-crawler"
}

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