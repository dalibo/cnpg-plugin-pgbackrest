---
sidebar_position: 5
---

import CodeBlock from '@theme/CodeBlock';
import Backup from '!!raw-loader!../../examples/backup.yaml';
import ScheduledBackup from '!!raw-loader!../../examples/schedule_backup.yaml';

# Backuping an instance

There are two ways to backup a PostgreSQL Cluster with this plugin through the CloudNativePG operator :

* One shot backup, equivalent to running it by hand but through a `Backup`
  object definition ;
* With `Scheduled Backup` object, equivalent to defining a crontab entry to run a
  backup periodically.

Whatever the kind of backup, users can list and see them with the
appropriate `kubectl` command :

```console
kubectl get backups.postgresql.cnpg.io
```

## One shot backup

Backup can be requested through a `Backup` object, using the default 
CloudNativePG CRD `Backup` definition. The pgbackrest plugin can be specified 
when declaring the `Backup` object. The `method` should be set to
`plugin` and the `pluginConfiguration.name` field to
`pgbackrest.dalibo.com`.

Here is a full example of a backup definition using the pgBackRest
plugin :

<CodeBlock language="yaml">{Backup}</CodeBlock>

It's also possible to use the `cnpg` plugin for `kubectl` to perform 
your backup :

```console
kubectl cnpg backup cluster-demo -m plugin --plugin-name pgbackrest.dalibo.com
```

When performing a backup, you can choose the repository to which to push
it. To do this, you need to define the `selectedRepository` key using
the number of the repository, according to its position in the list of
configured repositories. For example, to use the first repository:

```yaml title="backup.yaml"
[...]
  pluginConfiguration:
    name: pgbackrest.dalibo.com
    parameters:
      selectedRepository: "1"
```

Or with the `cnpg` plugin:

```console
kubectl cnpg backup cluster-demo -m plugin --plugin-name pgbackrest.dalibo.com \
  --plugin-parameters selectedRepository=1
```

## Scheduled backup

A scheduled backup uses almost the same definition as a one-shot backup. Only the kind should be changed to `ScheduledBackup`. When using this object, the schedule field (with a `crontab`-like annotation) should
also be defined under the specification (`spec`).

Here is a full example of a scheduled backup definition using the
pgbackrest plugin :

<CodeBlock language="yaml">{ScheduleBackup}</CodeBlock>
