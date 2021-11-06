# Flux multi-cluster example repository

This repository demonstrates how Flux can be leveraged for centralized management of multiple clusters and multiple tenants per cluster.

## Terminology

### Workspace

A cluster is managed by creating one workspace Kustomization on it. A workspace is considered as the primary grouping of multiple clusters and one cluster can only be in one workspace.

### Tenant

The second layer of grouping is by tenants. Each cluster may host multiple tenants. Each tenant can only be part of one workspace.
