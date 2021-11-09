#!/usr/bin/env bash
#
set -euo pipefail

# default args are '-a', '-b ARG'
ARGS="n:k:w:h:"

NAME=""
KUBECONFIG=""
HOST_KUBECONFIG=""
WORKSPACE=""

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
            exit 1
    esac
done

function usage() {
    echo "Usage: $(basename "$0") -n NAME -k KUBECONFIG -w WORKSPACE"
}

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
CLUSTERDIR="$CLONEDIR/clusters/$NAME"
git clone git@github.com:makkes/flux-mc-control-plane.git "$CLONEDIR"
cd "$CLONEDIR"
if [ -a "$CLUSTERDIR" ] ; then
    echo "Error: cluster dir already exists in repository"
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
  interval: 1m0s
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
cd remote
kustomize create --resources ../../../workspaces/"$WORKSPACE"

cd ../..
kustomize edit add resource "$NAME"

cd "$CLONEDIR"
git add .
git commit -m "attaching cluster $NAME"
git push
