#!/bin/bash

go build -tags=fetcherSched -o owlcrawler-fetcher-scheduler fetcher/owlcrawler_scheduler.go && \
go build -tags=fetcherExec -o owlcrawler-fetcher-executor fetcher/owlcrawler_executor.go && \
go build -tags=extractorSched -o owlcrawler-extractor-scheduler extractor/owlcrawler_scheduler.go && \
go build -tags=extractorExec -o owlcrawler-extractor-executor extractor/owlcrawler_executor.go 
