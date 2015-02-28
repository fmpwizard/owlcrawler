#!/bin/bash

go build -race -tags=fetcherSched -o owlcrawler-scheduler owlcrawler_scheduler.go && \
go build -race -tags=fetcherExec -o owlcrawler-executor-fetcher owlcrawler_executor_fetcher.go && \
go build -race -tags=extractorExec -o owlcrawler-executor-extractor owlcrawler_executor_extractor.go 
