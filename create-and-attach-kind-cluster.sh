#!/usr/bin/env bash

set -euo pipefail

ARGS="n:w:"

NAME=""
WORKSPACE=""

function usage() {
    echo "Usage: $(basename "$0") -n NAME -w WORKSPACE"
}

while getopts "${ARGS}" opts; do
    case "${opts}" in
        n)
            NAME=${OPTARG}
            ;;
        w)
            WORKSPACE=${OPTARG}
            ;;
        ?)
            usage
            exit 1
    esac
done

if [ -z "$NAME" ] ; then
    usage
    exit 1
fi

kind create cluster --name "$NAME" --kubeconfig $NAME-kubeconfig

TMPDIR=$(mktemp -d)
KUBECONFIG="$TMPDIR/kubeconfig"
HOST_KUBECONFIG="$TMPDIR/kubeconfig-host"
kind get kubeconfig --internal --name "$NAME" > "$KUBECONFIG"
kind get kubeconfig --name "$NAME" > "$HOST_KUBECONFIG"

./attach-cluster.sh -n "$NAME" -k "$KUBECONFIG" -h "$HOST_KUBECONFIG" -w "$WORKSPACE"
