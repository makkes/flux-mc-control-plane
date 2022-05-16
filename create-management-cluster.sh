#!/usr/bin/env bash

set -euo pipefail

kind create cluster --name management --config ~/kind-with-custom-registry.yaml
flux install
kubectl apply -f control-plane/gitrepo.yaml
kubectl apply -f control-plane/ks.yaml
