---
sidebar_position: 3
---

# Upgrades

Upgrading the pgBackRest plugin can actually be a bit tricky. Here is a
step-by-step guide to upgrading it:

- First, users need to apply the manifest for the new version they want
  to use.

- Then, to upgrade your Clusters to use the new plugin image, we
  actually need to restart the CNPG controller manager due to some
  limitations in our plugin. This can be achieved by running the
  following command:

  ``` console
  kubectl rollout restart deployment -n cnpg-system cnpg-controller-manager
  ```

  :::warning

  Beware: depending on your upgrade strategy, restarting the controller
  can cause your PostgreSQL Clusters to restart. This is because the
  definition of the Pods hosting your PostgreSQL instances is altered to
  use a new pgBackRest sidecar image.

  ::::

- Finally, you can verify that your Clusters and Pods are using the
  image of the newly requested version.

  For example, you can check the sidecar container images used the first
  Pod (`cluster-sample-1`) of the `cluster-sample` Cluster by running:

  ``` console
  kubectl get pods cluster-sample-1 -o jsonpath='{.spec.initContainers[*].image}'
  ```

<!--
    vim: spelllang=en spell
  -->
