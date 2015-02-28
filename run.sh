#!/bin/bash

set -e

./build.sh

./owlcrawler-scheduler \
--master=127.0.0.1:5050 \
--artifactPort=7070 \
--address=127.0.0.1 \
--logtostderr=true
