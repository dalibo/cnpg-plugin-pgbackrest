// SPDX-FileCopyrightText: 2025 Dalibo <contact@dalibo.com>
//
// SPDX-License-Identifier: Apache-2.0

package pgbackrest

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"

	pgbackrestapi "github.com/dalibo/cnpg-i-pgbackrest/internal/pgbackrest/api"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type BackupData struct {
	Backup []pgbackrestapi.BackupInfo `json:"backup"`
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

type CommandExecutor interface {
	CombinedOutput() ([]byte, error)
	Kill() error
	Run() error
	Start() error
	StderrPipe() (io.ReadCloser, error)
	StdoutPipe() (io.ReadCloser, error)
	Wait() error
	SetEnv(env []string)
}
type ExecCmd struct {
	*exec.Cmd
}

func (e *ExecCmd) Kill() error {
	if e.Process == nil {
		return fmt.Errorf("process does not exist")
	}
	return e.Process.Kill()
}
func (e *ExecCmd) SetEnv(env []string) {
	e.Env = env
}

type PgBackrest struct {
	cmdRunner func(name string, args ...string) CommandExecutor
	baseEnv   []string
}

func NewPgBackrest(env []string) *PgBackrest {
	return &PgBackrest{
		cmdRunner: func(name string, args ...string) CommandExecutor {
			return &ExecCmd{exec.Command(name, args...)}
		},
		baseEnv: env,
	}
}

func (p *PgBackrest) run(args []string, extraEnv []string) CommandExecutor {
	cmd := p.cmdRunner("pgbackrest", args...)
	cmd.SetEnv(append(os.Environ(), append(p.baseEnv, extraEnv...)...))
	return cmd
}

func (p *PgBackrest) runBackgroundTask(
	ctx context.Context,
	args []string,
	extraEnv []string,
) <-chan error {
	result := make(chan error, 1)
	logger := log.FromContext(ctx)
	cmd := p.run(args, extraEnv)
	go func() {
		defer close(result)

		stdoutPipe, _ := cmd.StdoutPipe()
		stderrPipe, _ := cmd.StderrPipe()

		if err := cmd.Start(); err != nil {
			logger.Error(err, "can't start pgbackrest task", "args", args)
			result <- err
			return
		}

		// Read stdout & stderr concurrently
		var wg sync.WaitGroup
		var errOut error

		wg.Add(1)
		go func() {
			defer wg.Done()
			scanner := bufio.NewScanner(stdoutPipe)
			for scanner.Scan() {
				l := scanner.Text()
				logger.Info(l)
			}
		}()
		wg.Add(1)
		go func() {
			defer wg.Done()
			scanner := bufio.NewScanner(stderrPipe)
			for scanner.Scan() {
				l := scanner.Text()
				errOut = errors.New(l)
				logger.Error(errOut, "error from pgbackrest")
			}
		}()

		// Wait for read completion
		wg.Wait()

		done := make(chan error, 1)
		go func() { done <- cmd.Wait() }()

		select {
		case err := <-done:
			if err != nil || errOut != nil {
				logger.Error(err, "pgbackrest task failed", "args", args)
				result <- err
				return
			}

		case <-ctx.Done():
			_ = cmd.Kill() // ensure subprocess tree is killed
			result <- ctx.Err()
		}

	}()

	return result
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
	return p.runBackgroundTask(ctx, []string{"archive-push", walName}, nil)
}

func (p *PgBackrest) GetWAL(
	ctx context.Context,
	walName string, dstPath string,
) <-chan error {
	return p.runBackgroundTask(ctx, []string{"archive-get", walName, dstPath}, nil)
}

func (p *PgBackrest) Backup() error {
	env := make([]string, 1)
	env = append(env, "PGBACKREST_ARCHIVE_CHECK=n")
	cmd := p.run([]string{"backup"}, env)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("can't backup: %s, error : %w", string(output), err)
	}
	return nil
}

func (p *PgBackrest) GetBackupInfo() ([]pgbackrestapi.BackupInfo, error) {
	cmd := p.run([]string{"info", "--output", "json"}, nil)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("can't get pgbackrest info: %s, %w", string(output), err)
	}
	var pgbackrestInfo []BackupData
	if err := json.Unmarshal(output, &pgbackrestInfo); err != nil {
		return nil, err
	}
	return pgbackrestInfo[0].Backup, nil

}

func LatestBackup(backups []pgbackrestapi.BackupInfo) *pgbackrestapi.BackupInfo {
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

func FirstBackup(backups []pgbackrestapi.BackupInfo) *pgbackrestapi.BackupInfo {
	if len(backups) < 1 {
		return nil
	}
	found := backups[0]
	for _, backup := range backups {
		if backup.Timestamp.Stop < found.Timestamp.Stop {
			found = backup
		}
	}
	return &found
}

func (p *PgBackrest) Restore(ctx context.Context) <-chan error {
	env := make([]string, 1)
	env = append(env, "PGBACKREST_ARCHIVE_CHECK=n")
	return p.runBackgroundTask(ctx, []string{"restore"}, env)
}
