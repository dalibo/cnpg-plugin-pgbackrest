---
sidebar_position: 6
---

# Configuration

This pgBackRest plugin supports three levels of configuration:

- **Managed configuration:** Options defined via the Stanza CRD.
- **Environment variables:** Free-form variables that can be specified
  using `CustomEnvVars`. `CustomEnvVars` is a list of variables.
- **Hard-coded options:** A few settings built directly into the plugin.

Additionally, the plugin can be configured to use a custom sidecar
image. For details, please see the installation documentation.

## Managed configuration and user's custom variables

The pgBackRest plugin is configured to run pgBackRest via environment
variables. Some of these variables are created and managed automatically
by the plugin based on the pgBackRest `Stanza` object associated with a
`Cluster`. Additional configuration can be specified freely through the
`CustomEnvVar` field, but these variables are only copied from there and
are no validation.

For example, the definition of that `Stanza`:

``` yaml
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
        cipherConfig:
          encryptionPass:
            name: minio
            key: ENCRYPTION_PASS
        secretRef:
          accessKeyId:
            name: minio
            key: ACCESS_KEY_ID
          secretAccessKey:
            name: minio
            key: ACCESS_SECRET_KEY
      CustomEnvVar:
        PGBACKREST_MY_CUSTOM: CNPG_ROCKS
```

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
