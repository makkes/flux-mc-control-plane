# Flux multi-cluster example repository

This repository demonstrates how Flux can be leveraged for centralized management of multiple clusters and multiple tenants per cluster. A management cluster is responsible for managing resource synchronization to attached clusters by means of applying Flux Kustomizations using each cluster's kubeconfig file.

## Terminology

### Workspace

A workspace is considered as the primary grouping of multiple clusters. One cluster can only be part of one workspace. All resources managed as part of a workspace (e.g. applications to be deployed) are synchronized across all clusters in that workspace, i.e. all clusters in a workspace will receive the same set of resources specific to that workspace.

### Tenant

The second layer of grouping is by tenants. Each cluster may host multiple tenants. Each tenant can only be part of one workspace.

## Quick Start

All of the following commands spin up kind clusters. Make sure you have the following commands installed:

* [kind](https://kind.sigs.k8s.io/)
* [flux](https://fluxcd.io/docs/get-started/#install-the-flux-cli)

First spin up the management cluster:

```sh
./create-management-cluster.sh
```

Then, create a managed cluster:

```sh
./create-and-attach-kind-cluster.sh -n attached -w dev
```

Now you have two clusters running, a management cluster and an attached cluster. Both are synchronized using this Git repository.

## Architecture

![architecture diagram](architecture.png)
