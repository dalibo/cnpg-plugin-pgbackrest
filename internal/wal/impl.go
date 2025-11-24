// SPDX-FileCopyrightText: 2025 Dalibo <contact@dalibo.com>
//
// SPDX-License-Identifier: Apache-2.0

package wal

import (
	"context"
	"fmt"

	"github.com/cloudnative-pg/cnpg-i/pkg/wal"
	"github.com/dalibo/cnpg-i-pgbackrest/internal/operator"
	"github.com/dalibo/cnpg-i-pgbackrest/internal/pgbackrest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type WALSrvImplementation struct {
	wal.UnimplementedWALServer
	Client         client.Client
	PGDataPath     string
	PGWALPath      string
	SpoolDirectory string
	// mutually exclusive with serverAddress
	PluginPath   string
	InstanceName string
}

// GetCapabilities gets the capabilities of the WAL service
func (WALSrvImplementation) GetCapabilities(
	context.Context,
	*wal.WALCapabilitiesRequest,
) (*wal.WALCapabilitiesResult, error) {
	return &wal.WALCapabilitiesResult{
		Capabilities: []*wal.WALCapability{
			{
				Type: &wal.WALCapability_Rpc{
					Rpc: &wal.WALCapability_RPC{
						Type: wal.WALCapability_RPC_TYPE_ARCHIVE_WAL,
					},
				},
			},
			// archive first, then we will see how to restore
			{
				Type: &wal.WALCapability_Rpc{
					Rpc: &wal.WALCapability_RPC{
						Type: wal.WALCapability_RPC_TYPE_RESTORE_WAL,
					},
				},
			},
		},
	}, nil
}

// Archive WAL through pgbackrest (currently via a S3 stanza)
func (w_impl WALSrvImplementation) Archive(
	ctx context.Context,
	request *wal.WALArchiveRequest,
) (*wal.WALArchiveResult, error) {
	contextLogger := log.FromContext(ctx)
	walName := request.GetSourceFileName()
	repo, err := operator.GetRepo(ctx,
		request,
		w_impl.Client,
		(*operator.PluginConfiguration).GetRepositoryRef,
	)
	if err != nil {
		return nil, err
	}
	env, err := operator.GetEnvVarConfig(ctx, *repo, w_impl.Client)
	if err != nil {
		return nil, err
	}
	pgb := pgbackrest.NewPgBackrest(env)
	created, err := pgb.EnsureStanzaExists(repo.Spec.Configuration.Stanza)
	if err != nil {
		return nil, fmt.Errorf("stanza verification failed stanza, error: %w", err)
	}
	if created {
		contextLogger.Info("stanza created while archiving", "WAL", walName)
	}
	_, err = pgb.PushWal(walName)
	if err != nil {
		return nil, fmt.Errorf("pgBackRest archive-push failed: %w", err)
	}
	contextLogger.Info("pgBackRest archive-push successful", "WAL", walName)
	return &wal.WALArchiveResult{}, nil
}

func (w WALSrvImplementation) Restore(
	ctx context.Context,
	request *wal.WALRestoreRequest,
) (*wal.WALRestoreResult, error) {
	logger := log.FromContext(ctx)

	walName := request.GetSourceWalName()
	destinationPath := request.GetDestinationFileName()

	repo, err := operator.GetRepo(ctx,
		request,
		w.Client,
		(*operator.PluginConfiguration).GetRecoveryRepositoryRef,
	)
	if err != nil {
		return nil, err
	}
	env, err := operator.GetEnvVarConfig(ctx, *repo, w.Client)
	if err != nil {
		return nil, err
	}
	logger.Info("Starting WAL restore via pgBackRest",
		"walName", walName,
		"destinationPath", destinationPath,
	)

	pgb := pgbackrest.NewPgBackrest(env)
	_, err = pgb.GetWAL(walName, destinationPath)
	if err != nil {
		return nil, fmt.Errorf("getting archive failed: %w", err)
	}

	logger.Info("Successfully restored WAL via pgBackRest",
		"walName",
		walName,
		"destinationPath",
		destinationPath)

	return &wal.WALRestoreResult{}, nil
}

func (WALSrvImplementation) SetFirstRequired(
	_ context.Context,
	_ *wal.SetFirstRequiredRequest,
) (*wal.SetFirstRequiredResult, error) {
	// TODO ask what the purpose of that method
	panic("implement me")
}

func (WALSrvImplementation) Status(
	_ context.Context,
	_ *wal.WALStatusRequest,
) (*wal.WALStatusResult, error) {
	// TODO ask what the purpose of that method
	panic("implement me")
}
