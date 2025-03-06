package pgbackrest

import (
	"encoding/json"
	"fmt"
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
		{"code ok", `[{"repo": [{"status": {"code": 0, "message": "OK" }}]}]`, true},
		{"code missing", `[{"repo": [{"status": {"message": "Machin" }}]}]`, false},
		{"code error", `[{"repo": [{"status": {"code": 2, "message": "BLA" }}]}]`, false},
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
