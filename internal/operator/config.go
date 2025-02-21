package operator

import (
	cnpgv1 "github.com/cloudnative-pg/cloudnative-pg/api/v1"
	"github.com/dalibo/cnpg-i-pgbackrest/internal/metadata"
)

type PluginConfiguration struct {
	Cluster    *cnpgv1.Cluster
	ServerName string
	RepoPath   string
	S3Bucket   string
	Stanza     string
}

type Plugin struct {
	Cluster *cnpgv1.Cluster
	// Parameters are the configuration parameters of this plugin
	Parameters  map[string]string
	PluginIndex int
}

// NewPlugin creates a new Plugin instance for the given cluster and plugin name.
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
func NewFromCluster(cluster *cnpgv1.Cluster) *PluginConfiguration {
	helper := NewPlugin(
		*cluster,
		metadata.PluginName,
	)

	serverName := cluster.Name
	result := &PluginConfiguration{
		Cluster:    cluster,
		ServerName: serverName,
		RepoPath:   helper.Parameters["repo-path"],
		S3Bucket:   helper.Parameters["s3Bucket"],
		Stanza:     helper.Parameters["stanza"],
	}

	return result
}
