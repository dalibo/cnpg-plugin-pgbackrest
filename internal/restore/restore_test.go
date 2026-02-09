// SPDX-FileCopyrightText: 2026 Dalibo <contact@dalibo.com>
//
// SPDX-License-Identifier: Apache-2.0
package restore

import (
	"testing"

	cnpgv1 "github.com/cloudnative-pg/cloudnative-pg/api/v1"
	"github.com/dalibo/cnpg-i-pgbackrest/internal/pgbackrest"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func newCluster(bootstrap *cnpgv1.BootstrapConfiguration) *cnpgv1.Cluster {
	return &cnpgv1.Cluster{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Cluster",
			APIVersion: "postgresql.cnpg.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cl",
			Namespace: "def",
		},
		Spec: cnpgv1.ClusterSpec{
			Bootstrap: bootstrap,
		},
	}
}

func TestRecoveryTargetToRestoreOption(t *testing.T) {
	testCases := []struct {
		desc    string
		cluster *cnpgv1.Cluster
		want    pgbackrest.RestoreOptions
	}{
		{
			desc:    "cluster without bootstrap section",
			cluster: newCluster(nil),
			want:    pgbackrest.RestoreOptions{},
		},
		{
			desc: "cluster with bootstrap section, but no target defined",
			cluster: newCluster(&cnpgv1.BootstrapConfiguration{
				Recovery: &cnpgv1.BootstrapRecovery{
					Source: "mysource",
				},
			}),
			want: pgbackrest.RestoreOptions{},
		},
		{
			desc: "cluster with bootstrap section and target LSN",
			cluster: newCluster(&cnpgv1.BootstrapConfiguration{
				Recovery: &cnpgv1.BootstrapRecovery{
					Source: "mysource",
					RecoveryTarget: &cnpgv1.RecoveryTarget{
						TargetLSN: "0/169EC40",
					},
				},
			}),
			want: pgbackrest.RestoreOptions{
				Target: "0/169EC40",
				Type:   "lsn",
			},
		},
		{
			desc: "cluster with bootstrap section and target time",
			cluster: newCluster(&cnpgv1.BootstrapConfiguration{
				Recovery: &cnpgv1.BootstrapRecovery{
					Source: "mysource",
					RecoveryTarget: &cnpgv1.RecoveryTarget{
						TargetTime: "2026-01-01 09:30:00+00",
					},
				},
			}),
			want: pgbackrest.RestoreOptions{
				Target: "2026-01-01 09:30:00+00",
				Type:   "time",
			},
		},
		{
			desc: "cluster with bootstrap section and specific timeline",
			cluster: newCluster(&cnpgv1.BootstrapConfiguration{
				Recovery: &cnpgv1.BootstrapRecovery{
					Source: "mysource",
					RecoveryTarget: &cnpgv1.RecoveryTarget{
						TargetLSN: "0/169EC40",
						TargetTLI: "2",
					},
				},
			}),
			want: pgbackrest.RestoreOptions{
				Target:         "0/169EC40",
				Type:           "lsn",
				TargetTimeline: "2",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			got := recoveryTargetToRestoreOptions(tc.cluster)
			if tc.want != got {
				t.Errorf("error want %v, got %v", tc.want, got)
			}
		})
	}
}
