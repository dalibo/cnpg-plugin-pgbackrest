// SPDX-FileCopyrightText: 2025 Dalibo <contact@dalibo.com>
//
// SPDX-License-Identifier: Apache-2.0

package pgbackrest

import (
	"github.com/dalibo/cnpg-i-pgbackrest/test/e2e/internal/kubernetes"
)

func Install(k8sClient kubernetes.K8sClient, installSpec kubernetes.InstallSpec) error {
	if err := kubernetes.Apply(installSpec); err != nil {
		return err
	}
	_, err := k8sClient.DeploymentIsReady("cnpg-system", "pgbackrest-controller", 15, 2)
	return err
}
