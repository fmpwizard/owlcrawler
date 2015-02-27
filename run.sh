#!/bin/bash

set -e

./build.sh

./owlcrawler-fetcher-scheduler \
--master=192.168.1.73:5050 \
--executor="owlcrawler-executor" \
--artifactPort=7070 \
--address=192.168.1.73 \
--logtostderr=true
