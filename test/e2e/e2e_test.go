// SPDX-FileCopyrightText: 2025 Dalibo <contact@dalibo.com>
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"io"
	"maps"
	"os"
	"strings"
	"testing"

	cloudnativepgv1 "github.com/cloudnative-pg/api/pkg/api/v1"
	"github.com/dalibo/cnpg-i-pgbackrest/test/e2e/internal/azurite"
	"github.com/dalibo/cnpg-i-pgbackrest/test/e2e/internal/certmanager"
	"github.com/dalibo/cnpg-i-pgbackrest/test/e2e/internal/cluster"
	"github.com/dalibo/cnpg-i-pgbackrest/test/e2e/internal/cnpg"
	"github.com/dalibo/cnpg-i-pgbackrest/test/e2e/internal/command"
	"github.com/dalibo/cnpg-i-pgbackrest/test/e2e/internal/kubernetes"
	"github.com/dalibo/cnpg-i-pgbackrest/test/e2e/internal/minio"
	"github.com/dalibo/cnpg-i-pgbackrest/test/e2e/internal/pgbackrest"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

const (
	NS string = "default"
)

var _S3_DATA_SECRET map[string]string = map[string]string{
	"ACCESS_KEY_ID":     minio.ACCESS_KEY,
	"ACCESS_SECRET_KEY": minio.SECRET_KEY,
	"ENCRYPTION_PASS":   "3nCrypTi0n",
}
var _AZURE_DATA_SECRET map[string]string = map[string]string{
	"KEY": azurite.ACCOUNT_SECRET,
}

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
	if err = azurite.Install(ctx, *k8sClient); err != nil {
		panic("can't install azurite")
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
}

func createSecret(
	ctx context.Context,
	k8sClient *kubernetes.K8sClient,
	namespace,
	name string,
	data map[string]string,
) (*corev1.Secret, error) {
	// TODO: move that ?
	secret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Type:       corev1.SecretTypeOpaque,
		StringData: data,
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
	if _, err = bi.IsDone(ctx, k8sClient, 115, 3); err != nil {
		t.Fatalf("error when trying to determine if backup is done, %v", err)
	}
	return b
}

// helper to determine if recovery window is valid after taking backup
func checkRecoveryWindow(
	ctx context.Context,
	t *testing.T,
	k8sClient *kubernetes.K8sClient,
	ns string,
	name string,
	same bool,
) {
	stanza, err := pgbackrest.GetStanza(ctx, k8sClient, name, ns)
	if err != nil {
		t.Fatalf("failed to get stanza: %v", err)
	}
	fBackup := stanza.Status.RecoveryWindow.FirstBackup
	lBackup := stanza.Status.RecoveryWindow.LastBackup
	if fBackup.Timestamp.Start == 0 || ((fBackup == lBackup) != same) {
		t.Fatal("registered backup information into recovery window are invalid")
	}
}

