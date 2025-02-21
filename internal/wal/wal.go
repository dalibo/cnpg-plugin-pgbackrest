package wal

import (
	"context"
	"fmt"
	"os"

	"github.com/cloudnative-pg/cnpg-i/pkg/wal"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// Archive copies one WAL file into the archive
func (w_impl WALSrvImplementation) Archive(
	ctx context.Context,
	request *wal.WALArchiveRequest,
) (*wal.WALArchiveResult, error) {
	contextLogger := log.FromContext(ctx)
	walName := request.GetSourceFileName()

	stanza, stanza_defined := os.LookupEnv("PGBACKREST_stanza")
	// ensure stanza exists before archiving, should be done earlier
	if !stanza_defined {
		return nil, fmt.Errorf("Stanza env var not found")
	}
	err := ensureStanzaExists(ctx, stanza)
	if err != nil {
		contextLogger.Info("error when ensuring stanza exists")
		return nil, fmt.Errorf("Stanza verification failed")
	}
	err = pushWal(ctx, walName)
	if err != nil {
		return nil, fmt.Errorf("pgBackRest archive-push failed: %w", err)
	}
	contextLogger.Info("pgBackRest archive-push successful")
	return &wal.WALArchiveResult{}, nil

}
func (WALSrvImplementation) Restore(
	ctx context.Context,
	request *wal.WALRestoreRequest,
) (*wal.WALRestoreResult, error) {

	contextLogger := log.FromContext(ctx)
	contextLogger.Info("Restoring WAL...")
	return &wal.WALRestoreResult{}, nil
}
