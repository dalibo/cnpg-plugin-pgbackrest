// SPDX-FileCopyrightText: 2026 Dalibo <contact@dalibo.com>
//
// SPDX-License-Identifier: Apache-2.0
package operator

import (
	"testing"

	machineryapi "github.com/cloudnative-pg/machinery/pkg/api"
	"github.com/cloudnative-pg/machinery/pkg/stringset"
	apipgbackrest "github.com/dalibo/cnpg-i-pgbackrest/api/v1"
	pgbackrest "github.com/dalibo/cnpg-i-pgbackrest/internal/pgbackrest/api"
)

func TestGetSecrets(t *testing.T) {
	t.Run("with secrets", func(t *testing.T) {
		stanza := apipgbackrest.Stanza{
			Spec: apipgbackrest.StanzaSpec{
				Configuration: pgbackrest.Stanza{
					S3Repositories: []pgbackrest.S3Repository{
						{
							SecretRef: &pgbackrest.S3SecretRef{
								AccessKeyIDReference: &machineryapi.SecretKeySelector{
									LocalObjectReference: machineryapi.LocalObjectReference{
										Name: "access-key-1",
									},
									Key: "key",
								},
								SecretAccessKeyReference: &machineryapi.SecretKeySelector{
									LocalObjectReference: machineryapi.LocalObjectReference{
										Name: "secret-key-1",
									},
									Key: "key",
								},
							},
						},
					},
					AzureRepositories: []pgbackrest.AzureRepository{
						{
							SecretRef: &pgbackrest.AzureSecretRef{
								KeyReference: &machineryapi.SecretKeySelector{
									LocalObjectReference: machineryapi.LocalObjectReference{
										Name: "azure-key-1",
									},
									Key: "key",
								},
							},
						},
					},
				},
			},
		}

		s := stringset.New()
		getSecrets(stanza, s)

		expected := []string{"access-key-1", "secret-key-1", "azure-key-1"}

		for _, name := range expected {
			if !s.Has(name) {
				t.Errorf("expected secret %q to be in set, but it was missing", name)
			}
		}

		if s.Len() != len(expected) {
			t.Errorf("expected %d secrets in set, got %d", len(expected), s.Len())
		}
	})

	t.Run("no secrets", func(t *testing.T) {
		stanza := apipgbackrest.Stanza{
			Spec: apipgbackrest.StanzaSpec{
				Configuration: pgbackrest.Stanza{
					S3Repositories:    []pgbackrest.S3Repository{{SecretRef: nil}},
					AzureRepositories: []pgbackrest.AzureRepository{{SecretRef: nil}},
				},
			},
		}

		s := stringset.New()
		getSecrets(stanza, s)

		if s.Len() != 0 {
			t.Errorf("expected no secrets in set, got %d: %v", s.Len(), s)
		}
	})
}
