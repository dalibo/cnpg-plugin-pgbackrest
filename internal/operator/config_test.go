// SPDX-FileCopyrightText: 2025 Dalibo <contact@dalibo.com>
//
// SPDX-License-Identifier: Apache-2.0

package operator

import (
	"context"
	"slices"
	"testing"

	machineryapi "github.com/cloudnative-pg/machinery/pkg/api"
	apipgbackrest "github.com/dalibo/cnpg-i-pgbackrest/api/v1"
	pgbackrest "github.com/dalibo/cnpg-i-pgbackrest/internal/pgbackrest/api"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func buildFakeClient() client.Client {
	aKey := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "access-key-secret",
			Namespace: "default",
		},
		Data: map[string][]byte{
			"key": []byte("AKIA123"),
		},
	}
	sKey := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "secret-key-secret",
			Namespace: "default",
		},
		Data: map[string][]byte{
			"key": []byte("SECRET123"),
		},
	}
	return fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(aKey, sKey).
		Build()
}
func buildRepo() *apipgbackrest.Stanza {
	return &apipgbackrest.Stanza{
		TypeMeta: metav1.TypeMeta{
			Kind:       "stanza",
			APIVersion: "pgbacrest.dalibo.com/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "stanza",
			Namespace: "default",
		},
		Spec: apipgbackrest.StanzaSpec{
			Configuration: pgbackrest.Stanza{
				Name: "myStanza",
				S3Repositories: []pgbackrest.S3Repository{
					{
						Bucket:   "mybucket",
						Endpoint: "https://s3.example.com",
						Region:   "us-east-1",
						RepoPath: "/backups",
						SecretRef: &pgbackrest.S3SecretRef{
							AccessKeyIDReference: &machineryapi.SecretKeySelector{
								LocalObjectReference: machineryapi.LocalObjectReference{
									Name: "access-key-secret",
								},
								Key: "key",
							},
							SecretAccessKeyReference: &machineryapi.SecretKeySelector{
								LocalObjectReference: machineryapi.LocalObjectReference{
									Name: "secret-key-secret",
								},
								Key: "key",
							},
						},
					},
				},
			},
		},
	}
}
func TestGetEnvVarConfig(t *testing.T) {
	ctx := context.Background()
	r := buildRepo()
	c := buildFakeClient()
	env, err := GetEnvVarConfig(ctx, *r, c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := []string{
		"PGBACKREST_REPO1_S3_BUCKET=mybucket",
		"PGBACKREST_REPO1_S3_KEY=AKIA123",
		"PGBACKREST_REPO1_S3_KEY_SECRET=SECRET123",
		"PGBACKREST_REPO1_TYPE=s3",
	}
	for _, e := range expected {
		if !slices.Contains(env, e) {
			t.Errorf("expected env var %v not found in: %v", e, env)
		}
	}
}
