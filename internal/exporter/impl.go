// SPDX-FileCopyrightText: 2026 Dalibo <contact@dalibo.com>
//
// SPDX-License-Identifier: Apache-2.0

package exporter

import (
	"context"

	cnpgv1 "github.com/cloudnative-pg/cloudnative-pg/api/v1"
	apipgbackrest "github.com/dalibo/cnpg-i-pgbackrest/api/v1"
	"github.com/dalibo/cnpg-i-pgbackrest/internal/config"
	"github.com/dalibo/cnpg-i-pgbackrest/internal/pgbackrest"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type PgbackrestSidecarServer struct {
	Client       client.Client
	InstanceName string
	Namespace    string
	ClusterName  string
}

func (p *PgbackrestSidecarServer) Start(ctx context.Context) error {
	contextLogger := log.FromContext(ctx).WithName("exporter")
	contextLogger.Info("starting exporter")
	// retrieve cluster
	clusterName := types.NamespacedName{
		Name:      p.ClusterName,
		Namespace: p.Namespace,
	}
	cluster := &cnpgv1.Cluster{}
	if err := p.Client.Get(ctx, clusterName, cluster); err != nil {
		return err
	}

	// retrieve stanza / pgbackrest configuration
	pluginConfig, err := config.NewFromCluster(cluster)
	if err != nil {
		return err
	}

	stanza, err := config.GetStanzaFromCluster(
		ctx,
		cluster,
		p.Client,
		(*config.PluginConfiguration).GetStanzaRef,
	)
	if err != nil {
		return err
	}

	// retrieve shared plugin config
	sharedPluginConfigName, err := pluginConfig.GetSharedPluginConfig()
	if err != nil {
		return err
	}
	sharedPluginConfig := &apipgbackrest.PluginConfig{}
	if err := p.Client.Get(ctx, *sharedPluginConfigName, sharedPluginConfig); err != nil {
		return err
	}

	// then launch the exporter
	env, err := config.GetEnvVarConfig(ctx, stanza, p.Client)
	if err != nil {
		return err
	}
	pgbaRunner := pgbackrest.NewPgBackrestExporterRunner(env)
	var args []string
	if ec := sharedPluginConfig.Spec.ExporterConfig; ec != nil {
		args = ec.ToArgs()
	}
	err = pgbaRunner.RunExporter(ctx, args)
	return err
}
