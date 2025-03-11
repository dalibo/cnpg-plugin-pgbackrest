package pgbackrest

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"reflect"
	"testing"
)

var backupInfo = []BackupInfo{
	{
		Archive: Archive{"000000010000000000000001", "000000010000000000000002"},
		Label:   "backup_20250306", Lsn: Lsn{"0/16B2D80", "0/16B2E00"},
		Prior: "backup_20250305", Timestamp: Timestamp{1710000000, 1710003600},
		Type: "full"},
}

func TestLatestBackup(t *testing.T) {
	backupsInfoTwo := append(backupInfo, BackupInfo{
		Archive: Archive{"000000010000000000000003", "000000010000000000000004"},
		Label:   "backup_20250307", Lsn: Lsn{"0/16C2D80", "0/16C2E00"},
		Prior: "backup_20250306", Timestamp: Timestamp{1810007200, 1810010800},
		Type: "incremental",
	})
	backupsInfoThree := append(backupsInfoTwo, BackupInfo{
		Archive: Archive{"000000010000000000000003", "000000010000000000000004"},
		Label:   "backup_20250307", Lsn: Lsn{"0/16C2D80", "0/16C2E00"},
		Prior: "backup_20250306", Timestamp: Timestamp{1910007200, 1910010800},
		Type: "incremental",
	})
	type TestCase struct {
		desc string
		data []BackupInfo
		want *BackupInfo
	}
	testCases := []TestCase{
		{"No backup", []BackupInfo{}, nil},
		{"Only one backup", backupInfo, &backupInfo[0]},
		{"Three backups", backupsInfoThree, &backupsInfoThree[2]},
		{"Two backups", backupsInfoTwo, &backupsInfoTwo[1]},
	}
	for _, tc := range testCases {
		f := func(t *testing.T) {
			got := LatestBackup(tc.data)
			if (tc.want == nil && got != tc.want) || (got != nil && tc.want != nil && *got != *tc.want) {
				t.Errorf("error %v\n%v", got, tc.data)
			}

		}
		t.Run(tc.desc, f)
	}
}

func TestParseDataForStanzaStatusCode(t *testing.T) {
	type TestCase struct {
		desc string
		data string
		want bool
	}
	testCases := []TestCase{
		{"code error", `[{"repo": [{"status": {"code": 2, "message": "BLA" }}]}]`, false},
		{"code missing", `[{"repo": [{"status": {"message": "Machin" }}]}]`, false},
		{"code ok", `[{"repo": [{"status": {"code": 0, "message": "OK" }}]}]`, true},
		{"empty repo", `[{"repo": []}]`, false},
		{"empty list", `[]`, false},
	}
	for _, tc := range testCases {
		f := func(t *testing.T) {
			var pgbackrestInfo []PgBackRestInfo
			err := json.Unmarshal([]byte(tc.data), &pgbackrestInfo)
			if err != nil {
				fmt.Println(err)
				panic("should not happen")
			}
			got := parseDataForStatusCode(pgbackrestInfo)
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
	return func(name string, args ...string) *exec.Cmd {
		e.execCalls = append(e.execCalls, fakeExec{cmdName: name, args: args})
		// Fake the command execution by returning a function that provides predefined output
		cmd := exec.Command("echo", output) // Fake command that outputs JSON
		if err != nil {
			return exec.Command("false") // Simulate failure
		}
		return cmd
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
				execCalls: []fakeExec{{cmdName: "pgbackrest", args: []string{"archive-push", "/machin"}}},
			},
		},
	}

	fakeExecCalls := execCalls{}
	backup := "" // we don't care about output here
	for _, tc := range testCases {
		f := func(t *testing.T) {
			PushWal(tc.walPath, fakeExecCalls.fakeCmdRunner(backup, nil))
			if !reflect.DeepEqual(fakeExecCalls, tc.want) {
				t.Errorf("error want %v, got %v", fakeExecCalls, tc.want)
			}
		}
		t.Run(tc.desc, f)
	}
}
