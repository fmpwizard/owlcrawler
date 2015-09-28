#!/bin/bash

go build  -tags=fetcherExec -o fetcher fetcher.go && \
go build  -tags=extractorExec -o extractor extractor.go 
