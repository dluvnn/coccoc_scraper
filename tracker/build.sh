#!/bin/bash

version=0.0.1
time=$(date)

echo $time': building tracker ...'
go build -o ../bin/tracker ./src/main.go
