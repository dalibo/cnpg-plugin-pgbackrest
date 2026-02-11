// SPDX-FileCopyrightText: 2025 Dalibo <contact@dalibo.com>
//
// SPDX-License-Identifier: Apache-2.0

package operator

import (
	"context"
	"fmt"

	cnpgv1 "github.com/cloudnative-pg/cloudnative-pg/api/v1"
	"github.com/cloudnative-pg/cnpg-i-machinery/pkg/pluginhelper/decoder"
	machineryapi "github.com/cloudnative-pg/machinery/pkg/api"
	"github.com/cloudnative-pg/machinery/pkg/stringset"
	apipgbackrest "github.com/dalibo/cnpg-i-pgbackrest/api/v1"
	"github.com/dalibo/cnpg-i-pgbackrest/internal/metadata"
	"github.com/dalibo/cnpg-i-pgbackrest/internal/pgbackrest/api"
	"github.com/dalibo/cnpg-i-pgbackrest/internal/utils"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type PluginConfiguration struct {
	Cluster           *cnpgv1.Cluster
	ServerName        string
	StanzaRef         string
	RecoveryStanzaRef string
	ReplicaStanzaRef  string
}

type Plugin struct {
	Cluster *cnpgv1.Cluster
	// Parameters are the configuration parameters of this plugin
	Parameters  map[string]string
	PluginIndex int
}

func NewPlugin(cluster cnpgv1.Cluster, pluginName string) *Plugin {
	result := &Plugin{Cluster: &cluster}

	result.PluginIndex = -1
	for idx, cfg := range result.Cluster.Spec.Plugins {
		if cfg.Name == pluginName {
			result.PluginIndex = idx
			result.Parameters = cfg.Parameters
		}
	}

	return result
}

func getRecovParams(cluster *cnpgv1.Cluster) map[string]string {
	if cluster.Spec.Bootstrap == nil || cluster.Spec.Bootstrap.Recovery == nil {
		return nil
	}

	recoveryConfig := cluster.Spec.Bootstrap.Recovery
	if len(recoveryConfig.Source) == 0 {
		// Plugin-based recovery is supported only with
		// An external cluster definition
		return nil
	}

	recoveryExternalCluster, found := cluster.ExternalCluster(recoveryConfig.Source)
	if !found {
		// This error should have already been detected
		// by the validating webhook.
		return nil
	}

	return recoveryExternalCluster.PluginConfiguration.Parameters
}

func getReplicaParams(cluster *cnpgv1.Cluster) map[string]string {

	if cluster.Spec.ReplicaCluster == nil || len(cluster.Spec.ReplicaCluster.Source) == 0 {
		return nil
	}

	replicaSource, found := cluster.ExternalCluster(
		cluster.Spec.ReplicaCluster.Source,
	)
	if !found || replicaSource.PluginConfiguration.Name != metadata.PluginName {
		return nil
	}

	return replicaSource.PluginConfiguration.Parameters

}

func NewFromClusterJSON(clusterJSON []byte) (*PluginConfiguration, error) {
	var res cnpgv1.Cluster
	if err := decoder.DecodeObjectLenient(clusterJSON, &res); err != nil {
		return nil, fmt.Errorf("cluster not found")
	}
	return NewFromCluster(&res)
}

func NewFromCluster(cluster *cnpgv1.Cluster) (*PluginConfiguration, error) {
	helper := NewPlugin(
		*cluster,
		metadata.PluginName,
	)
	serverName := cluster.Name
	recovObjName := ""
	if recovParams := getRecovParams(cluster); recovParams != nil {
		recovObjName = recovParams["stanzaRef"]
	}
	repliObjName := ""
	if repliParams := getReplicaParams(cluster); repliParams != nil {
		repliObjName = repliParams["stanzaRef"]
	}
	result := &PluginConfiguration{
		Cluster:           cluster,
		ServerName:        serverName,
		StanzaRef:         helper.Parameters["stanzaRef"],
		RecoveryStanzaRef: recovObjName,
		ReplicaStanzaRef:  repliObjName,
	}
	return result, nil
}

func (c *PluginConfiguration) GetReplicaStanzaRef() (*types.NamespacedName, error) {
	if len(c.ReplicaStanzaRef) > 0 {
		return &types.NamespacedName{
			Name:      c.ReplicaStanzaRef,
			Namespace: c.Cluster.Namespace,
		}, nil

	}
	return nil, fmt.Errorf("replica stanza not configured")
}

func (c *PluginConfiguration) GetStanzaRef() (*types.NamespacedName, error) {
	if len(c.StanzaRef) > 0 {
		return &types.NamespacedName{
			Name:      c.StanzaRef,
			Namespace: c.Cluster.Namespace,
		}, nil
	}
	return nil, fmt.Errorf("stanza not configured")
}

func (c *PluginConfiguration) GetRecoveryStanzaRef() (*types.NamespacedName, error) {
	if len(c.RecoveryStanzaRef) > 0 {
		return &types.NamespacedName{
			Name:      c.RecoveryStanzaRef,
			Namespace: c.Cluster.Namespace,
		}, nil
	}
	return nil, fmt.Errorf("recovery stanza not configured")
}

// GetReferredPgBackrestObjectKey the list of pgbackrest objects referred by this
// plugin configuration
func (c *PluginConfiguration) GetReferredPgBackrestObjectKey() []types.NamespacedName {
	objectNames := stringset.New()
	if len(c.StanzaRef) > 0 {
		objectNames.Put(c.StanzaRef)
	}
	if len(c.RecoveryStanzaRef) > 0 {
		objectNames.Put(c.RecoveryStanzaRef)
	}
	if len(c.ReplicaStanzaRef) > 0 {
		objectNames.Put(c.ReplicaStanzaRef)
	}
	res := make([]types.NamespacedName, 0, 3)
	for _, name := range objectNames.ToSortedList() {
		res = append(
			res, types.NamespacedName{
				Name:      name,
				Namespace: c.Cluster.Namespace,
			},
		)
	}
	return res
}

func GetEnvVarConfig(
	ctx context.Context,
	r apipgbackrest.Stanza,
	c client.Client,
) ([]string, error) {
	conf := r.Spec.Configuration
	env, err := conf.ToEnv()
	if err != nil {
		return nil, err
	}

	// helper to fetch secret values
	secretVal := func(ref *machineryapi.SecretKeySelector) (string, error) {
		raw, err := utils.GetValueFromSecret(ctx, c, r.Namespace, ref)
		if err != nil {
			return "", err
		}
		return string(raw), nil
	}
	for i, r := range conf.S3Repositories {
		sRef := r.SecretRef
		aKey, err := secretVal(sRef.AccessKeyIDReference)
		if err != nil {
			return nil, err
		}
		sKey, err := secretVal(sRef.SecretAccessKeyReference)
		if err != nil {
			return nil, err
		}
		prefix := fmt.Sprintf("PGBACKREST_REPO%d_", i+1)

		if r.Cipher != nil {
			encKey, err := secretVal(r.Cipher.PassReference)
			if err != nil {
				return nil, err
			}
			env = append(env, fmt.Sprintf("%sCIPHER_PASS=%s", prefix, encKey))
		}
		// build env var names
		env = append(
			env,
			fmt.Sprintf("%sS3_KEY=%s", prefix, aKey),
			fmt.Sprintf("%sS3_KEY_SECRET=%s", prefix, sKey),
			fmt.Sprintf("%sTYPE=%s", prefix, "s3"),
		)

	}
	return env, nil
}

type ClusterDefinitionGetter interface {
	GetClusterDefinition() []byte
}

type StanzaRefGetter func(*PluginConfiguration) (*client.ObjectKey, error)

func GetStanza(ctx context.Context,
	c ClusterDefinitionGetter,
	cl client.Client,
	getRef StanzaRefGetter,
) (*apipgbackrest.Stanza, error) {
	cDef := c.GetClusterDefinition()
	pluginConf, err := NewFromClusterJSON(cDef)
	if err != nil {
		return nil, err
	}
	serverName := pluginConf.ServerName
	stanzaFQDN, err := getRef(pluginConf)
	if err != nil {
		return nil, err
	}
	var stanza apipgbackrest.Stanza
	if err := cl.Get(ctx, *stanzaFQDN, &stanza); err != nil {
		return nil, err
	}

	// add the cluster name in the repo-path
	for r := range stanza.Spec.Configuration.S3Repositories {
		api.AppendToRepoPath(&stanza.Spec.Configuration.S3Repositories[r], serverName)
	}

	return &stanza, nil
}
