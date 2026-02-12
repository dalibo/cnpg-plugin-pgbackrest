---
sidebar_position: 3
---

# Using the plugin

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

``` yaml
---
apiVersion: v1
kind: Secret
metadata:
  name: pgbackrest-s3-secret
type: Opaque
stringData:
  ACCESS_KEY_ID: <key_to_replace>
  ACCESS_SECRET_KEY: <secret_to_replace>
```

2.  Create a pgBackRest `stanza` :

Example:

``` yaml
---
apiVersion: pgbackrest.dalibo.com/v1
kind: Stanza
metadata:
  name: stanza-sample
spec:
  stanzaConfiguration:
    name: main
    s3Repositories:
      - bucket: demo
        endpoint: s3.minio.svc.cluster.local
        region: us-east-1
        repoPath: /cluster-demo
        uriStyle: path
        verifyTLS: false
        retentionPolicy:
          full: 7
          fullType: count
          diff: 14
          archive: 2
          archiveType: full
          history: 30
        secretRef:
          accessKeyId:
            name: pgbackrest-s3-secret
            key: ACCESS_KEY_ID
          secretAccessKey:
            name: pgbackrest-s3-secret
            key: ACCESS_SECRET_KEY
```

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

``` yaml
---
apiVersion: postgresql.cnpg.io/v1
kind: Cluster
metadata:
  name: cluster-demo
spec:
  instances: 1
  plugins:
    - name: pgbackrest.dalibo.com
      isWALArchiver: true
      parameters:
        stanzaRef: stanza-sample
  storage:
    size: 1Gi
```

If it runs without errors, the `Pod` dedicated to the PostgreSQL
`Cluster` should have now two containers. One for the `postgres` service
(which is the default setup), an other one for the pgbackrest plugin,
named `pgbackrest-plugin`. The injected container now holds the
responsibility for archiving the WALs and triggering backups when a
backup request is made.

## Stanza Initialization

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
