// SPDX-FileCopyrightText: 2025 Dalibo <contact@dalibo.com>
//
// SPDX-License-Identifier: Apache-2.0

package wal

import (
	"context"
	"fmt"
	"os"

	"github.com/cloudnative-pg/cnpg-i/pkg/wal"
	"github.com/dalibo/cnpg-i-pgbackrest/internal/pgbackrest"
	"github.com/dalibo/cnpg-i-pgbackrest/internal/utils"
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
			///archive first, then we will see how to restore
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

	stanzaName, stanzaEnvVarDefined := os.LookupEnv("PGBACKREST_stanza")
	if !stanzaEnvVarDefined {
		return nil, fmt.Errorf("stanza env var not found")
	}
	created, err := pgbackrest.EnsureStanzaExists(stanzaName, utils.RealCmdRunner)
	if err != nil {
		return nil, fmt.Errorf("stanza verification failed stanza: %s error: %w", stanzaName, err)
	}
	if created {
		contextLogger.Info("stanza created while archiving", "WAL", walName, "stanza", stanzaName)
	}
	_, err = pgbackrest.PushWal(walName, utils.RealCmdRunner)
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

	logger.Info("Starting WAL restore via pgBackRest",
		"walName", walName,
		"destinationPath", destinationPath,
	)

	_, err := pgbackrest.GetWAL(walName, destinationPath, utils.RealCmdRunner)
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
