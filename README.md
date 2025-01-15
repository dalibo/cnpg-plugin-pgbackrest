# pgBackRest CNPG plugin

Experimental CNPG operator plugin to backup PostgreSQL instances with
pgBackRest.

## plugin anatomy

CNPG plugins are mainly specific pods / services / ... running on the
same namespace as the CNPG Operator (`cnpg-system` by default). They run
some kind of (small) specific "operator" dedicated to one task (eg:
adding and configuring backup to a PostgreSQL `Clusters` managed by
CNPG). Currently they are designed to run alongside the "main" CNPG
operator.

Once a plugin deployed, the CNPG operator register it (in realty it's
done when a service with some specific annotation is created). The
plugin operator should give some information about their
**capabilities** to the CNPG operator. The "main" operator keeps track
of plugins capabilities and calls (through a gRPC based protocol) them
when required.

More information about the architecture:

-   <https://github.com/cloudnative-pg/cnpg-i/blob/main/docs/protocol.md>
-   <https://github.com/cloudnative-pg/cnpg-i>

A plugin should define what **capabilities** are supported and
implements the logic behind. By example a developer can use the
`wal.WALCapability_RPC_TYPE_ARCHIVE_WAL` capability to build a specific
WAL plugin archiver and write the logic of what should be done when the
PostgreSQL archive command is executed (a specific hook is called by the
CNPG manager, more information
[here](https://github.com/cloudnative-pg/cloudnative-pg/blob/main/internal/cnpi/plugin/client/wal.go#L31)).

## plugin anatomy - pgBackRest example

Let's imagine a plugin to archive the WAL with pgBackrest. That plugin
will be split into 2 main components:

-   A minimalist (or nano controller) to inject some configuration (Eg:
    sidecar container) when initializing an instance. That component is
    visible when listing the Pods and Deployments on the `cnpg-system`
    namespace

-   A pgBackRest manager container (a sidecar container injected by the
    previous component) bound to a PostgreSQL / CNPG instance and
    initialized by the "nano" controller. This sidecar-container can be
    seen when observing the Pods dedicated to a PostgreSQL instance.

Here are more information about how this plugin should work:

-   When installing the plugin, a new deployment is created to run a Pod
    for the controller of our plugin on the same namespace as the CNPG
    operator.

-   The CNPG operator can detect the plugin when a dedicated Kubernetes
    Service (with some specific annotation) is created.

-   Our specialized operator / plugin can expose the supported
    capabilities (at least those required to manage the
    [lifecycle](https://pkg.go.dev/github.com/cloudnative-pg/cnpg-i@v0.0.0-20250113133225-d0f454f240a2/pkg/lifecycle)
    of our CNPG instances) to the CNPG operator.

-   When initializing a new CNPG / PostgreSQL instance, our specialized
    operator will be called by the CNPG operator (through gRPC) based on
    the plugin capabilities.

-   In some case, the plugin can alter the resources (Deployment / Pods
    / Jobs) requested by the CNPG operator (this is done before
    requesting the Kubernetes API).

    In our pgbackrest plugin example, the plugin will inject a sidecar
    container for `pgBackRest` within the PostgreSQL Pods. This sidecar
    will execute a manager dedicated to `pgBackRest` (which expose the
    required capabilities to backup the PostgreSQL instance).

-   Our newly created PostgreSQL instance will call the dedicated
    `pgBackRest` manager (on the side container) when the archive
    command is triggered.

<https://github.com/cloudnative-pg/cnpg-i/blob/main/docs/protocol.md#cnpgi-wal-v1-WALCapability-RPC>

## Dev

We start by configuring (and execute) a CNPG development environment,
then try our custom plugin on it:

-   Clone the main CNPG operator repository :
    `$ git clone https://github.com/cloudnative-pg/cloudnative-pg`

-   Move to the newly created directory: `$ cd cloudenative-pg`

-   Install the required dependencies (please follow the instruction
    within the README.md file)

-   Run a Kubernetes cluster with the development version of CNPG:
    `$ make deploy-locally`

-   Then install cert-manager, CNPG operator and the plugin will use
    certificates to communicate securely:
    `kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.16.2/cert-manager.yaml`

Then we can deploy our plugin on our Kubernetes cluster with the running
CloudNative PG operator:

-   Go back to the plugin directory `$ cd -`

-   Build the container image (with the cnpg-i-pgbackrest plugin
    embedded):
    `$ docker build -t pgbackrest-plugin:latest -f containers/Operator.container  .`

-   Load the resulting image to the Kubernetes cluster dedicated to the
    development environment:
    `kind load docker-image pgbackrest-plugin:latest  --name pg-operator-e2e-v1-31-2`

    The name of the cluster can be found by running:
    `$ kind get clusters`

-   The new plugin can now be deployed within the `cnpg-system`
    namespace, the manifest under the `kubernetes` directory can be
    applied: `kubectl apply -f ./kubernetes/`

-   The deployment of this plugin can be verified by checking the
    objects (Deployments, Pods,...) on the `cnpg-system`namespace

Now the plugin should run on a dedicated Pods alongside the CNPG
operator pod. The logs of both Pods can be inspected when creating a new
PostgreSQL instance managed by the CNPG operator.

## Implementation

TODO: document how we implement our plugin

## Install

TODO: give some hint about how to install our plugin

## Usage

TODO: give some hint about how to use our plugin

<!--
    vim: spelllang=en spell
  -->
