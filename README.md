<!--
SPDX-FileCopyrightText: 2025 Dalibo

SPDX-License-Identifier: Apache-2.0
-->

# pgBackRest plugin for CloudNativePG

This is an experimental CloudNativePG plugin to backup PostgreSQL instances with
pgBackRest. We hope this plugin to become production-ready as soon as possible !

The documentation is available at [https://plugin-pgbackrest.readthedocs.io/en/latest/](https://plugin-pgbackrest.readthedocs.io/en/latest/).

This plugin is currently developed by [Dalibo](https://dalibo.com).

## Features

- WALs archiving and restoring (asynchronous, using pgBackRest async feature)
- Taking and restoring physical backups
- Point-in-Time Recovery when restoring PostgreSQL Cluster
- Creating secondary based on logshipping
- OCI Container images for controller and sidecar containers

