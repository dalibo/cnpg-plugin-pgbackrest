package main

import (
	"context"
	"maps"
	"os"
	"reflect"
	"testing"

	"github.com/dalibo/cnpg-i-pgbackrest/test/e2e/internal/certmanager"
	"github.com/dalibo/cnpg-i-pgbackrest/test/e2e/internal/cluster"
	"github.com/dalibo/cnpg-i-pgbackrest/test/e2e/internal/cnpg"
	"github.com/dalibo/cnpg-i-pgbackrest/test/e2e/internal/kubernetes"
	"github.com/dalibo/cnpg-i-pgbackrest/test/e2e/internal/minio"
	"github.com/dalibo/cnpg-i-pgbackrest/test/e2e/internal/pgbackrest"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	logcr "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

// Deploy CNGP operator, certmanager, minio and our plugins
func setup() error {
	k8sClient, err := kubernetes.Client()
	if err != nil {
		panic("can't init kubernetes client")
	}
	s := kubernetes.InstallSpec{ManifestUrl: "https://raw.githubusercontent.com/cloudnative-pg/cloudnative-pg/release-1.25/releases/cnpg-1.25.1.yaml"}
	if err := cnpg.Install(*k8sClient, s); err != nil {
		panic("can't install CNPG")
	}
	s = kubernetes.InstallSpec{ManifestUrl: "https://github.com/cert-manager/cert-manager/releases/download/v1.16.2/cert-manager.yaml"}
	if err := certmanager.Install(*k8sClient, s); err != nil {
		panic("can't install certmanager")
	}
	if err = minio.Install(*k8sClient); err != nil {
		panic("can't install minio")
	}
	// install our pgbackrest plugin from kubernetes directory at the root
	// of the repository
	path, err := os.Getwd()
	s = kubernetes.InstallSpec{ManifestUrl: path + "/../../kubernetes"}
	if err := pgbackrest.Install(*k8sClient, s); err != nil {
		panic("can't deploy plugin")
	}
	return nil
}

func createSecret(ctx context.Context, k8sClient *kubernetes.K8sClient, namespace string) (*v1.Secret, error) {
	// TODO: move that ?
	secret := &v1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pgbackrest-s3-secret",
			Namespace: namespace,
		},
		Type: v1.SecretTypeOpaque,
		StringData: map[string]string{
			"key":        minio.ACCESS_KEY,
			"key-secret": minio.SECRET_KEY,
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
	logcr.SetLogger(zap.New(zap.WriteTo(os.Stdout), zap.UseDevMode(true)))
	// basic verification to ensure our plugin is present
	k8sClient, err := kubernetes.Client()
	if k8sClient == nil || err != nil {
		t.Errorf("error kubernetes client not initialized")
	}
	// basic check for deployment
	fqdn := types.NamespacedName{Name: "pgbackrest-controller", Namespace: "cnpg-system"}
	obj := &appsv1.Deployment{}
	k8sClient.Get(context.TODO(), fqdn, obj)
	var want int32 = 1
	got := obj.Status.ReadyReplicas
	if got != want {
		t.Errorf("error no Pod for pgbackrest plugin want: %v, got: %v", want, got)
	}

	// verify service creation
	fqdn = types.NamespacedName{Name: "pgbackrest", Namespace: "cnpg-system"}
	svc := &corev1.Service{}
	err = k8sClient.Get(context.TODO(), fqdn, svc)
	if err != nil {
		t.Errorf("error no service for pgbackrest found %s", err.Error())
	}
	want_labels := map[string]string{
		"app":                "pgbackrest-controller",
		"cnpg.io/pluginName": "pgbackrest.dalibo.com",
	}
	if !reflect.DeepEqual(svc.Labels, want_labels) {
		t.Errorf("error labels not valid  %v", svc.Labels)
	}
	k8sClient.Get(context.TODO(), fqdn, obj)
}

// basic verification to ensure we can use our plugin with a cluster
func TestDeployInstance(t *testing.T) {
	logcr.SetLogger(zap.New(zap.WriteTo(os.Stdout), zap.UseDevMode(true)))
	k8sClient, err := kubernetes.Client()
	if k8sClient == nil || err != nil {
		t.Errorf("error kubernetes client not initialized")
	}
	ns := "default"
	ctx := context.TODO()
	// first create a secret
	secret, err := createSecret(ctx, k8sClient, ns)
	if err != nil {
		t.Error(err.Error())
	}

	// create a test CloudNativePG Cluster
	clusterName := "cluster-demo"
	p := maps.Clone(cluster.DefaultParamater)
	p["s3-repo-path"] = "/" + clusterName
	c, err := cluster.Create(k8sClient, ns, clusterName, 1, "100M", p)
	if err != nil {
		t.Error(err.Error())
	}
	if ready, err := k8sClient.PodsIsReady(ns, clusterName+"-1", 30, 2); err != nil {
		t.Errorf("error when requesting pod status, %s", err.Error())
	} else if !ready {
		t.Error("pod not ready")
	}
	// more verification can be done here (before deleting the cluster)

	// delete created ressources
	k8sClient.Delete(ctx, c)
	k8sClient.Delete(ctx, secret)
}
