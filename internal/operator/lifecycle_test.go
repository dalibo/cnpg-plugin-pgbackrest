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

func TestInjectPluginVolumeMount(t *testing.T) {
	testCases := []struct {
		desc              string
		mainContainerName string
		spec              corev1.PodSpec
		wantVolume        bool
		wantMount         bool
	}{
		{
			desc:              "injects volume and mount into main container",
			mainContainerName: "postgres",
			spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name: "postgres",
					},
				},
			},
			wantVolume: true,
			wantMount:  true,
		},
		{
			desc:              "does not mount volume on non-main containers",
			mainContainerName: "postgres",
			spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name: "sidecar",
					},
					{
						Name: "postgres",
					},
				},
			},
			wantVolume: true,
			wantMount:  true,
		},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			injectPluginVolumeMount(&test.spec, test.mainContainerName)

			// Assert volume exists
			var foundVolume *corev1.Volume
			for i := range test.spec.Volumes {
				if test.spec.Volumes[i].Name == "plugins" {
					foundVolume = &test.spec.Volumes[i]
					break
				}
			}

			if test.wantVolume && foundVolume == nil {
				t.Fatalf("expected plugins volume to be injected")
			}

			// check volume mount is only on main container
			for _, c := range test.spec.Containers {
				hasMount := false
				for _, m := range c.VolumeMounts {
					if m.Name == "plugins" && m.MountPath == "/plugins" {
						hasMount = true
					}
				}

				if c.Name == test.mainContainerName && !hasMount {
					t.Errorf("expected plugins volume mount on main container %q", c.Name)
				}

				if c.Name != test.mainContainerName && hasMount {
					t.Errorf("did not expect plugins volume mount on non-main container %q", c.Name)
				}
			}
		})
	}
}

func TestEnsureVolume(t *testing.T) {
	testCases := []struct {
		desc       string
		existing   []corev1.Volume
		input      corev1.Volume
		wantLen    int
		wantVolume corev1.Volume
	}{
		{
			desc:     "add volume when not present",
			existing: []corev1.Volume{},
			input: corev1.Volume{
				Name: "plugins",
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{},
				},
			},
			wantLen: 1,
			wantVolume: corev1.Volume{
				Name: "plugins",
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{},
				},
			},
		},
		{
			desc: "update existing volume",
			existing: []corev1.Volume{
				{
					Name: "plugins",
					VolumeSource: corev1.VolumeSource{
						HostPath: &corev1.HostPathVolumeSource{
							Path: "/old",
						},
					},
				},
			},
			input: corev1.Volume{
				Name: "plugins",
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{},
				},
			},
			wantLen: 1,
			wantVolume: corev1.Volume{
				Name: "plugins",
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{},
				},
			},
		},
		{
			desc: "keep other volumes intact",
			existing: []corev1.Volume{
				{
					Name: "data",
				},
			},
			input: corev1.Volume{
				Name: "plugins",
			},
			wantLen: 2,
			wantVolume: corev1.Volume{
				Name: "plugins",
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			result := ensureVolume(test.existing, test.input)

			if len(result) != test.wantLen {
				t.Fatalf("expected %d volumes, got %d", test.wantLen, len(result))
			}

			var found *corev1.Volume
			for i := range result {
				if result[i].Name == test.input.Name {
					found = &result[i]
					break
				}
			}

			if found == nil {
				t.Fatalf("expected volume %q to be present", test.input.Name)
			}

			if !reflect.DeepEqual(*found, test.wantVolume) {
				t.Errorf("volume mismatch want: %v, got:  %v", test.wantVolume, *found)
			}
		})
	}
}

func TestAddVolumeMountsFromContainer(t *testing.T) {
	testCases := []struct {
		desc       string
		target     corev1.Container
		sourceName string
		containers []corev1.Container
		wantErr    bool
		wantMounts []corev1.VolumeMount
	}{
		{
			desc:       "add mounts from source container",
			target:     corev1.Container{Name: "target"},
			sourceName: "source",
			containers: []corev1.Container{
				{
					Name: "source",
					VolumeMounts: []corev1.VolumeMount{
						{Name: "vol1", MountPath: "/mnt/vol1"},
						{Name: "vol2", MountPath: "/mnt/vol2"},
					},
				},
			},
			wantErr: false,
			wantMounts: []corev1.VolumeMount{
				{Name: "vol1", MountPath: "/mnt/vol1"},
				{Name: "vol2", MountPath: "/mnt/vol2"},
			},
		},
		{
			desc:       "return error when source container not found",
			target:     corev1.Container{Name: "target"},
			sourceName: "nonexistent",
			containers: []corev1.Container{
				{Name: "other"},
			},
			wantErr:    true,
			wantMounts: nil,
		},
		{
			desc: "append mounts to existing ones",
			target: corev1.Container{
				Name: "target",
				VolumeMounts: []corev1.VolumeMount{
					{Name: "existing", MountPath: "/mnt/existing"},
				},
			},
			sourceName: "source",
			containers: []corev1.Container{
				{
					Name: "source",
					VolumeMounts: []corev1.VolumeMount{
						{Name: "new", MountPath: "/mnt/new"},
					},
				},
			},
			wantErr: false,
			wantMounts: []corev1.VolumeMount{
				{Name: "existing", MountPath: "/mnt/existing"},
				{Name: "new", MountPath: "/mnt/new"},
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.desc, func(t *testing.T) {
			err := addVolumeMountsFromContainer(&test.target, test.sourceName, test.containers)
			if test.wantErr && err == nil {
				t.Fatalf("expected error but got nil")
			}
			if !test.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !reflect.DeepEqual(test.target.VolumeMounts, test.wantMounts) {
				t.Errorf(
					"volume mounts mismatch want: %v, got: %v",
					test.wantMounts,
					test.target.VolumeMounts,
				)
			}
		})
	}
}
