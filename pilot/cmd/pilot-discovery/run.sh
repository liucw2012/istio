#!/usr/bin/env bash

./pilot-discovery discovery --monitoringAddr=:15014 --log_output_level=default:info --domain cluster.local --meshConfig=mesh.yaml --secureGrpcAddr=""
