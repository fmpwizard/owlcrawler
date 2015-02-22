#!/bin/bash

go build -tags=testSched -o owlcrawler-framework owlcrawler_framework.go && \
go build -tags=testExec -o owlcrawler-executor owlcrawler_executor.go && \
./owlcrawler-framework \
--master=192.168.1.73:5050 \
--executor="owlcrawler-executor" \
--artifactPort=7070 \
--address=192.168.1.73 \
--logtostderr=true
