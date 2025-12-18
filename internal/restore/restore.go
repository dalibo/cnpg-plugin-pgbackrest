// SPDX-FileCopyrightText: 2025 Dalibo <contact@dalibo.com>
//
// SPDX-License-Identifier: Apache-2.0
package restore

import (
	"context"
	"fmt"

	"github.com/cloudnative-pg/cloudnative-pg/pkg/postgres"
	restore "github.com/cloudnative-pg/cnpg-i/pkg/restore/job"
	"github.com/dalibo/cnpg-i-pgbackrest/internal/operator"
	"github.com/dalibo/cnpg-i-pgbackrest/internal/pgbackrest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type JobHookImpl struct {
	restore.UnimplementedRestoreJobHooksServer

	Client client.Client

	PgDataPath           string
	PgWalFolderToSymlink string
}

// GetCapabilities returns the capabilities of the restore job hooks
func (impl JobHookImpl) GetCapabilities(
	_ context.Context,
	_ *restore.RestoreJobHooksCapabilitiesRequest,
) (*restore.RestoreJobHooksCapabilitiesResult, error) {
	return &restore.RestoreJobHooksCapabilitiesResult{
		Capabilities: []*restore.RestoreJobHooksCapability{
			{
				Kind: restore.RestoreJobHooksCapability_KIND_RESTORE,
			},
		},
	}, nil
}

func (impl JobHookImpl) Restore(
	ctx context.Context,
	req *restore.RestoreRequest,
) (*restore.RestoreResponse, error) {
	contextLogger := log.FromContext(ctx)
	contextLogger.Info("Start restoring backup")
	r, err := operator.GetStanza(ctx,
		req,
		impl.Client,
		(*operator.PluginConfiguration).GetRecoveryStanzaRef,
	)
	if err != nil {
		return nil, err
	}
	env, err := operator.GetEnvVarConfig(ctx, *r, impl.Client)
	if err != nil {
		return nil, err
	}
	pgb := pgbackrest.NewPgBackrest(env)
	errCh := pgb.Restore(ctx)
	if err := <-errCh; err != nil {
		return nil, err
	}
	restoreCmd := fmt.Sprintf(
		"/controller/manager wal-restore --log-destination %s/%s.json %%f %%p",
		postgres.LogPath, postgres.LogFileName)
	config := fmt.Sprintf(
		"recovery_target_action = promote\n"+
			"restore_command = '%s'\n",
		restoreCmd)
	contextLogger.Info("Finished restoring backup, sending response", "config", config)
	return &restore.RestoreResponse{
		RestoreConfig: config,
		Envs:          nil,
	}, nil
}
