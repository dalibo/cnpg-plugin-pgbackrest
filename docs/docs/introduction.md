---
sidebar_position: 1
---

# Introduction

:::warning
This is an experimental CloudNativePG operator plugin to backup PostgreSQL instances with
[pgBackRest](https://pgbackrest.org/).
:::

## Features

- WALs archiving (using pgBackRest async feature)
- Taking and restoring backup
- Point-in-Time Recovery
- Creating secondary based on *Log Shipping*

This plugin is currently only compatible with `s3` and Azure Blob
storage and have been tested with :

- [minIO](https://min.io)
- [Scaleway Object Storage](https://www.scaleway.com/en/object-storage/)
- [Azure Blob
  Storage](https://azure.microsoft.com/en-us/products/storage/blobs)
  (without Workload Identity)

## Dependencies

To use this plugin, these dependencies should be installed on the target
Kubernetes cluster :

- [CloudNativePG](https://cloudnative-pg.io/) (1.27 or newer)
- [Cert-Manager](https://cert-manager.io/)
