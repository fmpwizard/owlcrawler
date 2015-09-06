#!/bin/bash

go build  -tags=fetcherSched -o owlcrawler-scheduler owlcrawler_scheduler.go && \
go build  -tags=fetcherExec -o owlcrawler-fetcher owlcrawler_executor_fetcher.go && \
go build  -tags=extractorExec -o owlcrawler-extractor owlcrawler_executor_extractor.go 
