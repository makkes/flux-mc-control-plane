#!/usr/bin/env bash

set -euo pipefail

kind create cluster
flux install
kubectl apply -f infra/flux-mc.yaml
kubectl apply -f infra/infra.yaml