// basic verification to ensure we can use our plugin with a cluster
func TestDeployInstance(t *testing.T) {
	log.SetLogger(zap.New(zap.WriteTo(os.Stdout), zap.UseDevMode(true)))
	k8sClient, err := kubernetes.Client()
	if k8sClient == nil || err != nil {
		t.Fatalf("kubernetes client not initialized: %v", err)
	}
	ctx := context.Background()
	log.FromContext(ctx)
	// first create a secret
	secret, err := createSecret(ctx, k8sClient, NS, "pgbackrest-s3-secret", _S3_DATA_SECRET)
	if err != nil {
		t.Fatalf("failed to create secret: %v", err)
	}
	defer func() {
		if err := k8sClient.Delete(ctx, secret); err != nil {
			t.Fatal("can't delete secret")
		}
	}()

	// Create a new stanza
	s, err := pgbackrest.CreateStanzaConfig(
		ctx,
		*k8sClient,
		"stanza",
		NS,
		minio.NewS3Repositories("stanza"),
		nil,
		true,
	)
	if err != nil {
		panic(err.Error())
	}
	defer func() {
		if err := k8sClient.Delete(ctx, s); err != nil {
			t.Fatal("can't delete stanza")
		}
	}()

	reqCpuLimit := "500m"
	reqMemoryLimit := "64Mi"
	pc, err := pgbackrest.CreatePluginConfig(
		ctx,
		*k8sClient,
		NS,
		"plugin-config",
		reqCpuLimit,
		reqMemoryLimit,
	)
	if err != nil {
		panic(err.Error())
	}
	defer func() {
		if err := k8sClient.Delete(ctx, pc); err != nil {
			t.Fatal("can't delete plugin configuration")
		}
	}()

	// create a test CloudNativePG Cluster
	clusterName := "cluster-demo"
	podName := clusterName + "-1"

	p := maps.Clone(cluster.DefaultParamater)
	p["pluginConfigRef"] = "plugin-config"

	c, err := cluster.Create(ctx, k8sClient, NS, clusterName, 1, "100M", p, nil)
	if err != nil {
		t.Fatalf("failed to create cluster: %v", err)
	}
	defer func() {
		if err := k8sClient.Delete(ctx, c); err != nil {
			t.Fatal("can't delete cluster")
		}
	}()
	if ready, err := k8sClient.PodIsReady(ctx, NS, podName, 150, 3); err != nil {
		t.Fatalf("error when requesting pod status, %s", err.Error())
	} else if !ready {
		t.Fatal("pod not ready")
	}

	// check if CPU limit is present and equivalent to what we set in plugin config
	pod := corev1.Pod{}
	if err := k8sClient.Get(ctx, types.NamespacedName{Namespace: NS, Name: podName}, &pod); err != nil {
		t.Fatalf("can't retrieve pod: %v", podName)
	}
	wantCpuLimit := resource.MustParse(reqCpuLimit)
	wantMemoryLimit := resource.MustParse(reqMemoryLimit)
	for _, ic := range pod.Spec.InitContainers {
		if ic.Name != "plugin-pgbackrest" {
			continue
		}

		cpuLimit, ok := ic.Resources.Limits[corev1.ResourceCPU]
		if !ok {
			t.Fatalf("CPU limit not set on plugin-pgbackrest container")
		}
		if cpuLimit.Cmp(wantCpuLimit) != 0 {
			t.Errorf(
				"CPU limit not based on plugin config, want: %v, got: %v",
				wantCpuLimit,
				cpuLimit,
			)
		}

		memoryLimit, ok := ic.Resources.Limits[corev1.ResourceMemory]
		if !ok {
			t.Fatalf("Memory limit not set on plugin-pgbackrest container")
		}
		if memoryLimit.Cmp(wantMemoryLimit) != 0 {
			t.Errorf(
				"Memory limit not based on plugin config, want: %v, got: %v",
				wantMemoryLimit,
				memoryLimit,
			)
		}
	}

	// check if pvc has been created (async enabled and MaxProcess>1
	pvc := corev1.PersistentVolumeClaim{}
	if err := k8sClient.Get(
		ctx,
		types.NamespacedName{Namespace: NS, Name: podName + "-pgbackrest-spool"},
		&pvc,
	); err != nil {
		t.Errorf("can't retrieve PVC for pgbackrest spooled WAL")

	}
	if pvc.Status.Phase != corev1.ClaimBound {
		t.Errorf("PVC for pgbackrest spooled WAL not bound")
	}

}

