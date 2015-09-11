# OwlCrawler

It's a distributed web crawler that uses [nats.io](http://nats.io) to coordinate work, written in Go.

## Dependencies

* CouchDB 1.x (tested on 1.6.1)
* gnatsd

## Building.

Build the two workers

```
go build  -tags=fetcherExec -o owlcrawler-fetcher fetcher.go && \
go build  -tags=extractorExec -o owlcrawler-extractor extractor.go
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

3. create a file `.gnatsd.json` and place it in your `$HOME` directory


Sample `.gnatsd.json`

```
{
	"URL": "nats://owlcrawler:natsd_password@127.0.0.1:4222"
}

```

4. Start gnatsd with a user and password (use a config file, but for a quick test
	you can pass parameters):

```
~/gnatsd --user owlcrawler --pass natsd_password
```

## On one terminal run:

```
./extractor -logtostderr=true -v=2
```

## On another terminal run:

```
./fetcher -logtostderr=true -v=2
```
