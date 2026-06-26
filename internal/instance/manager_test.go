// SPDX-FileCopyrightText: 2026 Dalibo <contact@dalibo.com>
//
// SPDX-License-Identifier: Apache-2.0
package instance

import (
	"context"
	"testing"

	"github.com/spf13/viper"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestGenerateScheme_RegistersMetaTypes(t *testing.T) {
	viper.Reset()
	group := "postgresql.cnpg.io"
	version := "v1"
	viper.Set("custom-cnpg-group", group)
	viper.Set("custom-cnpg-version", version)

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
			name: "CNPG Cluster Custom Resource Type",
			gvk:  schema.GroupVersionKind{Group: group, Version: version, Kind: "Cluster"},
		},
		{
			name: "metav1 GetOptions under CNPG GroupVersion",
			gvk:  schema.GroupVersionKind{Group: group, Version: version, Kind: "GetOptions"},
		},
		{
			name: "metav1 ListOptions under CNPG GroupVersion",
			gvk:  schema.GroupVersionKind{Group: group, Version: version, Kind: "ListOptions"},
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
