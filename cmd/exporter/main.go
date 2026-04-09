// SPDX-FileCopyrightText: 2025 Dalibo <contact@dalibo.com>
//
// SPDX-License-Identifier: Apache-2.0

package exporter

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	exporter_pgbackrest "github.com/dalibo/cnpg-i-pgbackrest/internal/exporter"
)

// NewCmd creates a new exporter command
func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "exporter",
		Short: "Starts the pgBackRest exporter sidecar plugin",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return exporter_pgbackrest.Start(cmd.Context())
		},
	}

	_ = viper.BindEnv("namespace", "NAMESPACE")
	_ = viper.BindEnv("pod-name", "POD_NAME")
	_ = viper.BindEnv("cluster-name", "CLUSTER_NAME")

	return cmd
}
