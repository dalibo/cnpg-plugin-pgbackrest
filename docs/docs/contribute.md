---
sidebar_position: 6
---

# Contribute

## Deploy CloudNativePG on kind

To contribute and test the pgBackRest plugin, a dedicated Kubernetes
cluster with the CloudNativePG operator is required. Contributors can use the `dev`
version of the CloudNativePG operator and follow those steps to prepare the
required environment.

1. Clone the main CloudNativePG operator repository :
```console
git clone https://github.com/cloudnative-pg/cloudnative-pg
```

2. Move to the newly created directory : 

```console
cd cloudenative-pg
```

3. Install the required dependencies (please follow the instruction
  within the README.md file, at least you will need
  [**kind**](https://kind.sigs.k8s.io/)).

4. Run a Kubernetes cluster with the development version of CloudNativePG :

```console
./hack/setup-cluster.sh create load deploy
```

:::info
Contributors and users can also refer to the CloudNativePG documentation for more
details on how it works and how to run the operator on
[**kind**](https://kind.sigs.k8s.io/).
:::

## Deploy pgBackRest plugin

The pgBackRest plugin can now be deployed on the newly created Kubernetes cluster.

1. Clone the pgBackRest plugin repository :

```console
git clone https://github.com/dalibo/cnpg-plugin-pgbackrest
```
2. Move to the newly created directory : 

```console
cd cd cnpg-plugin-pgbackrest 
```

3. Build the container images used for the plugin (one for the controller and
  another one for the sidecar container) :

```console
make build-images
```

4. The images must now be loaded into the registry dedicated the
  development environment. This is feasible with the `kind load` command :

```console
kind load docker-image pgbackrest-{controller,sidecar}:latest --name <KUBERNETES-CLUSTER-NAME>
```

It's possible to retrieve the name of the Kubernetes cluster by running :
```console
kind get clusters
```

5. The plugin controller can now be deployed within the `cnpg-system`
  namespace. For that the manifests on the `kubernetes` should be
  applied:

  ```console
  kubectl apply -k ./kubernetes/dev
  ```

6. A `pgbackrest-controller` deployment must be present (e.g., in `cnpg-system` namespace). The plugin
  controller should run on a dedicated `Pod` alongside the CloudNativePG operator `Pod`.

7. Your are ready to contribute !

## Executing E2E tests

E2E Tests can be run automatically. To do that, the easiest approach is to
use [**kind**](https://kind.sigs.k8s.io/) and the appropriate make
target :

```console
make test-e2e
```

That command will:

1. Create a dedicated Kubernetes cluster (managed by **kind** and named
  `e2e-cnpg-pgbackrest`)
2. Build the container images for the pgbackrest plugin
3. Load them on the Kubernetes Cluster
4 Run the tests defined on `test/e2e/e2e_test.go`, which also install
  the dependencies and our plugin.

To only run the tests (`test/e2e/e2e_test.go`), the `test-e2e-run-tests`
target can be used :

```console
make test-e2e-run-tests
```
