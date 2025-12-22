<!--
SPDX-FileCopyrightText: 2025 Dalibo

SPDX-License-Identifier: Apache-2.0
-->

# pgBackRest CNPG plugin

Experimental CNPG operator plugin to backup PostgreSQL instances with
pgBackRest.

## Features

- WALs archiving and restoring (asynchronous, using pgBackRest async feature)
- Taking and restoring backup
- PITR AKA Point-in-Time Recovery 
- Creating secondary based on logshipping

This plugin is currently only compatible with `s3` storage and have been
tested with:

- [minIO](https://min.io)
- [Scaleway Object Storage](https://www.scaleway.com/en/object-storage/)

## Dependencies

To use this plugin, these dependencies should be installed on the target
Kubernetes cluster:

- [CloudNativePG](https://cloudnative-pg.io/) 1.27 or newer (those
  versions add the cnpg-i support).
- [Cert-Manager](https://cert-manager.io/)

## How install and use the pgbackrest plugin

### Install

To install and use this plugin, Kubernetes and CNPG users should:

- Build the Docker images and load them to a registry that is accessible
  by the target Kubernetes cluster. You can build them with the
  `make build-images` command, which will execute the appropriate docker build
  commands.

- Install the plugin by applying the manifest located in the
  `kubernetes` directory

  ``` console
  $ kubectl apply -k ./kubernetes/dev
  ```

  > [!NOTE]  
  > Kustomize layers and overlays are available in the  Kubernetes directory.
  > You can add your own customisation to patch the resources provided by default.


  > [!NOTE]  
  > The image used by the CNPG instance sidecar container can be customised by
  > adding the `SIDECAR_IMAGE` environment variable to the pgbackrest plugin
  > controller container.
  >
  > For example, this patch can be used to add the 'SIDECAR_IMAGE' variable:
  >
  > ``` yaml
  > apiVersion: apps/v1
  > kind: Deployment
  > metadata:
  >   name: pgbackrest-controller
  >   namespace: cnpg-system
  > spec:
  >   template:
  >     spec:
  >       containers:
  >         - name: pgbackrest-controller
  >           env:
  >             - name: SIDECAR_IMAGE
  >               value: <my_own_image>
  > ```

  > [!NOTE]  
  > To use the latest testing or unstable version of this plugin,
  > apply the `test` kustomize overlay. It is configured to pull
  > the latest alpha/beta images from Docker Hub. You can Simply
  > run: `kubectl apply -k kubernetes/test`

- The installation can be verified by checking the presence and status
  of the `pgbackrest-controller` deployment in the namespace used by the
  CloudNativePG operator (e.g., `cnpg-system`), but also by confirming
  that the Custom Resource Definition `stanza.pgbackrest.dalibo.com`
  is installed.

### Initiate an instance with pgBackRest

The `examples` directory contains several pre-configured manifests
designed to work with kind (Eg: the pull policy is set to `Never`).
These files may require modifications to run on other types of
Kubernetes clusters.

To use this plugin with a **Cluster**, CNPG users must:

- Create a secret named `pgbackrest-s3-secret` in the namespace of the
  PostgreSQL Cluster, this secret must contain the `key` and
  `secret-key` for the `S3` bucket.

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

- Add a pgbackrest stanza definition:

  Example:

  ``` yaml
  ---
  apiVersion: pgbackrest.dalibo.com/v1
  kind: stanza
  metadata:
    name: stanza-sample
  spec:
    stanzaConfiguration:
      stanza: main
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

> [!NOTE]  
> The 's3Repositories' variable is a list, so you can
> configure multiple repositories. You can then select the repository to
> which your backup will be performed. By default, the first repository
> is selected for backup, WAL archiving always occurs on all
> repositories. See the backup chapter for more information.

- Adapt the PostgreSQL Cluster manifest by:

  - Adding the plugin definition `pgbackrest.dalibo.com` under the
    `plugins` entry.
  - Referencing the pgbackrest configuration directly under the plugin
    declaration

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
        parameters:
          stanzaRef: stanza-sample
    storage:
      size: 1Gi
  ```

- Then apply the manifest (`kubectl apply -f instance.yml`)

If it runs without errors, the Pod dedicated to the PostgreSQL Cluster
should have now two containers. One for the `postgres` service (which is
the default setup), an other one for the pgbackrest plugin, named
`pgbackrest-plugin`. The injected container now holds the responsibility
for archiving the WALs and triggering backups when a backup request is
made.

### Stanza Initialization

Stanzas are initialized when archiving the first WAL. Since the stanza
initialization state is tracked internally, restarting the sidecar
container will require running the `pgbackrest create-stanza` command
again.

### WAL Archiving

WAL archiving can be customized through the `pgbackrest` CRD. It is
possible to define the WAL archiving strategy (e.g. [using the
`asynchronous`
mode](https://pgbackrest.org/configuration.html#section-archive/option-archive-async))
as well as configure the `pgbackrest` queue size.

### Backup an instance

There are two ways to backup a PostgreSQL Cluster managed by the
CloudNativePG operator:

- One shot backup, equivalent to running it by hand but through a Backup
  object definition
- Scheduled backup, equivalent to defining a crontab entry to run a
  backup periodically

Whatever the kind of backup, users can list and see them with the
appropriate kubectl command:

``` console
$ kubectl get backups.postgresql.cnpg.io
```

#### One shot backup

Backup can be requested through a Backup object, using the default CNPG
CRD backup definition. The pgbackrest plugin can be specified when
declaring the backup object, for that the `method` should be set to
`plugin` and the `pluginConfiguration.name` field to
`pgbackrest.dalibo.com`.

Here is a full example of a backup definition using the pgbackrest
plugin:

``` yaml
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

It's also possible to use the `cnpg` plugin to perform your backup:

``` console
$ kubectl cnpg backup cluster-demo -m plugin --plugin-name pgbackrest.dalibo.com
```

When performing a backup, you can choose the repository to which to push
it. To do this, you need to define the `selectedRepository` key using
the number of the repository, according to its position in the list of
configured repositories. For example, to use the first repository:

``` yaml
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

Or with the CNPG plugin:

``` console
$ kubectl cnpg backup cluster-demo -m plugin --plugin-name pgbackrest.dalibo.com \
  --plugin-parameters selectedRepository=1
```

#### Scheduled backup

A scheduled backup uses almost the same definition as a "simple" backup,
only the kind should be adapted (to `ScheduledBackup`). When using that
kind of object, the schedule field (with a `crontab` annotation) should
also be defined under the specification (`spec`).

Here is a full example of a scheduled backup definition using the
pgbackrest plugin:

``` yaml
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

<https://github.com/cloudnative-pg/cnpg-i/blob/main/docs/protocol.md#cnpgi-wal-v1-WALCapability-RPC>

## Dev

To contribute and test the pgbackrest plugin a dedicated Kubernetes
cluster with the CNPG operator is required. Contributors can use the dev
version of the CNPG operator and follow those steps to prepare the
required environment.

- Clone the main CNPG operator repository :
  `$ git clone https://github.com/cloudnative-pg/cloudnative-pg`

- Move to the newly created directory: `$ cd cloudenative-pg`

- Install the required dependencies (please follow the instruction
  within the README.md file, at least you will need
  [**kind**](https://kind.sigs.k8s.io/))

- Run a Kubernetes cluster with the development version of CNPG:
  `$ ./hack/setup-cluster.sh create load deploy`

- Then install cert-manager, CNPG operator and the plugin will use
  certificates to communicate securely:
  `kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.16.2/cert-manager.yaml`

Contributors and users can also refer to the CNPG documentation for more
details on how it works and how to run the operator on
[**Kind**](https://kind.sigs.k8s.io/).

The plugin can now be deployed on that Kubernetes cluster:

- Go back to the plugin directory `$ cd -`

- Build the container images for the plugin (One for the controller and
  another one for the sidecar container):

  ``` console
  $ docker build --tag pgbackrest-controller:latest --target pgbackrest-controller  -f containers/pgbackrestPlugin.containers .
  $ docker build --tag pgbackrest-sidecar:latest --target pgbackrest-sidecar -f containers/pgbackrestPlugin.containers .
  ```

- The images should now be loaded into the registry dedicated the
  development environment:

  ``` console
  $ kind load docker-image pgbackrest-{controller,sidecar}:latest --name pg-operator-e2e-v1-31-2
  ```

  If needed, it's possible to retrieve the name of the cluster by
  running:

  ``` console
  $ kind get clusters
  ```

- The plugin controller can now be deployed within the `cnpg-system`
  namespace. For that the manifests on the `kubernetes` should be
  applied:

  ``` console
  $ kubectl apply -k ./kubernetes/dev
  ```

- Then the deployment can be verified by inspecting the objects
  (Deployments, Pods,...) on the `cnpg-system` namespace. A
  `pgbackrest-controller` deployment must be present. The plugin
  controller should run on a dedicated Pod alongside the CNPG operator
  Pod.

### Executing E2E tests

E2E Tests can be run automatically, for that the easiest approach is to
use [**kind**](https://kind.sigs.k8s.io/) and the appropriate make
target:

``` console
$ make test-e2e
```

That command will:

- Create a dedicated Kubernetes cluster (managed by **kind** and named
  `e2e-cnpg-pgbackrest`)
- Build the container images for the pgbackrest plugin
- Load them on the Kubernetes Cluster
- Run the tests defined on `test/e2e/e2e_test.go`, which also install
  the dependencies and our plugin.

To only run the tests (`test/e2e/e2e_test.go`), the `test-e2e-run-tests`
target can be used:

``` console
$ make test-e2e-run-tests
```

## Resources

This plugin is inspired by the:

- [Barman Cloud
  plugin](https://github.com/cloudnative-pg/plugin-barman-cloud)
- [Hello World
  plugin](https://github.com/cloudnative-pg/cnpg-i-hello-world)

<!--
    vim: spelllang=en spell
  -->
