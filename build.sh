#!/bin/bash

echo $(date)': building monitor ...'
go build -o bin/monitor monitor/src/main.go

echo $(date)': building tracker ...'
go build -o bin/tracker tracker/src/main.go

echo $(date)': building sampler ...'
go build -o bin/sampler sampler/src/main.go