package operator

import (
	"fmt"
	"strconv"

	cnpgv1 "github.com/cloudnative-pg/cloudnative-pg/api/v1"
	"github.com/dalibo/cnpg-i-pgbackrest/internal/metadata"
)

type PluginConfiguration struct {
	Cluster     *cnpgv1.Cluster
	ServerName  string
	S3Bucket    string
	S3Endpoint  string
	S3Region    string
	S3RepoPath  string
	S3Stanza    string
	S3UriStyle  string
	S3VerifyTls bool
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

func NewFromCluster(cluster *cnpgv1.Cluster) (*PluginConfiguration, error) {
	helper := NewPlugin(
		*cluster,
		metadata.PluginName,
	)

	serverName := cluster.Name
	result := &PluginConfiguration{
		Cluster:    cluster,
		ServerName: serverName,
		S3Bucket:   helper.Parameters["s3-bucket"],
		S3Endpoint: helper.Parameters["s3-endpoint"],
		S3Region:   helper.Parameters["s3-region"],
		S3RepoPath: helper.Parameters["s3-repo-path"],
		S3Stanza:   helper.Parameters["stanza"],
	}
	if val, ok := helper.Parameters["s3-uri-style"]; ok {
		result.S3UriStyle = val
	}
	if val, ok := helper.Parameters["s3-verify-tls"]; ok {
		r, err := strconv.ParseBool(val)
		if err != nil {
			return nil, fmt.Errorf("can't convert user input for s3-verify-tls field: %w", err)
		}
		result.S3VerifyTls = r
	} else {
		result.S3VerifyTls = true
	}
	return result, nil
}
