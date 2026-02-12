// SPDX-FileCopyrightText: 2025 Dalibo <contact@dalibo.com>
//
// SPDX-License-Identifier: Apache-2.0

package cluster

import (
	"context"
	"fmt"

	cloudnativepgv1 "github.com/cloudnative-pg/api/pkg/api/v1"
	"github.com/dalibo/cnpg-i-pgbackrest/test/e2e/internal/common"
	"github.com/dalibo/cnpg-i-pgbackrest/test/e2e/internal/kubernetes"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
)

var DefaultParamater map[string]string = map[string]string{
	"stanzaRef": "stanza",
}

type RestoreOption struct {
	RecoveryTarget *cloudnativepgv1.RecoveryTarget
}

func New(
	namespace string,
	name string,
	nbrInstances int,
	size string,
	pluginParam map[string]string,
	restore *RestoreOption,
) *cloudnativepgv1.Cluster {
	pluginConfig := []cloudnativepgv1.PluginConfiguration{
		{
			Name:          "pgbackrest.dalibo.com",
			IsWALArchiver: ptr.To(true),
			Parameters:    pluginParam,
		},
	}
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
			Plugins:         pluginConfig,
			PostgresConfiguration: cloudnativepgv1.PostgresConfiguration{
				Parameters: map[string]string{},
			},
			StorageConfiguration: cloudnativepgv1.StorageConfiguration{
				Size: size,
			},
		},
	}
	if restore != nil {
		externalName := "origin"
		cluster.Spec.Bootstrap = &cloudnativepgv1.BootstrapConfiguration{
			Recovery: &cloudnativepgv1.BootstrapRecovery{
				Source: externalName,
			},
		}
		if restore.RecoveryTarget != nil {
			cluster.Spec.Bootstrap.Recovery.RecoveryTarget = restore.RecoveryTarget
		}
		cluster.Spec.ExternalClusters = []cloudnativepgv1.ExternalCluster{
			{
				Name:                externalName,
				PluginConfiguration: &pluginConfig[0],
			},
		}
	}
	return cluster
}

func Create(
	ctx context.Context,
	k8sClient *kubernetes.K8sClient,
	namespace string,
	name string,
	nbrInstances int,
	size string,
	pluginParam map[string]string,
	restore *RestoreOption,
) (*cloudnativepgv1.Cluster, error) {
	m := New(namespace, name, nbrInstances, size, pluginParam, restore)
	if err := k8sClient.Create(ctx, m); err != nil {
		return nil, err
	}
	return m, nil
}

type BackupInfo struct {
	Cluster   string
	Name      string
	Namespace string
	Params    map[string]string
}

func (b BackupInfo) Backup(
	ctx context.Context,
	kClient *kubernetes.K8sClient,
) (*cloudnativepgv1.Backup, error) {
	backup := &cloudnativepgv1.Backup{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Backup",
			APIVersion: "postgresql.cnpg.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      b.Name,
			Namespace: b.Namespace,
		},
		Spec: cloudnativepgv1.BackupSpec{
			Cluster: cloudnativepgv1.LocalObjectReference{
				Name: b.Cluster,
			},
			Method: "plugin",
			PluginConfiguration: &cloudnativepgv1.BackupPluginConfiguration{
				Name:       "pgbackrest.dalibo.com",
				Parameters: b.Params,
			},
		},
	}
	if err := kClient.Create(ctx, backup); err != nil {
		return nil, err
	}
	return backup, nil
}

func (b BackupInfo) IsDone(
	ctx context.Context,
	kClient *kubernetes.K8sClient,
	r *common.Retrier,
) (bool, error) {
	waitedRessource := &cloudnativepgv1.Backup{}
	fqdn := types.NamespacedName{Name: b.Name, Namespace: b.Namespace}
	for range r.MaxRetry {
		err := kClient.Get(ctx, fqdn, waitedRessource)
		if errors.IsNotFound(err) {
			r.Wait()
			continue
		} else if err != nil {
			return false, err
		} else if waitedRessource.Status.Phase == "completed" {
			return true, nil
		}
		r.Wait()
	}
	return false, fmt.Errorf("%s", waitedRessource.Status.Phase)
}
