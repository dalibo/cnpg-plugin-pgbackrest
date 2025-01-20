// Package operator is the entrypoint of operator plugin
package operator

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/dalibo/cnpg-i-pgbackrest/internal/operator"
)

// NewCmd creates a new operator command
func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "operator",
		Short: "Starts the pgbackrest Cloud CNPG-i plugin",
		RunE: func(cmd *cobra.Command, _ []string) error {
			//			if len(viper.GetString("sidecar-image")) == 0 {
			//				return fmt.Errorf("missing required SIDECAR_IMAGE environment variable")
			//			}

			return operator.Start(cmd.Context())
		},
		PersistentPreRunE: func(_ *cobra.Command, _ []string) error {
			return nil
		},
	}

	cmd.Flags().String(
		"plugin-path",
		"",
		"The plugins socket path",
	)
	_ = viper.BindPFlag("plugin-path", cmd.Flags().Lookup("plugin-path"))

	cmd.Flags().String(
		"server-cert",
		"",
		"The public key to be used for the server process",
	)
	_ = viper.BindPFlag("server-cert", cmd.Flags().Lookup("server-cert"))

	cmd.Flags().String(
		"server-key",
		"",
		"The key to be used for the server process",
	)
	_ = viper.BindPFlag("server-key", cmd.Flags().Lookup("server-key"))

	cmd.Flags().String(
		"client-cert",
		"",
		"The client public key to verify the connection",
	)
	_ = viper.BindPFlag("client-cert", cmd.Flags().Lookup("client-cert"))

	cmd.Flags().String(
		"server-address",
		"",
		"The address where to listen (i.e. 0:9090)",
	)
	_ = viper.BindPFlag("server-address", cmd.Flags().Lookup("server-address"))

	_ = viper.BindEnv("sidecar-image", "SIDECAR_IMAGE")

	return cmd
}
