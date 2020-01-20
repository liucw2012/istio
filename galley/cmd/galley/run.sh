#!/usr/bin/env bash


config=""
config="$config --meshConfigFile=mesh.yaml"
config="$config --livenessProbeInterval=1s"
config="$config --readinessProbePath=/healthready"
config="$config --deployment-namespace=istio-system"
config="$config --insecure=true"
config="$config --validation-webhook-config-file=validating.yaml"
config="$config --monitoringPort=15015"
config="$config --log_output_level=default:info"
config="$config --enable-validation=false"
config="$config --kubeconfig=/root/.kube/config"
./galley server --enable-reconcileWebhookConfiguration=false $config

