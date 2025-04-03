// SPDX-FileCopyrightText: 2025 Dalibo <contact@dalibo.com>
//
// SPDX-License-Identifier: Apache-2.0

package metadata

import "github.com/cloudnative-pg/cnpg-i/pkg/identity"

// PluginName is the name of the plugin
const PluginName = "pgbackrest.dalibo.com"

// Data is the metadata of this plugin
var Data = identity.GetPluginMetadataResponse{
	Name:          PluginName,
	Version:       "0.0.1",
	DisplayName:   "pgBackRest demo / experimental plugin",
	ProjectUrl:    "https://github.com/dalibo/cnpg-i-pgbackrest",
	RepositoryUrl: "https://github.com/dalibo/cnpg-i-pgbackrest",
	License:       "Proprietary",
	LicenseUrl:    "https://github.com/dalibo/cnpg-i-pgbackrest/LICENSE",
	Maturity:      "alpha",
}
