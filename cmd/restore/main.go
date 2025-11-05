// SPDX-FileCopyrightText: 2025 Dalibo <contact@dalibo.com>
//
// SPDX-License-Identifier: Apache-2.0

package restore

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	restore_pgbackrest "github.com/dalibo/cnpg-i-pgbackrest/internal/restore"
)

// NewCmd creates a new instance command
func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "restore",
		Short: "Starts the restore job & sidecar plugin",
		RunE: func(cmd *cobra.Command, _ []string) error {
			requiredSettings := []string{
				"namespace",
				"pod-name",
			}

			for _, k := range requiredSettings {
				if len(viper.GetString(k)) == 0 {
					return fmt.Errorf("missing required %s setting", k)
				}
			}

			return restore_pgbackrest.Start(cmd.Context())
		},
	}

	_ = viper.BindEnv("namespace", "NAMESPACE")
	_ = viper.BindEnv("pod-name", "POD_NAME")
	_ = viper.BindEnv("pgdata", "PGDATA")

	return cmd
}
