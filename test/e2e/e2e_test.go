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
		ManifestUrl: "https://raw.githubusercontent.com/cloudnative-pg/cloudnative-pg/release-1.27/releases/cnpg-1.27.1.yaml",
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
	if _, err := pgbackrest.CreateRepoConfig(ctx, *k8sClient, "repository", "default"); err != nil {
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
	c, err := cluster.Create(ctx, k8sClient, ns, clusterName, 1, "100M", p)
	if err != nil {
		t.Fatalf("failed to create cluster: %v", err)
	}
	defer func() {
		if err := k8sClient.Delete(ctx, c); err != nil {
			t.Fatal("can't delete cluster")
		}
	}()
	if ready, err := k8sClient.PodsIsReady(ctx, ns, clusterName+"-1", 80, 3); err != nil {
		t.Fatalf("error when requesting pod status, %s", err.Error())
	} else if !ready {
		t.Fatal("pod not ready")
	}
	bi := cluster.BackupInfo{
		Cluster:   clusterName,
		Namespace: ns,
		Params:    p,
		Name:      "backup-01",
	}
	// more verification can be done here (before deleting the cluster)
	b, err := bi.Backup(ctx, k8sClient)
	if err != nil {
		t.Errorf("Error when executing backup %v", err.Error())

	}
	defer func() {
		if err := k8sClient.Delete(ctx, b); err != nil {
			t.Fatal("can't delete backup")
		}
	}()
	retrier, err := common.NewRetrier(80)
	if err != nil {
		panic("can't initiate retrier")
	}
	_, err = bi.IsDone(ctx, k8sClient, retrier)
	if err != nil {
		t.Errorf("Error when retrieving info for backup %v", err.Error())
	}

}
