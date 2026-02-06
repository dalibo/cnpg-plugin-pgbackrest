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
) *apipgbackrest.Stanza {
	verifyTls := false
	stanza := &apipgbackrest.Stanza{
		TypeMeta: metav1.TypeMeta{
			Kind:       "stanza",
			APIVersion: "pgbacrest.dalibo.com/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
		Spec: apipgbackrest.StanzaSpec{
			Configuration: pgbackrest.Stanza{
				Name: "my_stanza",
				S3Repositories: []pgbackrest.S3Repository{
					{
						Bucket:    minio.BUCKET_NAME,
						Endpoint:  minio.SVC_NAME,
						Region:    "us-east-1",
						VerifyTLS: &verifyTls,
						UriStyle:  "path",
						RepoPath:  "/repo01" + name,
						RetentionPolicy: pgbackrest.Retention{
							FullType: "count",
							Full:     7,
						},
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
						Cipher: &pgbackrest.CipherConfig{
							Type: "aes-256-cbc",
							PassReference: &api.SecretKeySelector{
								LocalObjectReference: api.LocalObjectReference{
									Name: "pgbackrest-s3-secret",
								},
								Key: "ENCRYPTION_PASS",
							},
						},
					},
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
	return stanza
}

func CreateStanzaConfig(
	ctx context.Context,
	k8sClient kubernetes.K8sClient,
	name string,
	ns string,
) (*apipgbackrest.Stanza, error) {
	stanza := NewStanzaConfig(k8sClient, name, ns)
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
