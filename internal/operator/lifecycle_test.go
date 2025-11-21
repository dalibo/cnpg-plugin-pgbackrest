// SPDX-FileCopyrightText: 2025 Dalibo <contact@dalibo.com>
//
// SPDX-License-Identifier: Apache-2.0

package operator

import (
	"reflect"
	"testing"

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
	srcPod := corev1.PodSpec{
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
