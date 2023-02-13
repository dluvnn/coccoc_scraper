#!/bin/bash

version=0.0.1
time=$(date)

echo $time': building sampler ...'
go build -o ../bin/sampler ./src/main.go
