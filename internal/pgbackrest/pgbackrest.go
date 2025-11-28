// SPDX-FileCopyrightText: 2025 Dalibo <contact@dalibo.com>
//
// SPDX-License-Identifier: Apache-2.0

package pgbackrest

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"

	"github.com/dalibo/cnpg-i-pgbackrest/internal/utils"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type Timestamp struct {
	Start int64 `json:"start"`
	Stop  int64 `json:"stop"`
}

type Lsn struct {
	Start string `json:"start"`
	Stop  string `json:"stop"`
}

type Archive struct {
	Start string `json:"start"`
	Stop  string `json:"stop"`
}

type BackupInfo struct {
	Archive   Archive   `json:"archive"`
	Label     string    `json:"label"`
	Lsn       Lsn       `json:"lsn"`
	Prior     string    `json:"prior"`
	Timestamp Timestamp `json:"timestamp"`
	Type      string    `json:"type"`
}

type BackupData struct {
	Backup []BackupInfo `json:"backup"`
}

type RepoStatus struct {
	Code    *int   `json:"code,omitempty"` // currently on that field is important
	Message string `json:"message"`
}

type Repo struct {
	Status RepoStatus `json:"status"`
}

type PgBackRestInfo struct {
	Repo []Repo `json:"repo"`
}

type CmdRunner func(name string, arg ...string) *exec.Cmd

type PgBackrest struct {
	cmdRunner CmdRunner
	baseEnv   []string
}

func NewPgBackrest(env []string) *PgBackrest {
	return &PgBackrest{
		cmdRunner: utils.RealCmdRunner,
		baseEnv:   env,
	}
}

func (p *PgBackrest) run(args []string, extraEnv []string) *exec.Cmd {
	cmd := p.cmdRunner("pgbackrest", args...)
	cmd.Env = append(os.Environ(), append(p.baseEnv, extraEnv...)...)
	return cmd
}

func (p *PgBackrest) StanzaExists() (bool, error) {
	cmd := p.run([]string{"info", "--output=json"}, nil)
	stdout, err := cmd.CombinedOutput()
	if err != nil {
		return false, fmt.Errorf("can't execute pgbackrest info command: %w", err)
	}
	var info []PgBackRestInfo
	if err := json.Unmarshal(stdout, &info); err != nil {
		return false, fmt.Errorf("can't parse pgbackrest JSON: %w", err)
	}
	return parseDataForStatusCode(info), nil
}

func parseDataForStatusCode(pgbackrestInfo []PgBackRestInfo) bool {
	for _, entry := range pgbackrestInfo {
		for _, repo := range entry.Repo {
			if repo.Status.Code != nil && *(repo.Status.Code) == 0 {
				return true
			}
		}
	}
	return false
}

func (p *PgBackrest) EnsureStanzaExists(stanza string) (bool, error) {
	stanzaExist, err := p.StanzaExists()
	if err != nil {
		return false, fmt.Errorf("can't determine if stanza exists, error %w", err)
	}
	if stanzaExist {
		return false, nil
	}
	cmd := p.run([]string{"stanza-create", "--stanza=" + stanza}, nil)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false, fmt.Errorf("can't create stanza, stdout: %s, error : %w", string(output), err)
	}
	return true, nil
}

func (p *PgBackrest) PushWal(ctx context.Context, walName string) <-chan error {
	cmd := p.run([]string{"archive-push", walName}, nil)
	result := make(chan error, 1)
	logger := log.FromContext(ctx)

	go func() {
		defer close(result)

		stdoutPipe, _ := cmd.StdoutPipe()
		stderrPipe, _ := cmd.StderrPipe()

		if err := cmd.Start(); err != nil {
			logger.Error(err, "can't start pgBackRest archive-push", "WAL", walName)
			result <- err
			return
		}

		// Read stdout & stderr concurrently
		var stdout, stderr []byte
		var outErr, errErr error
		var wg sync.WaitGroup

		wg.Add(2)
		go func() {
			defer wg.Done()
			stdout, outErr = io.ReadAll(stdoutPipe)
		}()
		go func() {
			defer wg.Done()
			stderr, errErr = io.ReadAll(stderrPipe)
		}()

		// Wait for read completion
		wg.Wait()

		done := make(chan error, 1)
		go func() { done <- cmd.Wait() }()

		select {
		case err := <-done:
			if err != nil || outErr != nil || errErr != nil {
				logger.Error(
					err,
					"pgBackRest archive-push failed",
					"WAL",
					walName,
					"stdout",
					string(stdout),
					"stderr",
					string(stderr),
				)
				result <- err
				return
			}

		case <-ctx.Done():
			_ = cmd.Process.Kill() // ensure subprocess tree is killed
			result <- ctx.Err()
		}

	}()

	return result
}

func (p *PgBackrest) GetWAL(walName string, dstPath string) (string, error) {
	cmd := p.run([]string{"archive-get", walName, dstPath}, nil)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("pgBackRest archive-get failed, output: %s %w", string(output), err)
	}
	return string(output), nil
}

func (p *PgBackrest) Backup(lockFile *string) (*BackupInfo, error) {
	env := make([]string, 2)
	env = append(env, "PGBACKREST_archive-check=n")
	if lockFile != nil && (*lockFile) != "" {
		env = append(env, "PGBACKREST_lock-path="+(*lockFile))
	}
	cmd := p.run([]string{"backup"}, env)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("can't backup: %s, error : %w", string(output), err)
	}
	backups, err := p.GetBackupInfo(env)
	if err != nil {
		return nil, err
	}
	latestBackup := LatestBackup(backups)
	return latestBackup, nil
}

func (p *PgBackrest) GetBackupInfo(env []string) ([]BackupInfo, error) {
	cmd := p.run([]string{"info", "--output", "json"}, env)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("can't get pgbackrest info: %s, %w", string(output), err)
	}
	var pgbackrestInfo []BackupData
	err = json.Unmarshal(output, &pgbackrestInfo)
	if err != nil {
		return nil, err
	}
	return pgbackrestInfo[0].Backup, nil

}

func LatestBackup(backups []BackupInfo) *BackupInfo {
	if len(backups) < 1 {
		return nil
	}
	found := backups[0]
	for _, backup := range backups {
		if backup.Timestamp.Stop > found.Timestamp.Stop {
			found = backup
		}
	}
	return &found
}

func (p *PgBackrest) Restore(ctx context.Context, lockFile *string) error {
	contextLogger := log.FromContext(ctx)
	env := make([]string, 2)
	env = append(env, "PGBACKREST_archive-check=n")
	if lockFile != nil && (*lockFile) != "" {
		env = append(env, "PGBACKREST_lock-path="+(*lockFile))
	}
	cmd := p.run([]string{"restore"}, env)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		contextLogger.Info(
			"pgbackrest restore error",
			"stdout",
			stdout.String(),
			"stderr",
			stderr.String(),
		)
		return fmt.Errorf("can't restore: %s, error : %w", stderr.String(), err)
	}
	return nil
}
