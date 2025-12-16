/*
Copyright © contributors to CloudNativePG, established as
CloudNativePG a Series of LF Projects, LLC.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

SPDX-License-Identifier: Apache-2.0
*/

package instance

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudnative-pg/cnpg-i/pkg/metrics"
	"github.com/cloudnative-pg/machinery/pkg/log"
	"github.com/dalibo/cnpg-i-pgbackrest/internal/metadata"
	"github.com/dalibo/cnpg-i-pgbackrest/internal/operator"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Sanitize the plugin name to be a valid Prometheus metric namespace
var metricsDomain = strings.NewReplacer(".", "_", "-", "_").Replace(metadata.PluginName)

type metricsImpl struct {
	// important the client should be one with a underlying cache
	Client client.Client
	metrics.UnimplementedMetricsServer
}

func buildFqName(name string) string {
	// Build the fully qualified name for the metric
	return fmt.Sprintf(
		"%s_%s",
		metricsDomain,
		strings.NewReplacer(".", "_", "-", "_").Replace(name),
	)
}

var (
	firstRecoverabilityPointMetricName     = buildFqName("first_recoverability_point")
	lastAvailableBackupTimestampMetricName = buildFqName("last_available_backup_timestamp")
	lastFailedBackupTimestampMetricName    = buildFqName("last_failed_backup_timestamp")
)

func (m metricsImpl) GetCapabilities(
	ctx context.Context,
	_ *metrics.MetricsCapabilitiesRequest,
) (*metrics.MetricsCapabilitiesResult, error) {
	contextLogger := log.FromContext(ctx)
	contextLogger.Info("metrics capabilities call received")

	return &metrics.MetricsCapabilitiesResult{
		Capabilities: []*metrics.MetricsCapability{
			{
				Type: &metrics.MetricsCapability_Rpc{
					Rpc: &metrics.MetricsCapability_RPC{
						Type: metrics.MetricsCapability_RPC_TYPE_METRICS,
					},
				},
			},
		},
	}, nil
}

func (m metricsImpl) Define(
	ctx context.Context,
	_ *metrics.DefineMetricsRequest,
) (*metrics.DefineMetricsResult, error) {
	contextLogger := log.FromContext(ctx)
	contextLogger.Debug("metrics define call received")

	return &metrics.DefineMetricsResult{
		Metrics: []*metrics.Metric{
			{
				FqName:    firstRecoverabilityPointMetricName,
				Help:      "First available pgBackRest stuff",
				ValueType: &metrics.MetricType{Type: metrics.MetricType_TYPE_GAUGE},
			},
			{
				FqName:    lastAvailableBackupTimestampMetricName,
				Help:      "Last available pgBackRest stuff",
				ValueType: &metrics.MetricType{Type: metrics.MetricType_TYPE_GAUGE},
			},
			{
				FqName:    lastFailedBackupTimestampMetricName,
				Help:      "Last failed pgBackRest backup",
				ValueType: &metrics.MetricType{Type: metrics.MetricType_TYPE_GAUGE},
			},
		},
	}, nil
}

func (m metricsImpl) Collect(
	ctx context.Context,
	req *metrics.CollectMetricsRequest,
) (*metrics.CollectMetricsResult, error) {
	contextLogger := log.FromContext(ctx)
	contextLogger.Debug("metrics collect call received")
	repo, err := operator.GetRepo(ctx,
		req,
		m.Client,
		(*operator.PluginConfiguration).GetRepositoryRef,
	)
	if err != nil {
		return nil, err
	}
	return &metrics.CollectMetricsResult{
		Metrics: []*metrics.CollectMetric{
			{
				FqName: firstRecoverabilityPointMetricName,
				Value:  float64(repo.Status.RecoveryWindow.FirstBackup.Timestamp.Stop),
			},
			{
				FqName: lastAvailableBackupTimestampMetricName,
				Value:  float64(repo.Status.RecoveryWindow.LastBackup.Timestamp.Stop),
			},
			{
				FqName: lastFailedBackupTimestampMetricName,
				Value:  float64(0),
			},
		},
	}, nil
}
