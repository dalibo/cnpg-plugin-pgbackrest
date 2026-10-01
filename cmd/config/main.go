// SPDX-FileCopyrightText: 2025 Dalibo <contact@dalibo.com>
//
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"fmt"
	"strings"

	cnpgv1 "github.com/cloudnative-pg/cloudnative-pg/api/v1"
	pgbackrestapi "github.com/dalibo/cnpg-i-pgbackrest/api/v1"
	config "github.com/dalibo/cnpg-i-pgbackrest/internal/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var scheme = runtime.NewScheme()

// NewCmd creates a new config command
func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Get the pgBackRest config from environment variables",
		RunE: func(cmd *cobra.Command, _ []string) error {
			requiredSettings := []string{
				"cluster-name",
				"namespace",
			}

			for _, k := range requiredSettings {
				if len(viper.GetString(k)) == 0 {
					return fmt.Errorf("missing required %s setting", k)
				}
			}

			err := corev1.AddToScheme(scheme)

			if err != nil {
				return err
			}

			err = cnpgv1.AddToScheme(scheme)

			if err != nil {
				return err
			}

			// initiate a client with cache disabled for stanza and cluster
			clientOpt := client.Options{
				Scheme: scheme,
				Cache: &client.CacheOptions{
					DisableFor: []client.Object{
						&pgbackrestapi.PluginConfig{},
						&cnpgv1.Cluster{},
					},
				},
			}

			// create client
			cl, err := client.New(ctrl.GetConfigOrDie(), clientOpt)
			if err != nil {
				return err
			}

			// retrieve cluster
			clusterName := types.NamespacedName{
				Name:      viper.GetString("cluster-name"),
				Namespace: viper.GetString("namespace"),
			}
			cluster := &cnpgv1.Cluster{}
			if err := cl.Get(cmd.Context(), clusterName, cluster); err != nil {
				return err
			}

			// retrieve stanza from cluster
			stanza, err := config.GetStanzaFromCluster(cmd.Context(), cluster, cl, (*config.PluginConfiguration).GetStanzaRef)
			if err != nil {
				return err
			}

			envvars, err := config.GetEnvVarConfig(cmd.Context(), stanza, cl)

			for i, _ := range envvars {
				if strings.Contains(envvars[i], "PGBACKREST_REPO1_S3_KEY_SECRET") {
					envvars[i] = "PGBACKREST_REPO1_S3_KEY_SECRET="
				}
			}

			fmt.Println(strings.Join(envvars[:], "\n"))

			return err
		},
	}

	_ = viper.BindEnv("namespace", "NAMESPACE")
	_ = viper.BindEnv("cluster-name", "CLUSTER_NAME")

	return cmd
}
