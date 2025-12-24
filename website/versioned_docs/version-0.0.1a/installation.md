---
sidebar_position: 2
---

# Installation

To install and use this plugin, Kubernetes and CloudNativePG users should :

- Build the Docker images and load them to a registry that is accessible
  by the target Kubernetes cluster. You can build them with the
  `make build-images` command, which will execute the appropriate `docker build`
  commands.

- Install the plugin by applying the manifest located in the
  `kubernetes` directory

  ``` console
  $ kubectl apply -k ./kubernetes/dev
  ```

:::note
Kustomize layers and overlays are available in the Kubernetes directory.
You can add your own customisation to patch the resources provided by default.
:::

:::note  
The image used by the CloudNativePG instance sidecar container can be customised 
by adding the `SIDECAR_IMAGE` environment variable to the pgbackrest plugin
controller container.

For example, this patch can be used to add the `SIDECAR_IMAGE` variable :

``` yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: pgbackrest-controller
  namespace: cnpg-system
spec:
  template:
    spec:
      containers:
        - name: pgbackrest-controller
          env:
            - name: SIDECAR_IMAGE
               value: <my_own_image>
```
:::

- The installation can be verified by checking the presence and status
  of the `pgbackrest-controller` deployment in the namespace used by the
  CloudNativePG operator (e.g., `cnpg-system`), but also by confirming
  that the `Custom Resource Definition` `stanza.pgbackrest.dalibo.com`
  is installed.


:::info
To use the latest testing or unstable version of this plugin,
apply the `test` kustomize overlay. It is configured to pull
the latest alpha/beta images from Docker Hub. You can Simply
run: `kubectl apply -k kubernetes/test`
:::
