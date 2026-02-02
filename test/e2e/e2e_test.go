// SPDX-FileCopyrightText: 2025 Dalibo <contact@dalibo.com>
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"io"
	"maps"
	"os"
	"testing"

	cloudnativepgv1 "github.com/cloudnative-pg/api/pkg/api/v1"
	apipgbackrest "github.com/dalibo/cnpg-i-pgbackrest/api/v1"
	"github.com/dalibo/cnpg-i-pgbackrest/test/e2e/internal/certmanager"
	"github.com/dalibo/cnpg-i-pgbackrest/test/e2e/internal/cluster"
	"github.com/dalibo/cnpg-i-pgbackrest/test/e2e/internal/cnpg"
	"github.com/dalibo/cnpg-i-pgbackrest/test/e2e/internal/common"
	"github.com/dalibo/cnpg-i-pgbackrest/test/e2e/internal/kubernetes"
	"github.com/dalibo/cnpg-i-pgbackrest/test/e2e/internal/minio"
	"github.com/dalibo/cnpg-i-pgbackrest/test/e2e/internal/pgbackrest"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

// Deploy CNGP operator, certmanager, minio and our plugins
func setup() {
	k8sClient, err := kubernetes.Client()
	logger := zap.New(zap.WriteTo(io.Discard), zap.UseDevMode(false))
	ctx := log.IntoContext(context.Background(), logger)
	if err != nil {
		panic("can't init kubernetes client")
	}
	s := kubernetes.InstallSpec{
		ManifestUrl: "https://raw.githubusercontent.com/cloudnative-pg/cloudnative-pg/release-1.28/releases/cnpg-1.28.0.yaml",
	}
	if err := cnpg.Install(ctx, *k8sClient, s); err != nil {
		panic("can't install CNPG")
	}
	s = kubernetes.InstallSpec{
		ManifestUrl: "https://github.com/cert-manager/cert-manager/releases/download/v1.19.1/cert-manager.yaml",
	}
	if err := certmanager.Install(ctx, *k8sClient, s); err != nil {
		panic("can't install certmanager")
	}
	if err = minio.Install(ctx, *k8sClient); err != nil {
		panic("can't install minio")
	}
	// install our pgbackrest plugin from kubernetes directory at the root
	// of the repository
	path, err := os.Getwd()
	if err != nil {
		panic("can't define current working dir")
	}
	s = kubernetes.InstallSpec{
		ManifestUrl:  path + "/../../kubernetes/dev/",
		UseKustomize: true,
	}
	if err := pgbackrest.Install(ctx, *k8sClient, s); err != nil {
		panic(err.Error())
	}
	if _, err := pgbackrest.CreateStanzaConfig(ctx, *k8sClient, "stanza", "default"); err != nil {
		panic(err.Error())
	}
}

func createSecret(
	ctx context.Context,
	k8sClient *kubernetes.K8sClient,
	namespace string,
) (*corev1.Secret, error) {
	// TODO: move that ?
	secret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pgbackrest-s3-secret",
			Namespace: namespace,
		},
		Type: corev1.SecretTypeOpaque,
		StringData: map[string]string{
			"ACCESS_KEY_ID":     minio.ACCESS_KEY,
			"ACCESS_SECRET_KEY": minio.SECRET_KEY,
			"ENCRYPTION_PASS":   "3nCrypTi0n",
		},
	}
	return secret, k8sClient.Create(ctx, secret)
}

func teardown() {
	// should remove ressources created on setup()
}

// TestMain is called before any tests are executed.
func TestMain(m *testing.M) {
	setup()
	defer teardown()
	m.Run()
}

func TestInstall(t *testing.T) {
	log.SetLogger(zap.New(zap.WriteTo(os.Stdout), zap.UseDevMode(true)))
	// basic verification to ensure our plugin is present
	k8sClient, err := kubernetes.Client()
	if k8sClient == nil || err != nil {
		t.Fatalf("error kubernetes client not initialized")
	}
	ctx := context.TODO()

	// basic check for deployment
	fqdn := types.NamespacedName{Name: "pgbackrest-controller", Namespace: "cnpg-system"}
	deployment := &appsv1.Deployment{}
	if err := k8sClient.Get(ctx, fqdn, deployment); err != nil {
		t.Errorf("can't get delployment")
	}
	if nRep := deployment.Status.ReadyReplicas; nRep != 1 {
		t.Errorf("error no Pod for pgbackrest plugin want: 1, got: %v", nRep)
	}

	// verify service creation
	fqdn = types.NamespacedName{Name: "pgbackrest", Namespace: "cnpg-system"}
	svc := &corev1.Service{}
	err = k8sClient.Get(ctx, fqdn, svc)
	if err != nil {
		t.Errorf("error no service for pgbackrest found %s", err.Error())
	}
	wantLabels := map[string]string{
		"app":                "pgbackrest-controller",
		"cnpg.io/pluginName": "pgbackrest.dalibo.com",
	}
	for k, v := range wantLabels {
		if svc.Labels[k] != v {
			t.Errorf("service label %s mismatch: want %s, got %s", k, v, svc.Labels[k])
		}
	}

}

