// SPDX-FileCopyrightText: 2025 Dalibo <contact@dalibo.com>
//
// SPDX-License-Identifier: Apache-2.0

package instance

import (
	"context"

	"github.com/cloudnative-pg/cnpg-i-machinery/pkg/pluginhelper/http"
	"github.com/cloudnative-pg/cnpg-i/pkg/backup"
	"github.com/cloudnative-pg/cnpg-i/pkg/wal"
	wal_pgbackrest "github.com/dalibo/cnpg-i-pgbackrest/internal/wal"
	"google.golang.org/grpc"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// PgbackrestPlugin is the implementation of the PostgreSQL sidecar
type PgbackrestPluginServer struct {
	Client     client.Client
	PGDataPath string
	PGWALPath  string
	// mutually exclusive with serverAddress
	PluginPath   string
	InstanceName string
}

// Start starts the GRPC service
func (c *PgbackrestPluginServer) Start(ctx context.Context) error {
	enrich := func(server *grpc.Server) error {
		wal.RegisterWALServer(server, &wal_pgbackrest.WALSrvImplementation{
			InstanceName: c.InstanceName,
			Client:       c.Client,
			PGDataPath:   c.PGDataPath,
			PGWALPath:    c.PGWALPath,
		})
		backup.RegisterBackupServer(server, BackupServiceImplementation{
			Client:       c.Client,
			InstanceName: c.InstanceName,
		})
		return nil
	}

	srv := http.Server{
		IdentityImpl: IdentityImplementation{
			Client: c.Client,
		},
		Enrichers:  []http.ServerEnricher{enrich},
		PluginPath: c.PluginPath,
	}

	return srv.Start(ctx)
}
