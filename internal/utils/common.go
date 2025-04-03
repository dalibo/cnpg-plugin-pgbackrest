// SPDX-FileCopyrightText: 2025 Dalibo <contact@dalibo.com>
//
// SPDX-License-Identifier: Apache-2.0

package utils

import "os/exec"

func RealCmdRunner(name string, arg ...string) *exec.Cmd {
	return exec.Command(name, arg...)
}
