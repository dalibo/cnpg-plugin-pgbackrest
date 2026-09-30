// SPDX-FileCopyrightText: 2026 Dalibo <contact@dalibo.com>
//
// SPDX-License-Identifier: Apache-2.0
package instance

import (
	"bytes"
	"context"
	"strings"
	"testing"

	cnpgv1 "github.com/cloudnative-pg/cloudnative-pg/api/v1"
	cnpglog "github.com/cloudnative-pg/machinery/pkg/log"
	pgbackrestapi "github.com/dalibo/cnpg-i-pgbackrest/api/v1"
	"github.com/dalibo/cnpg-i-pgbackrest/internal/metadata"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var sc = runtime.NewScheme()

func init() {
	_ = cnpgv1.AddToScheme(sc)
	pgbackrestapi.AddKnownTypes(sc)
}

func newFakeClient(initObjs ...client.Object) client.WithWatch {
	fc := fake.NewClientBuilder().
		WithScheme(sc).
		WithStatusSubresource(&pgbackrestapi.Stanza{}).
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
			initObjs := make([]client.Object, len(tt.cnpgBackup))
			for i := range tt.cnpgBackup {
				initObjs[i] = &tt.cnpgBackup[i]
			}

			fc := newFakeClient(initObjs...)

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

func TestStanzaMaintenanceRunOnceDoesNotLogError(t *testing.T) {
	const (
		clusterName = "test-cluster"
		namespace   = "default"
		podName     = "test-cluster-1"
		stanzaName  = "stanza"
	)

	cluster := &cnpgv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      clusterName,
			Namespace: namespace,
		},
		Spec: cnpgv1.ClusterSpec{
			Plugins: []cnpgv1.PluginConfiguration{
				{
					Name:          metadata.PluginName,
					IsWALArchiver: ptr.To(true),
					Parameters: map[string]string{
						"stanzaRef": stanzaName,
					},
				},
			},
		},
		Status: cnpgv1.ClusterStatus{
			CurrentPrimary: podName,
		},
	}

	stanza := &pgbackrestapi.Stanza{
		ObjectMeta: metav1.ObjectMeta{
			Name:      stanzaName,
			Namespace: namespace,
		},
	}

	backup := &cnpgv1.Backup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cnpg-1",
			Namespace: namespace,
			Labels:    map[string]string{"cnpg.io/cluster": clusterName},
		},
		Status: cnpgv1.BackupStatus{BackupName: "backup-1"},
	}

	fc := newFakeClient(cluster, stanza, backup)

	runnable := &StanzaMaintenanceRunnable{
		Client:         fc,
		ClusterKey:     types.NamespacedName{Name: clusterName, Namespace: namespace},
		CurrentPodName: podName,
		getBackupsInfoFn: func(_ context.Context, _ *pgbackrestapi.Stanza) ([]pgbackrestapi.BackupInfo, error) {
			return []pgbackrestapi.BackupInfo{{Label: "backup-1", Type: "full"}}, nil
		},
	}

	var logs bytes.Buffer
	previousLogger := cnpglog.GetLogger().GetLogger()
	cnpglog.SetLogger(zap.New(zap.WriteTo(&logs), zap.UseDevMode(false)))
	defer cnpglog.SetLogger(previousLogger)

	ctx := cnpglog.IntoContext(context.Background(), cnpglog.GetLogger())

	runnable.runOnce(ctx)

	if strings.Contains(logs.String(), "stanza maintenance failed") {
		t.Fatalf("unexpected maintenance error log: %s", logs.String())
	}
	if strings.Contains(logs.String(), "\"level\":\"error\"") {
		t.Fatalf("unexpected error-level log during maintenance: %s", logs.String())
	}
}
