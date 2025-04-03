// SPDX-FileCopyrightText: 2025 Dalibo <contact@dalibo.com>
//
// SPDX-License-Identifier: Apache-2.0

package pgbackrest

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
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

func StanzaExists(stanzaName string, cmdRunner CmdRunner) (bool, error) {
	cmd := cmdRunner("pgbackrest", "info", "--stanza="+stanzaName, "--output=json")
	stdout, err := cmd.CombinedOutput()
	if err != nil {
		return false, fmt.Errorf("can't execute pgbackrest info command: %w", err)
	}
	var info []PgBackRestInfo
	if err := json.Unmarshal(stdout, &info); err != nil {
		return false, fmt.Errorf("Error parsing pgbackrest JSON: %w", err)
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

func EnsureStanzaExists(stanzaName string, cmdRunner CmdRunner) (bool, error) {
	stanzaExist, err := StanzaExists(stanzaName, cmdRunner)
	if err != nil {
		return false, fmt.Errorf("can't determine if stanza: %s exists, error %w", stanzaName, err)
	}
	if stanzaExist {
		return false, nil
	}
	cmd := cmdRunner("pgbackrest", "stanza-create", "--stanza="+stanzaName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false, fmt.Errorf("can't create stanza: %s, stdout: %s, error : %w", stanzaName, string(output), err)
	}
	return true, nil
}

func PushWal(walName string, cmdRunner CmdRunner) (string, error) {
	cmd := cmdRunner("pgbackrest", "archive-push", walName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("pgBackRest archive-push failed, output: %s %w", string(output), err)
	}
	return string(output), nil
}

func Backup(lockFile *string, cmdRunner CmdRunner) (*BackupInfo, error) {
	cmd := cmdRunner("pgbackrest", "backup")
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env,
		"PGBACKREST_archive-check=n",
	)
	if lockFile != nil && (*lockFile) != "" {
		cmd.Env = append(cmd.Env, "PGBACKREST_lock-path="+(*lockFile))
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("can't backup: %s, error : %w", string(output), err)
	}
	backups, err := GetBackupInfo(cmdRunner)
	if err != nil {
		return nil, err
	}
	latestBackup := LatestBackup(backups)
	return latestBackup, nil
}

func GetBackupInfo(cmdRunner CmdRunner) ([]BackupInfo, error) {
	cmd := cmdRunner("pgbackrest", "info", "--output", "json")
	cmdOutput, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("can't get pgbackrest info: %s, %w", string(cmdOutput), err)
	}
	var pgbackrestInfo []BackupData
	err = json.Unmarshal(cmdOutput, &pgbackrestInfo)
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
