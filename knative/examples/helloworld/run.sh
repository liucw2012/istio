#!/usr/bin/env bash

for i in $(seq 1 100)
do
curl -H "Host:helloworld-go.default.example.com" http://172.31.71.183:31380;
done
