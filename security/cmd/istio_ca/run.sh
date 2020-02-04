#!/usr/bin/env bash

export CITADEL_ENABLE_NAMESPACES_BY_DEFAULT=true

config=""
config="$config --append-dns-names=true"
config="$config --grpc-port=8060"
config="$config --citadel-storage-namespace=istio-system"
config="$config --custom-dns-names=istio-pilot-service-account.istio-system:istio-pilot.istio-system"
config="$config --monitoring-port=15014"
config="$config --self-signed-ca=true"
config="$config --workload-cert-ttl=2160h"

./istio_ca $config

unset CITADEL_ENABLE_NAMESPACES_BY_DEFAULT

