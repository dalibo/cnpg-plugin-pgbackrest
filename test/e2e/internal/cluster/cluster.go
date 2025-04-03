// SPDX-FileCopyrightText: 2025 Dalibo <contact@dalibo.com>
//
// SPDX-License-Identifier: Apache-2.0

package cluster

import (
	"context"

	cloudnativepgv1 "github.com/cloudnative-pg/api/pkg/api/v1"
	"github.com/dalibo/cnpg-i-pgbackrest/test/e2e/internal/kubernetes"
	"github.com/dalibo/cnpg-i-pgbackrest/test/e2e/internal/minio"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
