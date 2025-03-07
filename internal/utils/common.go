package utils

import "os/exec"

func RealCmdRunner(name string, arg ...string) *exec.Cmd {
	return exec.Command(name, arg...)
}
