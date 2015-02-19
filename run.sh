#!/bin/bash

go build -tags=testSched -o owlcrawler-framework owlcrawler_framework.go && \
go build -tags=testExec -o owlcrawler-executor owlcrawler_executor.go && \
./owlcrawler-framework \
--master=127.0.0.1:5050 \
--executor="owlcrawler-executor" \
--logtostderr=true