func TestCreateAndRestoreInstance(t *testing.T) {
	log.SetLogger(zap.New(zap.WriteTo(os.Stdout), zap.UseDevMode(true)))
	k8sClient, err := kubernetes.Client()
	if k8sClient == nil || err != nil {
		t.Fatalf("kubernetes client not initialized: %v", err)
	}
	ctx := context.Background()
	log.FromContext(ctx)
	// first create a secret
	secret, err := createSecret(ctx, k8sClient, NS, "pgbackrest-s3-secret", _S3_DATA_SECRET)
	if err != nil {
		t.Fatalf("failed to create secret: %v", err)
	}
	defer func() {
		if err := k8sClient.Delete(ctx, secret); err != nil {
			t.Fatal("can't delete secret")
		}
	}()

	// Create a new stanza
	s, err := pgbackrest.CreateStanzaConfig(
		ctx,
		*k8sClient,
		"stanza-restored",
		NS,
		minio.NewS3Repositories("stanza-restored"),
		nil,
		false,
	)
	if err != nil {
		panic(err.Error())
	}
	defer func() {
		if err := k8sClient.Delete(ctx, s); err != nil {
			t.Fatal("can't delete stanza")
		}
	}()

	// create a test CloudNativePG Cluster
	// name must be different from previous Cluster to avoid conflict
	// in pgBackRest stanza

	clusterName := "cluster-restored"
	podName := clusterName + "-1"

	p := map[string]string{
		"stanzaRef": "stanza-restored",
	}

	c, err := cluster.Create(ctx, k8sClient, NS, clusterName, 1, "100M", p, nil)
	if err != nil {
		t.Fatalf("failed to create cluster: %v", err)
	}
	defer func() {
		if err := k8sClient.Delete(ctx, c); err != nil {
			t.Fatal("can't delete cluster")
		}
	}()
	if ready, err := k8sClient.PodIsReady(ctx, NS, podName, 150, 3); err != nil {
		t.Fatalf("error when requesting pod status, %s", err.Error())
	} else if !ready {
		t.Fatal("pod not ready")
	}

	// take a first backup
	b := takeBackup(ctx, t, k8sClient, NS, clusterName, "backup-01", p)
	defer func() {
		if delErr := k8sClient.Delete(ctx, b); delErr != nil {
			t.Fatalf("can't delete backup-01: %v", delErr)
		}
	}()

	// first & last backup on recovery window shoud be the same
	checkRecoveryWindow(ctx, t, k8sClient, NS, "stanza-restored", true)

	// few helpers func to create table, insert data,...
	createDumpData := func() {
		stdout, stderr, err := command.ExecutePSQLInPostgresContainer(
			ctx,
			*k8sClient.ClientSet,
			k8sClient.Cfg,
			NS,
			podName,
			`CREATE TABLE IF NOT EXISTS wal_test_insert (id SERIAL PRIMARY KEY, data TEXT);
			SELECT pg_switch_wal();
			INSERT INTO wal_test_insert(data) SELECT repeat('x', 1000) FROM generate_series(1, 5000);`,
		)
		if err != nil {
			t.Fatalf(
				"could not execute command, output %v, stderr %v, err %v",
				stdout,
				stderr,
				err.Error(),
			)
		}
	}
	countRow := func() string {
		stdout, stderr, err := command.ExecutePSQLInPostgresContainer(
			ctx,
			*k8sClient.ClientSet,
			k8sClient.Cfg,
			NS,
			podName,
			"SELECT count(*) FROM wal_test_insert;",
		)
		if err != nil {
			t.Fatalf("could not execute command, output %v, stderr %v", stdout, stderr)
		}
		return stdout
	}

	createDumpData()

	// get current date and number of row. now() result  will be used to restore to
	// this point
	curDate, stderr, err := command.ExecutePSQLInPostgresContainer(
		ctx,
		*k8sClient.ClientSet,
		k8sClient.Cfg,
		NS,
		podName,
		"SELECT now();",
	)
	if err != nil {
		t.Fatalf("could not execute command, output %v, stderr %v", curDate, stderr)
	}
	curDate = strings.TrimSpace(curDate)
	numOfRowAfterFirstInsert := countRow()

	// take a second backup
	b2 := takeBackup(ctx, t, k8sClient, NS, clusterName, "backup-02", p)
	defer func() {
		if delErr := k8sClient.Delete(ctx, b2); delErr != nil {
			t.Fatalf("can't delete backup-02: %v", delErr)
		}
	}()

	// then re-instert data to ensure we have some activity on the cluster
	createDumpData()

	// second backup first and last backup into recovery window must differ
	checkRecoveryWindow(ctx, t, k8sClient, NS, "stanza-restored", false)

	// delete cluster, we will recreate it from backup
	if err := k8sClient.Delete(ctx, c); err != nil {
		t.Fatal("can't delete cluster")
	}
	if _, err = k8sClient.PodIsAbsent(ctx, NS, podName, 10, 3); err != nil {
		t.Fatal("can't ensure cluster is absent")
	}

	// recreate cluster from backup (recovery: true)
	_, err = cluster.Create(
		ctx,
		k8sClient,
		NS,
		clusterName,
		1,
		"100M",
		p,
		&cluster.RestoreOption{
			RecoveryTarget: &cloudnativepgv1.RecoveryTarget{TargetTime: curDate},
		},
	)
	if err != nil {
		t.Fatalf("can't recreate cluster from backup, %v", err)
	}
	if ready, err := k8sClient.PodIsReady(ctx, NS, podName, 150, 3); err != nil {
		t.Fatalf("error when requesting pod status, %s", err.Error())
	} else if !ready {
		t.Fatal("pod not ready")
	}
	numberOfRowAferRestore := countRow()
	if numOfRowAfterFirstInsert != numberOfRowAferRestore {
		t.Fatalf(
			"restore error, number of row does not match %v, %v",
			numOfRowAfterFirstInsert,
			numberOfRowAferRestore,
		)
	}

}

