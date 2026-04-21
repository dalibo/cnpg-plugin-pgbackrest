// SPDX-FileCopyrightText: 2026 Dalibo <contact@dalibo.com>
//
// SPDX-License-Identifier: Apache-2.0
package utils

import (
	"slices"

	cnpgv1 "github.com/cloudnative-pg/cloudnative-pg/api/v1"
	"github.com/dalibo/cnpg-i-pgbackrest/internal/metadata"
)

func IsPluginEnabled(cluster *cnpgv1.Cluster) bool {
	e := cnpgv1.GetPluginConfigurationEnabledPluginNames(cluster.Spec.Plugins)
	return slices.Contains(e, metadata.PluginName)
}
