#!/bin/bash

cd ./bin

monitor_port=$1
if [ -z "$1" ]; then
    monitor_port="8090"
fi

tracker_port=$(($monitor_port + 1))
sampler_port=$(($monitor_port + 2))

./tracker -p="$tracker_port" &>> tracker.log &
./sampler -p="$sampler_port" &>> sampler.log &
./monitor -p="$monitor_port" -a="admin_token_example" -s="http://localhost:$sampler_port" -t="http://localhost:$tracker_port" &>> monitor.log &