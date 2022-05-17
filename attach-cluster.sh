#!/usr/bin/env bash
#
set -euo pipefail

ARGS="n:k:w:h:"

NAME=""
KUBECONFIG=""
HOST_KUBECONFIG=""
WORKSPACE=""

function usage() {
    echo "Usage: $(basename "$0") -n NAME -k KUBECONFIG -w WORKSPACE [-h HOST_KUBECONFIG]"
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
        k)
            KUBECONFIG=${OPTARG}
            ;;
        w)
            WORKSPACE=${OPTARG}
            ;;
        h)
            HOST_KUBECONFIG=${OPTARG}
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

if [ -z "$KUBECONFIG" ] ; then
    usage
    exit 1
fi
if [ ! -f "$KUBECONFIG" ] ; then
    echo "kubeconfig file $KUBECONFIG is not a file"
    exit 1
fi
if [ ! -r "$KUBECONFIG" ] ; then
    echo "kubeconfig file $KUBECONFIG is not readable"
    exit 1
fi

if [ -z "$WORKSPACE" ] ; then
    usage
    exit 1
fi

flux install --kubeconfig "${HOST_KUBECONFIG:-${KUBECONFIG}}"

CLONEDIR=$(mktemp -d)
trap 'cleanup $CLONEDIR' EXIT
CLUSTERDIR="$CLONEDIR/clusters/$NAME"
git clone git@github.com:makkes/flux-mc-control-plane.git "$CLONEDIR"

cd "$CLONEDIR"
if [ -a "$CLUSTERDIR" ] ; then
    echo "Error: cluster dir already exists in repository"
    exit 1
fi
mkdir -p "$CLUSTERDIR"

kubectl -n flux-system create secret generic "cluster-$NAME" --from-file=value="$KUBECONFIG" --dry-run=client -o yaml > "$CLUSTERDIR"/secret.yaml
cat <<EOF > "$CLUSTERDIR"/sync.yaml
apiVersion: kustomize.toolkit.fluxcd.io/v1beta2
kind: Kustomization
metadata:
  name: cluster-$NAME
  namespace: flux-system
spec:
  interval: 6h
  prune: true
  path: ./clusters/$NAME/remote
  kubeConfig:
    secretRef:
      name: cluster-$NAME
  sourceRef:
    kind: GitRepository
    name: control-plane
EOF
cd "$CLUSTERDIR"
kustomize create --autodetect

mkdir remote
cat <<EOF > "$CLUSTERDIR"/remote/sync.yaml
---
apiVersion: source.toolkit.fluxcd.io/v1beta1
kind: GitRepository
metadata:
  name: cluster
  namespace: flux-system
spec:
  interval: 1m0s
  ref:
    branch: cluster/$NAME
  url: https://github.com/makkes/flux-mc-control-plane
---
apiVersion: kustomize.toolkit.fluxcd.io/v1beta2
kind: Kustomization
metadata:
  name: cluster
  namespace: flux-system
spec:
  interval: 6h
  prune: true
  sourceRef:
    kind: GitRepository
    name: cluster
EOF
cd remote
kustomize create --resources ../../../workspaces/"$WORKSPACE",sync.yaml,../../common

cd ../..
kustomize edit add resource "$NAME"

cd "$CLONEDIR"
git add .
git commit -m "attaching cluster $NAME"
git push
flux reconcile source git control-plane
