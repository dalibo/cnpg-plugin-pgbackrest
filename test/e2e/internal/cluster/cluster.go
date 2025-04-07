// SPDX-FileCopyrightText: 2025 Dalibo <contact@dalibo.com>
//
// SPDX-License-Identifier: Apache-2.0

package cluster

import (
	"context"
	"fmt"
	"time"

	cloudnativepgv1 "github.com/cloudnative-pg/api/pkg/api/v1"
	"github.com/dalibo/cnpg-i-pgbackrest/test/e2e/internal/kubernetes"
	"github.com/dalibo/cnpg-i-pgbackrest/test/e2e/internal/minio"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
)

var DefaultParamater map[string]string = map[string]string{
	"s3-bucket":     minio.BUCKET_NAME,
	"s3-endpoint":   minio.SVC_NAME,
	"s3-region":     "us-east-1",
	"s3-verify-tls": "false",
	"s3-uri-style":  "path",
	"stanza":        "pgbackrest",
}

func New(namespace string, name string, nbrInstances int, size string, pluginParam map[string]string) *cloudnativepgv1.Cluster {

	cluster := &cloudnativepgv1.Cluster{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Cluster",
			APIVersion: "postgresql.cnpg.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: cloudnativepgv1.ClusterSpec{
			Instances:       nbrInstances,
			ImagePullPolicy: corev1.PullIfNotPresent,
			Plugins: []cloudnativepgv1.PluginConfiguration{
				{
					Name:       "pgbackrest.dalibo.com",
					Parameters: pluginParam,
				},
			},
			PostgresConfiguration: cloudnativepgv1.PostgresConfiguration{
				Parameters: map[string]string{},
			},
			StorageConfiguration: cloudnativepgv1.StorageConfiguration{
				Size: size,
			},
		}}
	return cluster
}

func Create(k8sClient *kubernetes.K8sClient, namespace string, name string, nbrInstances int, size string, pluginParam map[string]string) (*cloudnativepgv1.Cluster, error) {
	m := New(namespace, name, nbrInstances, size, pluginParam)
	if err := k8sClient.Create(context.TODO(), m); err != nil {
		return nil, err
	}
	return m, nil
}

func Backup(cl *kubernetes.K8sClient, cluster string, namespacedName types.NamespacedName) (*cloudnativepgv1.Backup, error) {
	b := cloudnativepgv1.Backup{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Backup",
			APIVersion: "postgresql.cnpg.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      namespacedName.Name,
			Namespace: namespacedName.Namespace,
		},
		Spec: cloudnativepgv1.BackupSpec{
			Cluster: cloudnativepgv1.LocalObjectReference{
				Name: cluster,
			},
			Method: "plugin",
			PluginConfiguration: &cloudnativepgv1.BackupPluginConfiguration{
				Name: "pgbackrest.dalibo.com",
			},
		},
	}
	if err := cl.Create(context.TODO(), &b); err != nil {
		return nil, err
	}
	return &b, nil
}

// TODO: Make it more generic
func BackupCompleted(ctx context.Context, cl *kubernetes.K8sClient, namespacedName types.NamespacedName, retryInterval time.Duration) error {
	waitedRessource := &cloudnativepgv1.Backup{}
	errMsg := "unable to check if %s is completed, error: %w"
	if err := wait.PollUntilContextCancel(ctx, retryInterval, false,
		func(ctx context.Context) (bool, error) {
			if err := cl.Get(ctx, namespacedName, waitedRessource); err != nil {
				return false, fmt.Errorf(errMsg, namespacedName, err)
			}
			if waitedRessource.Status.Phase == cloudnativepgv1.BackupPhaseCompleted {
				return true, nil
			}
			return false, nil
		},
	); err != nil {
		return fmt.Errorf(errMsg, namespacedName, err)
	}
	return nil
}
