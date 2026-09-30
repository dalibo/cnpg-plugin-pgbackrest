// SPDX-FileCopyrightText: 2026 Dalibo <contact@dalibo.com>
//
// SPDX-License-Identifier: Apache-2.0
package instance

import (
	"context"
	"testing"

	cnpgv1 "github.com/cloudnative-pg/cloudnative-pg/api/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestGenerateScheme_RegistersCNPGTypes(t *testing.T) {
	ctx := context.Background()
	scheme := generateScheme(ctx)

	if scheme == nil {
		t.Fatal("expected scheme to not be nil")
	}

	testsCases := []struct {
		name string
		gvk  schema.GroupVersionKind
	}{
		{
			name: "CNPG Cluster Type",
			gvk:  cnpgv1.SchemeGroupVersion.WithKind("Cluster"),
		},
		{
			name: "CNPG Backup Type",
			gvk:  cnpgv1.SchemeGroupVersion.WithKind("Backup"),
		},
		{
			name: "CNPG BackupList Type",
			gvk:  cnpgv1.SchemeGroupVersion.WithKind("BackupList"),
		},
		{
			name: "CNPG ScheduledBackup Type",
			gvk:  cnpgv1.SchemeGroupVersion.WithKind("ScheduledBackup"),
		},
	}

	for _, tt := range testsCases {
		t.Run(tt.name, func(t *testing.T) {
			if !scheme.Recognizes(tt.gvk) {
				t.Errorf(
					"Scheme is missing registration for GVK: %v. Your fix failed to add it.",
					tt.gvk,
				)
			}
			_, err := scheme.New(tt.gvk)
			if err != nil {
				t.Errorf("Failed to instantiate type for GVK %v: %v", tt.gvk, err)
			}
		})
	}
}
