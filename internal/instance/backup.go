// SPDX-FileCopyrightText: 2025 Dalibo <contact@dalibo.com>
//
// SPDX-License-Identifier: Apache-2.0

package instance

import (
	"context"
	"fmt"
	"strconv"

	"github.com/cloudnative-pg/cnpg-i/pkg/backup"
	apipgbackrestv1 "github.com/dalibo/cnpg-i-pgbackrest/api/v1"
	"github.com/dalibo/cnpg-i-pgbackrest/internal/metadata"
	"github.com/dalibo/cnpg-i-pgbackrest/internal/operator"
	"github.com/dalibo/cnpg-i-pgbackrest/internal/pgbackrest"
	apipgbackrest "github.com/dalibo/cnpg-i-pgbackrest/internal/pgbackrest/api"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type BackupServiceImplementation struct {
	Client       client.Client
	InstanceName string
	backup.UnimplementedBackupServer
}

func (b BackupServiceImplementation) GetCapabilities(
	_ context.Context, _ *backup.BackupCapabilitiesRequest,
) (*backup.BackupCapabilitiesResult, error) {
	return &backup.BackupCapabilitiesResult{
		Capabilities: []*backup.BackupCapability{
			{
				Type: &backup.BackupCapability_Rpc{
					Rpc: &backup.BackupCapability_RPC{
						Type: backup.BackupCapability_RPC_TYPE_BACKUP,
					},
				},
			},
		},
	}, nil
}

func getEnvVarBackupRepoDest(repo apipgbackrest.Stanza, selectedRepo string) (string, error) {
	sRepo, err := strconv.ParseUint(selectedRepo, 10, 64)
	if err != nil {
		return "", err
	}
	if sRepo != 1 && sRepo > uint64(len(repo.S3Repositories)) {
		return "", fmt.Errorf("can't parse selected repository: %s, %w", selectedRepo, err)
	}
	return fmt.Sprintf("PGBACKREST_REPO=%d", sRepo), nil
}

func updateBackupInfo(
	ctx context.Context,
	c client.Client,
	stanza *apipgbackrestv1.Stanza,
	firstBackup apipgbackrest.BackupInfo,
	lastBackup apipgbackrest.BackupInfo,
) error {
	return retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		stanza.Status.RecoveryWindow.FirstBackup = firstBackup
		stanza.Status.RecoveryWindow.LastBackup = lastBackup
		return c.Status().Update(ctx, stanza)
	})
}

func (b BackupServiceImplementation) Backup(
	ctx context.Context,
	request *backup.BackupRequest,
) (*backup.BackupResult, error) {
	contextLogger := log.FromContext(ctx)
	stanza, err := operator.GetStanza(
		ctx,
		request,
		b.Client,
		(*operator.PluginConfiguration).GetStanzaRef,
	)
	if err != nil {
		return nil, err
	}
	env, err := operator.GetEnvVarConfig(ctx, *stanza, b.Client)
	if err != nil {
		contextLogger.Error(err, "can't get envvar")
		return nil, err
	}
	selectedRepo, ok := request.Parameters["selectedRepository"]
	if !ok {
		selectedRepo = "1" // use first stanza by default
	}
	repoDestEnv, err := getEnvVarBackupRepoDest(stanza.Spec.Configuration, selectedRepo)
	if err != nil {
		return nil, err
	}
	env = append(env, repoDestEnv)
	contextLogger.Info("using repo", "repo", repoDestEnv)
	backupType := request.Parameters["backupType"]
	contextLogger.Info("Starting backup", "type", backupType)
	pgb := pgbackrest.NewPgBackrest(env)
	if err := pgb.Backup(backupType); err != nil {
		contextLogger.Error(err, "can't backup")
		return nil, err
	}
	backupsList, err := pgb.GetBackupInfo()
	if err != nil {
		return nil, err
	}
	lastBackup := pgbackrest.LatestBackup(backupsList)
	firstBackup := pgbackrest.FirstBackup(backupsList)
	if err := updateBackupInfo(ctx, b.Client, stanza, *firstBackup, *lastBackup); err != nil {
		contextLogger.Error(err, "can't update backup info")
		return nil, err
	}
	contextLogger.Info("Backup done!")
	return &backup.BackupResult{
		BackupName: lastBackup.Label,
		BeginLsn:   lastBackup.Lsn.Start,
		BeginWal:   lastBackup.Archive.Start,
		EndLsn:     lastBackup.Lsn.Stop,
		EndWal:     lastBackup.Archive.Stop,
		Online:     true,
		StartedAt:  lastBackup.Timestamp.Start,
		StoppedAt:  lastBackup.Timestamp.Stop,
		Metadata: map[string]string{
			"version":     metadata.Data.Version,
			"name":        metadata.Data.Name,
			"displayName": metadata.Data.DisplayName,
		},
	}, nil
}
