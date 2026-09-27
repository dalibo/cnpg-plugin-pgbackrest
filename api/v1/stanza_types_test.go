// SPDX-FileCopyrightText: 2026 Dalibo <contact@dalibo.com>
//
// SPDX-License-Identifier: Apache-2.0

package v1

import "testing"

func TestStanzaConfigurationEffectiveProcessMax(t *testing.T) {
	conf := StanzaConfiguration{
		ProcessMax:            2,
		BackupProcessMax:      3,
		RestoreProcessMax:     4,
		ArchivePushProcessMax: 5,
		ArchiveGetProcessMax:  6,
	}

	testCases := []struct {
		name    string
		command string
		want    uint
	}{
		{name: "backup override", command: "backup", want: 3},
		{name: "restore override", command: "restore", want: 4},
		{name: "archive push override", command: "archive-push", want: 5},
		{name: "archive get override", command: "archive-get", want: 6},
		{name: "fallback to global", command: "info", want: 2},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := conf.EffectiveProcessMax(tc.command); got != tc.want {
				t.Fatalf("EffectiveProcessMax(%q) = %d, want %d", tc.command, got, tc.want)
			}
		})
	}
}
