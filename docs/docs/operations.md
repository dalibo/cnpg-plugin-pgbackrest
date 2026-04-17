---
sidebar_position: 4
---

import CodeBlock from '@theme/CodeBlock';
import Secret from '!!raw-loader!../../examples/secret.yaml';
import Stanza from '!!raw-loader!../../examples/stanza.yaml';
import Cluster from '!!raw-loader!../../examples/cluster.yaml';
import ClusterRestored from '!!raw-loader!../../examples/cluster_restored.yaml';
import ClusterRestoredBackupID from '!!raw-loader!../../examples/cluster_restored_backupid.yaml';
import Backup from '!!raw-loader!../../examples/backup.yaml';
import ScheduleBackup from '!!raw-loader!../../examples/schedule_backup.yaml';

# Operations

## Create a Cluster with pgBackRest

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

2.  Create a pgBackRest `stanza`

    <CodeBlock language="yaml">{Stanza}</CodeBlock>

    :::note

    The `s3Repositories` variable is a list. You can configure multiple
    repositories. You can then select the repository to which your
    backup will be performed. By default :

    - the first repository is selected for backup ;
    - WAL archiving always occurs on all repositories.

    :::

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

## Backup a Cluster

There are two ways to backup a PostgreSQL Cluster with this plugin
through the CloudNativePG operator :

- One shot backup, equivalent to running it by hand but through a
  `Backup` object definition ;
- With `Scheduled Backup` object, equivalent to defining a crontab entry
  to run a backup periodically.

Whatever the kind of backup, users can list and see them with the
appropriate `kubectl` command :

``` console
kubectl get backups.postgresql.cnpg.io
```

### On-demand backups

Backup can be requested through a `Backup` object, using the default
CloudNativePG CRD `Backup` definition. The pgbackrest plugin can be
specified when declaring the `Backup` object. The `method` should be set
to `plugin` and the `pluginConfiguration.name` field to
`pgbackrest.dalibo.com`.

Here is a full example of a backup definition using the pgBackRest
plugin:

<CodeBlock language="yaml">{Backup}</CodeBlock>

It's also possible to use the `cnpg` plugin for `kubectl` to perform
your backup :

``` console
kubectl cnpg backup cluster-sample -m plugin --plugin-name pgbackrest.dalibo.com
```

When performing a backup, you can choose the repository to which to push
it. To do this, you need to define the `selectedRepository` key using
the number of the repository, according to its position in the list of
configured repositories. For example, to use the first repository:

``` yaml
[...]
  pluginConfiguration:
    name: pgbackrest.dalibo.com
    parameters:
      selectedRepository: "1"
```

Or with the `cnpg` plugin:

``` console
kubectl cnpg backup cluster-sample -m plugin --plugin-name pgbackrest.dalibo.com \
  --plugin-parameters selectedRepository=1
```

### Scheduled backup

A scheduled backup uses almost the same definition as a one-shot backup.
Only the kind should be changed to `ScheduledBackup`. When using this
object, the schedule field (with a `crontab`-like annotation) should
also be defined under the specification (`spec`).

Here is a full example of a scheduled backup definition using the
pgBackRest plugin:

<CodeBlock language="yaml">{ScheduleBackup}</CodeBlock>

## Restoring a Cluster

To restore a `Cluster` from a backup, create a new `Cluster` that
references the `Stanza` containing the backup. Below is an example:

<CodeBlock language="yaml">{ClusterRestored}</CodeBlock>

When using the recovery options, the `recoveryTarget` can be specified
to perform point-in-time recovery using a specific strategy (based on
time, LSN, etc.). If it is not specified, the recovery will continue up
to the latest available WAL.

<CodeBlock language="yaml">{ClusterRestoredBackupID}</CodeBlock>

If no specific backup (BackupID) is specified, the plugin lets
pgBackRest automatically choose the optimal backup using its standard
algorithm. For more details, see the [pgBackRest restore
documentation](https://pgbackrest.org/command.html#command-restore).

## WAL Archiving

WAL archiving can be customized through the `Stanza` CRD. It is possible
to define the WAL archiving strategy (e.g. [using the `asynchronous`
mode](https://pgbackrest.org/configuration.html#section-archive/option-archive-async))
as well as configure the `pgbackrest` queue size.

## Stanza Consideration

We chose to adhere to the concepts of the pgBackRest project, especially
regarding the scope of a `Stanza` object.

As stated in the
[documentation](https://pgbackrest.org/user-guide.html#quickstart/configure-stanza),
a *stanza* is specific to a PostgreSQL Cluster.

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