func takeBackup(
	ctx context.Context,
	t *testing.T,
	k8sClient *kubernetes.K8sClient,
	ns string,
	clusterName string,
	bName string,
	params map[string]string,
) *cloudnativepgv1.Backup {

	bi := cluster.BackupInfo{
		Cluster:   clusterName,
		Namespace: ns,
		Params:    params,
		Name:      bName,
	}
	// more verification can be done here (before deleting the cluster)
	b, err := bi.Backup(ctx, k8sClient)
	if err != nil {
		t.Fatalf("error when executing backup %v", err)
	}
	retrier, err := common.NewRetrier(80)
	if err != nil {
		panic("can't initiate retrier")
	}
	if _, err = bi.IsDone(ctx, k8sClient, retrier); err != nil {
		t.Fatalf("error when trying to determine if backup is done, %v", err)
	}
	return b
}

func getStanza(
	ctx context.Context,
	t *testing.T,
	k8sClient *kubernetes.K8sClient,
	ns string,
) *apipgbackrest.Stanza {
	stanza, err := pgbackrest.GetStanza(ctx, k8sClient, "stanza", ns)
	if err != nil {
		t.Fatalf("failed to get stanza: %v", err)
	}
	return stanza
}

// basic verification to ensure we can use our plugin with a cluster
func TestDeployInstance(t *testing.T) {
	log.SetLogger(zap.New(zap.WriteTo(os.Stdout), zap.UseDevMode(true)))
	k8sClient, err := kubernetes.Client()
	if k8sClient == nil || err != nil {
		t.Fatalf("kubernetes client not initialized: %v", err)
	}
	ns := "default"
	ctx := context.Background()
	log.FromContext(ctx)
	// first create a secret
	secret, err := createSecret(ctx, k8sClient, ns)
	if err != nil {
		t.Fatalf("failed to create secret: %v", err)
	}
	defer func() {
		if err := k8sClient.Delete(ctx, secret); err != nil {
			t.Fatal("can't delete secret")
		}
	}()

	// create a test CloudNativePG Cluster
	clusterName := "cluster-demo"
	p := maps.Clone(cluster.DefaultParamater)
	c, err := cluster.Create(ctx, k8sClient, ns, clusterName, 1, "100M", p, false)
	if err != nil {
		t.Fatalf("failed to create cluster: %v", err)
	}
	defer func() {
		if err := k8sClient.Delete(ctx, c); err != nil {
			t.Fatal("can't delete cluster")
		}
	}()
	if ready, err := k8sClient.PodIsReady(ctx, ns, clusterName+"-1", 80, 3); err != nil {
		t.Fatalf("error when requesting pod status, %s", err.Error())
	} else if !ready {
		t.Fatal("pod not ready")
	}

	// take a first backup
	b := takeBackup(ctx, t, k8sClient, ns, clusterName, "backup-01", p)
	defer func() {
		if delErr := k8sClient.Delete(ctx, b); delErr != nil {
			t.Fatalf("can't delete backup-01: %v", delErr)
		}
	}()

	// check stored backup info / status
	stanza := getStanza(ctx, t, k8sClient, ns)
	fBackup := stanza.Status.RecoveryWindow.FirstBackup
	lBackup := stanza.Status.RecoveryWindow.LastBackup
	if fBackup.Timestamp.Start == 0 || fBackup != lBackup {
		t.Fatal("registered backup data are invalid after first backup")
	}

	// take a second backup
	b2 := takeBackup(ctx, t, k8sClient, ns, clusterName, "backup-02", p)
	defer func() {
		if delErr := k8sClient.Delete(ctx, b2); delErr != nil {
			t.Fatalf("can't delete backup-02: %v", delErr)
		}
	}()

	// check stored backup info / status
	stanza = getStanza(ctx, t, k8sClient, ns)
	if err != nil {
		t.Fatalf("failed to get stanza after second backup: %v", err)
	}
	fBackup = stanza.Status.RecoveryWindow.FirstBackup
	lBackup = stanza.Status.RecoveryWindow.LastBackup
	// After the second backup, both ends of the window should NOT match the first case
	if fBackup.Timestamp.Start == 0 || fBackup == lBackup {
		t.Fatal("registered backup data are invalid after second backup")
	}
	// delete cluster, we will recreate it from backup
	if err := k8sClient.Delete(ctx, c); err != nil {
		t.Fatal("can't delete cluster")
	}
	if _, err = k8sClient.PodIsAbsent(ctx, ns, clusterName+"-1", 10, 3); err != nil {
		t.Fatal("can't ensure cluster is absent")
	}
	if _, err = cluster.Create(ctx, k8sClient, ns, clusterName, 1, "100M", p, true); err != nil {
		t.Fatal("can't recreate cluster from backup")
	}
	if ready, err := k8sClient.PodIsReady(ctx, ns, clusterName+"-1", 80, 3); err != nil {
		t.Fatalf("error when requesting pod status, %s", err.Error())
	} else if !ready {
		t.Fatal("pod not ready")
	}
}
