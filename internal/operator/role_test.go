// SPDX-FileCopyrightText: 2026 Dalibo <contact@dalibo.com>
//
// SPDX-License-Identifier: Apache-2.0
package operator

import (
	"slices"
	"testing"

	machineryapi "github.com/cloudnative-pg/machinery/pkg/api"
	"github.com/cloudnative-pg/machinery/pkg/stringset"
	pgbackrestapi "github.com/dalibo/cnpg-i-pgbackrest/api/v1"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGetSecrets(t *testing.T) {
	t.Run("with secrets", func(t *testing.T) {
		stanza := pgbackrestapi.Stanza{
			Spec: pgbackrestapi.StanzaSpec{
				Configuration: pgbackrestapi.StanzaConfiguration{
					S3Repositories: []pgbackrestapi.S3Repository{
						{
							SecretRef: &pgbackrestapi.S3SecretRef{
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
					AzureRepositories: []pgbackrestapi.AzureRepository{
						{
							SecretRef: &pgbackrestapi.AzureSecretRef{
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
		stanza := pgbackrestapi.Stanza{
			Spec: pgbackrestapi.StanzaSpec{
				Configuration: pgbackrestapi.StanzaConfiguration{
					S3Repositories:    []pgbackrestapi.S3Repository{{SecretRef: nil}},
					AzureRepositories: []pgbackrestapi.AzureRepository{{SecretRef: nil}},
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

func TestBuildK8SRole(t *testing.T) {
	testNs := "test-ns"
	testsCase := []struct {
		name          string
		ns            string
		clusterName   string
		stanzas       []pgbackrestapi.Stanza
		pluginconf    *pgbackrestapi.PluginConfig
		wantRuleCount int
	}{
		{
			name:        "with plugin config",
			ns:          testNs,
			clusterName: "cluster1",
			stanzas: []pgbackrestapi.Stanza{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "stanza1",
						Namespace: testNs,
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "stanza2",
						Namespace: testNs,
					},
				},
			},
			pluginconf: &pgbackrestapi.PluginConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "plugin1",
					Namespace: testNs,
				},
			},
			wantRuleCount: 4,
		},
		{
			name:        "without plugin config",
			ns:          testNs,
			clusterName: "cluster1",
			stanzas: []pgbackrestapi.Stanza{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "stanza1",
						Namespace: testNs,
					},
				},
			},
			pluginconf:    nil,
			wantRuleCount: 3,
		},
		{
			name:          "no stanzas",
			ns:            testNs,
			clusterName:   "cluster1",
			stanzas:       nil,
			pluginconf:    nil,
			wantRuleCount: 3,
		},
	}

	for _, tt := range testsCase {
		t.Run(tt.name, func(t *testing.T) {
			role := BuildK8SRole(tt.ns, tt.clusterName, tt.stanzas, tt.pluginconf)

			if role == nil {
				t.Fatalf("want role, got nil")
			}

			// Metadata checks
			if role.Namespace != tt.ns {
				t.Fatalf("want namespace %s, got %s", tt.ns, role.Namespace)
			}

			wantName := GetRBACName(tt.clusterName)
			if role.Name != wantName {
				t.Fatalf("want name %s, got %s", wantName, role.Name)
			}

			if len(role.Rules) != tt.wantRuleCount {
				t.Fatalf("want %d rules, got %d", tt.wantRuleCount, len(role.Rules))
			}

			// check stanza rule
			var stanzaRule *rbacv1.PolicyRule
			for i := range role.Rules {
				r := role.Rules[i]
				if slices.Contains(r.Resources, "stanzas") {
					stanzaRule = &r
					break
				}
			}

			if stanzaRule == nil {
				t.Fatalf("stanza rule not found")
			}

			wantStanzaNames := make([]string, len(tt.stanzas))
			for i, s := range tt.stanzas {
				wantStanzaNames[i] = s.Name
			}
			slices.Sort(stanzaRule.ResourceNames)
			slices.Sort(wantStanzaNames)

			if slices.Compare(stanzaRule.ResourceNames, wantStanzaNames) != 0 {
				t.Fatalf("want stanza names %v, got %v",
					wantStanzaNames, stanzaRule.ResourceNames)
			}

			if tt.pluginconf != nil {
				var pluginRule *rbacv1.PolicyRule
				for i := range role.Rules {
					r := role.Rules[i]
					if slices.Contains(r.Resources, "pluginconfigs") {
						pluginRule = &r
						break
					}
				}

				if pluginRule == nil {
					t.Fatalf("want plugin rule, not found")
				}

				want := []string{tt.pluginconf.Name}
				if slices.Compare(pluginRule.ResourceNames, want) != 0 {
					t.Fatalf("want plugin resource names %v, got %v",
						want, pluginRule.ResourceNames)
				}
			}
		})
	}
}
