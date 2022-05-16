#!/usr/bin/env bash
#
set -euo pipefail

ARGS="n:"

NAME=""
KUBECONFIG="${KUBECONFIG:-}"

function usage() {
    echo "Usage: $(basename "$0") -n NAME"
}

function cleanup() {
    dir=$1
    rm -rf "$dir"
}

while getopts "${ARGS}" opts; do
    case "${opts}" in
        n)
            NAME=${OPTARG}
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

CLONEDIR=$(mktemp -d)
trap 'cleanup $CLONEDIR' EXIT
CLUSTERDIR="$CLONEDIR/clusters/$NAME"
git clone git@github.com:makkes/flux-mc-control-plane.git "$CLONEDIR"

if [ ! -d "$CLUSTERDIR" ] ; then
    echo "Error: cluster dir doesn't exists in repository"
    exit 1
fi
cd "$CLUSTERDIR"

# stage 1 - remove sync Kustomization
git rm sync.yaml
git commit -m "prepare detachment of cluster $NAME"
git push

echo -n "waiting for cluster Kustomization to be removed..."
while kubectl -n flux-system get ks cluster-"$NAME" ; do
    sleep 1
done
echo "done"

# stage 2 - remove cluster from git

cd ..
git rm -rf "$NAME"
kustomize edit remove resource "$NAME"
git add kustomization.yaml
git commit -m "detach cluster $NAME"
git push
