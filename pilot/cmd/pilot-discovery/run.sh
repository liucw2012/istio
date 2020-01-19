#!/usr/bin/env bash

config=""
config="$config --httpAddr=:9090"
config="$config --monitoringAddr=:15014"
config="$config --log_output_level=default:info"
config="$config --domain cluster.local"
config="$config --meshConfig=mesh.yaml"

./pilot-discovery discovery --secureGrpcAddr="" $config

