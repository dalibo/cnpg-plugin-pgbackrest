package wal

import (
	"context"

	"github.com/cloudnative-pg/cnpg-i/pkg/wal"
	"sigs.k8s.io/controller-runtime/pkg/client"
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
