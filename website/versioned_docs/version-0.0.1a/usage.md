---
sidebar_position: 3
---

# Using the plugin

## Create an instance with pgBackRest

The `examples` directory contains several pre-configured manifests
designed to work with [`kind`](https://kind.sigs.k8s.io/)
(Eg: the pull policy is set to `Never`). 
These files may require modifications to run on other types of
Kubernetes clusters.

To use this plugin with a `Cluster`, CloudNativePG users must :

1. Create a secret named `pgbackrest-s3-secret` in the namespace of the
  PostgreSQL `Cluster`, this secret must contain the `key` and
  `secret-key` for the `s3` bucket.

  Example:

  ```yaml
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

2. Add a pgbackrest stanza definition:

  Example:

  ```yaml
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

:::note
The `s3Repositories` variable is a list, so you can
configure multiple repositories. You can then select the repository to
which your backup will be performed. By default, the first repository
is selected for backup, WAL archiving always occurs on all
repositories. See the backup chapter for more information.
:::

3. Adapt the PostgreSQL `Cluster` manifest by:
  * Adding the plugin definition `pgbackrest.dalibo.com` under the
    `plugins` entry ;
  * Referencing the pgbackrest configuration directly under the plugin
    declaration.

  Example:

  ```yaml
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

4. Then apply the manifest (`kubectl apply -f instance.yml`)

If it runs without errors, the `Pod` dedicated to the PostgreSQL `Cluster`
should have now two containers. One for the `postgres` service (which is
the default setup), an other one for the pgbackrest plugin, named
`pgbackrest-plugin`. The injected container now holds the responsibility
for archiving the WALs and triggering backups when a backup request is
made.

## Stanza Initialization

Stanzas are initialized when archiving the first WAL. Since the stanza
initialization state is tracked internally, restarting the sidecar
container will require running the `pgbackrest create-stanza` command
again.

## WAL Archiving

WAL archiving can be customized through the `pgbackrest` CRD. It is
possible to define the WAL archiving strategy (e.g. [using the
`asynchronous`
mode](https://pgbackrest.org/configuration.html#section-archive/option-archive-async))
as well as configure the `pgbackrest` queue size.

## Backup an instance

There are two ways to backup a PostgreSQL Cluster managed by the
CloudNativePG operator:

- One shot backup, equivalent to running it by hand but through a Backup
  object definition ;
- Scheduled backup, equivalent to defining a crontab entry to run a
  backup periodically.

Whatever the kind of backup, users can list and see them with the
appropriate kubectl command :

```console
$ kubectl get backups.postgresql.cnpg.io
```

### One shot backup

Backup can be requested through a `Backup` object, using the default 
CloudNativePG CRD backup definition. The pgbackrest plugin can be specified 
when declaring the `Backup`  object, for that the `method` should be set to
`plugin` and the `pluginConfiguration.name` field to
`pgbackrest.dalibo.com`.

Here is a full example of a backup definition using the pgbackrest
plugin :

```yaml
---
apiVersion: postgresql.cnpg.io/v1
kind: Backup
metadata:
  name: backup-example
spec:
  method: plugin
  cluster:
    name: cluster-demo
  pluginConfiguration:
    name: pgbackrest.dalibo.com
```

It's also possible to use the `cnpg` plugin for `kubectl` to perform 
your backup :

```console
$ kubectl cnpg backup cluster-demo -m plugin --plugin-name pgbackrest.dalibo.com
```

When performing a backup, you can choose the repository to which to push
it. To do this, you need to define the `selectedRepository` key using
the number of the repository, according to its position in the list of
configured repositories. For example, to use the first repository:

```yaml
---
apiVersion: postgresql.cnpg.io/v1
kind: Backup
metadata:
  name: backup-example
spec:
  method: plugin
  cluster:
    name: cluster-demo
  pluginConfiguration:
    name: pgbackrest.dalibo.com
    parameters:
      selectedRepository: "1"
```

Or with the `cnpg` plugin:

```console
$ kubectl cnpg backup cluster-demo -m plugin --plugin-name pgbackrest.dalibo.com \
  --plugin-parameters selectedRepository=1
```

### Scheduled backup

A scheduled backup uses almost the same definition as a "simple" backup,
only the kind should be changed to `ScheduledBackup`. When using that
kind of object, the schedule field (with a `crontab`-like annotation) should
also be defined under the specification (`spec`).

Here is a full example of a scheduled backup definition using the
pgbackrest plugin :

```yaml
---
apiVersion: postgresql.cnpg.io/v1
kind: ScheduledBackup
metadata:
  name: backup-example
spec:
  schedule: "0 30 * * * *"
  method: plugin
  cluster:
    name: cluster-demo
  pluginConfiguration:
    name: pgbackrest.dalibo.com
```
