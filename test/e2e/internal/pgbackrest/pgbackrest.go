// SPDX-FileCopyrightText: 2025 Dalibo <contact@dalibo.com>
//
// SPDX-License-Identifier: Apache-2.0

package pgbackrest

import (
	"context"

	"github.com/cloudnative-pg/machinery/pkg/api"
	apipgbackrest "github.com/dalibo/cnpg-i-pgbackrest/api/v1"
	pgbackrest "github.com/dalibo/cnpg-i-pgbackrest/internal/pgbackrest/api"
	"github.com/dalibo/cnpg-i-pgbackrest/test/e2e/internal/kubernetes"
	"github.com/dalibo/cnpg-i-pgbackrest/test/e2e/internal/minio"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Install(k8sClient kubernetes.K8sClient, installSpec kubernetes.InstallSpec) error {
	if err := kubernetes.Apply(installSpec); err != nil {
		return err
	}
	_, err := k8sClient.DeploymentIsReady("cnpg-system", "pgbackrest-controller", 15, 2)
	return err
}

func NewRepoConfig(
	k8sClient kubernetes.K8sClient,
	name string,
	ns string,
) *apipgbackrest.Repository {
	verifyTls := false
	repo := &apipgbackrest.Repository{
		TypeMeta: metav1.TypeMeta{
			Kind:       "repository",
			APIVersion: "pgbacrest.dalibo.com/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
		Spec: apipgbackrest.RepositorySpec{
			Configuration: pgbackrest.Repository{
				Stanza: "my_stanza",
				S3Repositories: []pgbackrest.S3Repository{
					{
						Bucket:    minio.BUCKET_NAME,
						Endpoint:  minio.SVC_NAME,
						Region:    "us-east-1",
						VerifyTLS: &verifyTls,
						UriStyle:  "path",
						RepoPath:  "/repo01",
						SecretRef: &pgbackrest.SecretRef{
							AccessKeyIDReference: &api.SecretKeySelector{
								LocalObjectReference: api.LocalObjectReference{
									Name: "pgbackrest-s3-secret",
								},
								Key: "ACCESS_KEY_ID",
							},
							SecretAccessKeyReference: &api.SecretKeySelector{
								LocalObjectReference: api.LocalObjectReference{
									Name: "pgbackrest-s3-secret",
								},
								Key: "ACCESS_SECRET_KEY",
							},
						},
					},
				},
			},
		},
	}
	return repo
}

func CreateRepoConfig(
	k8sClient kubernetes.K8sClient,
	name string,
	ns string,
) (*apipgbackrest.Repository, error) {
	repo := NewRepoConfig(k8sClient, name, ns)
	if err := k8sClient.Create(context.TODO(), repo); err != nil {
		return nil, err
	}
	return repo, nil
}
