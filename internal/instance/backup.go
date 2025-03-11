package instance

import (
	"context"

	"github.com/cloudnative-pg/cnpg-i/pkg/backup"
	"github.com/cloudnative-pg/machinery/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/dalibo/cnpg-i-pgbackrest/internal/metadata"
	"github.com/dalibo/cnpg-i-pgbackrest/internal/pgbackrest"
	"github.com/dalibo/cnpg-i-pgbackrest/internal/utils"
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

func (b BackupServiceImplementation) Backup(
	ctx context.Context,
	request *backup.BackupRequest,
) (*backup.BackupResult, error) {
	contextLogger := log.FromContext(ctx)

	contextLogger.Info("Starting backup")
	lockFile := "/tmp/pgbackrest-cnpg-plugin.lock"
	r, err := pgbackrest.Backup(&lockFile, utils.RealCmdRunner)
	if err != nil {
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
