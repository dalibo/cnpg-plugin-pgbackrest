// SPDX-FileCopyrightText: 2025 Dalibo <contact@dalibo.com>
//
// SPDX-License-Identifier: Apache-2.0

package wal

import (
	"context"
	"fmt"

	"github.com/cloudnative-pg/cnpg-i/pkg/wal"
	apipgbackrest "github.com/dalibo/cnpg-i-pgbackrest/api/v1"
	"github.com/dalibo/cnpg-i-pgbackrest/internal/operator"
	"github.com/dalibo/cnpg-i-pgbackrest/internal/pgbackrest"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type WALSrvImplementation struct {
	wal.UnimplementedWALServer
	Client     client.Client
	PGDataPath string
	PGWALPath  string
	// mutually exclusive with serverAddress
	PluginPath    string
	InstanceName  string
	StanzaCreated bool
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
func (w_impl *WALSrvImplementation) Archive(
	ctx context.Context,
	request *wal.WALArchiveRequest,
) (*wal.WALArchiveResult, error) {
	contextLogger := log.FromContext(ctx)
	walName := request.GetSourceFileName()
	stanza, err := operator.GetStanza(ctx,
		request,
		w_impl.Client,
		(*operator.PluginConfiguration).GetStanzaRef,
	)
	if err != nil {
		return nil, err
	}
	env, err := operator.GetEnvVarConfig(ctx, *stanza, w_impl.Client)
	if err != nil {
		return nil, err
	}
	pgb := pgbackrest.NewPgBackrest(env)
	if !w_impl.StanzaCreated {
		ok, err := pgb.EnsureStanzaExists(stanza.Spec.Configuration.Name)
		if err != nil {
			return nil, fmt.Errorf("stanza creation failed: %w", err)
		}
		if ok {
			w_impl.StanzaCreated = ok
			contextLogger.Info("stanza created while archiving", "WAL", walName)
		}
	} else {
		contextLogger.Info("stanza already exists, let's archive", "WAL", walName)
	}
	errCh := pgb.PushWal(context.Background(), walName)
	if err := <-errCh; err != nil {
		return nil, err
	}
	contextLogger.Info("pgBackRest archive-push successful", "WAL", walName)
	return &wal.WALArchiveResult{}, nil
}

func (w WALSrvImplementation) Restore(
	ctx context.Context,
	request *wal.WALRestoreRequest,
) (*wal.WALRestoreResult, error) {
	logger := log.FromContext(ctx)
	conf, err := operator.NewFromClusterJSON(request.ClusterDefinition)
	if err != nil {
		return nil, err
	}
	walName := request.GetSourceWalName()
	dstPath := request.GetDestinationFileName()

	var promotionToken string
	if conf.Cluster.Spec.ReplicaCluster != nil {
		promotionToken = conf.Cluster.Spec.ReplicaCluster.PromotionToken
	}

	var stanza *apipgbackrest.Stanza
	var getStanzaRef func(*operator.PluginConfiguration) (*types.NamespacedName, error)
	switch {

	case promotionToken != "" && conf.Cluster.Status.LastPromotionToken != promotionToken:
		getStanzaRef = func(pc *operator.PluginConfiguration) (*types.NamespacedName, error) {
			return pc.GetReplicaStanzaRef()
		}

	case conf.Cluster.IsReplica() && conf.Cluster.Status.CurrentPrimary == w.InstanceName:
		getStanzaRef = func(pc *operator.PluginConfiguration) (*types.NamespacedName, error) {
			return pc.GetReplicaStanzaRef()
		}

	case conf.Cluster.Status.CurrentPrimary == "":
		getStanzaRef = func(pc *operator.PluginConfiguration) (*types.NamespacedName, error) {
			return pc.GetRecoveryStanzaRef()
		}
	}
	if getStanzaRef == nil {
		return nil, fmt.Errorf("recovery not configured")
	}
	stanza, err = operator.GetStanza(
		ctx,
		request,
		w.Client,
		getStanzaRef,
	)
	if err != nil {
		return nil, err
	}
	env, err := operator.GetEnvVarConfig(ctx, *stanza, w.Client)
	if err != nil {
		return nil, err
	}
	logger.Info("Restoring WAL", "WAL", walName, "destination", dstPath)

	pgb := pgbackrest.NewPgBackrest(env)
	errCh := pgb.GetWAL(ctx, walName, dstPath)
	if err := <-errCh; err != nil {
		return nil, err
	}

	logger.Info("Successfully restored WAL", "WAL", walName, "destination", dstPath)

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
