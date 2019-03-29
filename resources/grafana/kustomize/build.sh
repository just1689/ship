#!/bin/sh

helm template ../chart --name-template metricviz $@  > base.yaml
kustomize build > patched-grafana.yaml
