// SPDX-FileCopyrightText: 2026 Dalibo <contact@dalibo.com>
//
// SPDX-License-Identifier: Apache-2.0
package instance

import (
	"context"
	"testing"

	cnpgv1 "github.com/cloudnative-pg/cloudnative-pg/api/v1"
	pgbackrestapi "github.com/dalibo/cnpg-i-pgbackrest/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var sc = runtime.NewScheme()

func init() {
	_ = cnpgv1.AddToScheme(sc)
}

func newFakeClient(cnpgBackups []cnpgv1.Backup) client.WithWatch {

	initObjs := make([]client.Object, len(cnpgBackups))
	for i := range cnpgBackups {
		initObjs[i] = &cnpgBackups[i]
	}

	fc := fake.NewClientBuilder().
		WithScheme(sc).
		WithStatusSubresource(&cnpgv1.Backup{}). // Ensure status is handled
		WithObjects(initObjs...).
		Build()

	return fc

}
func TestCleanOldCNPGBackups(t *testing.T) {

	clusterName := "test-cluster"
	namespace := "default"

	cluster := &cnpgv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      clusterName,
			Namespace: namespace,
		},
	}

	testCases := []struct {
		name       string
		realBackup []pgbackrestapi.BackupInfo
		cnpgBackup []cnpgv1.Backup
		wantLeft   int
	}{
		{
			name: "keep matching backup",
			realBackup: []pgbackrestapi.BackupInfo{
				{Label: "backup-1"},
			},
			cnpgBackup: []cnpgv1.Backup{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "cnpg-1",
						Namespace: namespace,
						Labels:    map[string]string{"cnpg.io/cluster": clusterName},
					},
					Status: cnpgv1.BackupStatus{BackupName: "backup-1"},
				},
			},
			wantLeft: 1,
		},
		{
			name: "keep matching backups, but remove other",
			realBackup: []pgbackrestapi.BackupInfo{
				{Label: "backup-1"},
			},
			cnpgBackup: []cnpgv1.Backup{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "cnpg-1",
						Namespace: namespace,
						Labels:    map[string]string{"cnpg.io/cluster": clusterName},
					},
					Status: cnpgv1.BackupStatus{BackupName: "backup-1"},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "cnpg-2",
						Namespace: namespace,
						Labels:    map[string]string{"cnpg.io/cluster": clusterName},
					},
					Status: cnpgv1.BackupStatus{BackupName: "backup-other"},
				},
			},
			wantLeft: 1,
		},
		{
			name: "delete orphaned backup",
			realBackup: []pgbackrestapi.BackupInfo{
				{Label: "backup-current"},
			},
			cnpgBackup: []cnpgv1.Backup{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "cnpg-old",
						Namespace: namespace,
						Labels:    map[string]string{"cnpg.io/cluster": clusterName},
					},
					Status: cnpgv1.BackupStatus{BackupName: "backup-old"},
				},
			},
			wantLeft: 0,
		},
		{
			name:       "do not touch backups from other clusters",
			realBackup: []pgbackrestapi.BackupInfo{},
			cnpgBackup: []cnpgv1.Backup{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "other-cluster-backup",
						Namespace: namespace,
						Labels:    map[string]string{"cnpg.io/cluster": "different-cluster"},
					},
					Status: cnpgv1.BackupStatus{BackupName: "backup-1"},
				},
			},
			wantLeft: 1,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {

			fc := newFakeClient(tt.cnpgBackup)

			runnable := &StanzaMaintenanceRunnable{
				Client:     fc,
				ClusterKey: types.NamespacedName{Name: clusterName, Namespace: namespace},
			}

			// Run function
			err := runnable.cleanOldCNPGBackups(context.Background(), tt.realBackup, cluster)
			if err != nil {
				t.Fatalf("cleanOldCNPGBackups() unexpected error: %v", err)
			}

			var remaining cnpgv1.BackupList
			if err := fc.List(context.Background(), &remaining); err != nil {
				t.Fatalf("failed to list backups after cleanup: %v", err)
			}

			if len(remaining.Items) != tt.wantLeft {
				t.Errorf(
					"expected %d backups to remain, but found %d",
					tt.wantLeft,
					len(remaining.Items),
				)
				for _, b := range remaining.Items {
					t.Logf(
						"remaining backup: %s (Status.BackupName: %s)",
						b.Name,
						b.Status.BackupName,
					)
				}
			}
		})
	}
}
