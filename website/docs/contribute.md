---
sidebar_position: 5
---

# Contribute

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

## Executing E2E tests

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
