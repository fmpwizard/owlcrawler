#!/bin/bash

go build -tags=testSched -o owlcrawler-framework owlcrawler_framework.go && \
go build -tags=testExec -o owlcrawler-executor owlcrawler_executor.go 
