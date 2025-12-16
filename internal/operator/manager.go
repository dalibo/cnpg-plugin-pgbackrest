// SPDX-FileCopyrightText: 2025 Dalibo <contact@dalibo.com>
//
// SPDX-License-Identifier: Apache-2.0

package operator

import (
	"context"
	"crypto/tls"

	// +kubebuilder:scaffold:imports
	cnpgv1 "github.com/cloudnative-pg/cloudnative-pg/api/v1"
	"github.com/cloudnative-pg/machinery/pkg/log"
	apipgbackrest "github.com/dalibo/cnpg-i-pgbackrest/api/v1"
	"github.com/spf13/viper"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/metrics/filters"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

var scheme = runtime.NewScheme()

func init() {
	utilruntime.Must(apipgbackrest.AddToScheme(scheme))
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(cnpgv1.AddToScheme(scheme))
	// +kubebuilder:scaffold:scheme
}

// Start starts the manager
func Start(ctx context.Context) error {
	setupLog := log.FromContext(ctx)

	var tlsOpts []func(*tls.Config)

	webhookServer := webhook.NewServer(webhook.Options{
		TLSOpts: tlsOpts,
	})
	metricsServerOptions := metricsserver.Options{
		BindAddress:   viper.GetString("metrics-bind-address"),
		SecureServing: viper.GetBool("metrics-secure"),
		TLSOpts:       tlsOpts,
	}

	if viper.GetBool("metrics-secure") {
		metricsServerOptions.FilterProvider = filters.WithAuthenticationAndAuthorization
	}
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                        scheme,
		WebhookServer:                 webhookServer,
		Metrics:                       metricsServerOptions,
		HealthProbeBindAddress:        viper.GetString("health-probe-bind-address"),
		LeaderElection:                true,
		LeaderElectionID:              "822e3f5c.cnpg.io",
		LeaderElectionReleaseOnCancel: true,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		return err
	}

	// +kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		return err
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		return err
	}

	if err := mgr.Add(&PgbackresControllerServer{
		Client:         mgr.GetClient(),
		PluginPath:     viper.GetString("plugin-path"),
		ServerCertPath: viper.GetString("server-cert"),
		ServerKeyPath:  viper.GetString("server-key"),
		ClientCertPath: viper.GetString("client-cert"),
		ServerAddress:  viper.GetString("server-address"),
	}); err != nil {
		setupLog.Error(err, "unable to create the pgbackrest runnable controller")
		return err
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctx); err != nil {
		return err
	}

	return nil
}
