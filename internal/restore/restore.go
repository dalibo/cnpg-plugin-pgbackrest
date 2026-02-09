// SPDX-FileCopyrightText: 2025 Dalibo <contact@dalibo.com>
//
// SPDX-License-Identifier: Apache-2.0
package restore

import (
	"context"
	"fmt"

	cnpgv1 "github.com/cloudnative-pg/cloudnative-pg/api/v1"
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

func recoveryTargetToRestoreOptions(
	cluster *cnpgv1.Cluster,
) pgbackrest.RestoreOptions {
	// empty slice if no bootstrap or recovery target
	b := cluster.Spec.Bootstrap
	if b == nil || b.Recovery == nil || b.Recovery.RecoveryTarget == nil {
		return pgbackrest.RestoreOptions{}
	}

	// otherwise try to convert the RecoveryTarget to
	// our own object type and envvar.
	rt := b.Recovery.RecoveryTarget
	res := pgbackrest.RestoreOptions{}

	// only one target can be used
	// TODO: handle TLI target ?
	// more information about exclusivity:
	// https://github.com/cloudnative-pg/cloudnative-pg/blob/main/api/v1/cluster_types.go#L1998
	switch {
	case rt.TargetLSN != "":
		res.Type = "lsn"
		res.Target = rt.TargetLSN
	case rt.TargetName != "":
		res.Type = "name"
		res.Target = rt.TargetName
	case rt.TargetXID != "":
		res.Type = "xid"
		res.Target = rt.TargetXID
	case rt.TargetTime != "":
		res.Type = "time"
		res.Target = rt.TargetTime
	}

	// TLI is not exclusive and can be define with other Target
	if tli := b.Recovery.RecoveryTarget.TargetTLI; tli != "" {
		res.TargetTimeline = tli
	}

	// then convert it to list of envvar
	// (should be done by the caller ?)
	return res
}

func (impl JobHookImpl) Restore(
	ctx context.Context,
	req *restore.RestoreRequest,
) (*restore.RestoreResponse, error) {
	contextLogger := log.FromContext(ctx)
	contextLogger.Info("Start restoring backup")
	stanza, err := operator.GetStanza(ctx,
		req,
		impl.Client,
		(*operator.PluginConfiguration).GetRecoveryStanzaRef,
	)
	if err != nil {
		return nil, err
	}
	env, err := operator.GetEnvVarConfig(ctx, *stanza, impl.Client)
	if err != nil {
		return nil, err
	}
	cConfig, err := operator.NewFromClusterJSON(req.ClusterDefinition)
	if err != nil {
		return nil, err
	}
	recovOption := recoveryTargetToRestoreOptions(cConfig.Cluster)
	recovEnv, err := recovOption.ToEnv()
	if err != nil {
		return nil, err
	}
	env = append(env, recovEnv...)
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