func TestAzure(t *testing.T) {
	ctx := context.Background()
	k8sClient, err := kubernetes.Client()
	clusterName := "cluster-azure"
	podName := clusterName + "-1"
	stanza := "stanza-azure"
	azContainer := "azcontainer"
	if err != nil {
		panic("can't init kubernetes client")
	}
	if err := azurite.CreateAzContainer(ctx, *k8sClient, "azurite", azContainer); err != nil {
		panic(err.Error())
	}
	secret, err := createSecret(ctx, k8sClient, NS, "pgbackrest-azure-secret", _AZURE_DATA_SECRET)
	if err != nil {
		panic(err.Error())

	}
	defer func() {
		if err := k8sClient.Delete(ctx, secret); err != nil {
			t.Fatal("can't delete secret")
		}
	}()

	s, err := pgbackrest.CreateStanzaConfig(
		ctx,
		*k8sClient,
		stanza,
		NS,
		nil,
		azurite.NewAzureRepositories(azContainer),
		false,
	)
	if err != nil {
		panic(err.Error())
	}
	defer func() {
		if err := k8sClient.Delete(ctx, s); err != nil {
			t.Fatal("can't delete stanza")
		}
	}()

	p := map[string]string{
		"stanzaRef": stanza,
	}

	c, err := cluster.Create(ctx, k8sClient, NS, clusterName, 1, "100M", p, nil)
	if err != nil {
		t.Fatalf("failed to create cluster: %v", err)
	}
	defer func() {
		if err := k8sClient.Delete(ctx, c); err != nil {
			t.Fatal("can't delete cluster")
		}
	}()
	if ready, err := k8sClient.PodIsReady(ctx, NS, podName, 150, 3); err != nil {
		t.Fatalf("error when requesting pod status, %s", err.Error())
	} else if !ready {
		t.Fatal("pod not ready")
	}

	b := takeBackup(ctx, t, k8sClient, NS, clusterName, "azure-backup-01", p)
	defer func() {
		if err := k8sClient.Delete(ctx, b); err != nil {
			t.Fatal("can't delete cluster")
		}
	}()
}
