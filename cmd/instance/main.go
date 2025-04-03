// SPDX-FileCopyrightText: 2025 Dalibo <contact@dalibo.com>
//
// SPDX-License-Identifier: Apache-2.0

package instance

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	instance_pgbackrest "github.com/dalibo/cnpg-i-pgbackrest/internal/instance"
)

// NewCmd creates a new instance command
func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "instance",
		Short: "Starts the pgbackrest sidecar plugin",
		RunE: func(cmd *cobra.Command, _ []string) error {
			//requiredSettings := []string{
			//		"namespace",
			//		"pod-name",
			//		"spool-directory",
			//	}

			//		for _, k := range requiredSettings {
			//				if len(viper.GetString(k)) == 0 {
			//					return fmt.Errorf("missing required %s setting", k)
			//				}
			//			}

			return instance_pgbackrest.Start(cmd.Context())
		},
	}

	_ = viper.BindEnv("namespace", "NAMESPACE")
	_ = viper.BindEnv("pod-name", "POD_NAME")
	_ = viper.BindEnv("pgdata", "PGDATA")
	_ = viper.BindEnv("spool-directory", "SPOOL_DIRECTORY")

	return cmd
}
