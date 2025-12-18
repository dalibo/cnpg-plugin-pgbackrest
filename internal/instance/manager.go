// SPDX-FileCopyrightText: 2025 Dalibo <contact@dalibo.com>
//
// SPDX-License-Identifier: Apache-2.0

package instance

import (
	"context"
	"path"

	apipgbackrest "github.com/dalibo/cnpg-i-pgbackrest/api/v1"
	extendedclient "github.com/dalibo/cnpg-i-pgbackrest/internal/instance/client"

	cnpgv1 "github.com/cloudnative-pg/cloudnative-pg/api/v1"
	"github.com/spf13/viper"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

// Start starts the sidecar informers and pgbackrest plugin
// server. This is the part running on the sidecar container.
func Start(ctx context.Context) error {
	setupLog := log.FromContext(ctx)
	setupLog.Info("Starting pgbackrest instance plugin")
	podName := viper.GetString("pod-name")

	sc := generateScheme(ctx)
	controllerOptions := ctrl.Options{
		Scheme: sc,
		Client: client.Options{
			Cache: &client.CacheOptions{
				DisableFor: []client.Object{
					&corev1.Secret{},
					&apipgbackrest.Stanza{},
					&cnpgv1.Cluster{},
					&cnpgv1.Backup{},
				},
			},
		},
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), controllerOptions)
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		return err
	}
	customCacheClient := extendedclient.NewExtendedClient(mgr.GetClient())
	if err := mgr.Add(&PgbackrestPluginServer{
		Client:       customCacheClient,
		InstanceName: podName,
		// TODO: improve
		PGDataPath: viper.GetString("pgdata"),
		PGWALPath:  path.Join(viper.GetString("pgdata"), "pg_wal"),
		PluginPath: viper.GetString("plugin-path"),
	}); err != nil {
		setupLog.Error(err, "unable to create pbacrest plugin runnable/server")
		return err
	}

	if err := mgr.Start(ctx); err != nil {
		return err
	}

	return nil
}

// generateScheme creates a runtime.Scheme object with all the
// definition needed to support the sidecar. This allows
// the plugin to be used in every CNPG-based operator.
func generateScheme(ctx context.Context) *runtime.Scheme {
	result := runtime.NewScheme()
	utilruntime.Must(apipgbackrest.AddToScheme(result))
	utilruntime.Must(clientgoscheme.AddToScheme(result))

	cnpgGroup := viper.GetString("custom-cnpg-group")
	cnpgVersion := viper.GetString("custom-cnpg-version")
	if len(cnpgGroup) == 0 {
		cnpgGroup = cnpgv1.SchemeGroupVersion.Group
	}
	if len(cnpgVersion) == 0 {
		cnpgVersion = cnpgv1.SchemeGroupVersion.Version
	}

	// Proceed with custom registration of the CNPG scheme
	schemeGroupVersion := schema.GroupVersion{Group: cnpgGroup, Version: cnpgVersion}
	schemeBuilder := &scheme.Builder{GroupVersion: schemeGroupVersion}
	schemeBuilder.Register(&cnpgv1.Cluster{}, &cnpgv1.ClusterList{})
	schemeBuilder.Register(&cnpgv1.Backup{}, &cnpgv1.BackupList{})
	schemeBuilder.Register(&cnpgv1.ScheduledBackup{}, &cnpgv1.ScheduledBackupList{})
	utilruntime.Must(schemeBuilder.AddToScheme(result))

	schemeLog := log.FromContext(ctx)
	schemeLog.Info("CNPG types registration", "schemeGroupVersion", schemeGroupVersion)

	return result
}
