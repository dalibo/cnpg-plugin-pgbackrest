// SPDX-FileCopyrightText: 2025 Dalibo <contact@dalibo.com>
//
// SPDX-License-Identifier: Apache-2.0

package cnpg

import (
	"github.com/dalibo/cnpg-i-pgbackrest/test/e2e/internal/kubernetes"
)

func Install(k8sClient kubernetes.K8sClient, installSpec kubernetes.InstallSpec) error {
	installSpec.CmdCustomOpt = []string{"--server-side"}
	if err := kubernetes.Apply(installSpec); err != nil {
		return err
	}
	_, err := k8sClient.DeploymentIsReady("cnpg-system", "cnpg-controller-manager", 15, 3)
	if err != nil {
		return err
	}
	return nil
}
