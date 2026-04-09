// SPDX-FileCopyrightText: 2025 Dalibo <contact@dalibo.com>
//
// SPDX-License-Identifier: Apache-2.0

package pgbackrest

import (
	"context"
	"os/exec"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/log"
)

type PgBackrestExporterRunner struct {
	baseRunner
}

func NewPgBackrestExporterRunner(env []string) *PgBackrestExporterRunner {
	command := "pgbackrest_exporter"
	return &PgBackrestExporterRunner{
		baseRunner: baseRunner{
			command: command,
			cmdRunner: func(args ...string) CommandExecutor {
				return &ExecCmd{exec.Command(command, args...)}
			},
			baseEnv: env,
		},
	}
}

func (p *PgBackrestExporterRunner) RunExporter(ctx context.Context, args []string) error {
	logger := log.FromContext(ctx)
	logger.Info("launching pgbackrest exporter with", "args", args)
	cmd := p.run(args, []string{})
	if err := cmd.Start(); err != nil {
		return err
	}
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			logger.Info("killing pgbackrest exporter")
			return cmd.Kill()
		case <-ticker.C:
			// TODO: Add some logic to check if the exporter is really running ?
			logger.Info("pgbackrest exporter running")
			continue
		}
	}
}
