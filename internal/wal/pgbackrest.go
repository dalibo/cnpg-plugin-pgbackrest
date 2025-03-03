package wal

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"

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
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type Repo struct {
	Status RepoStatus `json:"status"`
}

type PgBackRestInfo struct {
	Repo []Repo `json:"repo"`
}

func stanzaExists(stanza string) (bool, error) {
	cmd := exec.Command("pgbackrest", "info", "--stanza="+stanza, "--output=json")
	stdout, err := cmd.CombinedOutput()
	if err != nil {
		return false, fmt.Errorf("can't execute pgbackrest info command: %v", err)
	}
	var info []PgBackRestInfo
	if err := json.Unmarshal(stdout, &info); err != nil {
		return false, fmt.Errorf("Error parsing pgbackrest JSON: %v", err)
	}
	for _, entry := range info {
		for _, repo := range entry.Repo {
			if repo.Status.Code == 0 {
				return true, nil
			}
		}
	}
	return false, nil
}

func ensureStanzaExists(ctx context.Context, stanza string) error {
	contextLogger := log.FromContext(ctx)
	stanzaExist, err := stanzaExists(stanza)
	if err != nil {
		contextLogger.Info("pgBackRest stanza verification fails", "error", string(err.Error()))
		return err
	}

	if stanzaExist {
		contextLogger.Info("pgBackRest stanza already exists", "stanza", stanza)
		return nil
	}

	cmd := exec.Command("pgbackrest", "stanza-create", "--stanza="+stanza)

	output, err := cmd.CombinedOutput()
	if err != nil {
		contextLogger.Error(err, "can't create stanza", "output", string(output))
		return fmt.Errorf("can't create stanza: %v, error : %s", err, output)
	}

	return nil
}

func pushWal(ctx context.Context, walName string) error {
	contextLogger := log.FromContext(ctx)
	cmd := exec.Command("pgbackrest", "archive-push", walName)
	// add envvar here
	output, err := cmd.CombinedOutput()
	if err != nil {
		contextLogger.Error(err, "pgBackRest archive-push failed", "output", string(output))
		return fmt.Errorf("pgBackRest archive-push failed: %w", err)
	}
	return nil
}

func Backup(ctx context.Context) (*BackupInfo, error) {
	contextLogger := log.FromContext(ctx)
	cmd := exec.Command("pgbackrest", "backup")
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env,
		"PGBACKREST_archive-check=n",
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("can't create stanza: %v, error : %s", err, string(output))
	}
	backups, err := GetBackupInfo()
	if err != nil {
		return nil, err
	}
	latestBackup := LatestBackup(backups)
	contextLogger.Info("Backup done!", "backup command output: %s", string(output))
	return latestBackup, nil
}

func GetBackupInfo() ([]BackupInfo, error) {
	cmd := exec.Command("pgbackrest", "info", "--output", "json")
	o, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}
	var pgbackrestInfo []BackupData
	err = json.Unmarshal(o, &pgbackrestInfo)
	if err != nil {
		return nil, err
	}
	return pgbackrestInfo[0].Backup, nil

}

func LatestBackup(backups []BackupInfo) *BackupInfo {
	found := backups[0]
	for _, backup := range backups {
		if backup.Timestamp.Stop > found.Timestamp.Stop {
			found = backup
		}
	}
	return &found
}
