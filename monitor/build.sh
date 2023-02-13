#!/bin/bash

version=0.0.1
time=$(date)

echo $time': building monitor ...'
go build -o ../bin/monitor ./src/main.go
