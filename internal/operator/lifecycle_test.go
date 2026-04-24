// SPDX-FileCopyrightText: 2025 Dalibo <contact@dalibo.com>
//
// SPDX-License-Identifier: Apache-2.0

package operator

import (
	"reflect"
	"testing"

	cnpgv1 "github.com/cloudnative-pg/cloudnative-pg/api/v1"
	pluginv1 "github.com/dalibo/cnpg-i-pgbackrest/api/v1"
	corev1 "k8s.io/api/core/v1"
)

func envVarSliceToMap(envVars []corev1.EnvVar) map[string]string {
	// convert env vars slice to Map for comparaison
	envMap := make(map[string]string)
	for _, env := range envVars {
		envMap[env.Name] = env.Value
	}
	return envMap
}
func TestEnvFromContainer(t *testing.T) {
	srcPod := &corev1.PodSpec{
		Containers: []corev1.Container{
			{
				Name: "test-container-with-var",
				Env: []corev1.EnvVar{
					{Name: "V1", Value: "va1"},
					{Name: "V2", Value: "va2"},
					{Name: "V28", Value: "va99"},
				},
			},
			{
				Name: "test-container-without-shared-var",
				Env: []corev1.EnvVar{
					{Name: "V3ContainerOnly", Value: "va3"},
				},
			},
			{
				Name: "container-without-env",
			},
		},
	}
	testCases := []struct {
		desc          string
		containerName string
		srcData       []corev1.EnvVar
		want          []corev1.EnvVar
	}{
		{
			"merge non empty EnvVar on Pod",
			"test-container-with-var",
			[]corev1.EnvVar{
				{Name: "V1", Value: "va1"},
				{Name: "V28", Value: "Va128"}},
			[]corev1.EnvVar{
				{Name: "V2", Value: "va2"}, // this should be added from the container
			},
		},
		{
			"merge non empty (but no common) EnvVar on Pod",
			"test-container-without-shared-var",
			[]corev1.EnvVar{
				{Name: "V17", Value: "va19"},
				{Name: "V29", Value: "va29"},
			},
			[]corev1.EnvVar{
				{
					Name:  "V3ContainerOnly",
					Value: "va3",
				}, // this should be addeded (does not exist on srcData, but exists on container def)
			},
		},
		{
			"merge empty EnvVar on Pod",
			"test-container-without-env",
			[]corev1.EnvVar{ // no EnvVar on container def, all items are from data source
				{Name: "V1", Value: "va1"},
				{Name: "V2", Value: "va2"},
				{Name: "V28", Value: "Va128"}},
			[]corev1.EnvVar{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			r := envFromContainer(tc.containerName, srcPod, tc.srcData)
			if !reflect.DeepEqual(envVarSliceToMap(r), envVarSliceToMap(tc.want)) {
				t.Errorf("Expected %v, but got %v", envVarSliceToMap(tc.want), envVarSliceToMap(r))
			}
		})
	}
}

func TestGetSpoolWALSize(t *testing.T) {
	tests := []struct {
		name           string
		walStor        *cnpgv1.StorageConfiguration
		pluginStorConf *pluginv1.StorageConfig
		expected       string
	}{
		{
			name: "plugin size takes precedence over wal size",
			walStor: &cnpgv1.StorageConfiguration{
				Size: "2Gi",
			},
			pluginStorConf: &pluginv1.StorageConfig{
				Size: "5Gi",
			},
			expected: "5Gi",
		},
		{
			name:    "plugin size used when wal is nil",
			walStor: nil,
			pluginStorConf: &pluginv1.StorageConfig{
				Size: "3Gi",
			},
			expected: "3Gi",
		},
		{
			name: "wal size used when plugin is nil",
			walStor: &cnpgv1.StorageConfiguration{
				Size: "4Gi",
			},
			pluginStorConf: nil,
			expected:       "4Gi",
		},
		{
			name: "wal size used when plugin size is empty",
			walStor: &cnpgv1.StorageConfiguration{
				Size: "6Gi",
			},
			pluginStorConf: &pluginv1.StorageConfig{},
			expected:       "6Gi",
		},
		{
			name:           "default used when both are nil",
			walStor:        nil,
			pluginStorConf: nil,
			expected:       "1Gi",
		},
		{
			name:           "default used when both sizes are empty",
			walStor:        &cnpgv1.StorageConfiguration{},
			pluginStorConf: &pluginv1.StorageConfig{},
			expected:       "1Gi",
		},
		{
			name:           "default used when plugin empty and wal nil",
			walStor:        nil,
			pluginStorConf: &pluginv1.StorageConfig{},
			expected:       "1Gi",
		},
		{
			name:           "default used when wal empty and plugin nil",
			walStor:        &cnpgv1.StorageConfiguration{},
			pluginStorConf: nil,
			expected:       "1Gi",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getSpoolWALSize(tt.walStor, tt.pluginStorConf)
			if result != tt.expected {
				t.Fatalf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}
