// SPDX-FileCopyrightText: 2025 Dalibo <contact@dalibo.com>
//
// SPDX-License-Identifier: Apache-2.0

package instance

import (
	"context"
	"fmt"
	"strconv"

	"github.com/cloudnative-pg/cnpg-i/pkg/backup"
	"github.com/dalibo/cnpg-i-pgbackrest/internal/metadata"
	"github.com/dalibo/cnpg-i-pgbackrest/internal/operator"
	"github.com/dalibo/cnpg-i-pgbackrest/internal/pgbackrest"
	apipgbackrest "github.com/dalibo/cnpg-i-pgbackrest/internal/pgbackrest/api"
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

func getEnvVarBackupRepoDest(repo apipgbackrest.Repository, selectedRepo string) (string, error) {
	sRepo, err := strconv.ParseUint(selectedRepo, 10, 64)
	if err != nil {
		return "", err
	}
	if sRepo != 1 && sRepo > uint64(len(repo.S3Repositories)) {
		return "", fmt.Errorf("can't parse selected repository: %s, %w", selectedRepo, err)
	}
	return fmt.Sprintf("PGBACKREST_REPO=%d", sRepo), nil
}

func (b BackupServiceImplementation) Backup(
	ctx context.Context,
	request *backup.BackupRequest,
) (*backup.BackupResult, error) {
	contextLogger := log.FromContext(ctx)
	repo, err := operator.GetRepo(
		ctx,
		request,
		b.Client,
		(*operator.PluginConfiguration).GetRepositoryRef,
	)
	if err != nil {
		return nil, err
	}
	env, err := operator.GetEnvVarConfig(ctx, *repo, b.Client)
	if err != nil {
		contextLogger.Error(err, "can't get envvar")
		return nil, err
	}
	selectedRepo, ok := request.Parameters["selectedRepository"]
	if !ok {
		selectedRepo = "1" // use first repo by default
	}
	repoDestEnv, err := getEnvVarBackupRepoDest(repo.Spec.Configuration, selectedRepo)
	if err != nil {
		return nil, err
	}
	env = append(env, repoDestEnv)
	contextLogger.Info("using repo", "repo", repoDestEnv)
	contextLogger.Info("Starting backup")
	pgb := pgbackrest.NewPgBackrest(env)
	r, err := pgb.Backup()
	if err != nil {
		contextLogger.Error(err, "can't backup")
		return nil, err
	}
	contextLogger.Info("Backup done!")
	return &backup.BackupResult{
		BackupName: r.Label,
		BeginLsn:   r.Lsn.Start,
		BeginWal:   r.Archive.Start,
		EndLsn:     r.Lsn.Stop,
		EndWal:     r.Archive.Stop,
		Online:     true,
		StartedAt:  r.Timestamp.Start,
		StoppedAt:  r.Timestamp.Stop,
		Metadata: map[string]string{
			"version":     metadata.Data.Version,
			"name":        metadata.Data.Name,
			"displayName": metadata.Data.DisplayName,
		},
	}, nil
}
