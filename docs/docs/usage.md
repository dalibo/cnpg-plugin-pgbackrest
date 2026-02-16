---
sidebar_position: 3
---

# Using the plugin

## Importing test

import CodeBlock from '@theme/CodeBlock';
import Secret from '!!raw-loader!../../examples/secret.yaml';
import Stanza from '!!raw-loader!../../examples/stanza.yaml';
import Cluster from '!!raw-loader!../../examples/cluster.yaml';

## Create an instance with pgBackRest

The `examples` directory contains several pre-configured manifests
designed to work with [`kind`](https://kind.sigs.k8s.io/) (Eg: the pull
policy is set to `Never`). These files may require modifications to run
on other types of Kubernetes clusters.

To use this plugin with a `Cluster`, CloudNativePG users must :

1.  Create a `Secret` named `pgbackrest-s3-secret` in the namespace of
    the PostgreSQL `Cluster`. This secret must contain the `key` and
    `secret-key` of the `s3` bucket.

Example:

<CodeBlock language="yaml">{Secret}</CodeBlock>

2.  Create a pgBackRest `stanza` :

Example:

<CodeBlock language="yaml">{Stanza}</CodeBlock>

:::note The `s3Repositories` variable is a list. You can configure
multiple repositories. You can then select the repository to which your
backup will be performed. By default :

- the first repository is selected for backup ;
- WAL archiving always occurs on all repositories. :::

3.  Create the PostgreSQL `Cluster` and adapt the manifest by :

- adding the plugin definition `pgbackrest.dalibo.com` under the
  `plugins` entry;
- referencing the pgBackRest `stanza` resource with `stanzaRef`.

Example:

<CodeBlock language="yaml">{Cluster}</CodeBlock>

If it runs without errors, the `Pod` dedicated to the PostgreSQL
`Cluster` should have now two containers. One for the `postgres` service
(which is the default setup), an other one for the pgbackrest plugin,
named `pgbackrest-plugin`. The injected container now holds the
responsibility for archiving the WALs and triggering backups when a
backup request is made.

## Stanza Consideration

We chose to adhere to the concepts of the pgBackRest project, especially
regarding the scope of a `Stanza` object.

As stated in the
[documentation](https://pgbackrest.org/user-guide.html#quickstart/configure-stanza),
a *stanza* is specific to a PostgreSQL instance cluster.

> A stanza is the configuration for a PostgreSQL database cluster that
> defines where it is located, how it will be backed up, archiving
> options, etc.

Therefore, you will need to create as many `Stanza` objects as you have
deployed `Cluster`.

### Stanza Initialization (or create-stanza operation)

Stanzas are initialized when archiving the first WAL. Since the stanza
initialization state is tracked internally, restarting the sidecar
container will cause the `pgbackrest create-stanza` command to run
again.

Adding a new repository to a stanza can require running the
`create-stanza` command again. Currently, this is not done
automatically. Restarting the `pgbackrest-plugin` container will launch
the create-stanza command.

## WAL Archiving

WAL archiving can be customized through the `pgbackrest` CRD. It is
possible to define the WAL archiving strategy (e.g. [using the
`asynchronous`
mode](https://pgbackrest.org/configuration.html#section-archive/option-archive-async))
as well as configure the `pgbackrest` queue size.

## Restoring a Cluster

To restore a `Cluster` from a backup, create a new `Cluster` that
references the `Stanza` containing the backup. Below is an example:

``` yaml
---
apiVersion: postgresql.cnpg.io/v1
kind: Cluster
metadata:
  name: cluster-restored
spec:
  instances: 1
  plugins:
    - name: pgbackrest.dalibo.com
      parameters:
        stanzaRef: stanza-sample
  storage:
    size: 1Gi
  bootstrap:
    recovery:
      source: origin
  externalClusters:
    - name: origin
      plugin:
        name: pgbackrest.dalibo.com
        parameters:
          stanzaRef: stanza-sample
```

When using the recovery options, the `recoveryTarget` can be specified
to perform point-in-time recovery using a specific strategy (based on
time, LSN, etc.). If it is not specified, the recovery will continue up
to the latest available WAL.

``` yaml
---
apiVersion: postgresql.cnpg.io/v1
kind: Cluster
metadata:
  name: cluster-restored
spec:
  instances: 1
  plugins:
    - name: pgbackrest.dalibo.com
      parameters:
        stanzaRef: stanza-sample
  storage:
    size: 1Gi
  bootstrap:
    recovery:
      source: origin
      recoveryTarget:
        backupID: 20260210-101333F
  externalClusters:
    - name: origin
      plugin:
        name: pgbackrest.dalibo.com
        parameters:
          stanzaRef: stanza-sample
```

If no specific backup (BackupID) is specified, the plugin lets
pgBackRest automatically choose the optimal backup using its standard
algorithm. For more details, see the [pgBackRest restore
documentation](https://pgbackrest.org/command.html#command-restore).
