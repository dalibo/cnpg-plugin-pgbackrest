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
	"maps"
	"os/exec"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	pgbackrestapi "github.com/dalibo/cnpg-i-pgbackrest/api/v1"
)

type CmdRunner func(args ...string) CommandExecutor

func newPgBackrestWithRunner(env []string, runner CmdRunner) *PgBackrestRunner {
	return &PgBackrestRunner{
		baseRunner: baseRunner{
			command:   "pgbackrest",
			cmdRunner: runner,
			baseEnv:   env,
		},
	}
}

func newPgBackrestWithRunnerAndConfig(
	env []string,
	conf *pgbackrestapi.StanzaConfiguration,
	runner CmdRunner,
) *PgBackrestRunner {
	return &PgBackrestRunner{
		baseRunner: baseRunner{
			command:   "pgbackrest",
			cmdRunner: runner,
			baseEnv:   env,
		},
		stanzaConfiguration: conf,
	}
}

var backupInfo = []pgbackrestapi.BackupInfo{
	{
		Archive: pgbackrestapi.Archive{
			Start: "000000010000000000000001",
			Stop:  "000000010000000000000002",
		},
		Label: "backup_20250306",
		Lsn: pgbackrestapi.Lsn{
			Start: "0/16B2D80",
			Stop:  "0/16B2E00",
		},
		Prior: "backup_20250305",
		Timestamp: pgbackrestapi.Timestamp{
			Start: 1710000000,
			Stop:  1710003600,
		},
		Type: "full",
	},
}

func TestLatestBackup(t *testing.T) {
	backupsInfoTwo := append(backupInfo, pgbackrestapi.BackupInfo{
		Archive: pgbackrestapi.Archive{
			Start: "000000010000000000000003",
			Stop:  "000000010000000000000004",
		},
		Label: "backup_20250307",
		Lsn: pgbackrestapi.Lsn{
			Start: "0/16C2D80",
			Stop:  "0/16C2E00",
		},
		Prior: "backup_20250306",
		Timestamp: pgbackrestapi.Timestamp{
			Start: 1810007200,
			Stop:  1810010800,
		},
		Type: "incremental",
	})
	backupsInfoThree := append(backupsInfoTwo, pgbackrestapi.BackupInfo{
		Archive: pgbackrestapi.Archive{
			Start: "000000010000000000000003",
			Stop:  "000000010000000000000004",
		},
		Label: "backup_20250307",
		Lsn: pgbackrestapi.Lsn{
			Start: "0/16C2D80",
			Stop:  "0/16C2E00",
		},
		Prior: "backup_20250306",
		Timestamp: pgbackrestapi.Timestamp{
			Start: 1910007200,
			Stop:  1910010800,
		},
		Type: "incremental",
	})
	type TestCase struct {
		desc string
		data []pgbackrestapi.BackupInfo
		want *pgbackrestapi.BackupInfo
	}
	testCases := []TestCase{
		{"No backup", []pgbackrestapi.BackupInfo{}, nil},
		{"Only one backup", backupInfo, &backupInfo[0]},
		{"Three backups", backupsInfoThree, &backupsInfoThree[2]},
		{"Two backups", backupsInfoTwo, &backupsInfoTwo[1]},
	}
	for _, tc := range testCases {
		f := func(t *testing.T) {
			got := LatestBackup(tc.data)
			if (tc.want == nil && got != tc.want) ||
				(got != nil && tc.want != nil && *got != *tc.want) {
				t.Errorf("error %v\n%v", got, tc.data)
			}

		}
		t.Run(tc.desc, f)
	}
}

func TestAllReposHaveZeroStatusCode(t *testing.T) {
	testCases := []struct {
		desc string
		data string
		want bool
	}{
		{
			"code error",
			`[{"repo": [{"status": {"code": 2, "message": "BLA" }}]}]`,
			false,
		},
		{
			"code missing",
			`[{"repo": [{"status": {"message": "Machin" }}]}]`,
			false,
		},
		{
			"code ok",
			`[{"repo": [{"status": {"code": 0, "message": "OK" }}]}]`,
			true,
		},
		{
			"multiple repo, one is not yet configured",
			`[{"repo":[{"key":1,"status":{"code":0,"message":"ok"}},{"key":2,"status":{"code":1,"message":"missing stanza path"}}]}]`,
			false,
		},
		{
			"empty repo",
			`[{"repo": []}]`,
			false,
		},
		{
			"empty list",
			`[]`,
			false,
		},
	}
	for _, tc := range testCases {
		f := func(t *testing.T) {
			var pgbackrestInfo []PgBackRestInfo
			err := json.Unmarshal([]byte(tc.data), &pgbackrestInfo)
			if err != nil {
				fmt.Println(err)
				panic("should not happen")
			}
			got := allReposHaveZeroStatusCode(pgbackrestInfo)
			if got != tc.want {
				t.Errorf("error want: %v, got %v", tc.want, got)
			}
		}
		t.Run(tc.desc, f)
	}
}

