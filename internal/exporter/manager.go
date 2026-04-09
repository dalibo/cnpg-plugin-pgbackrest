// SPDX-FileCopyrightText: 2026 Dalibo <contact@dalibo.com>
//
// SPDX-License-Identifier: Apache-2.0

package exporter

import (
	"context"

	pgbackrestapi "github.com/dalibo/cnpg-i-pgbackrest/api/v1"

	cnpgv1 "github.com/cloudnative-pg/cloudnative-pg/api/v1"
	extendedclient "github.com/dalibo/cnpg-i-pgbackrest/internal/instance/client"
	"github.com/spf13/viper"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var scheme = runtime.NewScheme()

func init() {
	utilruntime.Must(pgbackrestapi.AddToScheme(scheme))
	utilruntime.Must(cnpgv1.AddToScheme(scheme))
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
}

// Start the sidecar informers and pgbackrest plugin
// server. This is the part running on the sidecar container.
func Start(ctx context.Context) error {
	setupLog := log.FromContext(ctx)
	setupLog.Info("Starting pgbackrest sidecar exporter instance plugin")

	podName := viper.GetString("pod-name")
	clusterName := viper.GetString("cluster-name")
	ns := viper.GetString("namespace")

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
	cl, err := client.New(ctrl.GetConfigOrDie(), clientOpt)
	if err != nil {
		return err
	}

	pgbaSidecarServer := PgbackrestSidecarServer{
		Client:       extendedclient.NewExtendedClient(cl),
		InstanceName: podName,
		Namespace:    ns,
		ClusterName:  clusterName,
	}

	return pgbaSidecarServer.Start(ctx)
}
