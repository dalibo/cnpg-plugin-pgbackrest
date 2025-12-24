---
sidebar_position: 2
---

# Installation

To install and use this plugin, Kubernetes and CloudNativePG users should :

1. Build the Docker images and load them to a registry that is accessible 
by your Kubernetes cluster.

You can build them locally with the command :
```console
make build-images
```

It will execute the appropriate `docker build` commands.

2. Install the plugin by applying the manifest located in the 
`kubernetes` directory :

``` console
kubectl apply -k ./kubernetes/dev
```

:::note
Kustomize layers and overlays are available in the `kubernetes` directory.
You can add your own customisation to patch the resources provided by default.
:::

# Verifications

The installation can be verified by checking the presence and status
of the `pgbackrest-controller` deployment in the namespace used by the
CloudNativePG operator (e.g., `cnpg-system`).

```console
kubectl get pod -n cnpg-system
NAME                                       READY   STATUS    RESTARTS   AGE
cnpg-controller-manager-65bfdb64c9-hwvpd   1/1     Running   0          9m4s
pgbackrest-controller-fc9f4d5f4-5w6wt      1/1     Running   0          15s
```

And also by confirming
that the `Custom Resource Definition` `stanza.pgbackrest.dalibo.com`
is installed.

```console
kubectl api-resources --api-group pgbackrest.dalibo.com
NAME      SHORTNAMES   APIVERSION                 NAMESPACED   KIND
stanzas                pgbackrest.dalibo.com/v1   true         Stanza
```

# Testing versions

To use the latest testing or unstable version of this plugin,
apply the `test` kustomize overlay. It is configured to pull
the latest alpha/beta images from Docker Hub. You can simply run :
```console
kubectl apply -k kubernetes/test
```

# Customisation

The image used by the CloudNativePG instance sidecar container can be customised 
by adding the `SIDECAR_IMAGE` environment variable to the pgbackrest plugin
controller container.

For example, this patch can be used to add the `SIDECAR_IMAGE` variable :

```yaml
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
