// SPDX-FileCopyrightText: 2025 Dalibo <contact@dalibo.com>
//
// SPDX-License-Identifier: Apache-2.0

package kubernetes

import (
	"fmt"
	"os/exec"
)

// TODO: probably rename for a more generic name ?!
type InstallSpec struct {
	ManifestUrl  string
	CmdCustomOpt []string
	UseKustomize bool
}

func Apply(s InstallSpec) error {
	manifestFlag := "-f"
	if s.UseKustomize {
		manifestFlag = "-k"
	}
	cmd := exec.Command("kubectl", "apply", manifestFlag, s.ManifestUrl)
	cmd.Args = append(cmd.Args, s.CmdCustomOpt...)
	if o, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf(
			"can't apply manifest: %s, output: %s, error: %w",
			s.ManifestUrl,
			string(o),
			err,
		)
	}
	return nil
}
