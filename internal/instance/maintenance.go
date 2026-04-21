// SPDX-FileCopyrightText: 2026 Dalibo <contact@dalibo.com>
//
// SPDX-License-Identifier: Apache-2.0

package instance

import (
	"context"
	"time"

	cnpgv1 "github.com/cloudnative-pg/cloudnative-pg/api/v1"
	"github.com/cloudnative-pg/machinery/pkg/log"
	pgbackrestapi "github.com/dalibo/cnpg-i-pgbackrest/api/v1"
	"github.com/dalibo/cnpg-i-pgbackrest/internal/operator"
	"github.com/dalibo/cnpg-i-pgbackrest/internal/pgbackrest"
	"github.com/dalibo/cnpg-i-pgbackrest/internal/utils"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const defaultRetentionPolicyInterval = time.Minute * 5

// StanzaMaintenanceRunnable executes all the pgbackrest
// stanza maintenance operations
type StanzaMaintenanceRunnable struct {
	Client         client.Client
	ClusterKey     types.NamespacedName
	CurrentPodName string
}

func (c *StanzaMaintenanceRunnable) Start(ctx context.Context) error {
	contextLogger := log.FromContext(ctx)
	contextLogger.Info("starting stanza maintenance runnable")

	for {
		err := c.cycle(ctx)
		if err != nil {
			contextLogger.Error(err, "stanza maintenance failed")
		}

		select {
		case <-time.After(defaultRetentionPolicyInterval):
		case <-ctx.Done():
			return nil
		}
	}
}

func (c *StanzaMaintenanceRunnable) cycle(ctx context.Context) error {
	contextLogger := log.FromContext(ctx)
	contextLogger.Info("running maintenance cycle")

	var cluster cnpgv1.Cluster
	if err := c.Client.Get(ctx, c.ClusterKey, &cluster); err != nil {
		return err
	}
	// check if plugin is enabled
	if !utils.IsPluginEnabled(&cluster) {
		contextLogger.Debug("skipping maintenance: plugin is not enabled for backups")
		return nil
	}
	stz, err := operator.GetStanzaFromCluster(
		ctx,
		&cluster,
		c.Client,
		(*operator.PluginConfiguration).GetStanzaRef,
	)
	if err != nil {
		return err
	}

	// execute maintenance on it
	if err := c.maintenance(ctx, &cluster, stz); err != nil {
		return err
	}

	return nil
}

func (c *StanzaMaintenanceRunnable) maintenance(
	ctx context.Context,
	cluster *cnpgv1.Cluster,
	stanza *pgbackrestapi.Stanza,
) error {
	contextLogger := log.FromContext(ctx)

	if cluster.Status.CurrentPrimary != c.CurrentPodName {
		contextLogger.Info(
			"skipping maintenance, not the current primary",
			"currentPrimary", cluster.Status.CurrentPrimary, "podName", c.CurrentPodName)
		return nil
	}

	backups, err := c.getBackupsInfo(ctx, stanza)
	if err != nil {
		return err
	}

	if err := c.updateBackupWindow(ctx, backups, stanza); err != nil {
		return err
	}

	return nil
}

func (c *StanzaMaintenanceRunnable) getBackupsInfo(
	ctx context.Context,
	stanza *pgbackrestapi.Stanza,
) ([]pgbackrestapi.BackupInfo, error) {
	env, err := operator.GetEnvVarConfig(ctx, *stanza, c.Client)
	if err != nil {
		return nil, err
	}
	pgbExec := pgbackrest.NewPgBackrest(env)
	return pgbExec.GetBackupInfo()
}

func (c *StanzaMaintenanceRunnable) updateBackupWindow(
	ctx context.Context,
	backups []pgbackrestapi.BackupInfo,
	stanza *pgbackrestapi.Stanza,
) error {
	l := pgbackrest.LatestBackup(backups)
	f := pgbackrest.FirstBackup(backups)
	return updateBackupInfo(ctx, c.Client, stanza, f, l)
}