type fakeExec struct {
	cmdName string
	args    []string
}

type execCalls struct {
	execCalls []fakeExec
}

func (e *execCalls) fakeCmdRunner(output string, err error) CmdRunner {
	return func(args ...string) CommandExecutor {
		// Track the command call
		e.execCalls = append(e.execCalls, fakeExec{cmdName: "pgbackrest", args: args})
		// Fake the command execution by returning a function that provides predefined output
		cmd := exec.Command("echo", output) // Fake command that outputs JSON
		if err != nil {
			cmd = exec.Command("false") // Simulate failure
		}
		return &ExecCmd{Cmd: cmd}
	}
}

func TestPushWal(t *testing.T) {
	type TestCase struct {
		desc    string
		walPath string
		want    execCalls
	}
	testCases := []TestCase{
		{
			desc: "push wal", walPath: "/machin",
			want: execCalls{
				execCalls: []fakeExec{
					{cmdName: "pgbackrest", args: []string{"archive-push", "/machin"}},
				},
			},
		},
	}
	output := ""
	for _, tc := range testCases {
		fExec := execCalls{}
		f := func(t *testing.T) {
			pgb := newPgBackrestWithRunner(nil, fExec.fakeCmdRunner(output, nil))
			errCh := pgb.PushWal(context.Background(), tc.walPath)
			if err := <-errCh; err != nil {
				t.Errorf("can't simulate push WAL, %v", err)
			}
			if !reflect.DeepEqual(fExec, tc.want) {
				t.Errorf("error want %v, got %v", fExec, tc.want)
			}
		}
		t.Run(tc.desc, f)
	}
}

func TestBackup(t *testing.T) {
	testCases := []struct {
		desc string
		want execCalls
	}{
		{
			desc: "run backup",
			want: execCalls{
				execCalls: []fakeExec{
					{cmdName: "pgbackrest", args: []string{"backup"}},
				},
			},
		},
	}

	backup := "" // we don't care about output here
	for _, tc := range testCases {
		fExec := execCalls{}
		t.Run(tc.desc, func(t *testing.T) {
			pgb := newPgBackrestWithRunner(nil, fExec.fakeCmdRunner(backup, nil))
			pgb.Backup("") //nolint:errcheck
			if !reflect.DeepEqual(fExec, tc.want) {
				t.Errorf("error want %v, got %v", fExec, tc.want)
			}
		})
	}
}

