package kubernetes

import (
	"fmt"
	"os/exec"
)

// TODO: probably rename for a more generic name ?!
type InstallSpec struct {
	ManifestUrl  string
	CmdCustomOpt []string
}

func Apply(s InstallSpec) error {

	cmd := exec.Command("kubectl", "apply", "-f", s.ManifestUrl)
	cmd.Args = append(cmd.Args, s.CmdCustomOpt...)
	if o, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("can't apply manifest: %s, output: %s, error: %w", s.ManifestUrl, string(o), err)
	}
	return nil
}
