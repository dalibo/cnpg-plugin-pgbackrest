---
sidebar_position: 6
---

import CodeBlock from '@theme/CodeBlock';
import StanzaCustom from '!!raw-loader!../../examples/stanza_custom.yaml';
import PluginConfig from '!!raw-loader!../../examples/plugin_config.yaml';

# Configuration

This pgBackRest plugin supports three levels of configuration:

- **Managed configuration:** Options defined via the Stanza or
  `PluginConfig` CRD.
- **Environment variables:** Free-form variables that can be specified
  using `CustomEnvVars`. `CustomEnvVars` is a list of variables.
- **Hard-coded options:** A few settings built directly into the plugin.

Additionally, the plugin can be configured to use a custom sidecar
image. For details, please see the installation documentation.

## Plugin configuration

The `PluginConfig` Custom Resource can be used to customize the behavior
of the plugin. For example, it allows to:

- Define resource limits (CPU, memory, etc.) for the pgBackRest sidecar
  container.
- Specify the `StorageClass` to use when creating a
  PersistentVolumeClaim (PVC) to store pgBackRest spooled WAL files when
  running in asynchronous mode. Using a dedicated PVC ensures that we
  don't lose WAL when operating in asynchronous mode in case the sidecar
  container crashes.

<CodeBlock language="yaml">{PluginConfig}</CodeBlock>

Resource requirements (`requests` and `limits`) can be defined for the
`plugin-pgbackrest` sidecar containers; however, it should be noted that
in Kubernetes the Pod's QoS depends on the resource configuration of all
containers, [different values may therefore downgrade the
QoS](https://kubernetes.io/docs/concepts/workloads/pods/pod-qos/#criteria).

## pgBackRest specific configuration

### Managed configuration and user's custom variables

The pgBackRest plugin is configured to run pgBackRest via environment
variables. Some of these variables are created and managed automatically
by the plugin based on the pgBackRest `Stanza` object associated with a
`Cluster`. Additional configuration can be specified freely through the
`CustomEnvVar` field, but these variables are only copied from there and
are no validation.

For example, the definition of that `Stanza`:

<CodeBlock language="yaml">{StanzaCustom}</CodeBlock>

Will result in pgbackrest running with those environment variables:

``` console
PGBACKREST_LOCK_PATH=/controller/tmp/pgbackrest-cnpg-plugin.lock
PGBACKREST_LOG_LEVEL_FILE=off
PGBACKREST_REPO1_PATH=/cluster-demo
PGBACKREST_REPO1_S3_
PGBACKREST_REPO1_S3_BUCKET=demo
PGBACKREST_REPO1_S3_ENDPOINT=s3.minio.svc.cluster.local
PGBACKREST_REPO1_S3_KEY=value_from_k8s_secret
PGBACKREST_REPO1_S3_KEY_SECRET=value_from_k8s_secret
PGBACKREST_REPO1_S3_REGION=us-east-1
PGBACKREST_REPO1_S3_URI_STYLE=path
PGBACKREST_REPO1_S3_VERIFY_TLS=n
PGBACKREST_STANZA=main
PGBACKREST_MY_CUSTOM=CNPG_ROCKS
```

To run pgBackRest with parameters not directly managed by this plugin,
the `CustomEnvVar` option can be used.

<!--
    vim: spelllang=en spell
  -->
