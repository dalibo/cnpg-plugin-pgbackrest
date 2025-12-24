---
sidebar_position: 4
---

# How it works
## How it works

Here are some basic informations about how this plugin should work:

- When installing the plugin, a new deployment is created to run a Pod
  for the controller (`pgbackrest-controller`) of our plugin in the same
  namespace as the CNPG operator.

- The CNPG operator detects the plugin when a dedicated Kubernetes
  Service (with some specific annotations) is created.

- Our specialized controller exposes the supported capabilities (at
  least those required to manage the
  [lifecycle](https://pkg.go.dev/github.com/cloudnative-pg/cnpg-i@v0.1.0/pkg/lifecycle)
  of our CNPG instances) to the CNPG operator.

- When initializing a new Cluster configured with our plugin, the
  pgBackRest controller will be called by the CloudNativePG operator.

- The plugin controller modifies the resources (Deployment / Pods /
  Jobs) requested by the CNPG operator (this is done before requesting
  the Kubernetes API), and inject some configuration if needed.

  For our pgbackrest plugin, the controller inject a sidecar container
  for `pgBackRest` within the PostgreSQL Pods. This sidecar container
  executes a manager dedicated to `pgBackRest` (which expose the
  required capabilities archive the WAL and backup the PostgreSQL
  instance).

- Our newly created PostgreSQL instance will call the dedicated
  `pgBackRest` manager (on the side container) when the archive command
  is triggered.

https://github.com/cloudnative-pg/cnpg-i/blob/main/docs/protocol.md#cnpgi-wal-v1-WALCapability-RPC
