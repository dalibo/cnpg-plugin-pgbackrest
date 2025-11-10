// SPDX-FileCopyrightText: 2025 Dalibo <contact@dalibo.com>
//
// SPDX-License-Identifier: Apache-2.0

package certmanager

import (
	"fmt"
	"os/exec"

	"github.com/dalibo/cnpg-i-pgbackrest/test/e2e/internal/kubernetes"
)

func Install(k8sClient kubernetes.K8sClient, installSpec kubernetes.InstallSpec) error {
	cmd := exec.Command("kubectl", "apply", "-f", installSpec.ManifestUrl)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("can't install cert-manager with manifest: %s, output, %s, error: %w", installSpec.ManifestUrl, string(output), err)
	}
	// TODO: do that in parallel
	for _, d := range []string{"cert-manager", "cert-manager-cainjector", "cert-manager-webhook"} {
		_, err := k8sClient.DeploymentIsReady("cert-manager", d, 15, 2)
		if err != nil {
			return fmt.Errorf("can't install certmanager, err: %w", err)
		}
	}
	return nil
}
