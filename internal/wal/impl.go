package wal

import (
	"context"
	"fmt"
	"os"

	"github.com/cloudnative-pg/cnpg-i/pkg/wal"
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
			//			{
			//				Type: &wal.WALCapability_Rpc{
			//					Rpc: &wal.WALCapability_RPC{
			//						Type: wal.WALCapability_RPC_TYPE_RESTORE_WAL,
			//					},
			//				},
			//			},
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
	created, err := pgbackrest.EnsureStanzaExists(stanzaName)
	if err != nil {
		return nil, fmt.Errorf("stanza verification failed stanza: %s error: %w", stanzaName, err)
	}
	if created {
		contextLogger.Info("stanza created while archiving", "WAL", walName, "stanza", stanzaName)
	}
	err = pgbackrest.PushWal(walName)
	if err != nil {
		return nil, fmt.Errorf("pgBackRest archive-push failed: %w", err)
	}
	contextLogger.Info("pgBackRest archive-push successful", "WAL", walName)
	return &wal.WALArchiveResult{}, nil
}

// Not yet implemented
func (WALSrvImplementation) Restore(
	ctx context.Context,
	request *wal.WALRestoreRequest,
) (*wal.WALRestoreResult, error) {

	contextLogger := log.FromContext(ctx)
	contextLogger.Info("Restoring WAL...")
	panic("implement me")
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
