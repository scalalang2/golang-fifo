#!/bin/bash

go test -c
./golang-fifo.test -test.v -test.run - -test.bench . -test.count 10 -test.benchmem -test.timeout 10h | tee out