func TestProcessMaxEnvByAction(t *testing.T) {
	stdout := io.NopCloser(bytes.NewBufferString("stdout line\n"))
	stderr := io.NopCloser(bytes.NewBufferString(""))

	newMockCmd := func() *MockCommandExecutor {
		return &MockCommandExecutor{
			stdout:         stdout,
			stderr:         stderr,
			combinedOutput: []byte("ok"),
		}
	}

	testCases := []struct {
		name       string
		command    string
		conf       pgbackrestapi.StanzaConfiguration
		run        func(*PgBackrestRunner) error
		wantEnvVar string
	}{
		{
			name:    "backup override",
			command: "backup",
			conf: pgbackrestapi.StanzaConfiguration{
				ProcessMax:       2,
				BackupProcessMax: 4,
			},
			run: func(p *PgBackrestRunner) error {
				return p.Backup("")
			},
			wantEnvVar: "PGBACKREST_PROCESS_MAX=4",
		},
		{
			name:    "restore override",
			command: "restore",
			conf: pgbackrestapi.StanzaConfiguration{
				ProcessMax:        2,
				RestoreProcessMax: 5,
			},
			run: func(p *PgBackrestRunner) error {
				return <-p.Restore(context.Background())
			},
			wantEnvVar: "PGBACKREST_PROCESS_MAX=5",
		},
		{
			name:    "archive push override",
			command: "archive-push",
			conf: pgbackrestapi.StanzaConfiguration{
				ProcessMax:            2,
				ArchivePushProcessMax: 6,
			},
			run: func(p *PgBackrestRunner) error {
				return <-p.PushWal(context.Background(), "000000010000000000000001")
			},
			wantEnvVar: "PGBACKREST_PROCESS_MAX=6",
		},
		{
			name:    "archive get override",
			command: "archive-get",
			conf: pgbackrestapi.StanzaConfiguration{
				ProcessMax:           2,
				ArchiveGetProcessMax: 7,
			},
			run: func(p *PgBackrestRunner) error {
				return <-p.GetWAL(context.Background(), "000000010000000000000001", "/tmp/wal")
			},
			wantEnvVar: "PGBACKREST_PROCESS_MAX=7",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockCmd := newMockCmd()
			cmdRunner := func(args ...string) CommandExecutor {
				return mockCmd
			}

			pg := newPgBackrestWithRunnerAndConfig([]string{"PGBACKREST_PROCESS_MAX=2"}, &tc.conf, cmdRunner)
			if err := tc.run(pg); err != nil {
				t.Fatalf("unexpected error for %s: %v", tc.command, err)
			}

			count := 0
			for _, entry := range mockCmd.env {
				if entry == tc.wantEnvVar {
					count++
				}
				if entry == "PGBACKREST_PROCESS_MAX=2" {
					t.Fatalf("unexpected global process max left in env: %v", mockCmd.env)
				}
			}
			if count != 1 {
				t.Fatalf("expected env %q exactly once in %v", tc.wantEnvVar, mockCmd.env)
			}
		})
	}
}

// MockCommandExecutor implements CommandExecutor for testing
type MockCommandExecutor struct {
	stdout         io.ReadCloser
	stderr         io.ReadCloser
	startErr       error
	waitErr        error
	killed         bool
	env            []string
	mu             sync.Mutex
	combinedOutput []byte
	combinedErr    error
}

func (m *MockCommandExecutor) Run() error                         { return nil }
func (m *MockCommandExecutor) Start() error                       { return m.startErr }
func (m *MockCommandExecutor) StdoutPipe() (io.ReadCloser, error) { return m.stdout, nil }
func (m *MockCommandExecutor) StderrPipe() (io.ReadCloser, error) { return m.stderr, nil }
func (m *MockCommandExecutor) Wait() error                        { return m.waitErr }
func (m *MockCommandExecutor) Kill() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.killed = true
	return nil
}

func (m *MockCommandExecutor) SetEnv(env []string) { m.env = env }

