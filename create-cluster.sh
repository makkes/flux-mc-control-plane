#!/usr/bin/env bash

set -euo pipefail

kind create cluster
flux install
flux create source git flux-mc --url=https://github.com/makkes/flux-mc --branch=main --interval=1m
flux create kustomization infra --source=GitRepository/flux-mc --path=infra
