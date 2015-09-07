#!/bin/bash

go build  -tags=fetcherExec -o owlcrawler-fetcher fetcher.go && \
go build  -tags=extractorExec -o owlcrawler-extractor extractor.go 
