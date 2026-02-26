// SPDX-FileCopyrightText: 2025 Dalibo <contact@dalibo.com>
//
// SPDX-License-Identifier: Apache-2.0

package pgbackrest

import (
	"context"

	apipgbackrest "github.com/dalibo/cnpg-i-pgbackrest/api/v1"
	pgbackrest "github.com/dalibo/cnpg-i-pgbackrest/internal/pgbackrest/api"
	"github.com/dalibo/cnpg-i-pgbackrest/test/e2e/internal/kubernetes"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
)

func Install(
	ctx context.Context,
	k8sClient kubernetes.K8sClient,
	installSpec kubernetes.InstallSpec,
) error {
	if err := kubernetes.Apply(installSpec); err != nil {
		return err
	}
	_, err := k8sClient.DeploymentIsReady(ctx, "cnpg-system", "pgbackrest-controller", 15, 2)
	return err
}

func NewStanzaConfig(
	k8sClient kubernetes.K8sClient,
	name string,
	ns string,
	s3Repo []pgbackrest.S3Repository,
	azRepo []pgbackrest.AzureRepository,
	async bool,
) *apipgbackrest.Stanza {
	stanza := &apipgbackrest.Stanza{
		TypeMeta: metav1.TypeMeta{
			Kind:       "stanza",
			APIVersion: "pgbackrest.dalibo.com/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
		Spec: apipgbackrest.StanzaSpec{
			Configuration: pgbackrest.Stanza{
				Name:              "my_stanza",
				S3Repositories:    s3Repo,
				AzureRepositories: azRepo,
				Archive: pgbackrest.ArchiveOption{
					Async: async,
				},
				Compress: &pgbackrest.CompressConfig{
					Type:  ptr.To("lz4"),
					Level: 7,
				},
				Delta: false,
				CustomEnvVar: map[string]string{
					"PGBACKREST_ARCHIVE_TIMEOUT": "30",
				},
			},
		},
	}
	if async {
		stanza.Spec.Configuration.ProcessMax = 2
	}
	return stanza
}

func CreateStanzaConfig(
	ctx context.Context,
	k8sClient kubernetes.K8sClient,
	name string,
	ns string,
	s3Repo []pgbackrest.S3Repository,
	azRepo []pgbackrest.AzureRepository,
	async bool,
) (*apipgbackrest.Stanza, error) {
	stanza := NewStanzaConfig(k8sClient, name, ns, s3Repo, azRepo, async)
	if err := k8sClient.Create(ctx, stanza); err != nil {
		return nil, err
	}
	return stanza, nil
}

func GetStanza(
	ctx context.Context,
	k8sClient *kubernetes.K8sClient,
	name string,
	ns string,
) (*apipgbackrest.Stanza, error) {
	var stanza apipgbackrest.Stanza
	fqdn := types.NamespacedName{
		Name:      name,
		Namespace: ns,
	}
	if err := k8sClient.Get(ctx, fqdn, &stanza); err != nil {
		return nil, err
	}
	return &stanza, nil
}

func NewPluginConfig(ns, name, cpu_limit, memory_limit string) *apipgbackrest.PluginConfig {
	return &apipgbackrest.PluginConfig{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PluginConfig",
			APIVersion: "pgbackrest.dalibo.com/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Name:      name,
		},
		Spec: apipgbackrest.PluginConfigSpec{
			Resources: &corev1.ResourceRequirements{
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse(cpu_limit),
					corev1.ResourceMemory: resource.MustParse(memory_limit),
				},
			},
			StorageConfig: &apipgbackrest.StorageConfig{
				StorageClass: "standard",
			},
		},
	}
}

func CreatePluginConfig(
	ctx context.Context,
	k8sClient kubernetes.K8sClient,
	ns, name, cpu_limit, memory_limit string,
) (*apipgbackrest.PluginConfig, error) {
	pc := NewPluginConfig(ns, name, cpu_limit, memory_limit)
	if err := k8sClient.Create(ctx, pc); err != nil {
		return nil, err
	}
	return pc, nil
}
