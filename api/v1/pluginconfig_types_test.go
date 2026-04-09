// SPDX-FileCopyrightText: 2026 Dalibo <contact@dalibo.com>
//
// SPDX-License-Identifier: Apache-2.0

package v1

import (
	"reflect"
	"testing"
)

func TestExporterConfig_ToArgs(t *testing.T) {
	testCases := []struct {
		name string
		conf ExporterConfig
		want []string
	}{
		{
			name: "empty struct",
			conf: ExporterConfig{},
			want: []string{},
		},
		{
			name: "zero interval returns empty args",
			conf: ExporterConfig{
				CollectInterval: 0,
			},
			want: []string{},
		},
		{
			name: "non-zero interval returns flag",
			conf: ExporterConfig{
				CollectInterval: 600,
			},
			want: []string{"--collect.interval=600"},
		},
		{
			name: "enabled does not affect args",
			conf: ExporterConfig{
				Enabled:         true,
				CollectInterval: 300,
			},
			want: []string{"--collect.interval=300"},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			args := tt.conf.ToArgs()

			if !reflect.DeepEqual(args, tt.want) {
				t.Fatalf("want %v, got %v", tt.want, args)
			}
		})
	}
}