func (m *MockCommandExecutor) CombinedOutput() ([]byte, error) {
	return m.combinedOutput, m.combinedErr
}
func TestRunBackgroundTask_Normal(t *testing.T) {
	// Prepare mock stdout/stderr
	stdout := io.NopCloser(bytes.NewBufferString("stdout line\n"))
	stderr := io.NopCloser(bytes.NewBufferString("stderr line\n"))

	mockCmd := &MockCommandExecutor{
		stdout: stdout,
		stderr: stderr,
	}

	cmdRunner := func(args ...string) CommandExecutor {
		return mockCmd
	}

	pg := &PgBackrestRunner{
		baseRunner: baseRunner{
			command:   "pgbackrest",
			cmdRunner: cmdRunner,
			baseEnv:   []string{"BASE=1"},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	errCh := pg.runBackgroundTask(ctx, []string{"--test"}, []string{"EXTRA=1"})

	err := <-errCh

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	foundBase := false
	foundExtra := false
	for _, v := range mockCmd.env {
		if v == "BASE=1" {
			foundBase = true
		}
		if v == "EXTRA=1" {
			foundExtra = true
		}
	}
	if !foundBase || !foundExtra {
		t.Errorf("expected BASE=1 and EXTRA=1 in env, got %v", mockCmd.env)
	}

	if mockCmd.killed {
		t.Errorf("command should not be killed on normal completion")
	}
}

func TestRunBackgroundTask_CommandFails(t *testing.T) {
	// Prepare mock stdout/stderr
	stdout := io.NopCloser(bytes.NewBufferString("stdout line\n"))
	stderr := io.NopCloser(bytes.NewBufferString("stderr line\n"))
	mockCmd := &MockCommandExecutor{
		stdout:         stdout,
		stderr:         stderr,
		combinedOutput: []byte("command failed"),
		combinedErr:    fmt.Errorf("simulated error"),
		waitErr:        fmt.Errorf("simulated error"),
	}

	// Command runner returns the mock command
	cmdRunner := func(args ...string) CommandExecutor {
		return mockCmd
	}

	pg := &PgBackrestRunner{
		baseRunner: baseRunner{
			command:   "pgbackrest",
			cmdRunner: cmdRunner,
			baseEnv:   []string{"BASE=1"},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	errCh := pg.runBackgroundTask(ctx, []string{"--test"}, []string{"EXTRA=1"})

	err := <-errCh

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !strings.Contains(err.Error(), "simulated error") {
		t.Errorf("unexpected error message: %v", err)
	}

	// Even on failure, the command should have been started
	foundBase := false
	foundExtra := false
	for _, v := range mockCmd.env {
		if v == "BASE=1" {
			foundBase = true
		}
		if v == "EXTRA=1" {
			foundExtra = true
		}
	}
	if !foundBase || !foundExtra {
		t.Errorf("expected BASE=1 and EXTRA=1 in env, got %v", mockCmd.env)
	}
}

func TestRestoreOptionToEnv(t *testing.T) {
	testCases := []struct {
		desc string
		data RestoreOptions
		want []string
		err  error
	}{
		{
			desc: "all field provided",
			data: RestoreOptions{
				Target: "2015-01-30 14:15:11 EST",
				Type:   "time",
			},
			want: []string{
				"PGBACKREST_TARGET=2015-01-30 14:15:11 EST",
				"PGBACKREST_TYPE=time",
			},
			err: nil,
		},
		{
			desc: "few fields missing",
			data: RestoreOptions{
				Type: "time",
			},
			want: []string{
				"PGBACKREST_TYPE=time",
			},
			err: nil,
		},
	}
	for _, tc := range testCases {
		f := func(t *testing.T) {

			got, err := tc.data.ToEnv()
			if err != tc.err {
				t.Errorf("error want: %v, got: %v", tc.err, err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("error want: %v, got: %v", tc.want, got)
			}
		}
		t.Run(tc.desc, f)
	}
}

func TestBackupValidation(t *testing.T) {
	testCases := []struct {
		desc       string
		backupType string
		wantErr    bool
	}{
		{"empty string is valid", "", false},
		{"full is valid", "full", false},
		{"diff is valid", "diff", false},
		{"incr is valid", "incr", false},
		{"typo ful is invalid", "ful", true},
		{"uppercase FULL is invalid", "FULL", true},
		{"incremental is invalid", "incremental", true},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			fExec := execCalls{}
			pgb := newPgBackrestWithRunner(nil, fExec.fakeCmdRunner("", nil))
			err := pgb.Backup(tc.backupType)
			if tc.wantErr && err == nil {
				t.Errorf("expected error for backup type %q, got nil", tc.backupType)
			}
			if !tc.wantErr && err != nil {
				t.Errorf("unexpected error for backup type %q: %v", tc.backupType, err)
			}
		})
	}
}

func TestCountByType(t *testing.T) {
	testCases := []struct {
		desc    string
		backups []pgbackrestapi.BackupInfo
		want    map[string]uint16
	}{
		{
			desc:    "no backup",
			backups: []pgbackrestapi.BackupInfo{},
			want:    make(map[string]uint16, 3),
		},
		{
			desc: "few full backups",
			backups: []pgbackrestapi.BackupInfo{
				{Type: "full"},
				{Type: "full"},
				{Type: "full"},
			},
			want: map[string]uint16{"full": 3},
		},
		{
			desc: "different types of backups",
			backups: []pgbackrestapi.BackupInfo{
				{Type: "full"},
				{Type: "full"},
				{Type: "full"},
				{Type: "diff"},
				{Type: "diff"},
				{Type: "incr"},
			},
			want: map[string]uint16{"full": 3, "diff": 2, "incr": 1},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {

			got := CountByType(tc.backups)
			if !maps.Equal(tc.want, got) {
				t.Errorf("count is invalid, want: %v, got: %v", tc.want, got)
			}
		})
	}
}
